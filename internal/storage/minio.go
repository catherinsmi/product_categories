package storage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Storage — обёртка над minio-go.
type Storage struct {
	client *minio.Client
	bucket string
}

// New создаёт клиента, проверяет существование бакета. Если бакета нет — возвращает ошибку.
func New(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*Storage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client init: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("minio bucket exists check: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("minio bucket %q does not exist (создайте его через `docker compose up -d` или `go run ./cmd/seed`)", bucket)
	}

	return &Storage{client: client, bucket: bucket}, nil
}

// Stat — метаданные объекта.
func (s *Storage) Stat(ctx context.Context, key string) (minio.ObjectInfo, error) {
	return s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
}

// Get — поток объекта.
func (s *Storage) Get(ctx context.Context, key string) (*minio.Object, error) {
	return s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
}

// Serve — Stat → Get → io.Copy с нужными заголовками.
// При ошибке Stat ничего в ResponseWriter не пишет, чтобы хендлер мог отдать 404.
func (s *Storage) Serve(w http.ResponseWriter, r *http.Request, key string) error {
	ctx := r.Context()

	info, err := s.Stat(ctx, key)
	if err != nil {
		return fmt.Errorf("stat %q: %w", key, err)
	}

	obj, err := s.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("get %q: %w", key, err)
	}
	defer obj.Close()

	ct := info.ContentType
	if ct == "" {
		ct = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ct)
	if info.Size > 0 {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size))
	}
	if info.ETag != "" {
		w.Header().Set("ETag", info.ETag)
	}
	w.Header().Set("Cache-Control", "public, max-age=3600")

	if _, err := io.Copy(w, obj); err != nil {
		return fmt.Errorf("copy %q: %w", key, err)
	}
	return nil
}
