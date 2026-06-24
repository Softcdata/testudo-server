package storage

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
)

type MinIOStorage struct{}

func NewMinIOStorage() *MinIOStorage {
	return &MinIOStorage{}
}

// newS3Client creates a new S3 client with the given credentials
func (s *MinIOStorage) newS3Client(ctx context.Context, endpoint, accessKey, secretKey, region string, addressingStyle dapisv1.StorageRepositoryAddressingStyle, caBundle []byte) (*s3.Client, error) {
	loadOptions := []func(*config.LoadOptions) error{
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	}
	httpClient, err := buildStorageHTTPClient(caBundle)
	if err != nil {
		return nil, err
	}
	if httpClient != nil {
		loadOptions = append(loadOptions, config.WithHTTPClient(httpClient))
	}

	cfg, err := config.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = normalizeAddressingStyle(addressingStyle) != dapisv1.StorageRepositoryAddressingStyleVirtualHostedStyle
	})
	return client, nil
}

func (s *MinIOStorage) GetDownloadURL(ctx context.Context, endpoint, accessKey, secretKey, bucket, region string, addressingStyle dapisv1.StorageRepositoryAddressingStyle, caBundle []byte, objectKey string, expiry time.Duration) (string, error) {
	client, err := s.newS3Client(ctx, endpoint, accessKey, secretKey, region, addressingStyle, caBundle)
	if err != nil {
		return "", err
	}

	presigner := s3.NewPresignClient(client)

	req, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey),
	}, func(o *s3.PresignOptions) {
		o.Expires = expiry
	})
	if err != nil {
		return "", err
	}

	return req.URL, nil
}

func (s *MinIOStorage) GetObject(ctx context.Context, endpoint, accessKey, secretKey, bucket, region string, addressingStyle dapisv1.StorageRepositoryAddressingStyle, caBundle []byte, objectKey, rangeHeader string) (*ObjectStream, error) {
	client, err := s.newS3Client(ctx, endpoint, accessKey, secretKey, region, addressingStyle, caBundle)
	if err != nil {
		return nil, err
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey),
	}
	if strings.TrimSpace(rangeHeader) != "" {
		input.Range = aws.String(strings.TrimSpace(rangeHeader))
	}

	resp, err := client.GetObject(ctx, input)
	if err != nil {
		return nil, err
	}
	if resp.Body == nil {
		return nil, fmt.Errorf("object %s has empty body", objectKey)
	}

	return &ObjectStream{
		Body:          resp.Body,
		Size:          aws.ToInt64(resp.ContentLength),
		ContentType:   aws.ToString(resp.ContentType),
		ContentRange:  aws.ToString(resp.ContentRange),
		AcceptRanges:  aws.ToString(resp.AcceptRanges),
		ETag:          aws.ToString(resp.ETag),
		LastModified:  aws.ToTime(resp.LastModified),
		ContentLength: aws.ToInt64(resp.ContentLength),
	}, nil
}

func (s *MinIOStorage) ListObjects(ctx context.Context, endpoint, accessKey, secretKey, bucket, region string, addressingStyle dapisv1.StorageRepositoryAddressingStyle, caBundle []byte, prefixes []string) ([]ObjectInfo, error) {
	client, err := s.newS3Client(ctx, endpoint, accessKey, secretKey, region, addressingStyle, caBundle)
	if err != nil {
		return nil, err
	}

	var objects []ObjectInfo
	for _, prefix := range prefixes {
		paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
			Bucket: aws.String(bucket),
			Prefix: aws.String(prefix),
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, obj := range page.Contents {
				// Skip directory entries
				if strings.HasSuffix(*obj.Key, "/") {
					continue
				}
				objects = append(objects, ObjectInfo{
					Key:  *obj.Key,
					Size: *obj.Size,
				})
			}
		}
	}

	return objects, nil
}

func normalizeAddressingStyle(style dapisv1.StorageRepositoryAddressingStyle) dapisv1.StorageRepositoryAddressingStyle {
	if style == "" {
		return dapisv1.StorageRepositoryAddressingStylePathStyle
	}
	return style
}

func buildStorageHTTPClient(caBundle []byte) (*http.Client, error) {
	if len(caBundle) == 0 {
		return nil, nil
	}

	rootCAs, err := x509.SystemCertPool()
	if err != nil || rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}
	if ok := rootCAs.AppendCertsFromPEM(caBundle); !ok {
		return nil, fmt.Errorf("failed to append custom CA bundle")
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	} else {
		transport.TLSClientConfig = transport.TLSClientConfig.Clone()
	}
	transport.TLSClientConfig.RootCAs = rootCAs

	return &http.Client{Transport: transport}, nil
}
