package cmd

import (
	"errors"
	"testing"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

// TestLoadStateErrors tests the loadState error path in every command.
func TestLoadStateErrors(t *testing.T) {
	setupMockStateErr(t, errors.New("connection failed"))

	tests := []struct {
		name string
		fn   func() error
	}{
		{"complete", func() error { return runComplete(nil, []string{"x"}) }},
		{"cancel", func() error { return runCancel(nil, []string{"x"}) }},
		{"trash", func() error { return runTrash(nil, []string{"x"}) }},
		{"untrash", func() error { return runUntrash(nil, []string{"x"}) }},
		{"reopen", func() error { return runReopen(nil, []string{"x"}) }},
		{"show", func() error { return runShow(nil, []string{"x"}) }},
		{"list", func() error { return runList(nil, nil) }},
		{"search", func() error { return runSearch(nil, []string{"x"}) }},
		{"logbook", func() error { return runLogbook(nil, nil) }},
		{"upcoming", func() error { return runUpcoming(nil, nil) }},
		{"projects", func() error { return runProjects(nil, nil) }},
		{"areas", func() error { return runAreas(nil, nil) }},
		{"tags", func() error { return runTags(nil, nil) }},
		{"info", func() error { return runInfo(nil, nil) }},
		{"summary", func() error { return runSummary(nil, nil) }},
		{"query", func() error { return runQuery(nil, nil) }},
		{"export", func() error { return runExport(nil, nil) }},
		{"edit", func() error {
			cmd := &cobra.Command{}
			cmd.Flags().StringVar(&flagEditTitle, "title", "", "")
			return runEdit(cmd, []string{"x"})
		}},
		{"move", func() error {
			cmd := &cobra.Command{}
			cmd.Flags().StringVar(&flagMoveArea, "area", "", "")
			return runMove(cmd, []string{"x"})
		}},
		{"reorder", func() error {
			old := flagReorderTop
			flagReorderTop = true
			defer func() { flagReorderTop = old }()
			return runReorder(nil, []string{"x"})
		}},
		{"repeat", func() error { return runRepeat(nil, []string{"x"}) }},
		{"duplicate", func() error { return runDuplicate(nil, []string{"x"}) }},
		{"convert", func() error { return runConvert(nil, []string{"x"}) }},
		{"tag", func() error { return runTag(nil, []string{"x", "y"}) }},
		{"untag", func() error { return runUntag(nil, []string{"x", "y"}) }},
		{"checklistAdd", func() error { return runChecklistAdd(nil, []string{"x", "y"}) }},
		{"checklistComplete", func() error { return runChecklistComplete(nil, []string{"x"}) }},
		{"checklistRemove", func() error { return runChecklistRemove(nil, []string{"x"}) }},
		{"checklistEdit", func() error { return runChecklistEdit(nil, []string{"x", "y"}) }},
		{"checklistToTasks", func() error { return runChecklistToTasks(nil, []string{"x"}) }},
		{"emptyTrash", func() error { return runEmptyTrash(nil, nil) }},
		{"createArea", func() error { return runCreateArea(nil, nil) }},
		{"createTag", func() error { return runCreateTag(nil, nil) }},
		{"editArea", func() error {
			cmd := &cobra.Command{}
			cmd.Flags().StringVar(&flagEditAreaTitle, "title", "", "")
			return runEditArea(cmd, []string{"x"})
		}},
		{"editTag", func() error {
			cmd := &cobra.Command{}
			cmd.Flags().StringVar(&flagEditTagTitle, "title", "", "")
			cmd.Flags().StringVar(&flagEditTagShortcut, "shortcut", "", "")
			return runEditTag(cmd, []string{"x"})
		}},
		{"deleteTag", func() error { return runDeleteTag(nil, []string{"x"}) }},
		{"reset", func() error {
			old := flagResetYes
			flagResetYes = true
			defer func() { flagResetYes = old }()
			return runReset(nil, nil)
		}},
		{"create", func() error {
			old := flagCreateTitle
			flagCreateTitle = "test"
			defer func() { flagCreateTitle = old }()
			cmd := &cobra.Command{}
			cmd.Flags().StringVar(&flagCreateTitle, "title", "", "")
			return runCreate(cmd, nil)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Fatal("expected error from loadState, got nil")
			}
		})
	}
}

