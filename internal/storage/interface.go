package storage

import (
	"context"
	"io"
	"time"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
)

// ObjectInfo represents metadata for an S3 object
type ObjectInfo struct {
	Key  string `json:"key"`
	Size int64  `json:"size"`
}

// ObjectStream represents an opened object body and selected response metadata.
type ObjectStream struct {
	Body          io.ReadCloser
	Size          int64
	ContentType   string
	ContentRange  string
	AcceptRanges  string
	ETag          string
	LastModified  time.Time
	ContentLength int64
}

// StorageService defines the interface for interacting with backup storage
type StorageService interface {
	// GetDownloadURL generates a presigned URL for downloading a backup file
	GetDownloadURL(ctx context.Context, endpoint, accessKey, secretKey, bucket, region string, addressingStyle dapisv1.StorageRepositoryAddressingStyle, caBundle []byte, objectKey string, expiry time.Duration) (string, error)
	// ListObjects lists all objects under the given prefixes
	ListObjects(ctx context.Context, endpoint, accessKey, secretKey, bucket, region string, addressingStyle dapisv1.StorageRepositoryAddressingStyle, caBundle []byte, prefixes []string) ([]ObjectInfo, error)
	// GetObject opens an object stream. rangeHeader may be empty or a valid HTTP Range value.
	GetObject(ctx context.Context, endpoint, accessKey, secretKey, bucket, region string, addressingStyle dapisv1.StorageRepositoryAddressingStyle, caBundle []byte, objectKey, rangeHeader string) (*ObjectStream, error)
}
