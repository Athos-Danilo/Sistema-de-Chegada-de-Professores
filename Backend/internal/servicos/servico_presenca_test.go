// Testes unitários da camada de regras de negócio (ServicoPresenca).
// Utiliza dublês em memória para o repositório PostgreSQL e o cliente do Telegram,
// validando entrada, saída, autenticação, tags inválidas e alertas off-line atrasados.
package servicos

import (
	"context"
	"errors"
	"testing"
	"time"

	"scp-backend/internal/banco"
	"scp-backend/internal/modelos"
	"scp-backend/internal/telegram"
)

type repositorioMemoria struct {
	dispositivo  *modelos.Dispositivo
	professor    *modelos.Professor
	ultimoEvento string
	aula         *modelos.AulaAgendada
	registros    []modelos.RegistroPresenca
	mensagens    []modelos.MensagemTelegramRegistrada
}

func (m *repositorioMemoria) BuscarDispositivo(_ context.Context, idDispositivo string) (*modelos.Dispositivo, error) {
	if m.dispositivo == nil || m.dispositivo.Id != idDispositivo {
		return nil, banco.ErrDispositivoNaoEncontrado
	}
	return m.dispositivo, nil
}

func (m *repositorioMemoria) BuscarProfessorPorTag(_ context.Context, uidNormalizado string) (*modelos.Professor, error) {
	if m.professor == nil || banco.NormalizarUidTag(m.professor.UidTag) != uidNormalizado {
		return nil, banco.ErrProfessorNaoEncontrado
	}
	return m.professor, nil
}

func (m *repositorioMemoria) BuscarUltimoEventoDoDia(_ context.Context, _ int, _ string, _, _ time.Time) (string, error) {
	return m.ultimoEvento, nil
}

func (m *repositorioMemoria) BuscarAulaPorHorario(_ context.Context, _ int, _ string, _ int, _ string) (*modelos.AulaAgendada, error) {
	return m.aula, nil
}

func (m *repositorioMemoria) RegistrarPresenca(_ context.Context, reg *modelos.RegistroPresenca) (int64, error) {
	m.registros = append(m.registros, *reg)
	m.ultimoEvento = reg.TipoEvento
	return int64(len(m.registros)), nil
}

func (m *repositorioMemoria) RegistrarTelemetria(_ context.Context, _ modelos.RequisicaoTelemetria) error {
	return nil
}

func (m *repositorioMemoria) RegistrarMensagemTelegram(_ context.Context, chatId string, messageId int64) error {
	m.mensagens = append(m.mensagens, modelos.MensagemTelegramRegistrada{
		Id:        int64(len(m.mensagens) + 1),
		ChatId:    chatId,
		MessageId: messageId,
		EnviadaEm: time.Now(),
	})
	return nil
}

func (m *repositorioMemoria) ListarMensagensTelegramParaLimpar(_ context.Context) ([]modelos.MensagemTelegramRegistrada, error) {
	copia := make([]modelos.MensagemTelegramRegistrada, len(m.mensagens))
	copy(copia, m.mensagens)
	return copia, nil
}

func (m *repositorioMemoria) RemoverMensagensTelegramDoChat(_ context.Context, chatId string, ateId int64) (int, error) {
	restantes := make([]modelos.MensagemTelegramRegistrada, 0, len(m.mensagens))
	removidos := 0
	for _, msg := range m.mensagens {
		if msg.ChatId == chatId && msg.Id <= ateId {
			removidos++
			continue
		}
		restantes = append(restantes, msg)
	}
	m.mensagens = restantes
	return removidos, nil
}

type notificadorFalso struct {
	enviados       []telegram.DadosNotificacao
	apagadasPorChat map[string][]int64
}

func (n *notificadorFalso) EnviarAlertaPresenca(_ context.Context, dados telegram.DadosNotificacao) (int64, error) {
	n.enviados = append(n.enviados, dados)
	return int64(100 + len(n.enviados)), nil
}

