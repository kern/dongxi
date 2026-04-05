package cmd

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/kern/dongxi/dongxi"
)

func TestRunConvertTaskToProject(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagConvertTo
	flagConvertTo = "project"
	defer func() { flagConvertTo = old }()

	err := runConvert(nil, []string{"task-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldType] != int(dongxi.TaskTypeProject) {
		t.Errorf("expected project type, got %v", commit.P[dongxi.FieldType])
	}
}

func TestRunConvertProjectToTask(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeProject("proj-1", "My Project"),
	})

	old := flagConvertTo
	flagConvertTo = "task"
	defer func() { flagConvertTo = old }()

	err := runConvert(nil, []string{"proj-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["proj-1"]
	if commit.P[dongxi.FieldType] != int(dongxi.TaskTypeTask) {
		t.Errorf("expected task type, got %v", commit.P[dongxi.FieldType])
	}
}

func TestRunConvertAlreadySameType(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagConvertTo
	flagConvertTo = "task"
	defer func() { flagConvertTo = old }()

	err := runConvert(nil, []string{"task-1"})
	if err == nil {
		t.Fatal("expected error when already same type")
	}
}

func TestRunConvertUnknownTargetType(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagConvertTo
	flagConvertTo = "heading"
	defer func() { flagConvertTo = old }()

	err := runConvert(nil, []string{"task-1"})
	if err == nil {
		t.Fatal("expected error for unknown target type")
	}
}

func TestRunConvertNotATask(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
	})

	old := flagConvertTo
	flagConvertTo = "project"
	defer func() { flagConvertTo = old }()

	err := runConvert(nil, []string{"area-1"})
	if err == nil {
		t.Fatal("expected error for non-task entity")
	}
}

func TestRunConvertNotFound(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagConvertTo
	flagConvertTo = "project"
	defer func() { flagConvertTo = old }()

	err := runConvert(nil, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent UUID")
	}
}

func TestRunConvertJSON(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	oldJSON := flagJSON
	flagJSON = true
	defer func() { flagJSON = oldJSON }()

	old := flagConvertTo
	flagConvertTo = "project"
	defer func() { flagConvertTo = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runConvert(nil, []string{"task-1"})

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

func TestRunConvertLoadStateErr(t *testing.T) {
	setupMockStateErr(t, fmt.Errorf("mock load error"))
	flagConvertTo = "project"
	err := runConvert(nil, []string{"task-1"})
	if err == nil || err.Error() != "mock load error" {
		t.Fatalf("expected load error, got %v", err)
	}
}

func TestRunConvertGetHistoryErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{makeTask("task-1", "Buy milk")})
	mock.getHistoryErr = fmt.Errorf("history error")
	old := flagConvertTo
	flagConvertTo = "project"
	defer func() { flagConvertTo = old }()
	err := runConvert(nil, []string{"task-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("history error")) {
		t.Fatalf("expected history error, got %v", err)
	}
}

func TestRunConvertCommitErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{makeTask("task-1", "Buy milk")})
	mock.commitErr = fmt.Errorf("commit error")
	old := flagConvertTo
	flagConvertTo = "project"
	defer func() { flagConvertTo = old }()
	err := runConvert(nil, []string{"task-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("commit error")) {
		t.Fatalf("expected commit error, got %v", err)
	}
}

func TestRunConvertUntitledTask(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", ""),
	})

	old := flagConvertTo
	flagConvertTo = "project"
	defer func() { flagConvertTo = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runConvert(nil, []string{"task-1"})

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
