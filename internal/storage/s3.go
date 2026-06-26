package storage

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Store struct {
	bucket string
	client *s3.Client
}

func NewS3Store(bucket, region string) *S3Store {
	if bucket == "" {
		return &S3Store{}
	}
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return &S3Store{bucket: bucket}
	}
	return &S3Store{bucket: bucket, client: s3.NewFromConfig(cfg)}
}

func (s *S3Store) Enabled() bool {
	return s != nil && s.bucket != "" && s.client != nil
}

func (s *S3Store) Put(ctx context.Context, key string, data []byte) (string, error) {
	if !s.Enabled() {
		return "", fmt.Errorf("s3 client not available")
	}
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/zip"),
	})
	if err != nil {
		return "", fmt.Errorf("s3 put: %w", err)
	}
	if endpoint := os.Getenv("AIVAR_S3_PUBLIC_URL"); endpoint != "" {
		return fmt.Sprintf("%s/%s/%s", strings.TrimSuffix(endpoint, "/"), s.bucket, key), nil
	}
	presign := s3.NewPresignClient(s.client)
	out, err := presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(7*24*time.Hour))
	if err != nil {
		return fmt.Sprintf("s3://%s/%s", s.bucket, key), nil
	}
	return out.URL, nil
}
