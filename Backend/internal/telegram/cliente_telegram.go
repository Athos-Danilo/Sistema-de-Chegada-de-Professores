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

// NotificadorTelegram define a interface de envio de alertas para facilitar testes.
type NotificadorTelegram interface {
	EnviarAlertaPresenca(ctx context.Context, dados DadosNotificacao) error
}

// ClienteTelegram envia mensagens reais usando o endpoint sendMessage do Telegram.
type ClienteTelegram struct {
	tokenBot   string
	clienteHttp *http.Client
	urlBase    string
}

// payloadEnvioTelegram representa o corpo JSON aceito pela Telegram Bot API.
type payloadEnvioTelegram struct {
	ChatId    string `json:"chat_id"`
	Texto     string `json:"text"`
	ParseMode string `json:"parse_mode"`
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

// EnviarAlertaPresenca formata e envia a notificação de entrada ou saída para o grupo.
func (c *ClienteTelegram) EnviarAlertaPresenca(ctx context.Context, dados DadosNotificacao) error {
	if strings.TrimSpace(dados.ChatId) == "" {
		return fmt.Errorf("chat_id do Telegram vazio: notificação ignorada")
	}

	texto := montarMensagemFormatada(dados)
	corpo, erro := json.Marshal(payloadEnvioTelegram{
		ChatId:    dados.ChatId,
		Texto:     texto,
		ParseMode: "HTML",
	})
	if erro != nil {
		return fmt.Errorf("erro ao serializar payload do Telegram: %w", erro)
	}

	endpoint := fmt.Sprintf("%s/bot%s/sendMessage", c.urlBase, c.tokenBot)
	requisicao, erro := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(corpo))
	if erro != nil {
		return fmt.Errorf("erro ao criar requisição para o Telegram: %w", erro)
	}
	requisicao.Header.Set("Content-Type", "application/json")

	resposta, erro := c.clienteHttp.Do(requisicao)
	if erro != nil {
		return fmt.Errorf("erro de rede ao chamar API do Telegram: %w", erro)
	}
	defer resposta.Body.Close()

	if resposta.StatusCode < 200 || resposta.StatusCode >= 300 {
		corpoErro, _ := io.ReadAll(io.LimitReader(resposta.Body, 512))
		return fmt.Errorf("API do Telegram retornou status %d: %s", resposta.StatusCode, string(corpoErro))
	}

	return nil
}

// montarMensagemFormatada gera uma mensagem clara em HTML para leitura rápida no Telegram.
func montarMensagemFormatada(dados DadosNotificacao) string {
	cabecalho := "✅ <b>CHEGADA REGISTRADA</b>"
	if dados.TipoEvento == modelos.EventoSaida {
		cabecalho = "👋 <b>SAÍDA REGISTRADA</b>"
	}

	local := escaparHTML(dados.Sala)
	if strings.TrimSpace(dados.Bloco) != "" {
		local = fmt.Sprintf("%s (%s)", escaparHTML(dados.Sala), escaparHTML(dados.Bloco))
	}

	var construtor strings.Builder
	construtor.WriteString(cabecalho + "\n\n")
	construtor.WriteString(fmt.Sprintf("👨‍🏫 <b>Professor:</b> %s\n", escaparHTML(dados.Professor)))
	construtor.WriteString(fmt.Sprintf("📍 <b>Local:</b> %s\n", local))

	if strings.TrimSpace(dados.Disciplina) != "" {
		construtor.WriteString(fmt.Sprintf("📚 <b>Disciplina:</b> %s\n", escaparHTML(dados.Disciplina)))
	}
	if strings.TrimSpace(dados.NomeTurma) != "" {
		construtor.WriteString(fmt.Sprintf("🎓 <b>Turma:</b> %s\n", escaparHTML(dados.NomeTurma)))
	}

	construtor.WriteString(fmt.Sprintf("⏰ <b>Horário:</b> %s", escaparHTML(dados.Horario)))
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
