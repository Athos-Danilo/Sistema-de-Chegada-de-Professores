// Pacote configuracao lê e valida as variáveis de ambiente da aplicação.
// Centraliza credenciais do banco PostgreSQL (Neon), token do Bot do Telegram,
// fuso horário regional e configurações de porta para execução local ou na Vercel.
package configuracao

import (
	"errors"
	"os"
	"strings"
	"time"
)

// Configuracao guarda todos os parâmetros necessários para o funcionamento do backend.
type Configuracao struct {
	UrlBancoDados        string
	TokenBotTelegram     string
	ChatIdTelegramPadrao string
	FusoHorario          *time.Location
	PortaServidor        string
}

// CarregarConfiguracao lê as variáveis de ambiente e retorna uma configuração validada.
func CarregarConfiguracao() (*Configuracao, error) {
	urlBanco := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if urlBanco == "" {
		return nil, errors.New("variável de ambiente DATABASE_URL não configurada")
	}

	tokenTelegram := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if tokenTelegram == "" {
		return nil, errors.New("variável de ambiente TELEGRAM_BOT_TOKEN não configurada")
	}

	chatPadrao := strings.TrimSpace(os.Getenv("TELEGRAM_CHAT_ID_PADRAO"))

	nomeFuso := strings.TrimSpace(os.Getenv("FUSO_HORARIO"))
	if nomeFuso == "" {
		nomeFuso = "America/Recife"
	}

	localizacao, erro := time.LoadLocation(nomeFuso)
	if erro != nil {
		// Garante fuso horário fixo GMT-3 caso o container não possua tzdata completo.
		localizacao = time.FixedZone("BRT", -3*60*60)
	}

	porta := strings.TrimSpace(os.Getenv("PORT"))
	if porta == "" {
		porta = "8080"
	}

	return &Configuracao{
		UrlBancoDados:        urlBanco,
		TokenBotTelegram:     tokenTelegram,
		ChatIdTelegramPadrao: chatPadrao,
		FusoHorario:          localizacao,
		PortaServidor:        porta,
	}, nil
}
