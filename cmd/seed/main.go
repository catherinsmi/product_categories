package main

import (
	"context"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Запускать из корня проекта: go run ./cmd/seed
// Читает файлы из resources/seed/* и загружает их в бакет под ключами seed/<filename>.
func main() {
	_ = godotenv.Load()

	endpoint := env("MINIO_ENDPOINT", "localhost:9000")
	accessKey := env("MINIO_ACCESS_KEY", "admin")
	secretKey := env("MINIO_SECRET_KEY", "admin12345")
	bucket := env("MINIO_BUCKET", "metoda-media")
	useSSL := env("MINIO_USE_SSL", "false") == "true"
	dir := env("SEED_DIR", "resources/seed")

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalf("minio client: %v", err)
	}

	ctx := context.Background()

	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		log.Fatalf("bucket exists check: %v", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			log.Fatalf("make bucket %q: %v", bucket, err)
		}
		log.Printf("bucket %q created", bucket)
	} else {
		log.Printf("bucket %q already exists", bucket)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Fatalf("read dir %q: %v", dir, err)
	}

	uploaded := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}

		ext := strings.ToLower(filepath.Ext(name))
		contentType := contentTypeForExt(ext)
		if contentType == "" {
			log.Printf("skip %s: unsupported extension %q", name, ext)
			continue
		}

		fullPath := filepath.Join(dir, name)
		f, err := os.Open(fullPath)
		if err != nil {
			log.Printf("open %s: %v", fullPath, err)
			continue
		}
		info, err := f.Stat()
		if err != nil {
			log.Printf("stat %s: %v", fullPath, err)
			f.Close()
			continue
		}

		key := "seed/" + name
		_, err = client.PutObject(ctx, bucket, key, f, info.Size(), minio.PutObjectOptions{
			ContentType: contentType,
		})
		f.Close()
		if err != nil {
			log.Printf("upload %s: %v", name, err)
			continue
		}
		log.Printf("uploaded %-24s -> %s (%s, %d bytes)", name, key, contentType, info.Size())
		uploaded++
	}

	log.Printf("done: %d file(s) uploaded to bucket %q", uploaded, bucket)
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func contentTypeForExt(ext string) string {
	switch ext {
	case ".jpg", ".jpeg":
		if ct := mime.TypeByExtension(ext); ct != "" {
			return ct
		}
		return "image/jpeg"
	case ".png":
		if ct := mime.TypeByExtension(ext); ct != "" {
			return ct
		}
		return "image/png"
	case ".mp4":
		if ct := mime.TypeByExtension(ext); ct != "" {
			return ct
		}
		return "video/mp4"
	case ".webm":
		if ct := mime.TypeByExtension(ext); ct != "" {
			return ct
		}
		return "video/webm"
	}
	return ""
}
