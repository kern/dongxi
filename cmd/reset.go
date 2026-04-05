package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

var flagResetYes bool

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset the Things Cloud history key",
	Long: `Reset the Things Cloud history key for your account.

WARNING: This is a destructive operation. Resetting the history key will:
  - Invalidate the current sync history
  - Force all Things clients to re-sync from scratch
  - Potentially cause data loss if clients have unsynced changes

This is typically used to resolve sync conflicts or corruption. Make sure
all your Things clients are synced before proceeding.`,
	RunE: runReset,
}

func init() {
	resetCmd.Flags().BoolVarP(&flagResetYes, "yes", "y", false, "Skip confirmation prompt")
}

func runReset(cmd *cobra.Command, args []string) error {
	_, client, historyKey, err := loadState()
	if err != nil {
		return err
	}

	acct, err := client.GetAccount("")
	if err != nil {
		return fmt.Errorf("fetch account: %w", err)
	}

	fmt.Println("WARNING: You are about to reset the Things Cloud history key.")
	fmt.Println()
	fmt.Printf("  Account: %s\n", acct.Email)
	fmt.Printf("  Current history key: %s\n", historyKey)
	fmt.Println()
	fmt.Println("This will invalidate the current sync history and force all Things")
	fmt.Println("clients to re-sync from scratch. Unsynced changes may be lost.")
	fmt.Println()

	if !flagResetYes {
		fmt.Print("Type 'yes' to confirm: ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("read confirmation: %w", err)
		}
		input = strings.TrimSpace(input)
		if input != "yes" {
			fmt.Println("Reset cancelled.")
			return nil
		}
	}

	fmt.Println("Resetting history key...")

	resp, err := client.ResetHistory(acct.Email)
	if err != nil {
		return fmt.Errorf("reset history: %w", err)
	}

	fmt.Printf("History key reset successfully.\n")
	fmt.Printf("New history key: %s\n", resp.NewHistoryKey)

	// Update local config with new history key.
	cfg, err := dongxi.LoadConfig()
	if err == nil {
		cfg.HistoryKey = resp.NewHistoryKey
		if err := dongxi.SaveConfig(cfg); err != nil {
			return fmt.Errorf("update config: %w\n\nThe server-side reset succeeded but the local config was not updated.", err)
		}
		fmt.Println("Config updated.")
	}

	fmt.Println("Restart Things on all your devices to complete the resync.")
	return nil
}
