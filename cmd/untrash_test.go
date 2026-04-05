package cmd

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/kern/dongxi/dongxi"
)

func TestRunUntrash(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk", func(p map[string]any) {
			p[dongxi.FieldTrashed] = true
		}),
	})

	err := runUntrash(nil, []string{"task-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldTrashed] != false {
		t.Errorf("expected trashed=false, got %v", commit.P[dongxi.FieldTrashed])
	}
}

func TestRunUntrashArea(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
	})

	err := runUntrash(nil, []string{"area-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["area-1"]
	if commit.E != dongxi.EntityArea {
		t.Errorf("expected entity area, got %v", commit.E)
	}
}

func TestRunUntrashNotATaskOrArea(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTag("tag-1", "Urgent"),
	})

	err := runUntrash(nil, []string{"tag-1"})
	if err == nil {
		t.Fatal("expected error for tag entity")
	}
}

func TestRunUntrashNotFound(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	err := runUntrash(nil, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunUntrashJSON(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runUntrash(nil, []string{"task-1"})

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

func TestRunUntrashLoadStateErr(t *testing.T) {
	setupMockStateErr(t, fmt.Errorf("mock load error"))
	err := runUntrash(nil, []string{"task-1"})
	if err == nil || err.Error() != "mock load error" {
		t.Fatalf("expected load error, got %v", err)
	}
}

func TestRunUntrashGetHistoryErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{makeTask("task-1", "Buy milk")})
	mock.getHistoryErr = fmt.Errorf("history error")
	err := runUntrash(nil, []string{"task-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("history error")) {
		t.Fatalf("expected history error, got %v", err)
	}
}

func TestRunUntrashCommitErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{makeTask("task-1", "Buy milk")})
	mock.commitErr = fmt.Errorf("commit error")
	err := runUntrash(nil, []string{"task-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("commit error")) {
		t.Fatalf("expected commit error, got %v", err)
	}
}

func TestRunUntrashUntitledTask(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", ""),
	})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runUntrash(nil, []string{"task-1"})

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
