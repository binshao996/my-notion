package file

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Service struct {
	Client   *minio.Client
	Bucket   string
	Endpoint string
}

func NewService() (*Service, error) {
	endpoint := os.Getenv("MINIO_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:9000"
	}

	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	if accessKey == "" {
		accessKey = "minioadmin"
	}

	secretKey := os.Getenv("MINIO_SECRET_KEY")
	if secretKey == "" {
		secretKey = "minioadmin"
	}

	bucket := os.Getenv("MINIO_BUCKET")
	if bucket == "" {
		bucket = "my-notion"
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, err
	}

	// Ensure bucket exists
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return &Service{
		Client:   client,
		Bucket:   bucket,
		Endpoint: endpoint,
	}, nil
}

func (s *Service) GetUploadURL(filename, contentType string) (string, string, error) {
	// Generate a unique key
	key := fmt.Sprintf("%d-%s", time.Now().UnixNano(), filename)

	ctx := context.Background()
	url, err := s.Client.PresignedPutObject(ctx, s.Bucket, key, 15*time.Minute)
	if err != nil {
		return "", "", err
	}

	// The public URL for the uploaded file
	publicURL := fmt.Sprintf("http://%s/%s/%s", s.Endpoint, s.Bucket, key)
	return url.String(), publicURL, nil
}
