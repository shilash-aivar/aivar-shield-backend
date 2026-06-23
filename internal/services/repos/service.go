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
