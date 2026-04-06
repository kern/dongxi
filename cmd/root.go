package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	flagEmail    string
	flagPassword string
	flagJSON     bool
	flagSkipSync bool
	flagSync     bool
)

var rootCmd = &cobra.Command{
	Use:   "dongxi",
	Short: "A CLI for Things Cloud",
	Long:  `dongxi is a command-line tool for interacting with Things Cloud.`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "Output in JSON format")
	rootCmd.PersistentFlags().BoolVar(&flagSkipSync, "skip-sync", false, "Use cached data only, do not contact Things Cloud")
	rootCmd.PersistentFlags().BoolVar(&flagSync, "sync", false, "Force a sync even if the throttle interval has not elapsed")

	// Auth
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(syncCmd)

	// Views
	rootCmd.AddCommand(areasCmd)
	rootCmd.AddCommand(projectsCmd)
	rootCmd.AddCommand(tagsCmd)
	rootCmd.AddCommand(logbookCmd)
	rootCmd.AddCommand(upcomingCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(summaryCmd)

	// CRUD
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(createAreaCmd)
	rootCmd.AddCommand(createTagCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(editAreaCmd)
	rootCmd.AddCommand(editTagCmd)
	rootCmd.AddCommand(completeCmd)
	rootCmd.AddCommand(reopenCmd)
	rootCmd.AddCommand(cancelCmd)
	rootCmd.AddCommand(trashCmd)
	rootCmd.AddCommand(untrashCmd)
	rootCmd.AddCommand(moveCmd)
	rootCmd.AddCommand(reorderCmd)
	rootCmd.AddCommand(repeatCmd)
	rootCmd.AddCommand(duplicateCmd)
	rootCmd.AddCommand(convertCmd)
	rootCmd.AddCommand(deleteTagCmd)
	rootCmd.AddCommand(emptyTrashCmd)

	// Tags
	rootCmd.AddCommand(tagCmd)
	rootCmd.AddCommand(untagCmd)

	// Checklist
	rootCmd.AddCommand(checklistCmd)

	// Batch
	rootCmd.AddCommand(batchCmd)

	// Export
	rootCmd.AddCommand(exportCmd)

	// Admin
	rootCmd.AddCommand(resetCmd)
}
