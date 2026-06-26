ALTER TABLE audit_log ADD COLUMN IF NOT EXISTS prev_hash TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS infra_reviews (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repo_id         UUID REFERENCES repos(id) ON DELETE SET NULL,
    repo_full_name  TEXT NOT NULL,
    org_id          UUID REFERENCES organizations(id) ON DELETE SET NULL,
    project_id      UUID REFERENCES projects(id) ON DELETE SET NULL,
    plan_summary    JSONB NOT NULL DEFAULT '{}',
    changes_add     INT NOT NULL DEFAULT 0,
    changes_change  INT NOT NULL DEFAULT 0,
    changes_destroy INT NOT NULL DEFAULT 0,
    status          TEXT NOT NULL DEFAULT 'pending',
    submitted_by    TEXT NOT NULL,
    approved_by     TEXT,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_infra_reviews_status ON infra_reviews(status);
CREATE INDEX IF NOT EXISTS idx_infra_reviews_repo ON infra_reviews(repo_full_name);
