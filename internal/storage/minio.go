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

// Storage — обёртка над minio-go. Может быть "выключенным":
// тогда Enabled() == false, а Serve() возвращает ошибку, и хендлер
// переключается на локальный fallback.
type Storage struct {
	client  *minio.Client
	bucket  string
	enabled bool
}

// New создаёт клиента. Если MinIO недоступен или бакета нет —
// возвращает хранилище с enabled=false и ошибку (для лога),
// но сам объект валидный, чтобы сервер мог стартовать.
func New(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*Storage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return &Storage{enabled: false}, fmt.Errorf("minio client init: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return &Storage{client: client, bucket: bucket, enabled: false},
			fmt.Errorf("minio bucket exists check: %w", err)
	}
	if !exists {
		return &Storage{client: client, bucket: bucket, enabled: false},
			fmt.Errorf("minio bucket %q does not exist", bucket)
	}

	return &Storage{client: client, bucket: bucket, enabled: true}, nil
}

// Enabled — true, если MinIO доступен и бакет найден.
func (s *Storage) Enabled() bool {
	return s != nil && s.enabled
}

// Stat — метаданные объекта.
func (s *Storage) Stat(ctx context.Context, key string) (minio.ObjectInfo, error) {
	if !s.Enabled() {
		return minio.ObjectInfo{}, fmt.Errorf("minio disabled")
	}
	return s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
}

// Get — поток объекта.
func (s *Storage) Get(ctx context.Context, key string) (*minio.Object, error) {
	if !s.Enabled() {
		return nil, fmt.Errorf("minio disabled")
	}
	return s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
}

// Serve — Stat → Get → io.Copy с заголовками. При любой ошибке
// в ResponseWriter ничего не пишется — вызывающий сам решает, что делать.
func (s *Storage) Serve(w http.ResponseWriter, r *http.Request, key string) error {
	if !s.Enabled() {
		return fmt.Errorf("minio disabled")
	}

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
