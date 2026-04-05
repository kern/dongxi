package cmd

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/kern/dongxi/dongxi"
)

func TestRunChecklistAdd(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	err := runChecklistAdd(nil, []string{"task-1", "Get whole milk"})
	if err != nil {
		t.Fatal(err)
	}

	if mock.lastCommit == nil {
		t.Fatal("expected a commit")
	}
	// Should have 2 entries: the new checklist item and task modification.
	if len(mock.lastCommit) != 2 {
		t.Fatalf("expected 2 commit items, got %d", len(mock.lastCommit))
	}
}

func TestRunChecklistAddNotATask(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
	})

	err := runChecklistAdd(nil, []string{"area-1", "Item"})
	if err == nil {
		t.Fatal("expected error for non-task entity")
	}
}

func TestRunChecklistAddNotFound(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	err := runChecklistAdd(nil, []string{"nonexistent", "Item"})
	if err == nil {
		t.Fatal("expected error for nonexistent UUID")
	}
}

func TestRunChecklistAddJSON(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runChecklistAdd(nil, []string{"task-1", "Item"})

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

func TestRunChecklistAddUntitledTask(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", ""),
	})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runChecklistAdd(nil, []string{"task-1", "Item"})

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

func TestRunChecklistComplete(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeChecklistItem("ci-1", "Step 1", "task-1"),
	})

	err := runChecklistComplete(nil, []string{"ci-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["ci-1"]
	if commit.P[dongxi.FieldStatus] != int(dongxi.TaskStatusCompleted) {
		t.Errorf("expected completed status, got %v", commit.P[dongxi.FieldStatus])
	}
}

func TestRunChecklistCompleteNotAChecklistItem(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	err := runChecklistComplete(nil, []string{"task-1"})
	if err == nil {
		t.Fatal("expected error for non-checklist item")
	}
}

func TestRunChecklistCompleteNotFound(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	err := runChecklistComplete(nil, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent UUID")
	}
}

func TestRunChecklistCompleteJSON(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeChecklistItem("ci-1", "Step 1", "task-1"),
	})

	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runChecklistComplete(nil, []string{"ci-1"})

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

func TestRunChecklistRemove(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeChecklistItem("ci-1", "Step 1", "task-1"),
	})

	err := runChecklistRemove(nil, []string{"ci-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["ci-1"]
	if commit.T != dongxi.ItemTypeDelete {
		t.Errorf("expected delete type, got %v", commit.T)
	}
}

func TestRunChecklistRemoveNotAChecklistItem(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	err := runChecklistRemove(nil, []string{"task-1"})
	if err == nil {
		t.Fatal("expected error for non-checklist item")
	}
}

func TestRunChecklistRemoveJSON(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeChecklistItem("ci-1", "Step 1", "task-1"),
	})

	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runChecklistRemove(nil, []string{"ci-1"})

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

func TestRunChecklistEdit(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeChecklistItem("ci-1", "Step 1", "task-1"),
	})

	err := runChecklistEdit(nil, []string{"ci-1", "Updated Step"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["ci-1"]
	if commit.P[dongxi.FieldTitle] != "Updated Step" {
		t.Errorf("expected title 'Updated Step', got %v", commit.P[dongxi.FieldTitle])
	}
}

func TestRunChecklistEditNotAChecklistItem(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	err := runChecklistEdit(nil, []string{"task-1", "New title"})
	if err == nil {
		t.Fatal("expected error for non-checklist item")
	}
}

func TestRunChecklistEditJSON(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeChecklistItem("ci-1", "Step 1", "task-1"),
	})

	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runChecklistEdit(nil, []string{"ci-1", "Updated"})

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

func TestRunChecklistToTasks(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeChecklistItem("ci-1", "Step 1", "task-1"),
		makeChecklistItem("ci-2", "Step 2", "task-1"),
	})

	err := runChecklistToTasks(nil, []string{"task-1"})
	if err != nil {
		t.Fatal(err)
	}

	if mock.lastCommit == nil {
		t.Fatal("expected a commit")
	}
	// Should have creates for new tasks, deletes for checklist items, and modify for parent.
	creates := 0
	deletes := 0
	for _, ci := range mock.lastCommit {
		switch ci.T {
		case dongxi.ItemTypeCreate:
			creates++
		case dongxi.ItemTypeDelete:
			deletes++
		}
	}
	if creates != 2 {
		t.Errorf("expected 2 creates, got %d", creates)
	}
	if deletes != 2 {
		t.Errorf("expected 2 deletes, got %d", deletes)
	}
}

func TestRunChecklistToTasksNoChecklist(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runChecklistToTasks(nil, []string{"task-1"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("(no checklist items)")) {
		t.Error("expected '(no checklist items)' message")
	}
}

func TestRunChecklistToTasksAllCompleted(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		{
			"ci-1": map[string]any{
				dongxi.CommitKeyType:   float64(dongxi.ItemTypeCreate),
				dongxi.CommitKeyEntity: string(dongxi.EntityChecklistItem),
				dongxi.CommitKeyPayload: map[string]any{
					dongxi.FieldTitle:   "Done step",
					dongxi.FieldStatus:  float64(dongxi.TaskStatusCompleted),
					dongxi.FieldTaskIDs: []any{"task-1"},
				},
			},
		},
	})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runChecklistToTasks(nil, []string{"task-1"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("(no open checklist items)")) {
		t.Error("expected '(no open checklist items)' message")
	}
}

func TestRunChecklistToTasksNotATask(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
	})

	err := runChecklistToTasks(nil, []string{"area-1"})
	if err == nil {
		t.Fatal("expected error for non-task entity")
	}
}

