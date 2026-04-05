package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Display Things Cloud account and sync state",
	Long:  `Display information about your Things Cloud account and history.`,
	RunE:  runInfo,
}

func runInfo(cmd *cobra.Command, args []string) error {
	_, client, historyKey, err := loadState()
	if err != nil {
		return err
	}

	acct, err := client.GetAccount("") // email not needed — already authenticated
	if err != nil {
		return fmt.Errorf("fetch account: %w", err)
	}

	histInfo, err := client.GetHistory(historyKey)
	if err != nil {
		return fmt.Errorf("fetch history info: %w", err)
	}

	if flagJSON {
		return printJSON(InfoOutput{
			Email:         acct.Email,
			Status:        acct.Status,
			MaildropEmail: acct.MaildropEmail,
			HistoryKey:    historyKey,
			ServerIndex:   histInfo.LatestServerIndex,
			SchemaVersion: histInfo.LatestSchemaVersion,
			IsEmpty:       histInfo.IsEmpty,
			ContentSize:   histInfo.LatestTotalContentSize,
		})
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintln(w, "DONGXI INFO")
	fmt.Fprintln(w, "===========")
	fmt.Fprintf(w, "Account Email:\t%s\n", acct.Email)
	fmt.Fprintf(w, "Account Status:\t%s\n", acct.Status)
	fmt.Fprintf(w, "Maildrop Email:\t%s\n", acct.MaildropEmail)
	fmt.Fprintf(w, "History Key:\t%s\n", historyKey)
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "SYNC STATE")
	fmt.Fprintln(w, "----------")
	fmt.Fprintf(w, "Server Index:\t%d\n", histInfo.LatestServerIndex)
	fmt.Fprintf(w, "Schema Version:\t%d\n", histInfo.LatestSchemaVersion)
	fmt.Fprintf(w, "Is Empty:\t%v\n", histInfo.IsEmpty)
	fmt.Fprintf(w, "Total Content Size:\t%d bytes\n", histInfo.LatestTotalContentSize)
	fmt.Fprintf(w, "Report Generated:\t%s\n", time.Now().Format(time.RFC3339))

	return w.Flush()
}
