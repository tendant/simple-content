package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/tendant/simple-content/pkg/simplecontent/config"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	rootCmd := NewRootCommand()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func NewRootCommand() *cobra.Command {
	var configFile string
	var verbose bool

	rootCmd := &cobra.Command{
		Use:   "agui",
		Short: "AGUI Protocol CLI - Content Management Client",
		Long: `AGUI Protocol Command Line Interface
		
A CLI tool for content management using the simplecontent service directly.
Supports file upload, download, and content management operations.

Uses in-memory storage by default for quick testing and development.`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	}

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "config file (optional)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Add subcommands
	rootCmd.AddCommand(NewUploadCommand())
	rootCmd.AddCommand(NewDownloadCommand())
	rootCmd.AddCommand(NewListCommand())
	rootCmd.AddCommand(NewDeleteCommand())
	rootCmd.AddCommand(NewMetadataCommand())

	return rootCmd
}

// NewServiceClientFromFlags creates a service client based on command flags and environment variables
func NewServiceClientFromFlags(cmd *cobra.Command) (*ServiceClient, error) {
	verbose, _ := cmd.Flags().GetBool("verbose")
	configFile, _ := cmd.Flags().GetString("config")

	if verbose {
		log.Println("Initializing simplecontent service")
	}

	var cfg *config.ServerConfig
	var err error

	if configFile != "" {
		// Load from config file
		if verbose {
			log.Printf("Loading config from file: %s\n", configFile)
		}
		// Load with environment variable overrides
		cfg, err = config.Load(config.WithEnv(""))
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
	} else {
		// Load configuration from environment variables
		// This will read STORAGE_URL, DATABASE_URL, and other env vars
		cfg, err = config.Load(config.WithEnv(""))
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
	}

	if verbose {
		log.Printf("Configuration loaded:")
		log.Printf("  Database Type: %s", cfg.DatabaseType)
		log.Printf("  Default Storage Backend: %s", cfg.DefaultStorageBackend)
		log.Printf("  URL Strategy: %s", cfg.URLStrategy)
		if cfg.DatabaseURL != "" {
			log.Printf("  Database URL: %s", maskPassword(cfg.DatabaseURL))
		}
	}

	service, err := cfg.BuildService()
	if err != nil {
		return nil, fmt.Errorf("failed to build service: %w", err)
	}

	return NewServiceClient(service, verbose), nil
}

// maskPassword masks the password in a database URL for logging
func maskPassword(url string) string {
	// Simple masking for postgres URLs: postgresql://user:password@host/db
	// Find the password part and replace it with ***
	start := 0
	for i := 0; i < len(url); i++ {
		if url[i] == ':' && i > 0 {
			// Check if this is the password separator (after //)
			if i >= 2 && url[i-2:i] == "//" {
				continue
			}
			start = i + 1
		}
		if url[i] == '@' && start > 0 {
			return url[:start] + "***" + url[i:]
		}
	}
	return url
}
