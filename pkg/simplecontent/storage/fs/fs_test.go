package fs

import (
    "bytes"
    "context"
    "io"
    "os"
    "path/filepath"
    "testing"
)

func TestFSBackend_BasicOps(t *testing.T) {
    tmp := t.TempDir()
    b, err := New(Config{BaseDir: tmp})
    if err != nil {
        t.Fatalf("new fs backend: %v", err)
    }
    backend := b

    ctx := context.Background()
    key := "C/parent/child/file.txt"

    // Upload
    data := []byte("hello fs")
    if err := backend.Upload(ctx, key, bytes.NewReader(data)); err != nil {
        t.Fatalf("upload: %v", err)
    }

    // GetObjectMeta
    meta, err := backend.GetObjectMeta(ctx, key)
    if err != nil {
        t.Fatalf("get meta: %v", err)
    }
    if meta.Size <= 0 {
        t.Fatalf("expected size > 0, got %d", meta.Size)
    }

    // Download
    rc, err := backend.Download(ctx, key)
    if err != nil {
        t.Fatalf("download: %v", err)
    }
    got, _ := io.ReadAll(rc)
    _ = rc.Close()
    if string(got) != string(data) {
        t.Fatalf("download mismatch: %q", string(got))
    }

    // Delete
    if err := backend.Delete(ctx, key); err != nil {
        t.Fatalf("delete: %v", err)
    }
    // Ensure file removed
    if _, err := os.Stat(filepath.Join(tmp, key)); !os.IsNotExist(err) {
        t.Fatalf("expected file removed, stat err=%v", err)
    }
}

func TestFSBackend_URLMethods_NoPrefix(t *testing.T) {
    tmp := t.TempDir()
    b, err := New(Config{BaseDir: tmp})
    if err != nil {
        t.Fatalf("new fs backend: %v", err)
    }
    backend := b
    ctx := context.Background()
    if _, err := backend.GetUploadURL(ctx, "a/b"); err == nil {
        t.Fatalf("expected error without urlPrefix")
    }
    if _, err := backend.GetDownloadURL(ctx, "a/b", ""); err == nil {
        t.Fatalf("expected error without urlPrefix")
    }
    if _, err := backend.GetPreviewURL(ctx, "a/b"); err == nil {
        t.Fatalf("expected error without urlPrefix")
    }
}

