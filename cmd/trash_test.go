package cmd

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/kern/dongxi/dongxi"
)

func TestRunTrash(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	err := runTrash(nil, []string{"task-1"})
	if err != nil {
		t.Fatal(err)
	}

	if mock.lastCommit == nil {
		t.Fatal("expected a commit")
	}
	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldTrashed] != true {
		t.Errorf("expected trashed=true, got %v", commit.P[dongxi.FieldTrashed])
	}
}

func TestRunTrashArea(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
	})

	err := runTrash(nil, []string{"area-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["area-1"]
	if commit.P[dongxi.FieldTrashed] != true {
		t.Error("expected trashed=true for area")
	}
	if commit.E != dongxi.EntityArea {
		t.Errorf("expected entity area, got %v", commit.E)
	}
}

func TestRunTrashMultiple(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeTask("task-2", "Call dentist"),
	})

	err := runTrash(nil, []string{"task-1", "task-2"})
	if err != nil {
		t.Fatal(err)
	}

	if len(mock.lastCommit) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(mock.lastCommit))
	}
}

func TestRunTrashNotFound(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	err := runTrash(nil, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent UUID")
	}
}

func TestRunTrashNotATaskOrArea(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTag("tag-1", "Urgent"),
	})

	err := runTrash(nil, []string{"tag-1"})
	if err == nil {
		t.Fatal("expected error for tag entity")
	}
}

func TestRunTrashJSON(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runTrash(nil, []string{"task-1"})

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

func TestRunTrashLoadStateErr(t *testing.T) {
	setupMockStateErr(t, fmt.Errorf("mock load error"))
	err := runTrash(nil, []string{"task-1"})
	if err == nil || err.Error() != "mock load error" {
		t.Fatalf("expected load error, got %v", err)
	}
}

func TestRunTrashGetHistoryErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{makeTask("task-1", "Buy milk")})
	mock.getHistoryErr = fmt.Errorf("history error")
	err := runTrash(nil, []string{"task-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("history error")) {
		t.Fatalf("expected history error, got %v", err)
	}
}

func TestRunTrashCommitErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{makeTask("task-1", "Buy milk")})
	mock.commitErr = fmt.Errorf("commit error")
	err := runTrash(nil, []string{"task-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("commit error")) {
		t.Fatalf("expected commit error, got %v", err)
	}
}

func TestRunTrashUntitledTask(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", ""),
	})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runTrash(nil, []string{"task-1"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("(untitled)")) {
		t.Error("expected (untitled) in output")
	}
}
