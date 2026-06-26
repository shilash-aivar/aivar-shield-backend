CREATE TABLE IF NOT EXISTS team_memberships (
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    team_id     UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    role        TEXT NOT NULL CHECK (role IN ('admin', 'approver', 'member')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, team_id)
);

CREATE INDEX IF NOT EXISTS idx_team_memberships_team ON team_memberships(team_id);

CREATE TABLE IF NOT EXISTS delivery_artifacts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repo        TEXT,
    org_id      UUID,
    project_id  UUID,
    storage_key TEXT NOT NULL UNIQUE,
    manifest_sig TEXT,
    byte_size   BIGINT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_delivery_artifacts_repo ON delivery_artifacts(repo);

ALTER TABLE infra_reviews ADD COLUMN IF NOT EXISTS estimated_monthly_delta NUMERIC;
