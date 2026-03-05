// Package cli provides the command-line interface for gosearch.
//
// It uses Cobra for command parsing and Viper for configuration management.
package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// backupCmd represents the backup command
var backupCmd = &cobra.Command{
	Use:   "backup [output-file]",
	Short: "Create a backup of the index",
	Long: `Create a backup of the index to a file.

If no output file is specified, a timestamped backup file will be created
in the data directory. The backup contains all index data including metadata,
postings lists, and document information.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runBackup,
}

func init() {
	rootCmd.AddCommand(backupCmd)

	backupCmd.Flags().StringP("output", "o", "", "output file path for the backup")
}

func runBackup(cmd *cobra.Command, args []string) error {
	// Get data directory
	dataDir := viper.GetString("data-dir")
	indexPath := filepath.Join(dataDir, "index", "index.db")

	// Check if index exists
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		return fmt.Errorf("index not found at %s. Run 'gosearch crawl' or 'gosearch index build' first", indexPath)
	}

	// Determine output file
	outputFile := ""
	if len(args) > 0 {
		outputFile = args[0]
	} else if flag, _ := cmd.Flags().GetString("output"); flag != "" {
		outputFile = flag
	}

	if outputFile == "" {
		// Create timestamped backup file
		timestamp := time.Now().Format("20060102-150405")
		backupDir := filepath.Join(dataDir, "backups")
		if err := os.MkdirAll(backupDir, 0o755); err != nil {
			return fmt.Errorf("failed to create backup directory: %w", err)
		}
		outputFile = filepath.Join(backupDir, fmt.Sprintf("index-backup-%s.db", timestamp))
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Perform backup
	fmt.Printf("Creating backup from %s to %s...\n", indexPath, outputFile)

	startTime := time.Now()
	if err := copyFile(indexPath, outputFile); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	duration := time.Since(startTime)

	// Get file size info
	srcInfo, _ := os.Stat(indexPath)
	dstInfo, _ := os.Stat(outputFile)

	fmt.Printf("Backup created successfully in %s\n", duration.Round(time.Millisecond))
	fmt.Printf("Source size: %.2f MB\n", float64(srcInfo.Size())/(1024*1024))
	fmt.Printf("Backup size: %.2f MB\n", float64(dstInfo.Size())/(1024*1024))
	fmt.Printf("Location: %s\n", outputFile)

	return nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	// Copy with buffer
	buf := make([]byte, 64*1024) // 64KB buffer
	for {
		n, err := source.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}
		if _, err := destination.Write(buf[:n]); err != nil {
			return err
		}
	}

	return nil
}
