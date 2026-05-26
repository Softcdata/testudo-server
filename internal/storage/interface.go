package storage

import (
	"context"
	"time"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
)

// ObjectInfo represents metadata for an S3 object
type ObjectInfo struct {
	Key  string `json:"key"`
	Size int64  `json:"size"`
}

// StorageService defines the interface for interacting with backup storage
type StorageService interface {
	// GetDownloadURL generates a presigned URL for downloading a backup file
	GetDownloadURL(ctx context.Context, endpoint, accessKey, secretKey, bucket, region string, addressingStyle dapisv1.StorageRepositoryAddressingStyle, caBundle []byte, objectKey string, expiry time.Duration) (string, error)
	// ListObjects lists all objects under the given prefixes
	ListObjects(ctx context.Context, endpoint, accessKey, secretKey, bucket, region string, addressingStyle dapisv1.StorageRepositoryAddressingStyle, caBundle []byte, prefixes []string) ([]ObjectInfo, error)
}
