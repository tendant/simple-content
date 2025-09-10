//go:build integration

package integration

import (
    "bytes"
    "context"
    "net/url"
    "os"
    "testing"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/google/uuid"
    simplecontent "github.com/tendant/simple-content/pkg/simplecontent"
    repopg "github.com/tendant/simple-content/pkg/simplecontent/repo/postgres"
    s3storage "github.com/tendant/simple-content/pkg/simplecontent/storage/s3"
)

func TestIntegration_Postgres_MinIO(t *testing.T) {
    // Postgres
    pgURL := getenv("DATABASE_URL", "postgres://content:pwd@localhost:5432/powercard_db?sslmode=disable")
    pool, err := pgxpool.New(context.Background(), pgURL)
    if err != nil {
        t.Skipf("postgres not available: %v", err)
    }
    defer pool.Close()

    // Ensure schema exists (assumes 'content' schema)
    if _, err := pool.Exec(context.Background(), "CREATE SCHEMA IF NOT EXISTS content"); err != nil {
        t.Fatalf("create schema: %v", err)
    }

    repo := repopg.NewWithPool(pool)

    // MinIO/S3
    endpoint := getenv("S3_ENDPOINT", "http://localhost:9000")
    if _, err := url.Parse(endpoint); err != nil {
        t.Skipf("minio endpoint invalid: %v", err)
    }
    store, err := s3storage.New(s3storage.Config{
        Region:          getenv("S3_REGION", "us-east-1"),
        Bucket:          getenv("S3_BUCKET", "content-bucket"),
        AccessKeyID:     getenv("S3_ACCESS_KEY_ID", "minioadmin"),
        SecretAccessKey: getenv("S3_SECRET_ACCESS_KEY", "minioadmin"),
        Endpoint:        endpoint,
        UseSSL:          false,
        UsePathStyle:    true,
        CreateBucketIfNotExist: true,
    })
    if err != nil {
        t.Skipf("minio not available: %v", err)
    }

    // Build service
    svc, err := simplecontent.New(
        simplecontent.WithRepository(repo),
        simplecontent.WithBlobStore("s3", store),
    )
    if err != nil { t.Fatalf("service: %v", err) }

    ctx := context.Background()

    // Create content and object, upload/download roundtrip
    content, err := svc.CreateContent(ctx, simplecontent.CreateContentRequest{OwnerID: uuid.New(), TenantID: uuid.New(), Name: "it"})
    if err != nil { t.Fatalf("create content: %v", err) }

    obj, err := svc.CreateObject(ctx, simplecontent.CreateObjectRequest{ContentID: content.ID, StorageBackendName: "s3", Version: 1})
    if err != nil { t.Fatalf("create object: %v", err) }

    if err := svc.UploadObject(ctx, obj.ID, bytes.NewBufferString("hello")); err != nil {
        t.Fatalf("upload: %v", err)
    }

    rc, err := svc.DownloadObject(ctx, obj.ID)
    if err != nil { t.Fatalf("download: %v", err) }
    _ = rc.Close()
}

func getenv(k, d string) string { if v := os.Getenv(k); v != "" { return v }; return d }

