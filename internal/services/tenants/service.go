package tenants

import (
	"context"
	"fmt"
	"strings"

	"github.com/aivar-shield/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) SeedDefaults(ctx context.Context) error {
	var orgID string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO organizations (slug, name, github_org_slug)
		VALUES ('aivar', 'Aivar', 'aivar')
		ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name
		RETURNING id
	`).Scan(&orgID)
	if err != nil {
		return fmt.Errorf("seed organization: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO teams (organization_id, slug, name)
		SELECT $1, 'platform', 'Platform Engineering'
		WHERE NOT EXISTS (
			SELECT 1 FROM teams WHERE organization_id = $1 AND slug = 'platform'
		)
	`, orgID)
	if err != nil {
		return fmt.Errorf("seed team: %w", err)
	}

	var teamID string
	if err := s.pool.QueryRow(ctx, `SELECT id FROM teams WHERE organization_id = $1 AND slug = 'platform'`, orgID).Scan(&teamID); err != nil {
		return fmt.Errorf("load seed team: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO projects (organization_id, team_id, slug, name)
		SELECT $1, $2, 'default', 'Default Project'
		WHERE NOT EXISTS (
			SELECT 1 FROM projects WHERE organization_id = $1 AND slug = 'default'
		)
	`, orgID, teamID)
	if err != nil {
		return fmt.Errorf("seed project: %w", err)
	}

	return nil
}

func (s *Service) ListOrganizations(ctx context.Context) ([]models.Organization, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, slug, name, github_org_slug, created_at
		FROM organizations ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}
	defer rows.Close()

	out := make([]models.Organization, 0)
	for rows.Next() {
		var org models.Organization
		var gh *string
		if err := rows.Scan(&org.ID, &org.Slug, &org.Name, &gh, &org.CreatedAt); err != nil {
			return nil, err
		}
		org.GitHubOrgSlug = deref(gh)
		out = append(out, org)
	}
	return out, rows.Err()
}

func (s *Service) ListTeams(ctx context.Context, orgID string) ([]models.Team, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, organization_id, slug, name, created_at
		FROM teams
		WHERE organization_id = $1
		ORDER BY name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list teams: %w", err)
	}
	defer rows.Close()

	out := make([]models.Team, 0)
	for rows.Next() {
		var team models.Team
		if err := rows.Scan(&team.ID, &team.OrganizationID, &team.Slug, &team.Name, &team.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, team)
	}
	return out, rows.Err()
}

func (s *Service) ListProjects(ctx context.Context, orgID, teamID string) ([]models.Project, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, organization_id, team_id, slug, name, created_at
		FROM projects
		WHERE organization_id = $1
		  AND ($2 = '' OR team_id::text = $2)
		ORDER BY name
	`, orgID, teamID)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	out := make([]models.Project, 0)
	for rows.Next() {
		var project models.Project
		var team *string
		if err := rows.Scan(&project.ID, &project.OrganizationID, &team, &project.Slug, &project.Name, &project.CreatedAt); err != nil {
			return nil, err
		}
		project.TeamID = deref(team)
		out = append(out, project)
	}
	return out, rows.Err()
}

func (s *Service) CreateOrganization(ctx context.Context, req models.CreateOrganizationRequest) (models.Organization, error) {
	id := uuid.NewString()
	row := s.pool.QueryRow(ctx, `
		INSERT INTO organizations (id, slug, name, github_org_slug)
		VALUES ($1, $2, $3, $4)
		RETURNING id, slug, name, github_org_slug, created_at
	`, id, req.Slug, req.Name, nullString(req.GitHubOrgSlug))

	var org models.Organization
	var gh *string
	if err := row.Scan(&org.ID, &org.Slug, &org.Name, &gh, &org.CreatedAt); err != nil {
		return models.Organization{}, fmt.Errorf("create organization: %w", err)
	}
	org.GitHubOrgSlug = deref(gh)
	return org, nil
}

func (s *Service) CreateTeam(ctx context.Context, orgID string, req models.CreateTeamRequest) (models.Team, error) {
	id := uuid.NewString()
	row := s.pool.QueryRow(ctx, `
		INSERT INTO teams (id, organization_id, slug, name)
		VALUES ($1, $2, $3, $4)
		RETURNING id, organization_id, slug, name, created_at
	`, id, orgID, req.Slug, req.Name)

	var team models.Team
	if err := row.Scan(&team.ID, &team.OrganizationID, &team.Slug, &team.Name, &team.CreatedAt); err != nil {
		return models.Team{}, fmt.Errorf("create team: %w", err)
	}
	return team, nil
}

func (s *Service) CreateProject(ctx context.Context, orgID string, req models.CreateProjectRequest) (models.Project, error) {
	id := uuid.NewString()
	row := s.pool.QueryRow(ctx, `
		INSERT INTO projects (id, organization_id, team_id, slug, name)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, organization_id, team_id, slug, name, created_at
	`, id, orgID, nullUUID(req.TeamID), req.Slug, req.Name)

	var project models.Project
	var team *string
	if err := row.Scan(&project.ID, &project.OrganizationID, &team, &project.Slug, &project.Name, &project.CreatedAt); err != nil {
		return models.Project{}, fmt.Errorf("create project: %w", err)
	}
	project.TeamID = deref(team)
	return project, nil
}

func (s *Service) ListMembers(ctx context.Context, orgID string) ([]models.Member, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT u.id, u.login, u.email, u.name, m.role
		FROM memberships m
		JOIN users u ON u.id = m.user_id
		WHERE m.organization_id = $1
		ORDER BY u.login
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	out := make([]models.Member, 0)
	for rows.Next() {
		var m models.Member
		var email, name *string
		if err := rows.Scan(&m.UserID, &m.Login, &email, &name, &m.Role); err != nil {
			return nil, err
		}
		m.Email = deref(email)
		m.Name = deref(name)
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Service) UpdateMemberRole(ctx context.Context, orgID, userID, role string) error {
	if role != "admin" && role != "approver" && role != "member" {
		return fmt.Errorf("invalid role: %s", role)
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE memberships SET role = $3
		WHERE organization_id = $1 AND user_id = $2
	`, orgID, userID, role)
	if err != nil {
		return fmt.Errorf("update member role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("membership not found")
	}
	return nil
}

func (s *Service) AddMember(ctx context.Context, orgID, login, role string) (models.Member, error) {
	login = strings.TrimSpace(login)
	if login == "" {
		return models.Member{}, fmt.Errorf("github_login is required")
	}
	if role != "admin" && role != "approver" && role != "member" {
		return models.Member{}, fmt.Errorf("invalid role: %s", role)
	}

	var userID string
	err := s.pool.QueryRow(ctx, `SELECT id FROM users WHERE lower(login) = lower($1)`, login).Scan(&userID)
	if err != nil {
		return models.Member{}, fmt.Errorf("user %q not found — they must sign in to the portal once first", login)
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO memberships (user_id, organization_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, organization_id) DO UPDATE SET role = EXCLUDED.role
	`, userID, orgID, role)
	if err != nil {
		return models.Member{}, fmt.Errorf("add member: %w", err)
	}

	members, err := s.ListMembers(ctx, orgID)
	if err != nil {
		return models.Member{}, err
	}
	for _, m := range members {
		if m.UserID == userID {
			return m, nil
		}
	}
	return models.Member{}, fmt.Errorf("member added but not found")
}

func nullString(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func nullUUID(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func deref(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
