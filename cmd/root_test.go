package cmd

import "testing"

func TestRootCmdHelp(t *testing.T) {
	rootCmd.SetArgs([]string{"--help"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("rootCmd --help returned error: %v", err)
	}
}

func TestRootCmdUnknownCommand(t *testing.T) {
	rootCmd.SetArgs([]string{"nonexistent-subcommand"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for unknown subcommand, got nil")
	}
}