func (n *notificadorFalso) ApagarMensagens(_ context.Context, chatId string, idsMensagens []int64) (int, error) {
	if n.apagadasPorChat == nil {
		n.apagadasPorChat = make(map[string][]int64)
	}
	n.apagadasPorChat[chatId] = append(n.apagadasPorChat[chatId], idsMensagens...)
	return len(idsMensagens), nil
}

func TestRegistrarLeituraAlternanciaEntradaSaidaENotificacao(t *testing.T) {
	fuso := time.FixedZone("BRT", -3*3600)
	agora := time.Date(2026, 11, 26, 19, 10, 0, 0, fuso)

	repo := &repositorioMemoria{
		dispositivo: &modelos.Dispositivo{
			Id:                "EMB-LAB-01",
			NomeSala:          "Laboratório 01",
			Bloco:             "Bloco A",
			TokenAutenticacao: "segredo-super-seguro",
			Ativo:             true,
		},
		professor: &modelos.Professor{
			Id:     1,
			Nome:   "Prof. Lauro Alves",
			UidTag: "15:35:D6:D6",
			Ativo:  true,
		},
		aula: &modelos.AulaAgendada{
			IdTurma:        10,
			NomeTurma:      "ADS 4º Período",
			ChatIdTelegram: "-100999888",
			Disciplina:     "Sistemas Embarcados",
			HorarioInicio:  "19:00",
			HorarioFim:     "22:00",
		},
	}
	notificador := &notificadorFalso{}
	servico := NovoServicoPresenca(repo, notificador, fuso, "-100000")
	servico.DefinirRelogio(func() time.Time { return agora })

	// Primeira leitura deve registrar ENTRADA e notificar no Telegram.
	res1, erro := servico.RegistrarLeitura(context.Background(), "segredo-super-seguro", modelos.RequisicaoPresenca{
		IdDispositivo:    "EMB-LAB-01",
		UidTag:           "1535d6d6",
		CarimboTempoLeit: agora.Unix(),
	})
	if erro != nil {
		t.Fatalf("erro inesperado na entrada: %v", erro)
	}
	if res1.TipoEvento != modelos.EventoEntrada || !res1.NotificacaoEnviada {
		t.Fatalf("esperado ENTRADA com notificação, obtido %+v", res1)
	}

	// Segunda leitura no mesmo dia deve alternar para SAIDA.
	res2, erro := servico.RegistrarLeitura(context.Background(), "segredo-super-seguro", modelos.RequisicaoPresenca{
		IdDispositivo:    "EMB-LAB-01",
		UidTag:           "15:35:D6:D6",
		CarimboTempoLeit: agora.Add(2 * time.Hour).Unix(),
	})
	if erro != nil {
		t.Fatalf("erro inesperado na saída: %v", erro)
	}
	if res2.TipoEvento != modelos.EventoSaida {
		t.Fatalf("esperado SAIDA, obtido %s", res2.TipoEvento)
	}
	if len(notificador.enviados) != 2 {
		t.Fatalf("esperadas 2 notificações enviadas, obtido %d", len(notificador.enviados))
	}
}

