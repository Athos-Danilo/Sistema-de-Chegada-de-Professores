// Este arquivo implementa o padrão Repositório para acesso ao PostgreSQL na Neon.
// Contém a interface RepositorioPresenca e todas as consultas SQL necessárias para
// autenticar dispositivos, localizar professores, cruzar grades e salvar registros.
package banco

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"scp-backend/internal/modelos"
)

// Erros padronizados retornados pela camada de persistência.
var (
	ErrDispositivoNaoEncontrado = errors.New("dispositivo não encontrado ou inativo")
	ErrProfessorNaoEncontrado   = errors.New("tag RFID não vinculada a nenhum professor ativo")
)

// RepositorioPresenca define o contrato de acesso a dados exigido pelos serviços.
type RepositorioPresenca interface {
	BuscarDispositivo(ctx context.Context, idDispositivo string) (*modelos.Dispositivo, error)
	BuscarProfessorPorTag(ctx context.Context, uidNormalizado string) (*modelos.Professor, error)
	BuscarUltimoEventoDoDia(ctx context.Context, idProfessor int, idDispositivo string, inicioDia time.Time, fimDia time.Time) (string, error)
	BuscarAulaPorHorario(ctx context.Context, idProfessor int, idDispositivo string, diaSemana int, horarioLocal string) (*modelos.AulaAgendada, error)
	RegistrarPresenca(ctx context.Context, registro *modelos.RegistroPresenca) (int64, error)
	RegistrarTelemetria(ctx context.Context, dados modelos.RequisicaoTelemetria) error
	RegistrarMensagemTelegram(ctx context.Context, chatId string, messageId int64) error
	ListarMensagensTelegramParaLimpar(ctx context.Context) ([]modelos.MensagemTelegramRegistrada, error)
	RemoverMensagensTelegramDoChat(ctx context.Context, chatId string, ateId int64) (int, error)
}

// RepositorioPostgres implementa RepositorioPresenca utilizando pgxpool.
type RepositorioPostgres struct {
	pool *pgxpool.Pool
}

// NovoRepositorioPostgres cria uma nova instância do repositório PostgreSQL.
func NovoRepositorioPostgres(pool *pgxpool.Pool) *RepositorioPostgres {
	return &RepositorioPostgres{pool: pool}
}

// NormalizarUidTag padroniza a tag removendo dois-pontos/espaços e convertendo para maiúsculas.
func NormalizarUidTag(uid string) string {
	limpo := strings.TrimSpace(uid)
	limpo = strings.ReplaceAll(limpo, ":", "")
	limpo = strings.ReplaceAll(limpo, "-", "")
	limpo = strings.ReplaceAll(limpo, " ", "")
	return strings.ToUpper(limpo)
}

// BuscarDispositivo consulta um leitor ESP32 cadastrado pelo seu identificador.
func (r *RepositorioPostgres) BuscarDispositivo(ctx context.Context, idDispositivo string) (*modelos.Dispositivo, error) {
	const consulta = `
		SELECT id, nome_sala, bloco, token_autenticacao, ativo
		FROM dispositivos
		WHERE id = $1 AND ativo = TRUE
	`

	var disp modelos.Dispositivo
	erro := r.pool.QueryRow(ctx, consulta, idDispositivo).Scan(
		&disp.Id,
		&disp.NomeSala,
		&disp.Bloco,
		&disp.TokenAutenticacao,
		&disp.Ativo,
	)
	if errors.Is(erro, pgx.ErrNoRows) {
		return nil, ErrDispositivoNaoEncontrado
	}
	if erro != nil {
		return nil, fmt.Errorf("erro ao buscar dispositivo: %w", erro)
	}

	return &disp, nil
}

