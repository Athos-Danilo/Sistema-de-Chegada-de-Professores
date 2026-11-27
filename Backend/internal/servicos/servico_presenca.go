// Pacote servicos concentra as regras de negócio do Sistema de Chegada de Professores.
// Orquestra a autenticação de dispositivos, alternância de entrada/saída, consulta à grade,
// prevenção de alertas atrasados em sincronização off-line e disparo para o Telegram.
package servicos

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"scp-backend/internal/banco"
	"scp-backend/internal/modelos"
	"scp-backend/internal/telegram"
)

// Erros de domínio tratados pela camada HTTP para devolver códigos adequados.
var (
	ErrTokenInvalido     = errors.New("token de autenticação do dispositivo inválido")
	ErrDadosInvalidos    = errors.New("campos obrigatórios ausentes ou inválidos")
	ErrTagNaoCadastrada  = errors.New("tag RFID não cadastrada para nenhum professor ativo")
)

// ServicoPresenca coordena o fluxo completo de leitura, persistência e notificação.
type ServicoPresenca struct {
	repositorio banco.RepositorioPresenca
	notificador telegram.NotificadorTelegram
	fusoHorario *time.Location
	chatPadrao  string
	relogio     func() time.Time
}

// NovoServicoPresenca cria o serviço com suas dependências injetadas.
func NovoServicoPresenca(
	repositorio banco.RepositorioPresenca,
	notificador telegram.NotificadorTelegram,
	fusoHorario *time.Location,
	chatPadrao string,
) *ServicoPresenca {
	if fusoHorario == nil {
		fusoHorario = time.FixedZone("BRT", -3*60*60)
	}
	return &ServicoPresenca{
		repositorio: repositorio,
		notificador: notificador,
		fusoHorario: fusoHorario,
		chatPadrao:  strings.TrimSpace(chatPadrao),
		relogio:     time.Now,
	}
}

// AutenticarDispositivo valida se o dispositivo existe e se o cabeçalho X-Device-Token confere.
func (s *ServicoPresenca) AutenticarDispositivo(
	ctx context.Context,
	idDispositivo string,
	tokenRecebido string,
) (*modelos.Dispositivo, error) {
	idLimpo := strings.TrimSpace(idDispositivo)
	tokenLimpo := strings.TrimSpace(tokenRecebido)
	if idLimpo == "" || tokenLimpo == "" {
		return nil, ErrTokenInvalido
	}

	dispositivo, erro := s.repositorio.BuscarDispositivo(ctx, idLimpo)
	if erro != nil {
		if errors.Is(erro, banco.ErrDispositivoNaoEncontrado) {
			return nil, ErrTokenInvalido
		}
		return nil, erro
	}

	tokenIgual := subtle.ConstantTimeCompare([]byte(dispositivo.TokenAutenticacao), []byte(tokenLimpo)) == 1
	if !tokenIgual {
		return nil, ErrTokenInvalido
	}

	return dispositivo, nil
}

