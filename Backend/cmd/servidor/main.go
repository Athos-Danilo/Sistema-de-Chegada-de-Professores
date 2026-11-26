// Servidor HTTP local utilizado para testar toda a API no computador antes do deploy.
// Monta exatamente as mesmas rotas da Vercel apontando para os controladores do sistema
// e carrega as configurações pelas variáveis de ambiente configuradas na máquina.
package main

import (
	"log"
	"net/http"

	"scp-backend/internal/configuracao"
	"scp-backend/internal/controladores"
)

func main() {
	cfg, erro := configuracao.CarregarConfiguracao()
	if erro != nil {
		log.Fatalf("Erro nas variáveis de ambiente: %v", erro)
	}

	controlador, erro := controladores.ObterControladorPadrao()
	if erro != nil {
		log.Fatalf("Erro ao conectar ao banco PostgreSQL (Neon): %v", erro)
	}

	roteador := http.NewServeMux()
	roteador.HandleFunc("/api/health", controladores.TratarSaude)
	roteador.HandleFunc("/api/v1/attendance", controlador.TratarPresenca)
	roteador.HandleFunc("/api/v1/attendance/sync-batch", controlador.TratarLotePresenca)
	roteador.HandleFunc("/api/v1/telemetry/heartbeat", controlador.TratarTelemetria)

	endereco := ":" + cfg.PortaServidor
	log.Printf("Servidor SCP local rodando em http://localhost%s", endereco)
	log.Fatal(http.ListenAndServe(endereco, roteador))
}