// BuscarProfessorPorTag localiza o docente ativo dono da tag RFID informada.
func (r *RepositorioPostgres) BuscarProfessorPorTag(ctx context.Context, uidNormalizado string) (*modelos.Professor, error) {
	const consulta = `
		SELECT id, nome, tag_uid, ativo
		FROM professores
		WHERE REPLACE(REPLACE(UPPER(TRIM(tag_uid)), ':', ''), '-', '') = $1
		  AND ativo = TRUE
	`

	var prof modelos.Professor
	erro := r.pool.QueryRow(ctx, consulta, uidNormalizado).Scan(
		&prof.Id,
		&prof.Nome,
		&prof.UidTag,
		&prof.Ativo,
	)
	if errors.Is(erro, pgx.ErrNoRows) {
		return nil, ErrProfessorNaoEncontrado
	}
	if erro != nil {
		return nil, fmt.Errorf("erro ao buscar professor pela tag: %w", erro)
	}

	return &prof, nil
}

// BuscarUltimoEventoDoDia retorna o último tipo_evento ('ENTRADA' ou 'SAIDA') registrado no dia.
func (r *RepositorioPostgres) BuscarUltimoEventoDoDia(
	ctx context.Context,
	idProfessor int,
	idDispositivo string,
	inicioDia time.Time,
	fimDia time.Time,
) (string, error) {
	const consulta = `
		SELECT tipo_evento
		FROM registros_presenca
		WHERE professor_id = $1
		  AND dispositivo_id = $2
		  AND leitura_em >= $3
		  AND leitura_em < $4
		ORDER BY leitura_em DESC, id DESC
		LIMIT 1
	`

	var ultimoEvento string
	erro := r.pool.QueryRow(ctx, consulta, idProfessor, idDispositivo, inicioDia, fimDia).Scan(&ultimoEvento)
	if errors.Is(erro, pgx.ErrNoRows) {
		return "", nil
	}
	if erro != nil {
		return "", fmt.Errorf("erro ao consultar último evento do dia: %w", erro)
	}

	return ultimoEvento, nil
}

// BuscarAulaPorHorario verifica se há aula cadastrada na grade para o professor, sala e horário.
func (r *RepositorioPostgres) BuscarAulaPorHorario(
	ctx context.Context,
	idProfessor int,
	idDispositivo string,
	diaSemana int,
	horarioLocal string,
) (*modelos.AulaAgendada, error) {
	const consulta = `
		SELECT t.id, t.nome, t.telegram_chat_id, g.disciplina,
		       TO_CHAR(g.horario_inicio, 'HH24:MI'),
		       TO_CHAR(g.horario_fim, 'HH24:MI')
		FROM grades_horarias g
		INNER JOIN turmas t ON t.id = g.turma_id
		WHERE g.professor_id = $1
		  AND g.dispositivo_id = $2
		  AND g.dia_semana = $3
		  AND $4::TIME BETWEEN g.horario_inicio AND g.horario_fim
		ORDER BY g.horario_inicio ASC
		LIMIT 1
	`

	var aula modelos.AulaAgendada
	erro := r.pool.QueryRow(ctx, consulta, idProfessor, idDispositivo, diaSemana, horarioLocal).Scan(
		&aula.IdTurma,
		&aula.NomeTurma,
		&aula.ChatIdTelegram,
		&aula.Disciplina,
		&aula.HorarioInicio,
		&aula.HorarioFim,
	)
	if errors.Is(erro, pgx.ErrNoRows) {
		return nil, nil
	}
	if erro != nil {
		return nil, fmt.Errorf("erro ao buscar aula na grade horária: %w", erro)
	}

	return &aula, nil
}

