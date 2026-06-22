package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                  string
	DatabaseURL           string
	GitHubClientID        string
	GitHubClientSecret    string
	SlackWebhookURL       string
	S3Bucket              string
	S3Region              string
	SuppressionSigningKey string
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
		SlackWebhookURL:       os.Getenv("AIVAR_SLACK_WEBHOOK_URL"),
		S3Bucket:              os.Getenv("AIVAR_S3_BUCKET"),
		S3Region:              os.Getenv("AIVAR_S3_REGION"),
		SuppressionSigningKey: os.Getenv("AIVAR_SUPPRESSION_SIGNING_KEY"),
	}
}
