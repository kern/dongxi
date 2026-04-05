package cmd

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/kern/dongxi/dongxi"
)

func TestRunDuplicate(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	err := runDuplicate(nil, []string{"task-1"})
	if err != nil {
		t.Fatal(err)
	}

	if mock.lastCommit == nil {
		t.Fatal("expected a commit")
	}
	// Should have created one new item (the duplicate).
	foundCreate := false
	for uuid, ci := range mock.lastCommit {
		if uuid != "task-1" && ci.T == dongxi.ItemTypeCreate {
			foundCreate = true
			if ci.P[dongxi.FieldTitle] != "Buy milk" {
				t.Errorf("expected duplicated title 'Buy milk', got %v", ci.P[dongxi.FieldTitle])
			}
		}
	}
	if !foundCreate {
		t.Error("expected a create commit for the duplicate")
	}
}

func TestRunDuplicateWithChecklist(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeChecklistItem("ci-1", "Step 1", "task-1"),
		makeChecklistItem("ci-2", "Step 2", "task-1"),
	})

	err := runDuplicate(nil, []string{"task-1"})
	if err != nil {
		t.Fatal(err)
	}

	// Should have the task + 2 checklist items = 3 creates.
	createCount := 0
	for _, ci := range mock.lastCommit {
		if ci.T == dongxi.ItemTypeCreate {
			createCount++
		}
	}
	if createCount != 3 {
		t.Errorf("expected 3 creates (task + 2 checklist), got %d", createCount)
	}
}

func TestRunDuplicateNotATask(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
	})

	err := runDuplicate(nil, []string{"area-1"})
	if err == nil {
		t.Fatal("expected error for non-task entity")
	}
}

func TestRunDuplicateNotFound(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	err := runDuplicate(nil, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent UUID")
	}
}

func TestRunDuplicateJSON(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDuplicate(nil, []string{"task-1"})

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

func TestRunDuplicateLoadStateErr(t *testing.T) {
	setupMockStateErr(t, fmt.Errorf("mock load error"))
	err := runDuplicate(nil, []string{"task-1"})
	if err == nil || err.Error() != "mock load error" {
		t.Fatalf("expected load error, got %v", err)
	}
}

func TestRunDuplicateGetHistoryErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{makeTask("task-1", "Buy milk")})
	mock.getHistoryErr = fmt.Errorf("history error")
	err := runDuplicate(nil, []string{"task-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("history error")) {
		t.Fatalf("expected history error, got %v", err)
	}
}

func TestRunDuplicateCommitErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{makeTask("task-1", "Buy milk")})
	mock.commitErr = fmt.Errorf("commit error")
	err := runDuplicate(nil, []string{"task-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("commit error")) {
		t.Fatalf("expected commit error, got %v", err)
	}
}

func TestRunDuplicateUntitledWithChecklist(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", ""),
		makeChecklistItem("ci-1", "Step 1", "task-1"),
	})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDuplicate(nil, []string{"task-1"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("(untitled)")) {
		t.Error("expected (untitled) in output")
	}
	if !bytes.Contains([]byte(output), []byte("checklist")) {
		t.Error("expected checklist count in output")
	}
}
