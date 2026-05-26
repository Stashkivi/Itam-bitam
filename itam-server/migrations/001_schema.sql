-- ============================================================
-- ITAM Server — initial schema
-- Run once against a fresh database. Idempotent via IF NOT EXISTS.
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ----------------------------------------------------------------
-- Enum types
-- ----------------------------------------------------------------
DO $$ BEGIN
    CREATE TYPE host_status     AS ENUM ('active','inactive','quarantined');
    CREATE TYPE diff_operation  AS ENUM ('added','removed','modified');
    CREATE TYPE entity_kind     AS ENUM ('service','process','port','interface','hardware');
    CREATE TYPE anomaly_severity AS ENUM ('CRITICAL','HIGH','MEDIUM','LOW');
    CREATE TYPE anomaly_status   AS ENUM ('open','acknowledged','resolved');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- ----------------------------------------------------------------
-- Organisations (multi-tenant root)
-- ----------------------------------------------------------------
CREATE TABLE IF NOT EXISTS organisations (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ----------------------------------------------------------------
-- Hosts — one row per enrolled endpoint, never deleted
-- ----------------------------------------------------------------
CREATE TABLE IF NOT EXISTS hosts (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id       UUID NOT NULL REFERENCES organisations(id),
    hostname     TEXT NOT NULL,
    os_family    TEXT NOT NULL,   -- 'linux' | 'windows'
    arch         TEXT NOT NULL,
    ip_primary   INET,
    enrolled_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at TIMESTAMPTZ,
    status       host_status NOT NULL DEFAULT 'active',
    UNIQUE (hostname, org_id)
);

CREATE INDEX IF NOT EXISTS idx_hosts_org        ON hosts(org_id);
CREATE INDEX IF NOT EXISTS idx_hosts_last_seen  ON hosts(last_seen_at DESC NULLS LAST);

-- ----------------------------------------------------------------
-- Host snapshots — append-only full scan results, partitioned monthly
-- ----------------------------------------------------------------
CREATE TABLE IF NOT EXISTS host_snapshots (
    id          BIGSERIAL,
    host_id     UUID         NOT NULL REFERENCES hosts(id),
    captured_at TIMESTAMPTZ  NOT NULL,
    payload     JSONB        NOT NULL,
    schema_ver  SMALLINT     NOT NULL DEFAULT 1,
    PRIMARY KEY (id, captured_at)
) PARTITION BY RANGE (captured_at);

-- ----------------------------------------------------------------
-- Config diffs — granular change log, partitioned monthly
-- ----------------------------------------------------------------
CREATE TABLE IF NOT EXISTS config_diffs (
    id          BIGSERIAL,
    host_id     UUID           NOT NULL,   -- intentionally no FK (partition key)
    changed_at  TIMESTAMPTZ    NOT NULL,
    entity_type entity_kind    NOT NULL,
    entity_key  TEXT           NOT NULL,   -- stable identity: "nginx:80/tcp"
    operation   diff_operation NOT NULL,
    old_value   JSONB,
    new_value   JSONB,
    diff_hash   TEXT           NOT NULL,   -- SHA-256(entity_key||old||new)
    PRIMARY KEY (id, changed_at)
) PARTITION BY RANGE (changed_at);

CREATE INDEX IF NOT EXISTS idx_diffs_host_time ON config_diffs(host_id, changed_at DESC);
CREATE INDEX IF NOT EXISTS idx_diffs_entity    ON config_diffs(entity_type, entity_key);

-- ----------------------------------------------------------------
-- Baselines — approved host state; compared against on each scan
-- ----------------------------------------------------------------
CREATE TABLE IF NOT EXISTS baselines (
    id          BIGSERIAL PRIMARY KEY,
    host_id     UUID        NOT NULL REFERENCES hosts(id),
    approved_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    approved_by TEXT,
    snapshot_id BIGINT,                -- FK skipped: cross-partition reference
    is_active   BOOLEAN     NOT NULL DEFAULT true,
    payload     JSONB       NOT NULL   -- denormalised for O(1) diff lookup
);

CREATE INDEX IF NOT EXISTS idx_baselines_host   ON baselines(host_id) WHERE is_active;

-- ----------------------------------------------------------------
-- Anomalies — detected rule violations
-- ----------------------------------------------------------------
CREATE TABLE IF NOT EXISTS anomalies (
    id          BIGSERIAL PRIMARY KEY,
    host_id     UUID             NOT NULL REFERENCES hosts(id),
    rule_id     TEXT             NOT NULL,
    severity    anomaly_severity NOT NULL,
    entity_type entity_kind,
    entity_key  TEXT,
    details     JSONB,
    detected_at TIMESTAMPTZ      NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ,
    status      anomaly_status   NOT NULL DEFAULT 'open',
    dedup_key   TEXT             NOT NULL  -- {host_uuid}:{rule_id}:{entity_key}
);

CREATE INDEX IF NOT EXISTS idx_anomalies_host   ON anomalies(host_id, status);
CREATE INDEX IF NOT EXISTS idx_anomalies_dedup  ON anomalies(dedup_key, status);
CREATE INDEX IF NOT EXISTS idx_anomalies_time   ON anomalies(detected_at DESC);

-- ----------------------------------------------------------------
-- Alert delivery tracking
-- ----------------------------------------------------------------
CREATE TABLE IF NOT EXISTS alert_deliveries (
    id           BIGSERIAL PRIMARY KEY,
    anomaly_id   BIGINT      NOT NULL REFERENCES anomalies(id),
    channel      TEXT        NOT NULL,   -- 'slack' | 'telegram'
    delivered_at TIMESTAMPTZ,
    failed_at    TIMESTAMPTZ,
    attempts     SMALLINT    NOT NULL DEFAULT 0,
    last_error   TEXT
);

-- ----------------------------------------------------------------
-- Enrollment tokens — admin-generated, single-use
-- ----------------------------------------------------------------
CREATE TABLE IF NOT EXISTS enrollment_tokens (
    id          BIGSERIAL   PRIMARY KEY,
    token_hash  TEXT        NOT NULL UNIQUE,  -- HMAC-SHA256 hex of raw token
    org_id      UUID        NOT NULL REFERENCES organisations(id),
    label       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    is_revoked  BOOLEAN     NOT NULL DEFAULT false
);

CREATE INDEX IF NOT EXISTS idx_tokens_org ON enrollment_tokens(org_id);

-- ----------------------------------------------------------------
-- Agent credentials — certificate metadata per enrolled host
-- ----------------------------------------------------------------
CREATE TABLE IF NOT EXISTS agent_credentials (
    host_id      UUID PRIMARY KEY REFERENCES hosts(id),
    cert_serial  TEXT        NOT NULL,
    cert_expires TIMESTAMPTZ NOT NULL,
    public_key   TEXT        NOT NULL,  -- PEM Ed25519
    issued_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ----------------------------------------------------------------
-- Org alert channels — Slack/Telegram config per org
-- ----------------------------------------------------------------
CREATE TABLE IF NOT EXISTS alert_channels (
    id           BIGSERIAL PRIMARY KEY,
    org_id       UUID    NOT NULL REFERENCES organisations(id),
    channel_type TEXT    NOT NULL CHECK (channel_type IN ('slack','telegram')),
    -- webhook_url and bot_token are stored AES-encrypted at app layer
    config_enc   BYTEA   NOT NULL,
    is_active    BOOLEAN NOT NULL DEFAULT true
);
