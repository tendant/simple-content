package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/tendant/simple-content/pkg/simplecontent"
	"github.com/tendant/simple-content/pkg/simplecontent/config"
	memoryrepo "github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
	memorystorage "github.com/tendant/simple-content/pkg/simplecontent/storage/memory"
)

// This example shows how to use simple-content as a LIBRARY in your own application
// Notice: We don't use Port or Environment from config - those are for cmd/server-configured only

func main() {
	fmt.Println("=== Library Usage Example ===")

	// Option 1: Direct service creation (most control)
	fmt.Println("Option 1: Direct Service Creation")
	svc1 := createServiceDirectly()
	testService(svc1, "direct-creation")
	fmt.Println()

	// Option 2: Use config.Load for convenience (recommended for library users)
	fmt.Println("Option 2: Using config.Load() for convenience")
	svc2 := createServiceViaConfig()
	testService(svc2, "via-config")
	fmt.Println()

	// Option 3: Embed in your own HTTP server
	fmt.Println("Option 3: Embedded in Custom HTTP Server")
	runCustomHTTPServer(svc1)
}

// Option 1: Direct service creation (most explicit)
func createServiceDirectly() simplecontent.Service {
	// Create components manually
	repo := memoryrepo.New()
	store := memorystorage.New()

	// Build service with explicit options
	svc, err := simplecontent.New(
		simplecontent.WithRepository(repo),
		simplecontent.WithBlobStore("memory", store),
		// Add other options as needed
	)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	fmt.Println("   ✓ Service created directly with explicit options")
	return svc
}

// Option 2: Use config package for convenience (library users can ignore Port/Environment)
func createServiceViaConfig() simplecontent.Service {
	// Build configuration using functional options
	// Note: Port and Environment are ignored - we run our own HTTP server
	cfg, err := config.Load(
		// Service-level configuration (what we care about)
		config.WithDatabase("memory", ""),
		config.WithMemoryStorage(""),
		config.WithDefaultStorage("memory"),
		config.WithContentBasedURLs("/api/v1"),
		config.WithEventLogging(true),
		config.WithPreviews(true),
		config.WithObjectKeyGenerator("git-like"),

		// Server-level configuration (ignored for library usage)
		// config.WithPort("8080"),        // ← IGNORED: We control our own server port
		// config.WithEnvironment("dev"),  // ← IGNORED: We manage our own environment
	)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Build service from configuration
	svc, err := cfg.BuildService()
	if err != nil {
		log.Fatalf("Failed to build service: %v", err)
	}

	fmt.Println("   ✓ Service created via config.Load() (ignoring Port/Environment)")
	return svc
}

// Test the service works
func testService(svc simplecontent.Service, label string) {
	ctx := context.Background()

	// Upload content
	content, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
		OwnerID:            uuid.New(),
		TenantID:           uuid.New(),
		Name:               fmt.Sprintf("Test Document (%s)", label),
		DocumentType:       "text/plain",
		StorageBackendName: "memory",
		Reader:             strings.NewReader("Hello from library usage example!"),
		FileName:           "test.txt",
	})
	if err != nil {
		log.Fatalf("Failed to upload content: %v", err)
	}
	fmt.Printf("   ✓ Content uploaded: %s\n", content.ID)

	// Get content details
	details, err := svc.GetContentDetails(ctx, content.ID)
	if err != nil {
		log.Fatalf("Failed to get details: %v", err)
	}
	fmt.Printf("   ✓ Content ready: %t\n", details.Ready)
}

// Option 3: Embed in your own HTTP server
func runCustomHTTPServer(svc simplecontent.Service) {
	// Your own HTTP server with YOUR own port
	mux := http.NewServeMux()

	// Add your custom handlers that use the service
	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		// Use svc.UploadContent(), etc.
		fmt.Fprintf(w, "Upload endpoint - uses simplecontent service internally")
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "OK")
	})

	// YOU control the port, not the config package
	myPort := ":3000" // Your application's port
	fmt.Printf("   ✓ Custom HTTP server would listen on %s\n", myPort)
	fmt.Println("   ✓ (not actually starting server in this example)")

	// In a real application:
	// log.Fatal(http.ListenAndServe(myPort, mux))
}
