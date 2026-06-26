package repos

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

func (s *Service) Register(ctx context.Context, req models.RegisterRepoRequest) (models.Repo, error) {
	parts := strings.Split(req.FullName, "/")
	if len(parts) != 2 {
		return models.Repo{}, fmt.Errorf("full_name must be org/repo")
	}

	id := uuid.NewString()
	row := s.pool.QueryRow(ctx, `
		INSERT INTO repos (id, org, name, full_name, repo_type, owner, tf_owner, project_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (full_name) DO UPDATE
		SET repo_type = EXCLUDED.repo_type,
		    owner = EXCLUDED.owner,
		    tf_owner = EXCLUDED.tf_owner,
		    project_id = COALESCE(EXCLUDED.project_id, repos.project_id),
		    updated_at = now()
		RETURNING id, org, name, full_name, repo_type, owner, tf_owner, project_id, created_at, updated_at
	`, id, parts[0], parts[1], req.FullName, req.Type, nullString(req.Owner), nullString(req.TFOwner), nullUUID(req.ProjectID))

	var repo models.Repo
	var owner, tfOwner, projectID *string
	if err := row.Scan(
		&repo.ID, &repo.Org, &repo.Name, &repo.FullName, &repo.RepoType,
		&owner, &tfOwner, &projectID, &repo.CreatedAt, &repo.UpdatedAt,
	); err != nil {
		return models.Repo{}, fmt.Errorf("register repo: %w", err)
	}
	repo.Owner = deref(owner)
	repo.TFOwner = deref(tfOwner)
	repo.ProjectID = deref(projectID)
	return repo, nil
}

func (s *Service) GetByFullName(ctx context.Context, fullName string) (models.Repo, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, org, name, full_name, repo_type, owner, tf_owner, project_id, created_at, updated_at
		FROM repos WHERE full_name = $1
	`, fullName)

	var repo models.Repo
	var owner, tfOwner, projectID *string
	if err := row.Scan(
		&repo.ID, &repo.Org, &repo.Name, &repo.FullName, &repo.RepoType,
		&owner, &tfOwner, &projectID, &repo.CreatedAt, &repo.UpdatedAt,
	); err != nil {
		return models.Repo{}, fmt.Errorf("get repo: %w", err)
	}
	repo.Owner = deref(owner)
	repo.TFOwner = deref(tfOwner)
	repo.ProjectID = deref(projectID)
	return repo, nil
}

func (s *Service) List(ctx context.Context, orgID, projectID string) ([]models.Repo, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT r.id, r.org, r.name, r.full_name, r.repo_type, r.owner, r.tf_owner, r.project_id, r.created_at, r.updated_at
		FROM repos r
		LEFT JOIN projects p ON p.id = r.project_id
		LEFT JOIN repo_projects rp ON rp.repo_id = r.id
		LEFT JOIN projects p2 ON p2.id = rp.project_id
		WHERE ($1 = '' OR COALESCE(p.organization_id, p2.organization_id)::text = $1)
		  AND ($2 = '' OR r.project_id::text = $2 OR rp.project_id::text = $2)
		ORDER BY r.full_name
	`, orgID, projectID)
	if err != nil {
		return nil, fmt.Errorf("list repos: %w", err)
	}
	defer rows.Close()

	out := make([]models.Repo, 0)
	for rows.Next() {
		var repo models.Repo
		var owner, tfOwner, projectIDVal *string
		if err := rows.Scan(
			&repo.ID, &repo.Org, &repo.Name, &repo.FullName, &repo.RepoType,
			&owner, &tfOwner, &projectIDVal, &repo.CreatedAt, &repo.UpdatedAt,
		); err != nil {
			return nil, err
		}
		repo.Owner = deref(owner)
		repo.TFOwner = deref(tfOwner)
		repo.ProjectID = deref(projectIDVal)
		out = append(out, repo)
	}
	return out, rows.Err()
}

func nullUUID(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func nullString(v string) *string {
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
