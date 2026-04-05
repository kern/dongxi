package cmd

import (
	"fmt"
	"time"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

var (
	flagCreateAreaTitle string
)

var createAreaCmd = &cobra.Command{
	Use:   "create-area",
	Short: "Create a new area of responsibility",
	RunE:  runCreateArea,
}

var editAreaCmd = &cobra.Command{
	Use:   "edit-area <uuid>",
	Short: "Edit an area's title",
	Args:  cobra.ExactArgs(1),
	RunE:  runEditArea,
}

var (
	flagEditAreaTitle string
)

func init() {
	createAreaCmd.Flags().StringVarP(&flagCreateAreaTitle, "title", "t", "", "Area title (required)")
	_ = createAreaCmd.MarkFlagRequired("title")

	editAreaCmd.Flags().StringVar(&flagEditAreaTitle, "title", "", "New title")
}

func runCreateArea(cmd *cobra.Command, args []string) error {
	_, client, historyKey, err := loadState()
	if err != nil {
		return err
	}

	histInfo, err := client.GetHistory(historyKey)
	if err != nil {
		return fmt.Errorf("fetch history info: %w", err)
	}

	now := float64(time.Now().UnixNano()) / 1e9
	areaUUID := newUUID()

	createCommit := map[string]dongxi.CommitItem{
		areaUUID: {
			T: dongxi.ItemTypeCreate,
			E: dongxi.EntityArea,
			P: map[string]any{
				dongxi.FieldTitle:    "",
				dongxi.FieldIndex:    dongxi.DefaultIndex,
				dongxi.FieldTrashed:  false,
				dongxi.FieldSyncMeta: dongxi.SyncMeta,
			},
		},
	}

	resp, err := client.Commit(historyKey, histInfo.LatestServerIndex, createCommit)
	if err != nil {
		return fmt.Errorf("commit create: %w", err)
	}

	modifyCommit := map[string]dongxi.CommitItem{
		areaUUID: {
			T: dongxi.ItemTypeModify,
			E: dongxi.EntityArea,
			P: map[string]any{
				dongxi.FieldTitle:            flagCreateAreaTitle,
				dongxi.FieldModificationDate: now,
			},
		},
	}

	resp, err = client.Commit(historyKey, resp.ServerHeadIndex, modifyCommit)
	if err != nil {
		return fmt.Errorf("commit modify: %w", err)
	}

	if flagJSON {
		return printJSON(CreateOutput{
			UUID:        areaUUID,
			Type:        "area",
			Title:       flagCreateAreaTitle,
			ServerIndex: resp.ServerHeadIndex,
		})
	}

	fmt.Printf("Created area: %q  [%s]\n", flagCreateAreaTitle, areaUUID)
	fmt.Printf("Server index: %d\n", resp.ServerHeadIndex)
	return nil
}

func runEditArea(cmd *cobra.Command, args []string) error {
	s, client, historyKey, err := loadState()
	if err != nil {
		return err
	}

	item, err := s.resolveUUID(args[0])
	if err != nil {
		return err
	}
	if item.entity != string(dongxi.EntityArea) {
		return fmt.Errorf("%s is not an area", args[0])
	}

	histInfo, err := client.GetHistory(historyKey)
	if err != nil {
		return fmt.Errorf("fetch history info: %w", err)
	}

	now := float64(time.Now().UnixNano()) / 1e9
	payload := map[string]any{dongxi.FieldModificationDate: now}
	changes := []string{}

	if cmd.Flags().Changed("title") {
		payload[dongxi.FieldTitle] = flagEditAreaTitle
		changes = append(changes, fmt.Sprintf("title -> %q", flagEditAreaTitle))
	}

	if len(changes) == 0 {
		return fmt.Errorf("no changes specified (use --title)")
	}

	commit := map[string]dongxi.CommitItem{
		item.uuid: {
			T: dongxi.ItemTypeModify,
			E: dongxi.EntityArea,
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
