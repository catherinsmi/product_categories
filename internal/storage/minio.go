package storage

import (
	"context"
	"fmt"
	"mime/multipart"
	"strings"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Storage struct {
	client *minio.Client
	bucket string
}

func New(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*Storage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client init: %w", err)
	}

	// Создаем бакет, если его нет
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("minio bucket check: %w", err)
	}
	if !exists {
		err = client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("minio make bucket: %w", err)
		}
	}

	return &Storage{client: client, bucket: bucket}, nil
}

// UploadFile загружает файл в MinIO и возвращает сгенерированный ключ (имя)
func (s *Storage) UploadFile(file *multipart.FileHeader) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	// Генерируем уникальное имя на латинице + расширение
	ext := file.Filename[strings.LastIndex(file.Filename, "."):]
	objectKey := fmt.Sprintf("%s%s", uuid.New().String(), ext)

	// Определяем Content-Type (упрощенно)
	contentType := "application/octet-stream"
	if strings.HasSuffix(objectKey, ".jpg") || strings.HasSuffix(objectKey, ".jpeg") {
		contentType = "image/jpeg"
	} else if strings.HasSuffix(objectKey, ".png") {
		contentType = "image/png"
	} else if strings.HasSuffix(objectKey, ".mp4") {
		contentType = "video/mp4"
	}

	ctx := context.Background()
	_, err = s.client.PutObject(ctx, s.bucket, objectKey, src, file.Size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}

	return objectKey, nil
}

// GetFileURL возвращает публичную ссылку на файл
func (s *Storage) GetFileURL(objectKey string) string {
	if objectKey == "" {
		return ""
	}
	// В реальном проекте тут может быть presigned URL, но для лабы сойдет прямая ссылка (если бакет публичный)
	// Или используем прокси-хендлер.
	return fmt.Sprintf("http://localhost:9000/%s/%s", s.bucket, objectKey)
}
