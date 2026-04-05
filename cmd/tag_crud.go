package cmd

import (
	"fmt"
	"time"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

var (
	flagCreateTagTitle    string
	flagCreateTagShortcut string
)

var createTagCmd = &cobra.Command{
	Use:   "create-tag",
	Short: "Create a new tag",
	RunE:  runCreateTag,
}

var editTagCmd = &cobra.Command{
	Use:   "edit-tag <uuid>",
	Short: "Rename a tag",
	Args:  cobra.ExactArgs(1),
	RunE:  runEditTag,
}

var deleteTagCmd = &cobra.Command{
	Use:   "delete-tag <uuid>",
	Short: "Delete a tag",
	Args:  cobra.ExactArgs(1),
	RunE:  runDeleteTag,
}

var (
	flagEditTagTitle    string
	flagEditTagShortcut string
)

func init() {
	createTagCmd.Flags().StringVarP(&flagCreateTagTitle, "title", "t", "", "Tag title (required)")
	createTagCmd.Flags().StringVar(&flagCreateTagShortcut, "shortcut", "", "Keyboard shortcut (single character)")
	_ = createTagCmd.MarkFlagRequired("title")

	editTagCmd.Flags().StringVar(&flagEditTagTitle, "title", "", "New title")
	editTagCmd.Flags().StringVar(&flagEditTagShortcut, "shortcut", "", "Keyboard shortcut (single character, or \"\" to clear)")
}

func runCreateTag(cmd *cobra.Command, args []string) error {
	_, client, historyKey, err := loadState()
	if err != nil {
		return err
	}

	histInfo, err := client.GetHistory(historyKey)
	if err != nil {
		return fmt.Errorf("fetch history info: %w", err)
	}

	now := float64(time.Now().UnixNano()) / 1e9
	tagUUID := newUUID()

	createCommit := map[string]dongxi.CommitItem{
		tagUUID: {
			T: dongxi.ItemTypeCreate,
			E: dongxi.EntityTag,
			P: map[string]any{
				dongxi.FieldTitle:    "",
				dongxi.FieldIndex:    dongxi.DefaultIndex,
				dongxi.FieldShortcut: nil,
				dongxi.FieldSyncMeta: dongxi.SyncMeta,
			},
		},
	}

	resp, err := client.Commit(historyKey, histInfo.LatestServerIndex, createCommit)
	if err != nil {
		return fmt.Errorf("commit create: %w", err)
	}

	modifyPayload := map[string]any{
		dongxi.FieldTitle:            flagCreateTagTitle,
		dongxi.FieldModificationDate: now,
	}
	if flagCreateTagShortcut != "" {
		modifyPayload[dongxi.FieldShortcut] = flagCreateTagShortcut
	}
	modifyCommit := map[string]dongxi.CommitItem{
		tagUUID: {
			T: dongxi.ItemTypeModify,
			E: dongxi.EntityTag,
			P: modifyPayload,
		},
	}

	resp, err = client.Commit(historyKey, resp.ServerHeadIndex, modifyCommit)
	if err != nil {
		return fmt.Errorf("commit modify: %w", err)
	}

	if flagJSON {
		return printJSON(CreateOutput{
			UUID:        tagUUID,
			Type:        "tag",
			Title:       flagCreateTagTitle,
			ServerIndex: resp.ServerHeadIndex,
		})
	}

	fmt.Printf("Created tag: %q  [%s]\n", flagCreateTagTitle, tagUUID)
	fmt.Printf("Server index: %d\n", resp.ServerHeadIndex)
	return nil
}

func runEditTag(cmd *cobra.Command, args []string) error {
	s, client, historyKey, err := loadState()
	if err != nil {
		return err
	}

	item, err := s.resolveUUID(args[0])
	if err != nil {
		return err
	}
	if item.entity != string(dongxi.EntityTag) {
		return fmt.Errorf("%s is not a tag", args[0])
	}

	histInfo, err := client.GetHistory(historyKey)
	if err != nil {
		return fmt.Errorf("fetch history info: %w", err)
	}

	now := float64(time.Now().UnixNano()) / 1e9
	payload := map[string]any{dongxi.FieldModificationDate: now}
	changes := []string{}

	if cmd.Flags().Changed("title") {
		payload[dongxi.FieldTitle] = flagEditTagTitle
		changes = append(changes, fmt.Sprintf("title -> %q", flagEditTagTitle))
	}

	if cmd.Flags().Changed("shortcut") {
		if flagEditTagShortcut == "" {
			payload[dongxi.FieldShortcut] = nil
			changes = append(changes, "cleared shortcut")
		} else {
			payload[dongxi.FieldShortcut] = flagEditTagShortcut
			changes = append(changes, fmt.Sprintf("shortcut -> %q", flagEditTagShortcut))
		}
	}

	if len(changes) == 0 {
		return fmt.Errorf("no changes specified (use --title or --shortcut)")
	}

	commit := map[string]dongxi.CommitItem{
		item.uuid: {
			T: dongxi.ItemTypeModify,
			E: dongxi.EntityTag,
			P: payload,
		},
	}

	resp, err := client.Commit(historyKey, histInfo.LatestServerIndex, commit)
	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	if flagJSON {
		return printJSON(EditOutput{
			UUID:        item.uuid,
			Title:       toStr(item.fields[dongxi.FieldTitle]),
			Changes:     changes,
			ServerIndex: resp.ServerHeadIndex,
		})
	}

	title := toStr(item.fields[dongxi.FieldTitle])
	if title == "" {
		title = "(untitled)"
	}
	for _, c := range changes {
		fmt.Printf("  %s: %s\n", title, c)
	}
	fmt.Printf("Server index: %d\n", resp.ServerHeadIndex)
	return nil
}

func runDeleteTag(cmd *cobra.Command, args []string) error {
	s, client, historyKey, err := loadState()
	if err != nil {
		return err
	}

	item, err := s.resolveUUID(args[0])
	if err != nil {
		return err
	}
	if item.entity != string(dongxi.EntityTag) {
		return fmt.Errorf("%s is not a tag", args[0])
	}

	histInfo, err := client.GetHistory(historyKey)
	if err != nil {
		return fmt.Errorf("fetch history info: %w", err)
	}

	commit := map[string]dongxi.CommitItem{
		item.uuid: {
			T: dongxi.ItemTypeDelete,
			E: dongxi.EntityTag,
		},
	}

	resp, err := client.Commit(historyKey, histInfo.LatestServerIndex, commit)
	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	title := toStr(item.fields[dongxi.FieldTitle])
	if flagJSON {
		return printJSON(ActionItemOutput{UUID: item.uuid, Title: title, Action: "deleted"})
	}

	if title == "" {
		title = "(untitled)"
	}
	fmt.Printf("  Deleted tag: %s\n", title)
	fmt.Printf("Server index: %d\n", resp.ServerHeadIndex)
	return nil
}
