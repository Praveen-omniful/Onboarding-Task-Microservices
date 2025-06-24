package upload

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/omniful/go_commons/uploads"
)

// Service provides upload functionality using go_commons/uploads
type Service struct {
	tempUploadService *uploads.TempUploadService
	// Using interface{} for now since we don't know the exact type
	ephemeralClient interface{}
	tempBucket      string
	permanentBucket string
	region          string
}

// Config contains the configuration for the upload service
type Config struct {
	TempBucket      string
	PermanentBucket string
	Region          string
}

// NewService creates a new upload service using go_commons/uploads
func NewService(cfg Config) (*Service, error) {
	// Initialize the temporary upload service for generating pre-signed URLs
	tempUploadService, err := uploads.NewTempUploadService(cfg.TempBucket, cfg.Region)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp upload service: %w", err)
	}

	// Initialize the ephemeral client for post-upload operations
	ephemeralClient, err := uploads.NewEphemeralDownloadClient(cfg.TempBucket)
	if err != nil {
		return nil, fmt.Errorf("failed to create ephemeral download client: %w", err)
	}

	return &Service{
		tempUploadService: tempUploadService,
		ephemeralClient:   ephemeralClient,
		tempBucket:        cfg.TempBucket,
		permanentBucket:   cfg.PermanentBucket,
		region:            cfg.Region,
	}, nil
}

// GenerateUploadURL creates a pre-signed URL for frontend uploads
func (s *Service) GenerateUploadURL(tenant, useCase, contentType, filename string) (*uploads.TempURLResponse, error) {
	return s.tempUploadService.GenerateTempURL(tenant, useCase, contentType, filename)
}

// DownloadUploadedFile downloads a file from the temporary bucket
func (s *Service) DownloadUploadedFile(ctx context.Context, uploadID string, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	err = s.ephemeralClient.DownloadObject(ctx, &uploads.DownloadObjectInput{
		EphemeralUploadID: uploadID,
		File:              file,
	})
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	return nil
}

// DownloadToReader downloads a file from the temporary bucket to an io.Writer
func (s *Service) DownloadToWriter(ctx context.Context, uploadID string, writer io.Writer) error {
	err := s.ephemeralClient.DownloadObject(ctx, &uploads.DownloadObjectInput{
		EphemeralUploadID: uploadID,
		File:              writer,
	})
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	return nil
}

// TransferToPermanentStorage transfers a file from temporary to permanent storage
func (s *Service) TransferToPermanentStorage(ctx context.Context, uploadID, directory string) (*uploads.UploadResponse, error) {
	response, err := s.ephemeralClient.Upload(
		ctx,
		s.permanentBucket,
		uploadID,
		directory,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to transfer file to permanent storage: %w", err)
	}

	return response, nil
}
