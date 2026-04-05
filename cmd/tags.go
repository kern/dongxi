package cmd

import (
	"fmt"
	"time"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

var tagsCmd = &cobra.Command{
	Use:   "tags",
	Short: "List all tags",
	RunE:  runTags,
}

var tagCmd = &cobra.Command{
	Use:   "tag <task-uuid> <tag-uuid>",
	Short: "Add a tag to a task",
	Args:  cobra.ExactArgs(2),
	RunE:  runTag,
}

var untagCmd = &cobra.Command{
	Use:   "untag <task-uuid> <tag-uuid>",
	Short: "Remove a tag from a task",
	Args:  cobra.ExactArgs(2),
	RunE:  runUntag,
}

func runTags(cmd *cobra.Command, args []string) error {
	s, _, _, err := loadState()
	if err != nil {
		return err
	}

	var filtered []replayedItem
	for _, item := range s.items {
		if item.entity != string(dongxi.EntityTag) {
			continue
		}
		filtered = append(filtered, item)
	}

	if flagJSON {
		var out []ItemOutput
		for _, item := range filtered {
			out = append(out, s.itemToOutput(&item))
		}
		return printJSON(out)
	}

	for _, item := range filtered {
		title := toStr(item.fields[dongxi.FieldTitle])
		if title == "" {
			title = "(untitled)"
		}
		shortcut := toStr(item.fields[dongxi.FieldShortcut])
		if shortcut != "" {
			fmt.Printf("  %s  (%s)  [%s]\n", title, shortcut, item.uuid)
		} else {
			fmt.Printf("  %s  [%s]\n", title, item.uuid)
		}
	}

	if len(filtered) == 0 {
		fmt.Println("  (no tags)")
	} else {
		fmt.Printf("\n%d tag(s)\n", len(filtered))
	}
	return nil
}

func runTag(cmd *cobra.Command, args []string) error {
	s, client, historyKey, err := loadState()
	if err != nil {
		return err
	}

	task, err := s.resolveUUID(args[0])
	if err != nil {
		return err
	}
	if task.entity != string(dongxi.EntityTask) {
		return fmt.Errorf("%s is not a task", args[0])
	}

	tag, err := s.resolveUUID(args[1])
	if err != nil {
		return err
	}
	if tag.entity != string(dongxi.EntityTag) {
		return fmt.Errorf("%s is not a tag", args[1])
	}

	// Get current tags and append.
	currentTags := toStringSlice(task.fields[dongxi.FieldTagIDs])
	for _, t := range currentTags {
		if t == tag.uuid {
			fmt.Printf("  %s already has tag %s\n", toStr(task.fields[dongxi.FieldTitle]), toStr(tag.fields[dongxi.FieldTitle]))
			return nil
		}
	}
	newTags := append(currentTags, tag.uuid)

	histInfo, err := client.GetHistory(historyKey)
	if err != nil {
		return fmt.Errorf("fetch history info: %w", err)
	}

	now := float64(time.Now().UnixNano()) / 1e9
	commit := map[string]dongxi.CommitItem{
		task.uuid: {
			T: dongxi.ItemTypeModify,
			E: dongxi.EntityTask,
			P: map[string]any{
				dongxi.FieldTagIDs:           newTags,
				dongxi.FieldModificationDate: now,
			},
		},
	}

	resp, err := client.Commit(historyKey, histInfo.LatestServerIndex, commit)
	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	if flagJSON {
		return printJSON(TagActionOutput{
			TaskUUID:    task.uuid,
			TagUUID:     tag.uuid,
			Action:      "tagged",
			ServerIndex: resp.ServerHeadIndex,
		})
	}

	fmt.Printf("  Tagged %s with %s\n", toStr(task.fields[dongxi.FieldTitle]), toStr(tag.fields[dongxi.FieldTitle]))
	fmt.Printf("Server index: %d\n", resp.ServerHeadIndex)
	return nil
}

func runUntag(cmd *cobra.Command, args []string) error {
	s, client, historyKey, err := loadState()
	if err != nil {
		return err
	}

	task, err := s.resolveUUID(args[0])
	if err != nil {
		return err
	}
	if task.entity != string(dongxi.EntityTask) {
		return fmt.Errorf("%s is not a task", args[0])
	}

	tag, err := s.resolveUUID(args[1])
	if err != nil {
		return err
	}

	currentTags := toStringSlice(task.fields[dongxi.FieldTagIDs])
	var newTags []string
	found := false
	for _, t := range currentTags {
		if t == tag.uuid {
			found = true
		} else {
			newTags = append(newTags, t)
		}
	}
	if !found {
		return fmt.Errorf("task does not have that tag")
	}

	histInfo, err := client.GetHistory(historyKey)
	if err != nil {
		return fmt.Errorf("fetch history info: %w", err)
	}

	now := float64(time.Now().UnixNano()) / 1e9
	commit := map[string]dongxi.CommitItem{
		task.uuid: {
			T: dongxi.ItemTypeModify,
			E: dongxi.EntityTask,
			P: map[string]any{
				dongxi.FieldTagIDs:           newTags,
				dongxi.FieldModificationDate: now,
			},
		},
	}

	resp, err := client.Commit(historyKey, histInfo.LatestServerIndex, commit)
	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	if flagJSON {
		return printJSON(TagActionOutput{
			TaskUUID:    task.uuid,
			TagUUID:     tag.uuid,
			Action:      "untagged",
			ServerIndex: resp.ServerHeadIndex,
		})
	}

	fmt.Printf("  Untagged %s from %s\n", toStr(tag.fields[dongxi.FieldTitle]), toStr(task.fields[dongxi.FieldTitle]))
	fmt.Printf("Server index: %d\n", resp.ServerHeadIndex)
	return nil
}

func toStringSlice(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	var result []string
	for _, item := range arr {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}