func TestRunChecklistToTasksJSON(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeChecklistItem("ci-1", "Step 1", "task-1"),
	})

	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runChecklistToTasks(nil, []string{"task-1"})

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

func TestRunChecklistRemoveNotFound(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	err := runChecklistRemove(nil, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent UUID")
	}
}

func TestRunChecklistEditNotFound(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	err := runChecklistEdit(nil, []string{"nonexistent", "New title"})
	if err == nil {
		t.Fatal("expected error for nonexistent UUID")
	}
}

func TestRunChecklistCompleteNotFound2(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	err := runChecklistComplete(nil, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent UUID")
	}
}

func TestRunChecklistToTasksNotFound(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	err := runChecklistToTasks(nil, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent UUID")
	}
}

// --- Error path tests for checklist commands ---

func TestRunChecklistAddLoadStateErr(t *testing.T) {
	setupMockStateErr(t, fmt.Errorf("mock load error"))
	err := runChecklistAdd(nil, []string{"task-1", "Item"})
	if err == nil || err.Error() != "mock load error" {
		t.Fatalf("expected load error, got %v", err)
	}
}

func TestRunChecklistAddGetHistoryErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{makeTask("task-1", "Buy milk")})
	mock.getHistoryErr = fmt.Errorf("history error")
	err := runChecklistAdd(nil, []string{"task-1", "Item"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("history error")) {
		t.Fatalf("expected history error, got %v", err)
	}
}

func TestRunChecklistAddCommitErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{makeTask("task-1", "Buy milk")})
	mock.commitErr = fmt.Errorf("commit error")
	err := runChecklistAdd(nil, []string{"task-1", "Item"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("commit error")) {
		t.Fatalf("expected commit error, got %v", err)
	}
}

func TestRunChecklistCompleteLoadStateErr(t *testing.T) {
	setupMockStateErr(t, fmt.Errorf("mock load error"))
	err := runChecklistComplete(nil, []string{"ci-1"})
	if err == nil || err.Error() != "mock load error" {
		t.Fatalf("expected load error, got %v", err)
	}
}

func TestRunChecklistCompleteGetHistoryErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeChecklistItem("ci-1", "Step 1", "task-1"),
	})
	mock.getHistoryErr = fmt.Errorf("history error")
	err := runChecklistComplete(nil, []string{"ci-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("history error")) {
		t.Fatalf("expected history error, got %v", err)
	}
}

func TestRunChecklistCompleteCommitErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeChecklistItem("ci-1", "Step 1", "task-1"),
	})
	mock.commitErr = fmt.Errorf("commit error")
	err := runChecklistComplete(nil, []string{"ci-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("commit error")) {
		t.Fatalf("expected commit error, got %v", err)
	}
}

func TestRunChecklistRemoveLoadStateErr(t *testing.T) {
	setupMockStateErr(t, fmt.Errorf("mock load error"))
	err := runChecklistRemove(nil, []string{"ci-1"})
	if err == nil || err.Error() != "mock load error" {
		t.Fatalf("expected load error, got %v", err)
	}
}

func TestRunChecklistRemoveGetHistoryErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeChecklistItem("ci-1", "Step 1", "task-1"),
	})
	mock.getHistoryErr = fmt.Errorf("history error")
	err := runChecklistRemove(nil, []string{"ci-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("history error")) {
		t.Fatalf("expected history error, got %v", err)
	}
}

func TestRunChecklistRemoveCommitErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeChecklistItem("ci-1", "Step 1", "task-1"),
	})
	mock.commitErr = fmt.Errorf("commit error")
	err := runChecklistRemove(nil, []string{"ci-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("commit error")) {
		t.Fatalf("expected commit error, got %v", err)
	}
}

func TestRunChecklistEditLoadStateErr(t *testing.T) {
	setupMockStateErr(t, fmt.Errorf("mock load error"))
	err := runChecklistEdit(nil, []string{"ci-1", "New"})
	if err == nil || err.Error() != "mock load error" {
		t.Fatalf("expected load error, got %v", err)
	}
}

func TestRunChecklistEditGetHistoryErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeChecklistItem("ci-1", "Step 1", "task-1"),
	})
	mock.getHistoryErr = fmt.Errorf("history error")
	err := runChecklistEdit(nil, []string{"ci-1", "New"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("history error")) {
		t.Fatalf("expected history error, got %v", err)
	}
}

func TestRunChecklistEditCommitErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeChecklistItem("ci-1", "Step 1", "task-1"),
	})
	mock.commitErr = fmt.Errorf("commit error")
	err := runChecklistEdit(nil, []string{"ci-1", "New"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("commit error")) {
		t.Fatalf("expected commit error, got %v", err)
	}
}

func TestRunChecklistToTasksLoadStateErr(t *testing.T) {
	setupMockStateErr(t, fmt.Errorf("mock load error"))
	err := runChecklistToTasks(nil, []string{"task-1"})
	if err == nil || err.Error() != "mock load error" {
		t.Fatalf("expected load error, got %v", err)
	}
}

func TestRunChecklistToTasksGetHistoryErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeChecklistItem("ci-1", "Step 1", "task-1"),
	})
	mock.getHistoryErr = fmt.Errorf("history error")
	err := runChecklistToTasks(nil, []string{"task-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("history error")) {
		t.Fatalf("expected history error, got %v", err)
	}
}

func TestRunChecklistToTasksCommitErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeChecklistItem("ci-1", "Step 1", "task-1"),
	})
	mock.commitErr = fmt.Errorf("commit error")
	err := runChecklistToTasks(nil, []string{"task-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("commit error")) {
		t.Fatalf("expected commit error, got %v", err)
	}
}
