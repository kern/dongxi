package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

// makeEditCmd creates a cobra.Command with the same flags as editCmd,
// and marks the given flag names as changed.
func makeEditCmd(changedFlags ...string) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().StringVar(&flagEditTitle, "title", "", "")
	cmd.Flags().StringVar(&flagEditNote, "note", "", "")
	cmd.Flags().StringVar(&flagEditScheduled, "scheduled", "", "")
	cmd.Flags().StringVar(&flagEditDeadline, "deadline", "", "")
	cmd.Flags().StringVar(&flagEditEvening, "evening", "", "")
	for _, name := range changedFlags {
		_ = cmd.Flags().Set(name, cmd.Flags().Lookup(name).DefValue)
	}
	return cmd
}

func TestRunEditTitle(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Old title"),
	})

	flagEditTitle = "New title"
	cmd := makeEditCmd("title")
	_ = cmd.Flags().Set("title", "New title")

	err := runEdit(cmd, []string{"task-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldTitle] != "New title" {
		t.Errorf("expected title 'New title', got %v", commit.P[dongxi.FieldTitle])
	}
}

func TestRunEditNote(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	flagEditNote = "Some notes"
	cmd := makeEditCmd("note")
	_ = cmd.Flags().Set("note", "Some notes")

	err := runEdit(cmd, []string{"task-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldNote] == nil {
		t.Error("expected note to be set")
	}
}

func TestRunEditClearNote(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	flagEditNote = ""
	cmd := makeEditCmd("note")

	err := runEdit(cmd, []string{"task-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldNote] == nil {
		t.Error("expected note to be set to empty note")
	}
}

func TestRunEditScheduled(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	flagEditScheduled = "2025-04-01"
	cmd := makeEditCmd("scheduled")
	_ = cmd.Flags().Set("scheduled", "2025-04-01")

	err := runEdit(cmd, []string{"task-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldScheduledDate] == nil {
		t.Error("expected scheduled date to be set")
	}
}

func TestRunEditClearScheduled(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	flagEditScheduled = ""
	cmd := makeEditCmd("scheduled")

	err := runEdit(cmd, []string{"task-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldScheduledDate] != nil {
		t.Error("expected scheduled date to be nil")
	}
}

func TestRunEditBadScheduled(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	flagEditScheduled = "not-a-date"
	cmd := makeEditCmd("scheduled")
	_ = cmd.Flags().Set("scheduled", "not-a-date")

	err := runEdit(cmd, []string{"task-1"})
	if err == nil {
		t.Fatal("expected error for bad scheduled date")
	}
}

func TestRunEditDeadline(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	flagEditDeadline = "2025-04-15"
	cmd := makeEditCmd("deadline")
	_ = cmd.Flags().Set("deadline", "2025-04-15")

	err := runEdit(cmd, []string{"task-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldDeadline] == nil {
		t.Error("expected deadline to be set")
	}
}

func TestRunEditClearDeadline(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	flagEditDeadline = ""
	cmd := makeEditCmd("deadline")

	err := runEdit(cmd, []string{"task-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldDeadline] != nil {
		t.Error("expected deadline to be nil")
	}
}

func TestRunEditBadDeadline(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	flagEditDeadline = "bad-date"
	cmd := makeEditCmd("deadline")
	_ = cmd.Flags().Set("deadline", "bad-date")

	err := runEdit(cmd, []string{"task-1"})
	if err == nil {
		t.Fatal("expected error for bad deadline date")
	}
}

func TestRunEditEveningOn(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	flagEditEvening = "true"
	cmd := makeEditCmd("evening")
	_ = cmd.Flags().Set("evening", "true")

	err := runEdit(cmd, []string{"task-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldStartBucket] != 1 {
		t.Errorf("expected start bucket 1, got %v", commit.P[dongxi.FieldStartBucket])
	}
}

func TestRunEditEveningOff(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	flagEditEvening = "false"
	cmd := makeEditCmd("evening")
	_ = cmd.Flags().Set("evening", "false")

	err := runEdit(cmd, []string{"task-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldStartBucket] != 0 {
		t.Errorf("expected start bucket 0, got %v", commit.P[dongxi.FieldStartBucket])
	}
}

func TestRunEditEveningBadValue(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	flagEditEvening = "maybe"
	cmd := makeEditCmd("evening")
	_ = cmd.Flags().Set("evening", "maybe")

	err := runEdit(cmd, []string{"task-1"})
	if err == nil {
		t.Fatal("expected error for bad evening value")
	}
}

func TestRunEditNoChanges(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	cmd := makeEditCmd() // no flags changed

	err := runEdit(cmd, []string{"task-1"})
	if err == nil {
		t.Fatal("expected error for no changes")
	}
}

func TestRunEditNotATask(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
	})

	flagEditTitle = "New title"
	cmd := makeEditCmd("title")
	_ = cmd.Flags().Set("title", "New title")

	err := runEdit(cmd, []string{"area-1"})
	if err == nil {
		t.Fatal("expected error for non-task entity")
	}
}

func TestRunEditNotFound(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	cmd := makeEditCmd("title")
	_ = cmd.Flags().Set("title", "New title")

	err := runEdit(cmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent UUID")
	}
}

func TestRunEditJSON(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	flagEditTitle = "New title"
	cmd := makeEditCmd("title")
	_ = cmd.Flags().Set("title", "New title")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runEdit(cmd, []string{"task-1"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if buf.Len() == 0 {
		t.Error("expected JSON output")
	}
}

func TestRunEditUntitledTask(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", ""),
	})

	flagEditTitle = "Now has title"
	cmd := makeEditCmd("title")
	_ = cmd.Flags().Set("title", "Now has title")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runEdit(cmd, []string{"task-1"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("(untitled)")) {
		t.Error("expected (untitled) in output for empty title")
	}
}
