CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE repos (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org         TEXT NOT NULL,
    name        TEXT NOT NULL,
    full_name   TEXT NOT NULL UNIQUE,
    repo_type   TEXT[] NOT NULL DEFAULT '{}',
    owner       TEXT,
    tf_owner    TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE rules (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tool        TEXT NOT NULL,
    rule_id     TEXT NOT NULL,
    severity    TEXT NOT NULL,
    title       TEXT NOT NULL,
    description TEXT,
    fix_guide   TEXT,
    docs_url    TEXT,
    version     INT NOT NULL DEFAULT 1,
    active      BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tool, rule_id)
);

CREATE TABLE suppressions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    platform_ref    TEXT NOT NULL UNIQUE,
    repo_id         UUID NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
    tool            TEXT NOT NULL,
    rule_id         TEXT NOT NULL,
    suppression_type TEXT NOT NULL,
    file_path       TEXT,
    line_number     INT,
    reason          TEXT NOT NULL,
    scope           TEXT NOT NULL DEFAULT 'repo',
    status          TEXT NOT NULL DEFAULT 'pending',
    severity        TEXT,
    requested_by    TEXT NOT NULL,
    approved_by     TEXT,
    expires_at      TIMESTAMPTZ,
    native_comment  TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_suppressions_repo_status ON suppressions(repo_id, status);
CREATE INDEX idx_suppressions_expires ON suppressions(expires_at) WHERE status = 'approved';

CREATE TABLE audit_log (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp       TIMESTAMPTZ NOT NULL DEFAULT now(),
    actor           TEXT NOT NULL,
    actor_type      TEXT NOT NULL,
    surface         TEXT NOT NULL,
    action          TEXT NOT NULL,
    resource_type   TEXT NOT NULL,
    resource_id     TEXT NOT NULL,
    repo            TEXT,
    rule_id         TEXT,
    tool            TEXT,
    severity        TEXT,
    details         JSONB NOT NULL DEFAULT '{}',
    signature       TEXT NOT NULL
);

CREATE INDEX idx_audit_log_repo ON audit_log(repo);
CREATE INDEX idx_audit_log_action ON audit_log(action);
CREATE INDEX idx_audit_log_timestamp ON audit_log(timestamp DESC);