func TestRegistroOfflineAtrasadoNaoEnviaTelegram(t *testing.T) {
	fuso := time.FixedZone("BRT", -3*3600)
	leituraPassada := time.Date(2026, 11, 26, 19, 15, 0, 0, fuso)
	agoraDepoisDaAula := time.Date(2026, 11, 26, 22, 30, 0, 0, fuso)

	repo := &repositorioMemoria{
		dispositivo: &modelos.Dispositivo{Id: "EMB-LAB-01", NomeSala: "Lab 01", TokenAutenticacao: "tok", Ativo: true},
		professor:   &modelos.Professor{Id: 1, Nome: "Prof. Lauro", UidTag: "AABBCCDD", Ativo: true},
		aula: &modelos.AulaAgendada{
			IdTurma: 1, NomeTurma: "ADS", ChatIdTelegram: "-100123",
			Disciplina: "Embarcados", HorarioInicio: "19:00", HorarioFim: "22:00",
		},
	}
	notificador := &notificadorFalso{}
	servico := NovoServicoPresenca(repo, notificador, fuso, "")
	servico.DefinirRelogio(func() time.Time { return agoraDepoisDaAula })

	res, erro := servico.RegistrarLeitura(context.Background(), "tok", modelos.RequisicaoPresenca{
		IdDispositivo:    "EMB-LAB-01",
		UidTag:           "AABBCCDD",
		CarimboTempoLeit: leituraPassada.Unix(),
		RegistroOffline:  true,
	})
	if erro != nil {
		t.Fatalf("erro inesperado: %v", erro)
	}
	if res.NotificacaoEnviada || len(notificador.enviados) != 0 {
		t.Fatalf("registro off-line após término da aula não deveria enviar alerta no Telegram")
	}
	if len(repo.registros) != 1 {
		t.Fatalf("registro histórico deveria permanecer salvo no banco")
	}
}

func TestAutenticacaoTokenInvalido(t *testing.T) {
	repo := &repositorioMemoria{
		dispositivo: &modelos.Dispositivo{Id: "EMB-LAB-01", TokenAutenticacao: "correto", Ativo: true},
	}
	servico := NovoServicoPresenca(repo, &notificadorFalso{}, time.UTC, "")

	_, erro := servico.RegistrarLeitura(context.Background(), "errado", modelos.RequisicaoPresenca{
		IdDispositivo: "EMB-LAB-01",
		UidTag:        "11223344",
	})
	if !errors.Is(erro, ErrTokenInvalido) {
		t.Fatalf("esperado ErrTokenInvalido, obtido %v", erro)
	}
}

func TestLimparMensagensTelegramPreservaHistoricoDoBanco(t *testing.T) {
	fuso := time.FixedZone("BRT", -3*3600)
	agora := time.Date(2026, 11, 26, 20, 0, 0, 0, fuso)

	repo := &repositorioMemoria{
		dispositivo: &modelos.Dispositivo{Id: "EMB-LAB-01", NomeSala: "Lab 01", TokenAutenticacao: "tok", Ativo: true},
		professor:   &modelos.Professor{Id: 1, Nome: "Prof. Paulo Pérris", UidTag: "04:51:2D:71:CF:2A:81", Ativo: true},
	}
	notificador := &notificadorFalso{}
	servico := NovoServicoPresenca(repo, notificador, fuso, "-1005901458470")
	servico.DefinirRelogio(func() time.Time { return agora })

	_, erro := servico.RegistrarLeitura(context.Background(), "tok", modelos.RequisicaoPresenca{
		IdDispositivo:    "EMB-LAB-01",
		UidTag:           "04:51:2D:71:CF:2A:81",
		CarimboTempoLeit: agora.Unix(),
	})
	if erro != nil {
		t.Fatalf("erro inesperado ao registrar leitura: %v", erro)
	}
	if len(repo.mensagens) != 1 {
		t.Fatalf("esperado 1 message_id registrado na fila auxiliar, obtido %d", len(repo.mensagens))
	}

	repo.mensagens[0].EnviadaEm = agora.Add(-2 * time.Hour)
	resLimpeza, erro := servico.LimparMensagensTelegram(context.Background())
	if erro != nil {
		t.Fatalf("erro inesperado ao limpar mensagens: %v", erro)
	}
	if resLimpeza.MensagensApagadas != 1 || resLimpeza.RegistrosRemovidos != 1 {
		t.Fatalf("resumo de limpeza inesperado: %+v", resLimpeza)
	}
	if len(repo.mensagens) != 0 {
		t.Fatalf("fila auxiliar deveria estar vazia após limpeza")
	}
	if len(repo.registros) != 1 {
		t.Fatalf("histórico oficial de presenças deveria permanecer intacto (obtido %d)", len(repo.registros))
	}
}
