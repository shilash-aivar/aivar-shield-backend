package reports

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

type BundleManifest struct {
	Version     string    `json:"version"`
	GeneratedAt time.Time `json:"generated_at"`
	Repo        string    `json:"repo,omitempty"`
	ProjectID   string    `json:"project_id,omitempty"`
	OrgID       string    `json:"org_id,omitempty"`
	ContentHash string    `json:"content_hash"`
	Signature   string    `json:"signature"`
	Files       []string  `json:"files"`
}

// BuildBundle builds a signed zip delivery proof bundle.
func (s *Service) BuildBundle(ctx context.Context, repo, projectID, orgID, signingKey string) ([]byte, BundleManifest, error) {
	report, err := s.Delivery(ctx, repo, projectID, orgID)
	if err != nil {
		return nil, BundleManifest{}, err
	}
	return buildBundleZip(report, repo, projectID, orgID, signingKey)
}

func buildBundleZip(report DeliveryReport, repo, projectID, orgID, signingKey string) ([]byte, BundleManifest, error) {
	reportJSON, _ := json.MarshalIndent(report, "", "  ")
	suppJSON, _ := json.MarshalIndent(report.Suppressions, "", "  ")
	auditJSON, _ := json.MarshalIndent(report.Audit, "", "  ")
	html := []byte(RenderDeliveryHTML(report))

	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	files := map[string][]byte{
		"delivery-report.json": reportJSON,
		"suppressions.json":    suppJSON,
		"audit-log.json":       auditJSON,
		"delivery-report.html": html,
		"sbom/README.txt":      []byte("Include sbom.json or cyclonedx.json from syft/grype in your CI packaging step.\n"),
	}
	names := make([]string, 0, len(files))
	for name, data := range files {
		names = append(names, name)
		w, err := zw.Create(name)
		if err != nil {
			return nil, BundleManifest{}, err
		}
		if _, err := w.Write(data); err != nil {
			return nil, BundleManifest{}, err
		}
	}

	contentHash := sha256.Sum256(bytes.Join([][]byte{reportJSON, suppJSON, auditJSON}, nil))
	manifest := BundleManifest{
		Version:     "1.0",
		GeneratedAt: time.Now().UTC(),
		Repo:        repo,
		ProjectID:   projectID,
		OrgID:       orgID,
		ContentHash: "sha256:" + hex.EncodeToString(contentHash[:]),
		Files:       names,
	}
	if signingKey != "" {
		mac := hmac.New(sha256.New, []byte(signingKey))
		payload, _ := json.Marshal(manifest)
		mac.Write(payload)
		manifest.Signature = "sha256:" + hex.EncodeToString(mac.Sum(nil))
	}

	manifestJSON, _ := json.MarshalIndent(manifest, "", "  ")
	mw, err := zw.Create("manifest.json")
	if err != nil {
		return nil, BundleManifest{}, err
	}
	if _, err := mw.Write(manifestJSON); err != nil {
		return nil, BundleManifest{}, err
	}
	manifest.Files = append(manifest.Files, "manifest.json")

	if err := zw.Close(); err != nil {
		return nil, BundleManifest{}, fmt.Errorf("close zip: %w", err)
	}
	return buf.Bytes(), manifest, nil
}
