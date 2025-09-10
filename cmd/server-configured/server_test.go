package main

import (
    "bytes"
    "encoding/json"
    "io"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/google/uuid"
    "github.com/tendant/simple-content/pkg/simplecontent"
    memoryrepo "github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
    memorystorage "github.com/tendant/simple-content/pkg/simplecontent/storage/memory"
    "github.com/tendant/simple-content/pkg/simplecontent/config"
)

func newTestServer(t *testing.T) *HTTPServer {
    t.Helper()
    repo := memoryrepo.New()
    store := memorystorage.New()
    svc, err := simplecontent.New(
        simplecontent.WithRepository(repo),
        simplecontent.WithBlobStore("memory", store),
        simplecontent.WithBlobStore("default", store),
    )
    if err != nil {
        t.Fatalf("service create error: %v", err)
    }
    cfg := &config.ServerConfig{Environment: "testing", DefaultStorageBackend: "memory"}
    return NewHTTPServer(svc, cfg)
}

func doJSON(t *testing.T, ts *HTTPServer, method, path string, body any) *httptest.ResponseRecorder {
    t.Helper()
    var buf io.Reader
    if body != nil {
        b, err := json.Marshal(body)
        if err != nil {
            t.Fatalf("json marshal: %v", err)
        }
        buf = bytes.NewReader(b)
    }
    req := httptest.NewRequest(method, path, buf)
    req.Header.Set("Content-Type", "application/json")
    rr := httptest.NewRecorder()
    ts.Routes().ServeHTTP(rr, req)
    return rr
}

func TestCreateContentAndList(t *testing.T) {
    ts := newTestServer(t)
    ownerID := uuid.New().String()
    tenantID := uuid.New().String()

    // Create content
    rr := doJSON(t, ts, http.MethodPost, "/api/v1/contents", map[string]any{
        "owner_id": ownerID,
        "tenant_id": tenantID,
        "name": "test content",
        "description": "desc",
        "document_type": "text/plain",
    })
    if rr.Code != http.StatusCreated {
        t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
    }

    // List
    rr = doJSON(t, ts, http.MethodGet, "/api/v1/contents?owner_id="+ownerID+"&tenant_id="+tenantID, nil)
    if rr.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
    }
}

func TestObjectUploadDownload(t *testing.T) {
    ts := newTestServer(t)
    ownerID := uuid.New().String()
    tenantID := uuid.New().String()

    // Create content via API
    rr := doJSON(t, ts, http.MethodPost, "/api/v1/contents", map[string]any{
        "owner_id": ownerID,
        "tenant_id": tenantID,
        "name": "demo",
    })
    if rr.Code != http.StatusCreated {
        t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
    }
    var created struct{ ID string `json:"id"` }
    if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil || created.ID == "" {
        t.Fatalf("invalid create content response: %v, body=%s", err, rr.Body.String())
    }

    // Create object via API
    rr = doJSON(t, ts, http.MethodPost, "/api/v1/contents/"+created.ID+"/objects", map[string]any{"version": 1})
    if rr.Code != http.StatusCreated {
        t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
    }
    var obj struct{ ID string `json:"id"` }
    if err := json.Unmarshal(rr.Body.Bytes(), &obj); err != nil || obj.ID == "" {
        t.Fatalf("invalid create object response: %v, body=%s", err, rr.Body.String())
    }

    // Upload object (direct)
    rr = doRaw(t, ts, http.MethodPost, "/api/v1/objects/"+obj.ID+"/upload", "text/plain", bytes.NewBufferString("hello"))
    if rr.Code != http.StatusNoContent {
        t.Fatalf("expected 204, got %d: %s", rr.Code, rr.Body.String())
    }

    // Download and verify
    req := httptest.NewRequest(http.MethodGet, "/api/v1/objects/"+obj.ID+"/download", nil)
    rec := httptest.NewRecorder()
    ts.Routes().ServeHTTP(rec, req)
    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
    }
    if rec.Body.String() != "hello" {
        t.Fatalf("unexpected download body: %q", rec.Body.String())
    }
}

func TestCreateDerivedContentEndpoint(t *testing.T) {
    ts := newTestServer(t)
    ownerID := uuid.New().String()
    tenantID := uuid.New().String()

    // Create parent content
    rr := doJSON(t, ts, http.MethodPost, "/api/v1/contents", map[string]any{
        "owner_id": ownerID,
        "tenant_id": tenantID,
        "name": "parent",
    })
    if rr.Code != http.StatusCreated {
        t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
    }
    var parent struct{ ID string `json:"id"` }
    _ = json.Unmarshal(rr.Body.Bytes(), &parent)

    // Create derived content
    body := map[string]any{
        "owner_id": ownerID,
        "tenant_id": tenantID,
        "derivation_type": "thumbnail",
        "variant": "thumbnail_256",
        "metadata": map[string]any{"width": 256},
    }
    rr = doJSON(t, ts, http.MethodPost, "/api/v1/contents/"+parent.ID+"/derived", body)
    if rr.Code != http.StatusCreated {
        t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
    }
    var resp struct{ DerivationType string `json:"derivation_type"`; Variant string `json:"variant"` }
    _ = json.Unmarshal(rr.Body.Bytes(), &resp)
    if resp.DerivationType != "thumbnail" || resp.Variant != "thumbnail_256" {
        t.Fatalf("unexpected response: %s", rr.Body.String())
    }
}

func doRaw(t *testing.T, ts *HTTPServer, method, path, contentType string, body io.Reader) *httptest.ResponseRecorder {
    t.Helper()
    req := httptest.NewRequest(method, path, body)
    if contentType != "" {
        req.Header.Set("Content-Type", contentType)
    }
    rr := httptest.NewRecorder()
    ts.Routes().ServeHTTP(rr, req)
    return rr
}
