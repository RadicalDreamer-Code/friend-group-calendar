// Package blob stores files in S3-compatible object storage: RustFS in
// Docker locally, Cloudflare R2 or AWS S3 in production.
package blob

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/config"
)

var ErrNotFound = errors.New("object not found")

type Store interface {
	Put(ctx context.Context, key string, data []byte, contentType string) error
	// Get returns the object's content; the caller closes it.
	Get(ctx context.Context, key string) (io.ReadCloser, int64, error)
	Delete(ctx context.Context, key string) error
}

type S3 struct {
	client *minio.Client
	bucket string
}

func NewS3(ctx context.Context, cfg config.S3Config) (*S3, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("s3 client: %w", err)
	}
	if cfg.CreateBucket {
		if err := ensureBucket(ctx, client, cfg.Bucket, cfg.Region); err != nil {
			return nil, err
		}
	}
	return &S3{client: client, bucket: cfg.Bucket}, nil
}

// ensureBucket creates the bucket if it is missing. In Docker Compose the
// storage container may still be starting, so it retries for a while.
func ensureBucket(ctx context.Context, client *minio.Client, bucket, region string) error {
	deadline := time.Now().Add(30 * time.Second)
	for {
		exists, err := client.BucketExists(ctx, bucket)
		if err == nil && !exists {
			err = client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: region})
		}
		if err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("create bucket %q: %w", bucket, err)
		}
		slog.Info("waiting for object storage", "err", err)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

func (s *S3) Put(ctx context.Context, key string, data []byte, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("put %q: %w", key, err)
	}
	return nil
}

func (s *S3) Get(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, 0, fmt.Errorf("get %q: %w", key, err)
	}
	// GetObject is lazy; Stat performs the request and surfaces a missing key.
	info, err := obj.Stat()
	if err != nil {
		obj.Close()
		if minio.ToErrorResponse(err).Code == minio.NoSuchKey {
			return nil, 0, ErrNotFound
		}
		return nil, 0, fmt.Errorf("stat %q: %w", key, err)
	}
	return obj, info.Size, nil
}

func (s *S3) Delete(ctx context.Context, key string) error {
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("delete %q: %w", key, err)
	}
	return nil
}
