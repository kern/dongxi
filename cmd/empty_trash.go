package cmd

import (
	"fmt"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

var flagEmptyTrashConfirm bool

var emptyTrashCmd = &cobra.Command{
	Use:   "empty-trash",
	Short: "Permanently delete all trashed items",
	Long: `Permanently delete all items in the trash. This cannot be undone.

Use --yes to skip the confirmation prompt.`,
	RunE: runEmptyTrash,
}

func init() {
	emptyTrashCmd.Flags().BoolVar(&flagEmptyTrashConfirm, "yes", false, "Skip confirmation prompt")
}

func runEmptyTrash(cmd *cobra.Command, args []string) error {
	s, client, historyKey, err := loadState()
	if err != nil {
		return err
	}

	// Find all trashed items.
	var trashed []replayedItem
	for _, item := range s.items {
		if !toBool(item.fields[dongxi.FieldTrashed]) {
			continue
		}
		trashed = append(trashed, item)
	}

	if len(trashed) == 0 {
		fmt.Println("Trash is empty.")
		return nil
	}

	if !flagEmptyTrashConfirm {
		return fmt.Errorf("this will permanently delete %d item(s); use --yes to confirm", len(trashed))
	}

	histInfo, err := client.GetHistory(historyKey)
	if err != nil {
		return fmt.Errorf("fetch history info: %w", err)
	}

	commit := map[string]dongxi.CommitItem{}
	for _, item := range trashed {
		commit[item.uuid] = dongxi.CommitItem{
			T: dongxi.ItemTypeDelete,
			E: dongxi.EntityType(item.entity),
		}
	}

	resp, err := client.Commit(historyKey, histInfo.LatestServerIndex, commit)
	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	if flagJSON {
		return printJSON(BulkActionOutput{
			Items:       []ActionItemOutput{{Action: "emptied trash", Title: fmt.Sprintf("%d items", len(trashed))}},
			ServerIndex: resp.ServerHeadIndex,
		})
	}

	fmt.Printf("Permanently deleted %d item(s).\n", len(trashed))
	fmt.Printf("Server index: %d\n", resp.ServerHeadIndex)
	return nil
}
