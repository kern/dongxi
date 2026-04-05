package cmd

import (
	"fmt"
	"time"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

var checklistCmd = &cobra.Command{
	Use:   "checklist",
	Short: "Manage checklist items on a task",
}

var checklistAddCmd = &cobra.Command{
	Use:   "add <task-uuid> <title>",
	Short: "Add a checklist item to a task",
	Args:  cobra.ExactArgs(2),
	RunE:  runChecklistAdd,
}

var checklistCompleteCmd = &cobra.Command{
	Use:   "complete <checklist-uuid>",
	Short: "Complete a checklist item",
	Args:  cobra.ExactArgs(1),
	RunE:  runChecklistComplete,
}

var checklistRemoveCmd = &cobra.Command{
	Use:   "remove <checklist-uuid>",
	Short: "Remove a checklist item",
	Args:  cobra.ExactArgs(1),
	RunE:  runChecklistRemove,
}

var checklistEditCmd = &cobra.Command{
	Use:   "edit <checklist-uuid> <new-title>",
	Short: "Edit a checklist item's title",
	Args:  cobra.ExactArgs(2),
	RunE:  runChecklistEdit,
}

var checklistToTasksCmd = &cobra.Command{
	Use:   "to-tasks <task-uuid>",
	Short: "Convert a task's checklist items into standalone tasks",
	Long: `Convert all open checklist items on a task into standalone tasks.
The new tasks inherit the parent task's area, project, and destination.
The original checklist items are removed.`,
	Args: cobra.ExactArgs(1),
	RunE: runChecklistToTasks,
}

func init() {
	checklistCmd.AddCommand(checklistAddCmd)
	checklistCmd.AddCommand(checklistCompleteCmd)
	checklistCmd.AddCommand(checklistRemoveCmd)
	checklistCmd.AddCommand(checklistEditCmd)
	checklistCmd.AddCommand(checklistToTasksCmd)
}

func runChecklistAdd(cmd *cobra.Command, args []string) error {
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

	histInfo, err := client.GetHistory(historyKey)
	if err != nil {
		return fmt.Errorf("fetch history info: %w", err)
	}

	// Count existing checklist items to determine index.
	existing := s.checklistForTask(task.uuid)
	idx := len(existing)

	now := float64(time.Now().UnixNano()) / 1e9
	ciUUID := newUUID()

	commit := map[string]dongxi.CommitItem{
		ciUUID: {
			T: dongxi.ItemTypeCreate,
			E: dongxi.EntityChecklistItem,
			P: map[string]any{
				dongxi.FieldTitle:            args[1],
				dongxi.FieldStatus:           int(dongxi.TaskStatusOpen),
				dongxi.FieldTaskIDs:          []string{task.uuid},
				dongxi.FieldCreationDate:     now,
				dongxi.FieldModificationDate: now,
				dongxi.FieldIndex:            idx,
				dongxi.FieldStopDate:         nil,
				dongxi.FieldLateTask:         false,
				dongxi.FieldSyncMeta:         dongxi.SyncMeta,
			},
		},
		task.uuid: {
			T: dongxi.ItemTypeModify,
			E: dongxi.EntityTask,
			P: map[string]any{
				dongxi.FieldModificationDate: now,
				dongxi.FieldChecklistCount:   idx + 1,
			},
		},
	}

	resp, err := client.Commit(historyKey, histInfo.LatestServerIndex, commit)
	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	if flagJSON {
		return printJSON(ChecklistActionOutput{
			UUID:        ciUUID,
			Title:       args[1],
			TaskUUID:    task.uuid,
			Action:      "added",
			ServerIndex: resp.ServerHeadIndex,
		})
	}

	taskTitle := toStr(task.fields[dongxi.FieldTitle])
	if taskTitle == "" {
		taskTitle = "(untitled)"
	}
	fmt.Printf("  Added checklist item %q to %s\n", args[1], taskTitle)
	fmt.Printf("Server index: %d\n", resp.ServerHeadIndex)
	return nil
}

func runChecklistComplete(cmd *cobra.Command, args []string) error {
	s, client, historyKey, err := loadState()
	if err != nil {
		return err
	}

	item, err := s.resolveUUID(args[0])
	if err != nil {
		return err
	}
	if item.entity != string(dongxi.EntityChecklistItem) {
		return fmt.Errorf("%s is not a checklist item", args[0])
	}

	histInfo, err := client.GetHistory(historyKey)
	if err != nil {
		return fmt.Errorf("fetch history info: %w", err)
	}

	now := float64(time.Now().UnixNano()) / 1e9
	commit := map[string]dongxi.CommitItem{
		item.uuid: {
			T: dongxi.ItemTypeModify,
			E: dongxi.EntityChecklistItem,
			P: map[string]any{
				dongxi.FieldStatus:           int(dongxi.TaskStatusCompleted),
				dongxi.FieldStopDate:         now,
				dongxi.FieldModificationDate: now,
			},
		},
	}

	resp, err := client.Commit(historyKey, histInfo.LatestServerIndex, commit)
	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	if flagJSON {
		return printJSON(ChecklistActionOutput{
			UUID:        item.uuid,
			Title:       toStr(item.fields[dongxi.FieldTitle]),
			Action:      "completed",
			ServerIndex: resp.ServerHeadIndex,
		})
	}

	title := toStr(item.fields[dongxi.FieldTitle])
	fmt.Printf("  Completed: %s\n", title)
	fmt.Printf("Server index: %d\n", resp.ServerHeadIndex)
	return nil
}

func runChecklistRemove(cmd *cobra.Command, args []string) error {
	s, client, historyKey, err := loadState()
	if err != nil {
		return err
	}

	item, err := s.resolveUUID(args[0])
	if err != nil {
		return err
	}
	if item.entity != string(dongxi.EntityChecklistItem) {
		return fmt.Errorf("%s is not a checklist item", args[0])
	}

	histInfo, err := client.GetHistory(historyKey)
	if err != nil {
		return fmt.Errorf("fetch history info: %w", err)
	}

	commit := map[string]dongxi.CommitItem{
		item.uuid: {
			T: dongxi.ItemTypeDelete,
			E: dongxi.EntityChecklistItem,
		},
	}

	resp, err := client.Commit(historyKey, histInfo.LatestServerIndex, commit)
	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	if flagJSON {
		return printJSON(ChecklistActionOutput{
			UUID:        item.uuid,
			Title:       toStr(item.fields[dongxi.FieldTitle]),
			Action:      "removed",
			ServerIndex: resp.ServerHeadIndex,
		})
	}

	title := toStr(item.fields[dongxi.FieldTitle])
	fmt.Printf("  Removed: %s\n", title)
	fmt.Printf("Server index: %d\n", resp.ServerHeadIndex)
	return nil
}

func runChecklistEdit(cmd *cobra.Command, args []string) error {
	s, client, historyKey, err := loadState()
	if err != nil {
		return err
	}

	item, err := s.resolveUUID(args[0])
	if err != nil {
		return err
	}
	if item.entity != string(dongxi.EntityChecklistItem) {
		return fmt.Errorf("%s is not a checklist item", args[0])
	}

	histInfo, err := client.GetHistory(historyKey)
	if err != nil {
		return fmt.Errorf("fetch history info: %w", err)
	}

	now := float64(time.Now().UnixNano()) / 1e9
	commit := map[string]dongxi.CommitItem{
		item.uuid: {
			T: dongxi.ItemTypeModify,
			E: dongxi.EntityChecklistItem,
			P: map[string]any{
				dongxi.FieldTitle:            args[1],
				dongxi.FieldModificationDate: now,
			},
		},
	}

	resp, err := client.Commit(historyKey, histInfo.LatestServerIndex, commit)
	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	if flagJSON {
		return printJSON(ChecklistActionOutput{
			UUID:        item.uuid,
			Title:       args[1],
			Action:      "edited",
			ServerIndex: resp.ServerHeadIndex,
		})
	}

	fmt.Printf("  Renamed: %q -> %q\n", toStr(item.fields[dongxi.FieldTitle]), args[1])
	fmt.Printf("Server index: %d\n", resp.ServerHeadIndex)
	return nil
}

func runChecklistToTasks(cmd *cobra.Command, args []string) error {
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

	checklistItems := s.checklistForTask(task.uuid)
	if len(checklistItems) == 0 {
		fmt.Println("  (no checklist items)")
		return nil
	}

	// Filter to open items only.
	var openItems []replayedItem
	for _, ci := range checklistItems {
		if toInt(ci.fields[dongxi.FieldStatus]) == int(dongxi.TaskStatusOpen) {
			openItems = append(openItems, ci)
		}
	}
	if len(openItems) == 0 {
		fmt.Println("  (no open checklist items)")
		return nil
	}

	histInfo, err := client.GetHistory(historyKey)
	if err != nil {
		return fmt.Errorf("fetch history info: %w", err)
	}

	now := float64(time.Now().UnixNano()) / 1e9

	// Inherit parent task's area, project, destination.
	areaIDs := task.fields[dongxi.FieldAreaIDs]
	if areaIDs == nil {
		areaIDs = []string{}
	}
	projectIDs := task.fields[dongxi.FieldProjectIDs]
	if projectIDs == nil {
		projectIDs = []string{}
	}
	destination := toInt(task.fields[dongxi.FieldDestination])

	commit := map[string]dongxi.CommitItem{}
	var created []string

	for _, ci := range openItems {
		taskUUID := newUUID()
		created = append(created, toStr(ci.fields[dongxi.FieldTitle]))

		commit[taskUUID] = dongxi.CommitItem{
			T: dongxi.ItemTypeCreate,
			E: dongxi.EntityTask,
			P: map[string]any{
				dongxi.FieldTitle:              toStr(ci.fields[dongxi.FieldTitle]),
				dongxi.FieldStatus:             int(dongxi.TaskStatusOpen),
				dongxi.FieldDestination:        destination,
				dongxi.FieldType:               int(dongxi.TaskTypeTask),
				dongxi.FieldCreationDate:       now,
				dongxi.FieldModificationDate:   now,
				dongxi.FieldTrashed:            false,
				dongxi.FieldIndex:              dongxi.DefaultIndex,
				dongxi.FieldTodayIndex:         dongxi.DefaultTodayIndex,
				dongxi.FieldStartBucket:        0,
				dongxi.FieldDueOrder:           0,
				dongxi.FieldChecklistCount:     0,
				dongxi.FieldChecklistComplete:  false,
				dongxi.FieldLateTask:           false,
				dongxi.FieldAreaIDs:            areaIDs,
				dongxi.FieldProjectIDs:         projectIDs,
				dongxi.FieldTagIDs:             []string{},
				dongxi.FieldHeadingIDs:         []string{},
				dongxi.FieldActionGroupIDs:     []string{},
				dongxi.FieldReminders:          []any{},
				dongxi.FieldStopDate:           nil,
				dongxi.FieldDeadline:           nil,
				dongxi.FieldScheduledDate:      nil,
				dongxi.FieldTodayIndexRef:      nil,
				dongxi.FieldAutoTimeOffer:      nil,
				dongxi.FieldNote:               dongxi.EmptyNote,
				dongxi.FieldRepeatRule:         nil,
				dongxi.FieldRepeatPaused:       nil,
				dongxi.FieldRepeatMethodDate:   nil,
				dongxi.FieldDueDate:            nil,
				dongxi.FieldLastAlarmInteract:  nil,
				dongxi.FieldInstanceCreatedSrc: nil,
				dongxi.FieldAutoCompRepeatDate: nil,
				dongxi.FieldSyncMeta:           dongxi.SyncMeta,
			},
		}

		// Delete the original checklist item.
		commit[ci.uuid] = dongxi.CommitItem{
			T: dongxi.ItemTypeDelete,
			E: dongxi.EntityChecklistItem,
		}
	}

	// Update parent task's checklist count.
	remaining := len(checklistItems) - len(openItems)
	commit[task.uuid] = dongxi.CommitItem{
		T: dongxi.ItemTypeModify,
		E: dongxi.EntityTask,
		P: map[string]any{
			dongxi.FieldModificationDate: now,
			dongxi.FieldChecklistCount:   remaining,
		},
	}

	resp, err := client.Commit(historyKey, histInfo.LatestServerIndex, commit)
	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	if flagJSON {
		var items []ActionItemOutput
		for _, title := range created {
			items = append(items, ActionItemOutput{Title: title, Action: "created from checklist"})
		}
		return printJSON(BulkActionOutput{Items: items, ServerIndex: resp.ServerHeadIndex})
	}

	for _, title := range created {
		fmt.Printf("  Created task: %s\n", title)
	}
	fmt.Printf("\nConverted %d checklist item(s) to tasks.\n", len(created))
	fmt.Printf("Server index: %d\n", resp.ServerHeadIndex)
	return nil
}
