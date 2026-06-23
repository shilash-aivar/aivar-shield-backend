package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func splitCSV(v string) []string {
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

type Config struct {
	Port                  string
	DatabaseURL           string
	GitHubClientID        string
	GitHubClientSecret    string
	SessionSecret         string
	UIURL                 string
	APIPublicURL          string
	AdminGitHubLogins     []string
	ApproverGitHubLogins  []string
	SlackWebhookURL       string
	SMTPHost              string
	SMTPPort              string
	SMTPUser              string
	SMTPPassword          string
	EmailFrom             string
	NotifyEmails          []string
	S3Bucket              string
	S3Region              string
	SuppressionSigningKey string
}

func (c Config) AuthEnabled() bool {
	return c.GitHubClientID != "" && c.GitHubClientSecret != ""
}

func Load() Config {
	_ = godotenv.Load()

	port := os.Getenv("AIVAR_API_PORT")
	if port == "" {
		port = "8080"
	}

	return Config{
		Port:                  port,
		DatabaseURL:           os.Getenv("AIVAR_DATABASE_URL"),
		GitHubClientID:        os.Getenv("AIVAR_GITHUB_CLIENT_ID"),
		GitHubClientSecret:    os.Getenv("AIVAR_GITHUB_CLIENT_SECRET"),
		SessionSecret:         envOr("AIVAR_SESSION_SECRET", "dev-session-secret-change-me"),
		UIURL:                 envOr("AIVAR_UI_URL", "http://localhost:5173"),
		APIPublicURL:          envOr("AIVAR_API_URL", "http://localhost:8080"),
		AdminGitHubLogins:     splitCSV(os.Getenv("AIVAR_ADMIN_GITHUB_LOGINS")),
		ApproverGitHubLogins:  splitCSV(os.Getenv("AIVAR_APPROVER_GITHUB_LOGINS")),
		SlackWebhookURL:       os.Getenv("AIVAR_SLACK_WEBHOOK_URL"),
		SMTPHost:              os.Getenv("AIVAR_SMTP_HOST"),
		SMTPPort:              envOr("AIVAR_SMTP_PORT", "587"),
		SMTPUser:              os.Getenv("AIVAR_SMTP_USER"),
		SMTPPassword:          os.Getenv("AIVAR_SMTP_PASSWORD"),
		EmailFrom:             os.Getenv("AIVAR_EMAIL_FROM"),
		NotifyEmails:          splitCSV(os.Getenv("AIVAR_NOTIFY_EMAILS")),
		S3Bucket:              os.Getenv("AIVAR_S3_BUCKET"),
		S3Region:              os.Getenv("AIVAR_S3_REGION"),
		SuppressionSigningKey: os.Getenv("AIVAR_SUPPRESSION_SIGNING_KEY"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