// RegistrarLeitura processa uma leitura individual enviada pelo ESP32 e envia aviso no Telegram.
func (s *ServicoPresenca) RegistrarLeitura(
	ctx context.Context,
	tokenRecebido string,
	req modelos.RequisicaoPresenca,
) (*modelos.ResultadoPresenca, error) {
	dispositivo, erro := s.AutenticarDispositivo(ctx, req.IdDispositivo, tokenRecebido)
	if erro != nil {
		return nil, erro
	}

	uidNormalizado := banco.NormalizarUidTag(req.UidTag)
	if uidNormalizado == "" {
		return nil, ErrDadosInvalidos
	}

	professor, erro := s.repositorio.BuscarProfessorPorTag(ctx, uidNormalizado)
	if erro != nil {
		if errors.Is(erro, banco.ErrProfessorNaoEncontrado) {
			return nil, ErrTagNaoCadastrada
		}
		return nil, erro
	}

	leituraEm := s.converterCarimboTempo(req.CarimboTempoLeit)
	leituraLocal := leituraEm.In(s.fusoHorario)

	tipoEvento, erro := s.determinarTipoEvento(ctx, professor.Id, dispositivo.Id, leituraLocal)
	if erro != nil {
		return nil, erro
	}

	aula, erro := s.buscarAulaDaLeitura(ctx, professor.Id, dispositivo.Id, leituraLocal)
	if erro != nil {
		return nil, erro
	}

	var idTurma *int
	chatDestino := s.chatPadrao
	nomeTurma := ""
	disciplina := ""
	if aula != nil {
		idTurma = &aula.IdTurma
		if strings.TrimSpace(aula.ChatIdTelegram) != "" {
			chatDestino = aula.ChatIdTelegram
		}
		nomeTurma = aula.NomeTurma
		disciplina = aula.Disciplina
	}

	notificacaoEnviada := false
	deveNotificar := s.deveNotificarTelegram(req.RegistroOffline, aula, leituraLocal)
	if deveNotificar && chatDestino != "" {
		idMensagem, erroEnvio := s.notificador.EnviarAlertaPresenca(ctx, telegram.DadosNotificacao{
			ChatId:     chatDestino,
			Professor:  professor.Nome,
			Sala:       dispositivo.NomeSala,
			Bloco:      dispositivo.Bloco,
			Disciplina: disciplina,
			NomeTurma:  nomeTurma,
			TipoEvento: tipoEvento,
			Data:       leituraLocal.Format("02/01/2006"),
			Horario:    leituraLocal.Format("15:04:05"),
		})
		if erroEnvio != nil {
			// A falha no Telegram não deve impedir o salvamento do ponto no PostgreSQL.
			log.Printf("[AVISO] Falha ao enviar notificação no Telegram: %v", erroEnvio)
		} else {
			notificacaoEnviada = true
			if idMensagem > 0 {
				if erroFila := s.repositorio.RegistrarMensagemTelegram(ctx, chatDestino, idMensagem); erroFila != nil {
					log.Printf("[AVISO] Falha ao registrar message_id para limpeza noturna: %v", erroFila)
				}
			}
		}
	}

	idRegistro, erro := s.repositorio.RegistrarPresenca(ctx, &modelos.RegistroPresenca{
		IdProfessor:        professor.Id,
		IdDispositivo:      dispositivo.Id,
		IdTurma:            idTurma,
		TipoEvento:         tipoEvento,
		LeituraEm:          leituraEm,
		RegistroOffline:    req.RegistroOffline,
		NotificacaoEnviada: notificacaoEnviada,
	})
	if erro != nil {
		return nil, erro
	}

	return &modelos.ResultadoPresenca{
		IdRegistro:         idRegistro,
		Professor:          professor.Nome,
		Sala:               dispositivo.NomeSala,
		TipoEvento:         tipoEvento,
		Horario:            leituraLocal.Format("02/01/2006 15:04:05"),
		NotificacaoEnviada: notificacaoEnviada,
	}, nil
}

// ProcessarLoteOffline processa em ordem cronológica os registros acumulados sem internet.
func (s *ServicoPresenca) ProcessarLoteOffline(
	ctx context.Context,
	tokenRecebido string,
	lote modelos.RequisicaoLotePresenca,
) ([]modelos.ResultadoPresenca, error) {
	if _, erro := s.AutenticarDispositivo(ctx, lote.IdDispositivo, tokenRecebido); erro != nil {
		return nil, erro
	}

	if len(lote.Registros) == 0 {
		return nil, ErrDadosInvalidos
	}

	resultados := make([]modelos.ResultadoPresenca, 0, len(lote.Registros))
	for _, item := range lote.Registros {
		resultado, erro := s.RegistrarLeitura(ctx, tokenRecebido, modelos.RequisicaoPresenca{
			IdDispositivo:    lote.IdDispositivo,
			UidTag:           item.UidTag,
			CarimboTempoLeit: item.CarimboTempoLeit,
			RegistroOffline:  true,
		})
		if erro != nil {
			if errors.Is(erro, ErrTagNaoCadastrada) || errors.Is(erro, ErrDadosInvalidos) {
				log.Printf("[AVISO] Item inválido ignorado no lote off-line (tag=%s): %v", item.UidTag, erro)
				continue
			}
			return nil, erro
		}
		resultados = append(resultados, *resultado)
	}

	return resultados, nil
}

// RegistrarTelemetria valida o dispositivo e persiste seu sinal de vida (heartbeat).
func (s *ServicoPresenca) RegistrarTelemetria(
	ctx context.Context,
	tokenRecebido string,
	dados modelos.RequisicaoTelemetria,
) error {
	if _, erro := s.AutenticarDispositivo(ctx, dados.IdDispositivo, tokenRecebido); erro != nil {
		return erro
	}

	if strings.TrimSpace(dados.VersaoFirmware) == "" {
		return ErrDadosInvalidos
	}

	return s.repositorio.RegistrarTelemetria(ctx, dados)
}

