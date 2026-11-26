// Servidor HTTP principal da aplicação SCP para execução na Vercel e localmente.
// Registra todas as rotas REST da API (/api/health e /api/v1/*) e escuta na porta
// fornecida pela variável PORT, inicializando banco e Telegram sob demanda.
package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"scp-backend/internal/controladores"
)

func registrarRotaControlador(
	roteador *http.ServeMux,
	caminho string,
	acao func(*controladores.ControladorAPI, http.ResponseWriter, *http.Request),
) {
	roteador.HandleFunc(caminho, func(w http.ResponseWriter, r *http.Request) {
		if controlador, ok := controladores.ObterControladorOuResponderErro(w); ok {
			acao(controlador, w, r)
		}
	})
}

func main() {
	roteador := http.NewServeMux()
	roteador.HandleFunc("/api/health", controladores.TratarSaude)
	registrarRotaControlador(roteador, "/api/v1/attendance", (*controladores.ControladorAPI).TratarPresenca)
	registrarRotaControlador(roteador, "/api/v1/attendance/sync-batch", (*controladores.ControladorAPI).TratarLotePresenca)
	registrarRotaControlador(roteador, "/api/v1/telemetry/heartbeat", (*controladores.ControladorAPI).TratarTelemetria)
	registrarRotaControlador(roteador, "/api/cron/limpar-telegram", (*controladores.ControladorAPI).TratarLimpezaNoturnaTelegram)

	porta := strings.TrimSpace(os.Getenv("PORT"))
	if porta == "" {
		porta = "8080"
	}

	log.Printf("Servidor SCP em Go iniciado na porta %s", porta)
	log.Fatal(http.ListenAndServe(":"+porta, roteador))
}
