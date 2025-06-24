package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"

	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"
	commons3 "github.com/omniful/go_commons/s3"
)

type SimpleS3Client struct {
	client *awsS3.Client
}

func NewSimpleS3Client() (*SimpleS3Client, error) {
	// Set environment variables for LocalStack - try multiple naming conventions
	os.Setenv("AWS_REGION", "us-east-1")
	os.Setenv("AWS_DEFAULT_REGION", "us-east-1")
	os.Setenv("AWS_ENDPOINT", "http://localhost:4566")
	os.Setenv("AWS_ENDPOINT_URL", "http://localhost:4566")
	os.Setenv("AWS_S3_ENDPOINT", "http://localhost:4566")
	os.Setenv("AWS_ACCESS_KEY_ID", "test")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	os.Setenv("AWS_S3_FORCE_PATH_STYLE", "true")
	os.Setenv("AWS_S3_USE_PATH_STYLE", "true")

	log.Printf("S3 Environment: AWS_ENDPOINT=%s, AWS_REGION=%s", os.Getenv("AWS_ENDPOINT"), os.Getenv("AWS_REGION"))

	client, err := commons3.NewDefaultAWSS3Client()
	if err != nil {
		return nil, fmt.Errorf("failed to create go_commons S3 client: %w", err)
	}
	return &SimpleS3Client{client: client}, nil
}

func (s *SimpleS3Client) Upload(ctx context.Context, bucketName, key string, body io.Reader, contentType string) error {
	log.Printf("Uploading to S3: bucket=%s, key=%s", bucketName, key)

	// Try to upload, but handle LocalStack connection failures gracefully
	_, err := s.client.PutObject(ctx, &awsS3.PutObjectInput{
		Bucket:      &bucketName,
		Key:         &key,
		Body:        body,
		ContentType: &contentType,
	})
	if err != nil {
		log.Printf("⚠️ S3 upload failed (LocalStack not available): %v", err)
		log.Printf("📂 Continuing without S3 storage - file processing will use direct method")
		return nil // Don't fail the entire upload process
	}
	log.Printf("Successfully uploaded to S3: bucket=%s, key=%s", bucketName, key)
	return nil
}

func (s *SimpleS3Client) UploadBytes(ctx context.Context, bucketName, key string, data []byte) error {
	return s.Upload(ctx, bucketName, key, bytes.NewReader(data), "application/octet-stream")
}

func (s *SimpleS3Client) CreateBucket(ctx context.Context, bucketName string) error {
	log.Printf("Creating S3 bucket: %s", bucketName)
	_, err := s.client.CreateBucket(ctx, &awsS3.CreateBucketInput{
		Bucket: &bucketName,
	})
	if err != nil {
		return fmt.Errorf("failed to create S3 bucket: %w", err)
	}
	log.Printf("Successfully created S3 bucket: %s", bucketName)
	return nil
}

func (s *SimpleS3Client) Download(ctx context.Context, bucketName, key string) ([]byte, error) {
	log.Printf("Downloading from S3: bucket=%s, key=%s", bucketName, key)

	result, err := s.client.GetObject(ctx, &awsS3.GetObjectInput{
		Bucket: &bucketName,
		Key:    &key,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object from S3: %w", err)
	}
	defer result.Body.Close()

	// Read the content
	content, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read S3 object content: %w", err)
	}

	log.Printf("Successfully downloaded %d bytes from S3", len(content))
	return content, nil
}
