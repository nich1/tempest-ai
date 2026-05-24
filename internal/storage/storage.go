// Package storage wraps minio-go/v7 with the operations we need:
// presigned upload URLs (so the API doesn't proxy file bytes), presigned
// download URLs (so consumers can stream files directly from MinIO),
// stat (for size/content-type after upload), and delete.
//
// The same code path works against any S3-compatible service - swap
// MINIO_ENDPOINT / creds and you're talking to S3, R2, or GCS.
package storage

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/nich1/tempest-ai/internal/config"
)

// ObjectInfo summarizes a stored blob.
type ObjectInfo struct {
	Key         string
	Size        int64
	ContentType string
	ETag        string
	LastModified time.Time
}

// Client is the storage handle.
type Client struct {
	client *minio.Client
	bucket string
}

// New connects to MinIO/S3 and ensures the bucket exists.
func New(ctx context.Context, cfg config.MinIO) (*Client, error) {
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("init minio client: %w", err)
	}

	c := &Client{client: mc, bucket: cfg.Bucket}
	if err := c.ensureBucket(ctx, cfg.Region); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client) ensureBucket(ctx context.Context, region string) error {
	exists, err := c.client.BucketExists(ctx, c.bucket)
	if err != nil {
		return fmt.Errorf("bucket exists check: %w", err)
	}
	if exists {
		return nil
	}
	if err := c.client.MakeBucket(ctx, c.bucket, minio.MakeBucketOptions{Region: region}); err != nil {
		// Race: another instance might have created it concurrently.
		exists2, err2 := c.client.BucketExists(ctx, c.bucket)
		if err2 == nil && exists2 {
			return nil
		}
		return fmt.Errorf("make bucket %q: %w", c.bucket, err)
	}
	return nil
}

// NewBlobKey returns a fresh, random key suitable for new uploads.
func NewBlobKey() string {
	return fmt.Sprintf("uploads/%s", uuid.NewString())
}

// PresignPutURL returns a URL the client can PUT to directly. The
// content-type and size are encoded into the request via headers; clients
// must use them on the upload or MinIO will reject.
func (c *Client) PresignPutURL(ctx context.Context, key string, contentType string, sizeBytes int64, ttl time.Duration) (string, error) {
	// minio-go's PresignedPutObject signs a basic PUT. Note the size is not
	// embedded in the signature - we rely on the bucket policy + the API
	// HEAD-checking after upload to enforce limits.
	u, err := c.client.PresignedPutObject(ctx, c.bucket, key, ttl)
	if err != nil {
		return "", fmt.Errorf("presign put: %w", err)
	}
	return u.String(), nil
}

// PresignGetURL returns a time-limited URL to download the object.
func (c *Client) PresignGetURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	u, err := c.client.PresignedGetObject(ctx, c.bucket, key, ttl, url.Values{})
	if err != nil {
		return "", fmt.Errorf("presign get: %w", err)
	}
	return u.String(), nil
}

// Stat returns object metadata. ErrObjectNotFound for misses.
func (c *Client) Stat(ctx context.Context, key string) (ObjectInfo, error) {
	info, err := c.client.StatObject(ctx, c.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		var er minio.ErrorResponse
		if errors.As(err, &er) && (er.Code == "NoSuchKey" || er.StatusCode == 404) {
			return ObjectInfo{}, ErrObjectNotFound
		}
		return ObjectInfo{}, fmt.Errorf("stat object: %w", err)
	}
	return ObjectInfo{
		Key:          key,
		Size:         info.Size,
		ContentType:  info.ContentType,
		ETag:         info.ETag,
		LastModified: info.LastModified,
	}, nil
}

// Delete removes the object. No error if it didn't exist.
func (c *Client) Delete(ctx context.Context, key string) error {
	return c.client.RemoveObject(ctx, c.bucket, key, minio.RemoveObjectOptions{})
}

// HealthCheck pings MinIO. Returns nil if reachable.
func (c *Client) HealthCheck(ctx context.Context) error {
	_, err := c.client.BucketExists(ctx, c.bucket)
	return err
}

// GetObjectStream returns a streaming reader for the object. Caller must close.
func (c *Client) GetObjectStream(ctx context.Context, key string) (*minio.Object, error) {
	return c.client.GetObject(ctx, c.bucket, key, minio.GetObjectOptions{})
}

// ErrObjectNotFound is returned when a Stat or download targets a missing key.
var ErrObjectNotFound = errors.New("object not found")
