package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/mark3labs/mcp-go/server"
	"github.com/tendant/simple-content/internal/mcp"
)

type Config struct {
	Host    string `env:"HOST" env-default:"localhost"`
	Port    uint16 `env:"PORT" env-default:"8000"`
	BaseUrl string `env:"BASE_URL" env-default:"http://localhost:8000"`
}

func main() {
	// Define command line flags for database configs and server mode

	// Server mode flags
	var mode = flag.String("mode", "stdio", "Server mode: 'stdio', 'sse', or 'http'")

	// Parse command line flags
	flag.Parse()

	var cfg Config
	cleanenv.ReadEnv(&cfg)

	// Create MCP server with appropriate options based on mode
	s := server.NewMCPServer(
		"Content Server Mcp",
		"1.0.0",
		server.WithResourceCapabilities(true, true), // Enable SSE and JSON-RPC
	)

	// Register hello content tools
	helloHandler := mcp.NewHelloContentHandler()
	helloHandler.RegisterTools(s)

	// Start the server based on the selected mode
	switch *mode {
	case "sse":
		// Construct base URL from host and port
		sseServer := server.NewSSEServer(s, server.WithBaseURL(cfg.BaseUrl))
		slog.Info("Starting SSE server", "base url", cfg.BaseUrl)
		if err := sseServer.Start(fmt.Sprintf(":%d", cfg.Port)); err != nil {
			slog.Error("Failed to start SSE server", "err", err)
			os.Exit(-1)
		}
	case "http":
		httpServer := server.NewStreamableHTTPServer(s)
		log.Printf("HTTP server listening on :%d", cfg.Port)
		if err := httpServer.Start(fmt.Sprintf(":%d", cfg.Port)); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	default:
		// Default to stdio mode
		slog.Info("Starting in stdio mode")
		if err := server.ServeStdio(s); err != nil {
			slog.Error("Failed to start stdio server", "err", err)
			os.Exit(-1)
		}
	}
}
