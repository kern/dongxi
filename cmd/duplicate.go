package cmd

import (
	"fmt"
	"time"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

var duplicateCmd = &cobra.Command{
	Use:   "duplicate <uuid>",
	Short: "Duplicate a task",
	Args:  cobra.ExactArgs(1),
	RunE:  runDuplicate,
}

func runDuplicate(cmd *cobra.Command, args []string) error {
	s, client, historyKey, err := loadState()
	if err != nil {
		return err
	}

	item, err := s.resolveUUID(args[0])
	if err != nil {
		return err
	}
	if item.entity != string(dongxi.EntityTask) {
		return fmt.Errorf("%s is a %s, not a task", args[0], item.entity)
	}

	histInfo, err := client.GetHistory(historyKey)
	if err != nil {
		return fmt.Errorf("fetch history info: %w", err)
	}

	now := float64(time.Now().UnixNano()) / 1e9
	dupUUID := newUUID()

	// Copy all fields from original.
	createPayload := copyMap(item.fields)
	createPayload[dongxi.FieldCreationDate] = now
	createPayload[dongxi.FieldModificationDate] = now
	createPayload[dongxi.FieldStatus] = int(dongxi.TaskStatusOpen)
	createPayload[dongxi.FieldStopDate] = nil

	commit := map[string]dongxi.CommitItem{
		dupUUID: {
			T: dongxi.ItemTypeCreate,
			E: dongxi.EntityTask,
			P: createPayload,
		},
	}

	// Duplicate checklist items too.
	checklistItems := s.checklistForTask(item.uuid)
	for _, ci := range checklistItems {
		ciUUID := newUUID()
		ciPayload := copyMap(ci.fields)
		ciPayload[dongxi.FieldCreationDate] = now
		ciPayload[dongxi.FieldModificationDate] = now
		ciPayload[dongxi.FieldStatus] = int(dongxi.TaskStatusOpen)
		ciPayload[dongxi.FieldStopDate] = nil
		ciPayload[dongxi.FieldTaskIDs] = []string{dupUUID}
		commit[ciUUID] = dongxi.CommitItem{
			T: dongxi.ItemTypeCreate,
			E: dongxi.EntityChecklistItem,
			P: ciPayload,
		}
	}

	resp, err := client.Commit(historyKey, histInfo.LatestServerIndex, commit)
	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	title := toStr(item.fields[dongxi.FieldTitle])
	if flagJSON {
		return printJSON(CreateOutput{
			UUID:           dupUUID,
			Type:           "task",
			Title:          title,
			ChecklistCount: len(checklistItems),
			ServerIndex:    resp.ServerHeadIndex,
		})
	}

	if title == "" {
		title = "(untitled)"
	}
	fmt.Printf("  Duplicated: %s  [%s]\n", title, dupUUID)
	if len(checklistItems) > 0 {
		fmt.Printf("  with %d checklist item(s)\n", len(checklistItems))
	}
	fmt.Printf("Server index: %d\n", resp.ServerHeadIndex)
	return nil
}
