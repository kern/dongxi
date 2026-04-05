package cmd

import (
	"fmt"
	"time"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

var reopenCmd = &cobra.Command{
	Use:   "reopen <uuid>...",
	Short: "Reopen completed or cancelled tasks",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runReopen,
}

func runReopen(cmd *cobra.Command, args []string) error {
	s, client, historyKey, err := loadState()
	if err != nil {
		return err
	}

	histInfo, err := client.GetHistory(historyKey)
	if err != nil {
		return fmt.Errorf("fetch history info: %w", err)
	}

	now := float64(time.Now().UnixNano()) / 1e9
	commit := map[string]dongxi.CommitItem{}

	for _, query := range args {
		item, err := s.resolveUUID(query)
		if err != nil {
			return err
		}
		if item.entity != string(dongxi.EntityTask) {
			return fmt.Errorf("%s is a %s, not a task", query, item.entity)
		}
		commit[item.uuid] = dongxi.CommitItem{
			T: dongxi.ItemTypeModify,
			E: dongxi.EntityTask,
			P: map[string]any{
				dongxi.FieldStatus:           int(dongxi.TaskStatusOpen),
				dongxi.FieldStopDate:         nil,
				dongxi.FieldModificationDate: now,
			},
		}
	}

	resp, err := client.Commit(historyKey, histInfo.LatestServerIndex, commit)
	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	if flagJSON {
		var items []ActionItemOutput
		for _, query := range args {
			item, _ := s.resolveUUID(query)
			items = append(items, ActionItemOutput{UUID: item.uuid, Title: toStr(item.fields[dongxi.FieldTitle]), Action: "reopened"})
		}
		return printJSON(BulkActionOutput{Items: items, ServerIndex: resp.ServerHeadIndex})
	}

	for _, query := range args {
		item, _ := s.resolveUUID(query)
		title := toStr(item.fields[dongxi.FieldTitle])
		if title == "" {
			title = "(untitled)"
		}
		fmt.Printf("  Reopened: %s\n", title)
	}
	fmt.Printf("Server index: %d\n", resp.ServerHeadIndex)
	return nil
}
