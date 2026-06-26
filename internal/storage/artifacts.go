package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Artifacts struct {
	localDir string
	s3       *S3Store
}

func NewArtifacts(localDir, bucket, region string) *Artifacts {
	a := &Artifacts{localDir: localDir}
	if bucket != "" {
		a.s3 = NewS3Store(bucket, region)
	}
	return a
}

func (a *Artifacts) Enabled() bool {
	return a.localDir != "" || (a.s3 != nil && a.s3.Enabled())
}

type StoredObject struct {
	Key       string `json:"key"`
	URL       string `json:"url,omitempty"`
	ByteSize  int64  `json:"byte_size"`
	Storage   string `json:"storage"`
	CreatedAt time.Time `json:"created_at"`
}

func (a *Artifacts) Put(ctx context.Context, prefix string, data []byte) (StoredObject, error) {
	key := strings.Trim(prefix, "/") + "/" + uuid.NewString() + ".zip"
	obj := StoredObject{Key: key, ByteSize: int64(len(data)), CreatedAt: time.Now().UTC()}

	if a.s3 != nil && a.s3.Enabled() {
		url, err := a.s3.Put(ctx, key, data)
		if err != nil {
			return StoredObject{}, err
		}
		obj.Storage = "s3"
		obj.URL = url
		return obj, nil
	}

	if a.localDir == "" {
		return StoredObject{}, fmt.Errorf("artifact storage not configured (set AIVAR_ARTIFACTS_DIR or AIVAR_S3_BUCKET)")
	}
	path := filepath.Join(a.localDir, key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return StoredObject{}, err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return StoredObject{}, err
	}
	obj.Storage = "local"
	obj.URL = "/api/v1/artifacts/" + key
	return obj, nil
}

func (a *Artifacts) GetLocal(key string) ([]byte, error) {
	if a.localDir == "" {
		return nil, fmt.Errorf("local artifacts dir not configured")
	}
	key = strings.TrimPrefix(key, "/")
	return os.ReadFile(filepath.Join(a.localDir, key))
}