// RegistrarPresenca insere um novo evento de entrada ou saída no banco de dados.
func (r *RepositorioPostgres) RegistrarPresenca(ctx context.Context, registro *modelos.RegistroPresenca) (int64, error) {
	const consulta = `
		INSERT INTO registros_presenca (
			professor_id, dispositivo_id, turma_id, tipo_evento,
			leitura_em, registro_offline, notificacao_enviada
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	var idGerado int64
	erro := r.pool.QueryRow(
		ctx,
		consulta,
		registro.IdProfessor,
		registro.IdDispositivo,
		registro.IdTurma,
		registro.TipoEvento,
		registro.LeituraEm,
		registro.RegistroOffline,
		registro.NotificacaoEnviada,
	).Scan(&idGerado)
	if erro != nil {
		return 0, fmt.Errorf("erro ao gravar registro de presença: %w", erro)
	}

	return idGerado, nil
}

// RegistrarTelemetria salva o heartbeat do ESP32 e atualiza o último contato do dispositivo.
func (r *RepositorioPostgres) RegistrarTelemetria(ctx context.Context, dados modelos.RequisicaoTelemetria) error {
	const inserirTelemetria = `
		INSERT INTO telemetria_dispositivos (
			dispositivo_id, versao_firmware, tempo_ativo_segundos,
			sinal_wifi_rssi, memoria_livre_bytes, pendentes_offline
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	if _, erro := r.pool.Exec(
		ctx,
		inserirTelemetria,
		dados.IdDispositivo,
		dados.VersaoFirmware,
		dados.TempoAtivoSegundos,
		dados.SinalWifiRssi,
		dados.MemoriaLivreBytes,
		dados.PendentesOffline,
	); erro != nil {
		return fmt.Errorf("erro ao gravar telemetria: %w", erro)
	}

	const atualizarDispositivo = `UPDATE dispositivos SET ultimo_heartbeat = NOW() WHERE id = $1`
	if _, erro := r.pool.Exec(ctx, atualizarDispositivo, dados.IdDispositivo); erro != nil {
		return fmt.Errorf("erro ao atualizar último heartbeat: %w", erro)
	}

	return nil
}

// RegistrarMensagemTelegram salva o identificador da mensagem enviada para exclusão noturna no Telegram.
func (r *RepositorioPostgres) RegistrarMensagemTelegram(ctx context.Context, chatId string, messageId int64) error {
	const consulta = `
		INSERT INTO mensagens_telegram (chat_id, message_id)
		VALUES ($1, $2)
		ON CONFLICT (chat_id, message_id) DO NOTHING
	`
	if _, erro := r.pool.Exec(ctx, consulta, chatId, messageId); erro != nil {
		return fmt.Errorf("erro ao registrar mensagem do Telegram para limpeza: %w", erro)
	}
	return nil
}

// ListarMensagensTelegramParaLimpar retorna os identificadores de mensagens pendentes de limpeza.
func (r *RepositorioPostgres) ListarMensagensTelegramParaLimpar(ctx context.Context) ([]modelos.MensagemTelegramRegistrada, error) {
	const consulta = `
		SELECT id, chat_id, message_id, enviada_em
		FROM mensagens_telegram
		ORDER BY chat_id ASC, id ASC
	`
	linhas, erro := r.pool.Query(ctx, consulta)
	if erro != nil {
		return nil, fmt.Errorf("erro ao listar mensagens do Telegram para limpeza: %w", erro)
	}
	defer linhas.Close()

	var mensagens []modelos.MensagemTelegramRegistrada
	for linhas.Next() {
		var item modelos.MensagemTelegramRegistrada
		if erro := linhas.Scan(&item.Id, &item.ChatId, &item.MessageId, &item.EnviadaEm); erro != nil {
			return nil, fmt.Errorf("erro ao ler registro de mensagem do Telegram: %w", erro)
		}
		mensagens = append(mensagens, item)
	}
	if erro := linhas.Err(); erro != nil {
		return nil, fmt.Errorf("erro ao percorrer mensagens do Telegram: %w", erro)
	}

	return mensagens, nil
}

// RemoverMensagensTelegramDoChat limpa apenas a fila auxiliar após exclusão no Telegram, mantendo o histórico de presença intacto.
func (r *RepositorioPostgres) RemoverMensagensTelegramDoChat(ctx context.Context, chatId string, ateId int64) (int, error) {
	const consulta = `
		DELETE FROM mensagens_telegram
		WHERE chat_id = $1 AND id <= $2
	`
	resultado, erro := r.pool.Exec(ctx, consulta, chatId, ateId)
	if erro != nil {
		return 0, fmt.Errorf("erro ao remover fila de mensagens do Telegram: %w", erro)
	}
	return int(resultado.RowsAffected()), nil
}
