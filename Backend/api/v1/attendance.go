// Ponto de entrada da função serverless /api/v1/attendance na Vercel.
// Inicializa o controlador compartilhado e encaminha as leituras em tempo real
// enviadas pelo ESP32 para registro no PostgreSQL (Neon) e aviso no Telegram.
package v1

import (
	"net/http"

	"scp-backend/internal/controladores"
)

// Handler atende às chamadas HTTP enviadas para /api/v1/attendance na Vercel.
func Handler(w http.ResponseWriter, r *http.Request) {
	if controlador, ok := controladores.ObterControladorOuResponderErro(w); ok {
		controlador.TratarPresenca(w, r)
	}
}
