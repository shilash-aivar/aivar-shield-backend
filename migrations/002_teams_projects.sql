CREATE TABLE organizations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug            TEXT NOT NULL UNIQUE,
    name            TEXT NOT NULL,
    github_org_slug TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE teams (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    slug            TEXT NOT NULL,
    name            TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (organization_id, slug)
);

CREATE TABLE projects (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    team_id         UUID REFERENCES teams(id) ON DELETE SET NULL,
    slug            TEXT NOT NULL,
    name            TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (organization_id, slug)
);

ALTER TABLE repos ADD COLUMN IF NOT EXISTS project_id UUID REFERENCES projects(id) ON DELETE SET NULL;

CREATE TABLE repo_projects (
    repo_id     UUID NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    PRIMARY KEY (repo_id, project_id)
);

CREATE INDEX idx_teams_org ON teams(organization_id);
CREATE INDEX idx_projects_org ON projects(organization_id);
CREATE INDEX idx_projects_team ON projects(team_id);
CREATE INDEX idx_repos_project ON repos(project_id);

INSERT INTO organizations (slug, name, github_org_slug)
VALUES ('aivar', 'Aivar', 'aivar')
ON CONFLICT (slug) DO NOTHING;