// TestGetHistoryErrors tests the GetHistory error path in commands that call it.
func TestGetHistoryErrors(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeTask("trashed-1", "Trashed task", func(p map[string]any) {
			p[dongxi.FieldTrashed] = true
		}),
		makeArea("area-1", "Work"),
		makeTag("tag-1", "Urgent"),
		makeChecklistItem("cl-1", "Step 1", "task-1"),
	})
	mock.getHistoryErr = errors.New("history unavailable")

	tests := []struct {
		name string
		fn   func() error
	}{
		{"complete", func() error { return runComplete(nil, []string{"task-1"}) }},
		{"cancel", func() error { return runCancel(nil, []string{"task-1"}) }},
		{"trash", func() error { return runTrash(nil, []string{"task-1"}) }},
		{"untrash", func() error { return runUntrash(nil, []string{"task-1"}) }},
		{"reopen", func() error { return runReopen(nil, []string{"task-1"}) }},
		{"edit", func() error {
			cmd := &cobra.Command{}
			cmd.Flags().StringVar(&flagEditTitle, "title", "", "")
			_ = cmd.Flags().Set("title", "New")
			return runEdit(cmd, []string{"task-1"})
		}},
		{"move", func() error {
			cmd := &cobra.Command{}
			cmd.Flags().StringVar(&flagMoveDestination, "destination", "", "")
			flagMoveDestination = "today"
			return runMove(cmd, []string{"task-1"})
		}},
		{"reorder", func() error {
			old := flagReorderTop
			flagReorderTop = true
			defer func() { flagReorderTop = old }()
			return runReorder(nil, []string{"task-1"})
		}},
		{"repeat", func() error { return runRepeat(nil, []string{"task-1"}) }},
		{"duplicate", func() error { return runDuplicate(nil, []string{"task-1"}) }},
		{"convert", func() error { return runConvert(nil, []string{"task-1"}) }},
		{"tag", func() error { return runTag(nil, []string{"task-1", "tag-1"}) }},
		{"untag", func() error {
			// Need the task to have the tag for untag to proceed to GetHistory
			return runUntag(nil, []string{"task-1", "tag-1"})
		}},
		{"checklistAdd", func() error { return runChecklistAdd(nil, []string{"task-1", "New item"}) }},
		{"checklistComplete", func() error { return runChecklistComplete(nil, []string{"cl-1"}) }},
		{"checklistRemove", func() error { return runChecklistRemove(nil, []string{"cl-1"}) }},
		{"checklistEdit", func() error { return runChecklistEdit(nil, []string{"cl-1", "Updated"}) }},
		{"checklistToTasks", func() error { return runChecklistToTasks(nil, []string{"task-1"}) }},
		{"emptyTrash", func() error {
			old := flagEmptyTrashConfirm
			flagEmptyTrashConfirm = true
			defer func() { flagEmptyTrashConfirm = old }()
			return runEmptyTrash(nil, nil)
		}},
		{"createArea", func() error { return runCreateArea(nil, nil) }},
		{"createTag", func() error { return runCreateTag(nil, nil) }},
		{"editArea", func() error {
			cmd := &cobra.Command{}
			cmd.Flags().StringVar(&flagEditAreaTitle, "title", "", "")
			_ = cmd.Flags().Set("title", "New")
			return runEditArea(cmd, []string{"area-1"})
		}},
		{"editTag", func() error {
			cmd := &cobra.Command{}
			cmd.Flags().StringVar(&flagEditTagTitle, "title", "", "")
			cmd.Flags().StringVar(&flagEditTagShortcut, "shortcut", "", "")
			_ = cmd.Flags().Set("title", "New")
			return runEditTag(cmd, []string{"tag-1"})
		}},
		{"deleteTag", func() error { return runDeleteTag(nil, []string{"tag-1"}) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Fatal("expected error from GetHistory, got nil")
			}
		})
	}
}

// TestCommitErrors tests the Commit error path in commands.
func TestCommitErrors(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeChecklistItem("cl-1", "Step 1", "task-1"),
	})
	mock.commitErr = errors.New("commit failed")

	tests := []struct {
		name string
		fn   func() error
	}{
		{"complete", func() error { return runComplete(nil, []string{"task-1"}) }},
		{"cancel", func() error { return runCancel(nil, []string{"task-1"}) }},
		{"trash", func() error { return runTrash(nil, []string{"task-1"}) }},
		{"untrash", func() error { return runUntrash(nil, []string{"task-1"}) }},
		{"reopen", func() error { return runReopen(nil, []string{"task-1"}) }},
		{"edit", func() error {
			cmd := &cobra.Command{}
			cmd.Flags().StringVar(&flagEditTitle, "title", "", "")
			_ = cmd.Flags().Set("title", "New")
			return runEdit(cmd, []string{"task-1"})
		}},
		{"reorder", func() error {
			old := flagReorderTop
			flagReorderTop = true
			defer func() { flagReorderTop = old }()
			return runReorder(nil, []string{"task-1"})
		}},
		{"duplicate", func() error { return runDuplicate(nil, []string{"task-1"}) }},
		{"convert", func() error { return runConvert(nil, []string{"task-1"}) }},
		{"checklistComplete", func() error { return runChecklistComplete(nil, []string{"cl-1"}) }},
		{"checklistRemove", func() error { return runChecklistRemove(nil, []string{"cl-1"}) }},
		{"checklistEdit", func() error { return runChecklistEdit(nil, []string{"cl-1", "Updated"}) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Fatal("expected error from Commit, got nil")
			}
		})
	}
}

// TestGetAccountError tests the GetAccount error path in info and reset.
func TestGetAccountError(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})
	mock.getAccountErr = errors.New("account unavailable")

	t.Run("info", func(t *testing.T) {
		err := runInfo(nil, nil)
		if err == nil {
			t.Fatal("expected error from GetAccount")
		}
	})

	t.Run("reset", func(t *testing.T) {
		old := flagResetYes
		flagResetYes = true
		defer func() { flagResetYes = old }()
		err := runReset(nil, nil)
		if err == nil {
			t.Fatal("expected error from GetAccount")
		}
	})
}

// TestResetHistoryError tests the ResetHistory error path.
func TestResetHistoryError(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})
	mock.resetHistErr = errors.New("reset failed")

	old := flagResetYes
	flagResetYes = true
	defer func() { flagResetYes = old }()

	err := runReset(nil, nil)
	if err == nil {
		t.Fatal("expected error from ResetHistory")
	}
}
