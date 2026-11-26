// Ponto de entrada da função serverless /api/v1/telemetry/heartbeat na Vercel.
// Recebe os pings periódicos de saúde do ESP32 (sinal Wi-Fi, memória livre e uptime)
// para monitoramento contínuo dos dispositivos armazenado no PostgreSQL da Neon.
package telemetry

import (
	"net/http"

	"scp-backend/internal/controladores"
)

// Handler atende às chamadas HTTP enviadas para /api/v1/telemetry/heartbeat na Vercel.
func Handler(w http.ResponseWriter, r *http.Request) {
	if controlador, ok := controladores.ObterControladorOuResponderErro(w); ok {
		controlador.TratarTelemetria(w, r)
	}
}
