package cmd

import (
	"fmt"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

// newClientFunc creates a CloudClient. Tests can replace this.
var newClientFunc = func(email, password string) CloudClient {
	return dongxi.NewClient(email, password)
}

// saveConfigFunc persists a Config to disk. Tests can replace this.
var saveConfigFunc = func(cfg *dongxi.Config) error {
	return dongxi.SaveConfig(cfg)
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Save Things Cloud credentials",
	Long:  `Save your Things Cloud email and password to ~/.config/dongxi/config.json.`,
	RunE:  runLogin,
}

func init() {
	loginCmd.Flags().StringVar(&flagEmail, "email", "", "Things Cloud account email (required)")
	loginCmd.Flags().StringVar(&flagPassword, "password", "", "Things Cloud account password (required)")
	_ = loginCmd.MarkFlagRequired("email")
	_ = loginCmd.MarkFlagRequired("password")
}

func runLogin(cmd *cobra.Command, args []string) error {
	client := newClientFunc(flagEmail, flagPassword)

	// Verify credentials by fetching account info.
	acct, err := client.GetAccount(flagEmail)
	if err != nil {
		return fmt.Errorf("verify credentials: %w", err)
	}

	cfg := &dongxi.Config{
		Email:      flagEmail,
		Password:   flagPassword,
		HistoryKey: acct.HistoryKey,
	}

	if err := saveConfigFunc(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Logged in as %s\n", flagEmail)
	fmt.Fprintf(cmd.OutOrStdout(), "History key: %s\n", acct.HistoryKey)
	return nil
}