// LimparMensagensTelegram apaga no Telegram as mensagens enviadas pelo bot e limpa a fila auxiliar,
// mantendo 100% do histórico oficial de presenças preservado na tabela registros_presenca.
func (s *ServicoPresenca) LimparMensagensTelegram(ctx context.Context) (*modelos.ResultadoLimpezaTelegram, error) {
	mensagens, erro := s.repositorio.ListarMensagensTelegramParaLimpar(ctx)
	if erro != nil {
		return nil, erro
	}

	resultado := &modelos.ResultadoLimpezaTelegram{
		MensagensEncontradas: len(mensagens),
	}
	if len(mensagens) == 0 {
		return resultado, nil
	}

	type grupoChat struct {
		idsTelegram []int64
		maiorId     int64
	}
	porChat := make(map[string]*grupoChat)
	ordemChats := make([]string, 0)
	agora := s.relogio()
	const limiteTelegram = 47*time.Hour + 50*time.Minute

	for _, msg := range mensagens {
		grupo, existe := porChat[msg.ChatId]
		if !existe {
			grupo = &grupoChat{}
			porChat[msg.ChatId] = grupo
			ordemChats = append(ordemChats, msg.ChatId)
		}
		if msg.Id > grupo.maiorId {
			grupo.maiorId = msg.Id
		}
		if agora.Sub(msg.EnviadaEm) < limiteTelegram {
			grupo.idsTelegram = append(grupo.idsTelegram, msg.MessageId)
		}
	}

	for _, chatId := range ordemChats {
		grupo := porChat[chatId]
		if len(grupo.idsTelegram) > 0 {
			apagadas, erroExclusao := s.notificador.ApagarMensagens(ctx, chatId, grupo.idsTelegram)
			resultado.MensagensApagadas += apagadas
			if erroExclusao != nil {
				return resultado, erroExclusao
			}
		}

		removidos, erroBanco := s.repositorio.RemoverMensagensTelegramDoChat(ctx, chatId, grupo.maiorId)
		if erroBanco != nil {
			return resultado, erroBanco
		}
		resultado.RegistrosRemovidos += removidos
	}

	return resultado, nil
}

func (s *ServicoPresenca) converterCarimboTempo(carimboUnix int64) time.Time {
	if carimboUnix <= 0 {
		return s.relogio().UTC()
	}
	return time.Unix(carimboUnix, 0).UTC()
}

func (s *ServicoPresenca) determinarTipoEvento(
	ctx context.Context,
	idProfessor int,
	idDispositivo string,
	leituraLocal time.Time,
) (string, error) {
	inicioDia := time.Date(leituraLocal.Year(), leituraLocal.Month(), leituraLocal.Day(), 0, 0, 0, 0, s.fusoHorario)
	fimDia := inicioDia.Add(24 * time.Hour)

	ultimoEvento, erro := s.repositorio.BuscarUltimoEventoDoDia(ctx, idProfessor, idDispositivo, inicioDia.UTC(), fimDia.UTC())
	if erro != nil {
		return "", erro
	}

	if ultimoEvento == modelos.EventoEntrada {
		return modelos.EventoSaida, nil
	}
	return modelos.EventoEntrada, nil
}

func (s *ServicoPresenca) buscarAulaDaLeitura(
	ctx context.Context,
	idProfessor int,
	idDispositivo string,
	leituraLocal time.Time,
) (*modelos.AulaAgendada, error) {
	diaSemana := int(leituraLocal.Weekday())
	horarioTexto := leituraLocal.Format("15:04:05")
	return s.repositorio.BuscarAulaPorHorario(ctx, idProfessor, idDispositivo, diaSemana, horarioTexto)
}

// deveNotificarTelegram descarta alertas de sincronização off-line se a aula já terminou.
func (s *ServicoPresenca) deveNotificarTelegram(
	registroOffline bool,
	aula *modelos.AulaAgendada,
	leituraLocal time.Time,
) bool {
	if !registroOffline {
		return true
	}

	agoraLocal := s.relogio().In(s.fusoHorario)
	if aula == nil {
		// Sem aula vinculada, evita alertas atrasados de outro dia/horário antigo.
		return agoraLocal.Sub(leituraLocal) <= 15*time.Minute
	}

	horarioFim, erro := time.Parse("15:04", aula.HorarioFim)
	if erro != nil {
		return false
	}

	fimAula := time.Date(
		leituraLocal.Year(), leituraLocal.Month(), leituraLocal.Day(),
		horarioFim.Hour(), horarioFim.Minute(), 0, 0, s.fusoHorario,
	)
	return !agoraLocal.After(fimAula)
}

// DefinirRelogio permite substituir o relógio em testes unitários automatizados.
func (s *ServicoPresenca) DefinirRelogio(fn func() time.Time) {
	if fn != nil {
		s.relogio = fn
	}
}

// FormatarResumoLote gera uma mensagem resumida para retorno HTTP da sincronização.
func FormatarResumoLote(total int) string {
	return fmt.Sprintf("Lote sincronizado com sucesso (%d registros gravados)", total)
}
