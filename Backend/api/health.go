// Ponto de entrada da função serverless /api/health na Vercel.
// Permite verificar rapidamente pelo navegador se o backend em Go está ativo
// sem exigir autenticação nem abertura de conexão prévia com o banco.
package api

import (
	"net/http"

	"scp-backend/internal/controladores"
)

// Handler atende às chamadas HTTP enviadas para /api/health na Vercel.
func Handler(w http.ResponseWriter, r *http.Request) {
	controladores.TratarSaude(w, r)
}
