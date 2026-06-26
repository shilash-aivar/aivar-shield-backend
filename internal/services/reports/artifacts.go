package reports

import (
	"context"

	"github.com/google/uuid"
)

func (s *Service) RecordArtifact(ctx context.Context, id, repo, orgID, projectID, key, sig string, size int64) error {
	if id == "" {
		id = uuid.NewString()
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO delivery_artifacts (id, repo, org_id, project_id, storage_key, manifest_sig, byte_size)
		VALUES ($1, $2, NULLIF($3,'')::uuid, NULLIF($4,'')::uuid, $5, $6, $7)
	`, id, nullStr(repo), orgID, projectID, key, nullStr(sig), size)
	return err
}

func nullStr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
