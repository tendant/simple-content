// Package main demonstrates how to use the presigned package as a standalone library
// in your own Go application without using the full simple-content service.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/tendant/simple-content/pkg/simplecontent/presigned"
)

// Config holds application configuration
type Config struct {
	SecretKey     string
	StorageDir    string
	Port          string
	MaxUploadSize int64
}

// Server wraps the HTTP server with presigned upload support
type Server struct {
	signer     *presigned.Signer
	config     Config
	uploadDir  string
}

func main() {
	// Load configuration
	config := Config{
		SecretKey:     getEnv("SECRET_KEY", generateSecretKey()),
		StorageDir:    getEnv("STORAGE_DIR", "./uploads"),
		Port:          getEnv("PORT", "8080"),
		MaxUploadSize: 100 * 1024 * 1024, // 100MB
	}

	log.Printf("Starting presigned upload server on port %s", config.Port)
	log.Printf("Storage directory: %s", config.StorageDir)
	log.Printf("Secret key: %s... (first 10 chars)", config.SecretKey[:10])

	// Create server
	srv, err := NewServer(config)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start HTTP server
	log.Printf("Server ready at http://localhost:%s", config.Port)
	log.Printf("Try: curl http://localhost:%s/demo", config.Port)
	if err := http.ListenAndServe(":"+config.Port, srv.Routes()); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// NewServer creates a new server instance
func NewServer(config Config) (*Server, error) {
	// Ensure storage directory exists
	if err := os.MkdirAll(config.StorageDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Create presigned signer
	signer := presigned.New(
		presigned.WithSecretKey(config.SecretKey),
		presigned.WithDefaultExpiration(15*time.Minute),
		presigned.WithURLPattern("/upload/{key}"),
	)

	return &Server{
		signer:    signer,
		config:    config,
		uploadDir: config.StorageDir,
	}, nil
}

// Routes sets up HTTP routes
func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	// API routes
	r.Get("/", s.handleIndex)
	r.Get("/demo", s.handleDemo)
	r.Post("/get-upload-url", s.handleGetUploadURL)
	r.Get("/files", s.handleListFiles)
	r.Get("/download/{filename}", s.handleDownload)

	// Upload endpoint with presigned validation
	r.Put("/upload/*", presigned.ValidateHandler(s.config.SecretKey, s.handleUpload))

	return r
}

// handleIndex serves a simple HTML page
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Presigned Upload Demo</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        button { background: #007bff; color: white; padding: 10px 20px; border: none; cursor: pointer; }
        button:hover { background: #0056b3; }
        .result { margin-top: 20px; padding: 10px; border-radius: 4px; }
        .success { background: #d4edda; color: #155724; }
        .error { background: #f8d7da; color: #721c24; }
    </style>
</head>
<body>
    <h1>Presigned Upload Demo</h1>
    <p>This demonstrates the <code>presigned</code> package as a standalone library.</p>

    <h2>Upload File</h2>
    <input type="file" id="fileInput">
    <button onclick="uploadFile()">Upload</button>

    <div id="result"></div>

    <h2>API Endpoints</h2>
    <ul>
        <li><code>POST /get-upload-url</code> - Get presigned upload URL</li>
        <li><code>PUT /upload/{filename}</code> - Upload file (presigned)</li>
        <li><code>GET /files</code> - List uploaded files</li>
        <li><code>GET /download/{filename}</code> - Download file</li>
        <li><code>GET /demo</code> - Run demo workflow</li>
    </ul>

    <script>
    async function uploadFile() {
        const fileInput = document.getElementById('fileInput');
        const file = fileInput.files[0];
        if (!file) {
            alert('Please select a file');
            return;
        }

        try {
            // Step 1: Get presigned URL
            const urlResponse = await fetch('/get-upload-url', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ filename: file.name })
            });
            const { upload_url } = await urlResponse.json();

            // Step 2: Upload to presigned URL
            const uploadResponse = await fetch(upload_url, {
                method: 'PUT',
                body: file
            });

            if (uploadResponse.ok) {
                document.getElementById('result').innerHTML =
                    '<div class="result success">Upload successful!</div>';
            } else {
                throw new Error('Upload failed: ' + uploadResponse.statusText);
            }
        } catch (error) {
            document.getElementById('result').innerHTML =
                '<div class="result error">Error: ' + error.message + '</div>';
        }
    }
    </script>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// handleGetUploadURL generates a presigned upload URL
func (s *Server) handleGetUploadURL(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Filename string `json:"filename"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Filename == "" {
		http.Error(w, "filename is required", http.StatusBadRequest)
		return
	}

	// Clean filename
	filename := filepath.Base(req.Filename)

	// Generate presigned URL
	baseURL := fmt.Sprintf("http://localhost:%s", s.config.Port)
	uploadURL, err := s.signer.SignURLWithBase(
		baseURL,
		"PUT",
		"/upload/"+filename,
		15*time.Minute,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"upload_url": uploadURL,
		"filename":   filename,
		"expires_in": "900", // 15 minutes
	})
}

// handleUpload handles validated upload requests
func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	// Extract validated filename from context
	filename := presigned.ObjectKeyFromContext(r.Context())
	if filename == "" {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	// Check file size
	if r.ContentLength > s.config.MaxUploadSize {
		http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
		return
	}

	// Save file
	filePath := filepath.Join(s.uploadDir, filename)
	file, err := os.Create(filePath)
	if err != nil {
		log.Printf("Failed to create file: %v", err)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	_, err = io.Copy(file, r.Body)
	if err != nil {
		log.Printf("Failed to write file: %v", err)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	log.Printf("File uploaded successfully: %s", filename)
	w.WriteHeader(http.StatusOK)
}

// handleListFiles lists uploaded files
func (s *Server) handleListFiles(w http.ResponseWriter, r *http.Request) {
	files, err := os.ReadDir(s.uploadDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fileList := []map[string]interface{}{}
	for _, file := range files {
		if !file.IsDir() {
			info, _ := file.Info()
			fileList = append(fileList, map[string]interface{}{
				"name": file.Name(),
				"size": info.Size(),
				"modified": info.ModTime().Format(time.RFC3339),
			})
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"files": fileList,
		"count": len(fileList),
	})
}

// handleDownload downloads a file
func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	filename = filepath.Base(filename) // Security: prevent directory traversal

	filePath := filepath.Join(s.uploadDir, filename)
	http.ServeFile(w, r, filePath)
}

// handleDemo runs a complete demo workflow
func (s *Server) handleDemo(w http.ResponseWriter, r *http.Request) {
	// Generate presigned URL
	baseURL := fmt.Sprintf("http://localhost:%s", s.config.Port)
	uploadURL, _ := s.signer.SignURLWithBase(
		baseURL,
		"PUT",
		"/upload/demo.txt",
		15*time.Minute,
	)

	// Simulate client upload
	client := presigned.NewClient()
	demoData := []byte("Hello from presigned package demo!")

	// Create a reader with progress tracking
	progressClient := presigned.NewClient(
		presigned.WithProgress(func(bytes int64) {
			log.Printf("Upload progress: %d bytes", bytes)
		}),
	)

	err := progressClient.Upload(
		context.Background(),
		uploadURL,
		io.NopCloser(bytes.NewReader(demoData)),
		presigned.WithContentType("text/plain"),
	)

	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"message":     "Demo completed successfully",
		"upload_url":  uploadURL,
		"uploaded":    "demo.txt",
		"client_used": "presigned.Client with progress tracking",
	})
}

// Utility functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func generateSecretKey() string {
	// In production, generate with crypto/rand
	// For demo purposes, use a simple key
	return "demo-secret-key-min-32-chars-abcdef0123456789"
}

// Additional import needed
import "bytes"
