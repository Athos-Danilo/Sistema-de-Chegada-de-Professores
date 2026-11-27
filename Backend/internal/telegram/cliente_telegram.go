// Pacote telegram implementa a comunicação oficial com a Telegram Bot API.
// Formata as mensagens de chegada e saída de professores com dados da sala,
// disciplina e horário local e realiza o disparo HTTP para os grupos das turmas.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"scp-backend/internal/modelos"
)

// DadosNotificacao reúne as informações exibidas na mensagem do grupo no Telegram.
type DadosNotificacao struct {
	ChatId     string
	Professor  string
	Sala       string
	Bloco      string
	Disciplina string
	NomeTurma  string
	TipoEvento string
	Horario    string
}

// NotificadorTelegram define a interface de envio e limpeza de alertas para facilitar testes.
type NotificadorTelegram interface {
	EnviarAlertaPresenca(ctx context.Context, dados DadosNotificacao) (int64, error)
	ApagarMensagens(ctx context.Context, chatId string, idsMensagens []int64) (int, error)
}

// ClienteTelegram envia mensagens reais usando a Telegram Bot API.
type ClienteTelegram struct {
	tokenBot    string
	clienteHttp *http.Client
	urlBase     string
}

// payloadEnvioTelegram representa o corpo JSON aceito pelo método sendMessage.
type payloadEnvioTelegram struct {
	ChatId    string `json:"chat_id"`
	Texto     string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

type respostaEnvioTelegram struct {
	Ok     bool `json:"ok"`
	Result struct {
		MessageId int64 `json:"message_id"`
	} `json:"result"`
}

type payloadExclusaoLoteTelegram struct {
	ChatId     string  `json:"chat_id"`
	MessageIds []int64 `json:"message_ids"`
}

type payloadExclusaoUnicaTelegram struct {
	ChatId    string `json:"chat_id"`
	MessageId int64  `json:"message_id"`
}

// NovoClienteTelegram cria um cliente HTTP configurado para o Bot do Telegram.
func NovoClienteTelegram(tokenBot string) *ClienteTelegram {
	return &ClienteTelegram{
		tokenBot: tokenBot,
		urlBase:  "https://api.telegram.org",
		clienteHttp: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

// EnviarAlertaPresenca formata e envia a notificação de entrada ou saída e retorna o message_id gerado.
func (c *ClienteTelegram) EnviarAlertaPresenca(ctx context.Context, dados DadosNotificacao) (int64, error) {
	if strings.TrimSpace(dados.ChatId) == "" {
		return 0, fmt.Errorf("chat_id do Telegram vazio: notificação ignorada")
	}

	corpoResposta, codigoStatus, erro := c.postarJSON(ctx, "sendMessage", payloadEnvioTelegram{
		ChatId:    dados.ChatId,
		Texto:     montarMensagemFormatada(dados),
		ParseMode: "HTML",
	})
	if erro != nil {
		return 0, erro
	}
	if codigoStatus < 200 || codigoStatus >= 300 {
		return 0, fmt.Errorf("API do Telegram retornou status %d: %s", codigoStatus, corpoResposta)
	}

	var resposta respostaEnvioTelegram
	if erro := json.Unmarshal([]byte(corpoResposta), &resposta); erro != nil {
		return 0, nil
	}
	return resposta.Result.MessageId, nil
}

// ApagarMensagens remove em lote até 100 mensagens por requisição no canal/grupo informado.
func (c *ClienteTelegram) ApagarMensagens(ctx context.Context, chatId string, idsMensagens []int64) (int, error) {
	if strings.TrimSpace(chatId) == "" || len(idsMensagens) == 0 {
		return 0, nil
	}

	apagadas := 0
	const tamanhoLote = 100
	for inicio := 0; inicio < len(idsMensagens); inicio += tamanhoLote {
		fim := inicio + tamanhoLote
		if fim > len(idsMensagens) {
			fim = len(idsMensagens)
		}
		lote := idsMensagens[inicio:fim]

		corpoResposta, codigoStatus, erro := c.postarJSON(ctx, "deleteMessages", payloadExclusaoLoteTelegram{
			ChatId:     chatId,
			MessageIds: lote,
		})
		if erro != nil {
			return apagadas, erro
		}
		if codigoStatus >= 200 && codigoStatus < 300 {
			apagadas += len(lote)
			continue
		}
		if erroPermissaoOuFatal(codigoStatus, corpoResposta) {
			return apagadas, fmt.Errorf("API do Telegram recusou exclusão em lote (%d): %s", codigoStatus, corpoResposta)
		}

		// Se alguma mensagem do lote já foi removida manualmente ou passou de 48h, tenta individualmente.
		removidasIndividuais, erroIndiv := c.apagarIndividualmente(ctx, chatId, lote)
		apagadas += removidasIndividuais
		if erroIndiv != nil {
			return apagadas, erroIndiv
		}
	}

	return apagadas, nil
}

func (c *ClienteTelegram) apagarIndividualmente(ctx context.Context, chatId string, ids []int64) (int, error) {
	apagadas := 0
	for _, idMensagem := range ids {
		corpoResposta, codigoStatus, erro := c.postarJSON(ctx, "deleteMessage", payloadExclusaoUnicaTelegram{
			ChatId:    chatId,
			MessageId: idMensagem,
		})
		if erro != nil {
			return apagadas, erro
		}
		if codigoStatus >= 200 && codigoStatus < 300 {
			apagadas++
			continue
		}
		if erroPermissaoOuFatal(codigoStatus, corpoResposta) {
			return apagadas, fmt.Errorf("API do Telegram recusou exclusão (%d): %s", codigoStatus, corpoResposta)
		}
	}
	return apagadas, nil
}

func (c *ClienteTelegram) postarJSON(ctx context.Context, metodo string, payload interface{}) (string, int, error) {
	corpo, erro := json.Marshal(payload)
	if erro != nil {
		return "", 0, fmt.Errorf("erro ao serializar payload (%s): %w", metodo, erro)
	}

	endpoint := fmt.Sprintf("%s/bot%s/%s", c.urlBase, c.tokenBot, metodo)
	requisicao, erro := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(corpo))
	if erro != nil {
		return "", 0, fmt.Errorf("erro ao criar requisição para o Telegram (%s): %w", metodo, erro)
	}
	requisicao.Header.Set("Content-Type", "application/json")

	resposta, erro := c.clienteHttp.Do(requisicao)
	if erro != nil {
		return "", 0, fmt.Errorf("erro de rede ao chamar API do Telegram (%s): %w", metodo, erro)
	}
	defer resposta.Body.Close()

	bytesResposta, _ := io.ReadAll(io.LimitReader(resposta.Body, 4096))
	return string(bytesResposta), resposta.StatusCode, nil
}

func erroPermissaoOuFatal(codigoStatus int, corpo string) bool {
	if codigoStatus == http.StatusUnauthorized || codigoStatus == http.StatusForbidden || codigoStatus >= 500 {
		return true
	}
	texto := strings.ToLower(corpo)
	return strings.Contains(texto, "right") || strings.Contains(texto, "administrator")
}

// montarMensagemFormatada gera uma mensagem limpa em HTML com bloco visual para leitura rápida no Telegram.
func montarMensagemFormatada(dados DadosNotificacao) string {
	cabecalho := "🟢 <b>CHEGADA REGISTRADA</b>"
	if dados.TipoEvento == modelos.EventoSaida {
		cabecalho = "🔴 <b>SAÍDA REGISTRADA</b>"
	}

	local := escaparHTML(dados.Sala)
	if strings.TrimSpace(dados.Bloco) != "" {
		local = fmt.Sprintf("%s (%s)", escaparHTML(dados.Sala), escaparHTML(dados.Bloco))
	}

	var construtor strings.Builder
	construtor.WriteString(cabecalho + "\n\n<blockquote>")
	construtor.WriteString(fmt.Sprintf("<b>Professor:</b> %s\n", escaparHTML(dados.Professor)))

	if strings.TrimSpace(dados.Disciplina) != "" {
		construtor.WriteString(fmt.Sprintf("<b>Disciplina:</b> %s\n", escaparHTML(dados.Disciplina)))
	}

	construtor.WriteString(fmt.Sprintf("<b>Local:</b> %s\n", local))

	if strings.TrimSpace(dados.NomeTurma) != "" {
		construtor.WriteString(fmt.Sprintf("<b>Turma:</b> %s\n", escaparHTML(dados.NomeTurma)))
	}

	construtor.WriteString(fmt.Sprintf("<b>Horário:</b> <code>%s</code></blockquote>", escaparHTML(dados.Horario)))
	return construtor.String()
}

// escaparHTML evita quebra de formatação caso nomes possuam caracteres especiais.
func escaparHTML(texto string) string {
	substituidor := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
	)
	return substituidor.Replace(texto)
}
