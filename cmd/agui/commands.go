package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewUploadCommand creates the upload command
func NewUploadCommand() *cobra.Command {
	var analysisType string
	var metadata map[string]interface{}

	cmd := &cobra.Command{
		Use:   "upload <file>",
		Short: "Upload a file to the content service",
		Long:  `Upload a file to the content service and receive a content ID and download URL.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]

			// Check if file exists
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				return fmt.Errorf("file does not exist: %s", filePath)
			}

			client, err := NewServiceClientFromFlags(cmd)
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			verbose, _ := cmd.Flags().GetBool("verbose")
			if verbose {
				fmt.Printf("Uploading file: %s\n", filePath)
			}

			resp, err := client.UploadFile(filePath, analysisType, metadata)
			if err != nil {
				return fmt.Errorf("upload failed: %w", err)
			}

			fmt.Printf("Upload successful!\n")
			fmt.Printf("Content ID: %s\n", resp.ID)
			fmt.Printf("Download URL: %s\n", resp.URL)
			if resp.UploadURL != "" {
				fmt.Printf("Upload URL: %s\n", resp.UploadURL)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&analysisType, "analysis-type", "", "Type of analysis to perform")
	// Note: metadata flag would need custom parsing for map[string]interface{}

	return cmd
}

// NewDownloadCommand creates the download command
func NewDownloadCommand() *cobra.Command {
	var outputPath string

	cmd := &cobra.Command{
		Use:   "download <content-id>",
		Short: "Download content by ID",
		Long:  `Download content from the service using its content ID.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			contentID := args[0]

			client, err := NewServiceClientFromFlags(cmd)
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			// If no output path specified, use content ID as filename
			if outputPath == "" {
				outputPath = fmt.Sprintf("content-%s", contentID)
			}

			verbose, _ := cmd.Flags().GetBool("verbose")
			if verbose {
				fmt.Printf("Downloading content: %s\n", contentID)
			}

			err = client.DownloadContent([]string{contentID}, outputPath)
			if err != nil {
				return fmt.Errorf("download failed: %w", err)
			}

			fmt.Printf("Download successful!\n")
			fmt.Printf("Saved to: %s\n", outputPath)

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path (default: content-<id>)")

	return cmd
}

// NewListCommand creates the list command
func NewListCommand() *cobra.Command {
	var limit int
	var offset int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all uploaded contents",
		Long:  `List all contents that have been uploaded to the service.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := NewServiceClientFromFlags(cmd)
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			verbose, _ := cmd.Flags().GetBool("verbose")
			if verbose {
				fmt.Printf("Listing contents (limit: %d, offset: %d)\n", limit, offset)
			}

			resp, err := client.ListContents(limit, offset)
			if err != nil {
				return fmt.Errorf("list failed: %w", err)
			}

			fmt.Printf("Total contents: %d\n", resp.Total)
			fmt.Printf("Showing: %d\n\n", len(resp.Contents))

			for i, content := range resp.Contents {
				fmt.Printf("%d. ID: %s\n", i+1, content.ID)
				fmt.Printf("   URL: %s\n", content.URL)
				if content.UploadURL != "" {
					fmt.Printf("   Upload URL: %s\n", content.UploadURL)
				}
				fmt.Println()
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum number of results to return")
	cmd.Flags().IntVar(&offset, "offset", 0, "Number of results to skip")

	return cmd
}

// NewDeleteCommand creates the delete command
func NewDeleteCommand() *cobra.Command {
	var confirm bool

	cmd := &cobra.Command{
		Use:   "delete <content-id>",
		Short: "Delete content by ID",
		Long:  `Delete content from the service using its content ID.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			contentID := args[0]

			// Require confirmation unless --confirm flag is set
			if !confirm {
				fmt.Printf("Are you sure you want to delete content %s? (y/N): ", contentID)
				var response string
				fmt.Scanln(&response)
				if response != "y" && response != "Y" {
					fmt.Println("Delete cancelled.")
					return nil
				}
			}

			client, err := NewServiceClientFromFlags(cmd)
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			verbose, _ := cmd.Flags().GetBool("verbose")
			if verbose {
				fmt.Printf("Deleting content: %s\n", contentID)
			}

			err = client.DeleteContent(contentID)
			if err != nil {
				return fmt.Errorf("delete failed: %w", err)
			}

			fmt.Printf("Content %s deleted successfully!\n", contentID)

			return nil
		},
	}

	cmd.Flags().BoolVarP(&confirm, "confirm", "y", false, "Skip confirmation prompt")

	return cmd
}

// NewMetadataCommand creates the metadata command
func NewMetadataCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metadata <content-id>",
		Short: "Get metadata for content",
		Long:  `Retrieve metadata information for a specific content ID.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			contentID := args[0]

			client, err := NewServiceClientFromFlags(cmd)
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			verbose, _ := cmd.Flags().GetBool("verbose")
			if verbose {
				fmt.Printf("Getting metadata for content: %s\n", contentID)
			}

			metadata, err := client.GetContentMetadata(contentID)
			if err != nil {
				return fmt.Errorf("failed to get metadata: %w", err)
			}

			fmt.Printf("Content Metadata:\n")
			fmt.Printf("  ID:        %s\n", metadata.ID)
			fmt.Printf("  Filename:  %s\n", metadata.Filename)
			fmt.Printf("  MIME Type: %s\n", metadata.MimeType)
			fmt.Printf("  Size:      %d bytes\n", metadata.Size)
			fmt.Printf("  Created:   %s\n", metadata.CreatedAt.Format("2006-01-02 15:04:05"))
			if metadata.ExpiresAt != nil {
				fmt.Printf("  Expires:   %s\n", metadata.ExpiresAt.Format("2006-01-02 15:04:05"))
			}

			return nil
		},
	}

	return cmd
}
