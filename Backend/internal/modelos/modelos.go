// Pacote modelos define as entidades de domínio do sistema e os contratos JSON
// trocados entre o dispositivo ESP32, o backend em Go e o banco PostgreSQL.
// Mantém as tags JSON compatíveis com a especificação REST da documentação do SCP.
package modelos

import "time"

// Constantes que representam o tipo de movimentação do docente na sala.
const (
	EventoEntrada = "ENTRADA"
	EventoSaida   = "SAIDA"
)

// RequisicaoPresenca representa o JSON enviado pelo ESP32 ao ler uma tag RFID.
type RequisicaoPresenca struct {
	IdDispositivo    string `json:"device_id"`
	UidTag           string `json:"tag_uid"`
	CarimboTempoLeit int64  `json:"read_timestamp"`
	SinalWifiRssi    int    `json:"rssi"`
	RegistroOffline  bool   `json:"is_offline_record"`
}

// ItemLotePresenca representa cada leitura armazenada durante operação off-line.
type ItemLotePresenca struct {
	UidTag           string `json:"tag_uid"`
	CarimboTempoLeit int64  `json:"read_timestamp"`
}

// RequisicaoLotePresenca representa o envio em lote após reconexão do ESP32.
type RequisicaoLotePresenca struct {
	IdDispositivo   string             `json:"device_id"`
	QuantidadeLote  int                `json:"batch_count"`
	Registros       []ItemLotePresenca `json:"records"`
}

// RequisicaoTelemetria representa o ping periódico de saúde (heartbeat) do ESP32.
type RequisicaoTelemetria struct {
	IdDispositivo       string `json:"device_id"`
	VersaoFirmware      string `json:"firmware_version"`
	TempoAtivoSegundos  int64  `json:"uptime_seconds"`
	SinalWifiRssi       int    `json:"wifi_rssi"`
	MemoriaLivreBytes   int64  `json:"free_heap_bytes"`
	PendentesOffline    int    `json:"offline_pending_count"`
}

// Professor representa o docente cadastrado e vinculado a uma tag RFID.
type Professor struct {
	Id     int
	Nome   string
	UidTag string
	Ativo  bool
}

// Dispositivo representa o leitor físico ESP32 instalado em uma sala ou laboratório.
type Dispositivo struct {
	Id                string
	NomeSala          string
	Bloco             string
	TokenAutenticacao string
	Ativo             bool
}

// AulaAgendada representa o cruzamento da grade horária ativa com a turma no Telegram.
type AulaAgendada struct {
	IdTurma        int
	NomeTurma      string
	ChatIdTelegram string
	Disciplina     string
	HorarioInicio  string
	HorarioFim     string
}

// RegistroPresenca representa o evento de chegada ou saída persistido no banco.
type RegistroPresenca struct {
	Id                 int64
	IdProfessor        int
	IdDispositivo      string
	IdTurma            *int
	TipoEvento         string
	LeituraEm          time.Time
	RegistroOffline    bool
	NotificacaoEnviada bool
}

// ResultadoPresenca resume o processamento de uma leitura para retorno da API.
type ResultadoPresenca struct {
	IdRegistro         int64  `json:"id_registro"`
	Professor          string `json:"professor"`
	Sala               string `json:"sala"`
	TipoEvento         string `json:"tipo_evento"`
	Horario            string `json:"horario"`
	NotificacaoEnviada bool   `json:"notificacao_enviada"`
}

// MensagemTelegramRegistrada guarda o ID da mensagem enviada pelo bot para limpeza noturna.
type MensagemTelegramRegistrada struct {
	Id        int64
	ChatId    string
	MessageId int64
	EnviadaEm time.Time
}

// ResultadoLimpezaTelegram resume a rotina noturna de exclusão de mensagens no Telegram.
type ResultadoLimpezaTelegram struct {
	MensagensEncontradas int `json:"mensagens_encontradas"`
	MensagensApagadas    int `json:"mensagens_apagadas"`
	RegistrosRemovidos   int `json:"registros_removidos"`
}

// RespostaAPI padroniza todas as respostas JSON devolvidas pelos endpoints HTTP.
type RespostaAPI struct {
	Sucesso  bool        `json:"sucesso"`
	Mensagem string      `json:"mensagem"`
	Dados    interface{} `json:"dados,omitempty"`
}
