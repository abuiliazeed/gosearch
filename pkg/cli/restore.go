// Package cli provides the command-line interface for gosearch.
//
// It uses Cobra for command parsing and Viper for configuration management.
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// restoreCmd represents the restore command
var restoreCmd = &cobra.Command{
	Use:   "restore <backup-file>",
	Short: "Restore index from a backup file",
	Long: `Restore index from a backup file.

This will replace the current index with the backup. It's recommended to
create a backup of the current index before restoring.`,
	Args: cobra.ExactArgs(1),
	RunE: runRestore,
}

func init() {
	rootCmd.AddCommand(restoreCmd)

	restoreCmd.Flags().BoolP("force", "f", false, "force restore without confirmation")
	restoreCmd.Flags().BoolP("backup-current", "b", true, "create backup of current index before restore")
}

func runRestore(cmd *cobra.Command, args []string) error {
	backupFile := args[0]

	// Check if backup file exists
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found: %s", backupFile)
	}

	// Get data directory
	dataDir := viper.GetString("data-dir")
	indexPath := filepath.Join(dataDir, "index", "index.db")

	// Check if current index exists
	currentIndexExists := false
	if _, err := os.Stat(indexPath); err == nil {
		currentIndexExists = true
	}

	// Backup current index if it exists and --backup-current is true
	backupCurrent, _ := cmd.Flags().GetBool("backup-current")
	if currentIndexExists && backupCurrent {
		timestamp := time.Now().Format("20060102-150405")
		autoBackupFile := filepath.Join(dataDir, "backups", fmt.Sprintf("pre-restore-%s.db", timestamp))

		fmt.Printf("Creating backup of current index to %s...\n", autoBackupFile)
		if err := os.MkdirAll(filepath.Dir(autoBackupFile), 0755); err != nil {
			return fmt.Errorf("failed to create backup directory: %w", err)
		}

		if err := copyFile(indexPath, autoBackupFile); err != nil {
			return fmt.Errorf("failed to create backup of current index: %w", err)
		}
		fmt.Printf("Current index backed up to: %s\n", autoBackupFile)
	}

	// Confirm restore unless --force is set
	force, _ := cmd.Flags().GetBool("force")
	if !force {
		fmt.Printf("This will replace the current index with: %s\n", backupFile)
		fmt.Print("Continue? (y/N): ")

		var response string
		_, _ = fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Restore cancelled")
			return nil
		}
	}

	// Ensure index directory exists
	indexDir := filepath.Dir(indexPath)
	if err := os.MkdirAll(indexDir, 0755); err != nil {
		return fmt.Errorf("failed to create index directory: %w", err)
	}

	// Perform restore
	fmt.Printf("Restoring index from %s to %s...\n", backupFile, indexPath)

	startTime := time.Now()
	if err := copyFile(backupFile, indexPath); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	duration := time.Since(startTime)

	// Get file size info
	srcInfo, _ := os.Stat(backupFile)
	dstInfo, _ := os.Stat(indexPath)

	fmt.Printf("Index restored successfully in %s\n", duration.Round(time.Millisecond))
	fmt.Printf("Backup size: %.2f MB\n", float64(srcInfo.Size())/(1024*1024))
	fmt.Printf("Restored size: %.2f MB\n", float64(dstInfo.Size())/(1024*1024))

	return nil
}
