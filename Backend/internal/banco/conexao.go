// Pacote banco gerencia a conexão com o banco PostgreSQL hospedado na Neon
// e executa as operações de leitura e escrita usando pool adaptado ao ambiente
// serverless da Vercel, evitando vazamento de conexões e conflitos de cache.
package banco

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AbrirPoolConexoes inicializa o pool de conexões com o PostgreSQL da Neon.
func AbrirPoolConexoes(ctx context.Context, urlBanco string) (*pgxpool.Pool, error) {
	configPool, erro := pgxpool.ParseConfig(urlBanco)
	if erro != nil {
		return nil, fmt.Errorf("falha ao interpretar URL do banco Neon: %w", erro)
	}

	// Limites enxutos recomendados para funções serverless na Vercel + Neon.
	configPool.MaxConns = 5
	configPool.MinConns = 0
	configPool.MaxConnLifetime = 30 * time.Minute
	configPool.MaxConnIdleTime = 5 * time.Minute
	configPool.ConnConfig.ConnectTimeout = 10 * time.Second
	configPool.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	pool, erro := pgxpool.NewWithConfig(ctx, configPool)
	if erro != nil {
		return nil, fmt.Errorf("falha ao criar pool PostgreSQL: %w", erro)
	}

	if erro := pool.Ping(ctx); erro != nil {
		pool.Close()
		return nil, fmt.Errorf("falha ao testar conexão com a Neon: %w", erro)
	}

	return pool, nil
}
