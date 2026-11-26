-- ============================================================================
-- Esquema relacional do banco de dados PostgreSQL (Neon) para o projeto SCP.
-- Define as tabelas de professores, dispositivos ESP32, turmas, grades horárias,
-- registros de presença (entrada/saída) e telemetria de saúde dos leitores.
-- ============================================================================

CREATE TABLE IF NOT EXISTS professores (
    id SERIAL PRIMARY KEY,
    nome VARCHAR(120) NOT NULL,
    tag_uid VARCHAR(32) NOT NULL UNIQUE,
    ativo BOOLEAN NOT NULL DEFAULT TRUE,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS dispositivos (
    id VARCHAR(50) PRIMARY KEY,
    nome_sala VARCHAR(100) NOT NULL,
    bloco VARCHAR(50) NOT NULL DEFAULT '',
    token_autenticacao VARCHAR(128) NOT NULL,
    ativo BOOLEAN NOT NULL DEFAULT TRUE,
    ultimo_heartbeat TIMESTAMPTZ,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS turmas (
    id SERIAL PRIMARY KEY,
    nome VARCHAR(100) NOT NULL,
    telegram_chat_id VARCHAR(64) NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS grades_horarias (
    id SERIAL PRIMARY KEY,
    professor_id INTEGER NOT NULL REFERENCES professores(id) ON DELETE CASCADE,
    dispositivo_id VARCHAR(50) NOT NULL REFERENCES dispositivos(id) ON DELETE CASCADE,
    turma_id INTEGER NOT NULL REFERENCES turmas(id) ON DELETE CASCADE,
    disciplina VARCHAR(120) NOT NULL,
    dia_semana SMALLINT NOT NULL CHECK (dia_semana BETWEEN 0 AND 6), -- 0=Domingo .. 6=Sábado
    horario_inicio TIME NOT NULL,
    horario_fim TIME NOT NULL
);

CREATE TABLE IF NOT EXISTS registros_presenca (
    id BIGSERIAL PRIMARY KEY,
    professor_id INTEGER NOT NULL REFERENCES professores(id) ON DELETE CASCADE,
    dispositivo_id VARCHAR(50) NOT NULL REFERENCES dispositivos(id) ON DELETE CASCADE,
    turma_id INTEGER REFERENCES turmas(id) ON DELETE SET NULL,
    tipo_evento VARCHAR(15) NOT NULL CHECK (tipo_evento IN ('ENTRADA', 'SAIDA')),
    leitura_em TIMESTAMPTZ NOT NULL,
    registro_offline BOOLEAN NOT NULL DEFAULT FALSE,
    notificacao_enviada BOOLEAN NOT NULL DEFAULT FALSE,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS telemetria_dispositivos (
    id BIGSERIAL PRIMARY KEY,
    dispositivo_id VARCHAR(50) NOT NULL REFERENCES dispositivos(id) ON DELETE CASCADE,
    versao_firmware VARCHAR(30) NOT NULL,
    tempo_ativo_segundos BIGINT NOT NULL,
    sinal_wifi_rssi INTEGER NOT NULL,
    memoria_livre_bytes BIGINT NOT NULL,
    pendentes_offline INTEGER NOT NULL DEFAULT 0,
    registrado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_professores_tag_uid ON professores(tag_uid);
CREATE INDEX IF NOT EXISTS idx_registros_prof_disp_data ON registros_presenca(professor_id, dispositivo_id, leitura_em DESC);
CREATE INDEX IF NOT EXISTS idx_grades_busca ON grades_horarias(professor_id, dispositivo_id, dia_semana);
