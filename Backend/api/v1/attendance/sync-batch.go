// Ponto de entrada da função serverless /api/v1/attendance/sync-batch na Vercel.
// Recebe os lotes de registros acumulados em modo off-line no ESP32 e
// delega ao controlador a gravação cronológica no PostgreSQL da Neon.
package attendance

import (
	"net/http"

	"scp-backend/internal/controladores"
)

// Handler atende às chamadas HTTP enviadas para /api/v1/attendance/sync-batch na Vercel.
func Handler(w http.ResponseWriter, r *http.Request) {
	if controlador, ok := controladores.ObterControladorOuResponderErro(w); ok {
		controlador.TratarLotePresenca(w, r)
	}
}
