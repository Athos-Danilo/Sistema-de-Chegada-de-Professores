// Pacote controladores traduz requisições e respostas HTTP para a camada de serviço.
// Gerencia o ciclo de inicialização única (singleton) das dependências no ambiente
// serverless da Vercel e padroniza respostas JSON, validação de métodos e erros.
package controladores

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"scp-backend/internal/banco"
	"scp-backend/internal/configuracao"
	"scp-backend/internal/modelos"
	"scp-backend/internal/servicos"
	"scp-backend/internal/telegram"
)

// ControladorAPI agrupa os manipuladores HTTP dos endpoints do sistema SCP.
type ControladorAPI struct {
	servico *servicos.ServicoPresenca
}

var (
	instanciaPadrao *ControladorAPI
	erroInicial     error
	inicializacao   sync.Once
)

// NovoControladorAPI cria um controlador conectado ao serviço de presença informado.
func NovoControladorAPI(servico *servicos.ServicoPresenca) *ControladorAPI {
	return &ControladorAPI{servico: servico}
}

// ObterControladorPadrao reutiliza a conexão com a Neon entre chamadas quentes na Vercel.
func ObterControladorPadrao() (*ControladorAPI, error) {
	inicializacao.Do(func() {
		cfg, erro := configuracao.CarregarConfiguracao()
		if erro != nil {
			erroInicial = erro
			return
		}

		ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelar()

		pool, erro := banco.AbrirPoolConexoes(ctx, cfg.UrlBancoDados)
		if erro != nil {
			erroInicial = erro
			return
		}

		repositorio := banco.NovoRepositorioPostgres(pool)
		notificador := telegram.NovoClienteTelegram(cfg.TokenBotTelegram)
		servico := servicos.NovoServicoPresenca(repositorio, notificador, cfg.FusoHorario, cfg.ChatIdTelegramPadrao)
		instanciaPadrao = NovoControladorAPI(servico)
	})

	return instanciaPadrao, erroInicial
}

// ObterControladorOuResponderErro inicializa o controlador ou grava resposta HTTP 500 padronizada.
func ObterControladorOuResponderErro(w http.ResponseWriter) (*ControladorAPI, bool) {
	controlador, erro := ObterControladorPadrao()
	if erro != nil {
		responderErro(w, http.StatusInternalServerError, "Falha ao inicializar serviço: "+erro.Error())
		return nil, false
	}
	return controlador, true
}

// TratarSaude responde ao endpoint de verificação de disponibilidade do serviço.
func TratarSaude(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		responderErro(w, http.StatusMethodNotAllowed, "Método não permitido; utilize GET")
		return
	}

	responderJSON(w, http.StatusOK, modelos.RespostaAPI{
		Sucesso:  true,
		Mensagem: "API SCP em Go operando normalmente na nuvem",
		Dados: map[string]string{
			"status": "ativo",
		},
	})
}

// TratarPresenca processa a rota POST /api/v1/attendance enviada pelo ESP32.
func (c *ControladorAPI) TratarPresenca(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		responderErro(w, http.StatusMethodNotAllowed, "Método não permitido; utilize POST")
		return
	}

	var requisicao modelos.RequisicaoPresenca
	if erro := lerJSON(r, &requisicao); erro != nil {
		responderErro(w, http.StatusBadRequest, "JSON da requisição inválido")
		return
	}

	token := r.Header.Get("X-Device-Token")
	resultado, erro := c.servico.RegistrarLeitura(r.Context(), token, requisicao)
	if erro != nil {
		responderErroDominio(w, erro)
		return
	}

	responderJSON(w, http.StatusOK, modelos.RespostaAPI{
		Sucesso:  true,
		Mensagem: "Presença registrada com sucesso",
		Dados:    resultado,
	})
}

// TratarLotePresenca processa a rota POST /api/v1/attendance/sync-batch.
func (c *ControladorAPI) TratarLotePresenca(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		responderErro(w, http.StatusMethodNotAllowed, "Método não permitido; utilize POST")
		return
	}

	var lote modelos.RequisicaoLotePresenca
	if erro := lerJSON(r, &lote); erro != nil {
		responderErro(w, http.StatusBadRequest, "JSON do lote inválido")
		return
	}

	token := r.Header.Get("X-Device-Token")
	resultados, erro := c.servico.ProcessarLoteOffline(r.Context(), token, lote)
	if erro != nil {
		responderErroDominio(w, erro)
		return
	}

	responderJSON(w, http.StatusOK, modelos.RespostaAPI{
		Sucesso:  true,
		Mensagem: servicos.FormatarResumoLote(len(resultados)),
		Dados:    resultados,
	})
}

// TratarTelemetria processa a rota POST /api/v1/telemetry/heartbeat.
func (c *ControladorAPI) TratarTelemetria(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		responderErro(w, http.StatusMethodNotAllowed, "Método não permitido; utilize POST")
		return
	}

	var dados modelos.RequisicaoTelemetria
	if erro := lerJSON(r, &dados); erro != nil {
		responderErro(w, http.StatusBadRequest, "JSON de telemetria inválido")
		return
	}

	token := r.Header.Get("X-Device-Token")
	if erro := c.servico.RegistrarTelemetria(r.Context(), token, dados); erro != nil {
		responderErroDominio(w, erro)
		return
	}

	responderJSON(w, http.StatusOK, modelos.RespostaAPI{
		Sucesso:  true,
		Mensagem: "Heartbeat registrado com sucesso",
	})
}

func lerJSON(r *http.Request, destino interface{}) error {
	defer r.Body.Close()
	leitorLimitado := io.LimitReader(r.Body, 1<<20) // Limite de 1 MB por segurança.
	decodificador := json.NewDecoder(leitorLimitado)
	return decodificador.Decode(destino)
}

func responderErroDominio(w http.ResponseWriter, erro error) {
	switch {
	case errors.Is(erro, servicos.ErrTokenInvalido):
		responderErro(w, http.StatusUnauthorized, erro.Error())
	case errors.Is(erro, servicos.ErrDadosInvalidos):
		responderErro(w, http.StatusBadRequest, erro.Error())
	case errors.Is(erro, servicos.ErrTagNaoCadastrada):
		responderErro(w, http.StatusNotFound, erro.Error())
	default:
		responderErro(w, http.StatusInternalServerError, "Erro interno ao processar requisição")
	}
}

func responderErro(w http.ResponseWriter, codigoHTTP int, mensagem string) {
	responderJSON(w, codigoHTTP, modelos.RespostaAPI{
		Sucesso:  false,
		Mensagem: mensagem,
	})
}

func responderJSON(w http.ResponseWriter, codigoHTTP int, corpo interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(codigoHTTP)
	_ = json.NewEncoder(w).Encode(corpo)
}
