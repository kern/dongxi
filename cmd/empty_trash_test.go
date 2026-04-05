package cmd

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/kern/dongxi/dongxi"
)

func TestRunEmptyTrash(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Trashed task", func(p map[string]any) {
			p[dongxi.FieldTrashed] = true
		}),
		makeTask("task-2", "Open task"),
	})

	old := flagEmptyTrashConfirm
	flagEmptyTrashConfirm = true
	defer func() { flagEmptyTrashConfirm = old }()

	err := runEmptyTrash(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Should only delete the trashed item.
	if len(mock.lastCommit) != 1 {
		t.Fatalf("expected 1 delete, got %d", len(mock.lastCommit))
	}
	commit := mock.lastCommit["task-1"]
	if commit.T != dongxi.ItemTypeDelete {
		t.Errorf("expected delete type, got %v", commit.T)
	}
}

func TestRunEmptyTrashEmpty(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Open task"),
	})

	old := flagEmptyTrashConfirm
	flagEmptyTrashConfirm = true
	defer func() { flagEmptyTrashConfirm = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runEmptyTrash(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("Trash is empty")) {
		t.Error("expected 'Trash is empty' message")
	}
}

func TestRunEmptyTrashRequiresConfirmation(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Trashed task", func(p map[string]any) {
			p[dongxi.FieldTrashed] = true
		}),
	})

	old := flagEmptyTrashConfirm
	flagEmptyTrashConfirm = false
	defer func() { flagEmptyTrashConfirm = old }()

	err := runEmptyTrash(nil, nil)
	if err == nil {
		t.Fatal("expected error when --yes not specified")
	}
}

func TestRunEmptyTrashJSON(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Trashed task", func(p map[string]any) {
			p[dongxi.FieldTrashed] = true
		}),
	})

	oldJSON := flagJSON
	flagJSON = true
	defer func() { flagJSON = oldJSON }()

	old := flagEmptyTrashConfirm
	flagEmptyTrashConfirm = true
	defer func() { flagEmptyTrashConfirm = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runEmptyTrash(nil, nil)

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

func TestRunEmptyTrashLoadStateErr(t *testing.T) {
	setupMockStateErr(t, fmt.Errorf("mock load error"))
	err := runEmptyTrash(nil, nil)
	if err == nil || err.Error() != "mock load error" {
		t.Fatalf("expected load error, got %v", err)
	}
}

func TestRunEmptyTrashGetHistoryErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Trashed", func(p map[string]any) { p[dongxi.FieldTrashed] = true }),
	})
	mock.getHistoryErr = fmt.Errorf("history error")
	old := flagEmptyTrashConfirm
	flagEmptyTrashConfirm = true
	defer func() { flagEmptyTrashConfirm = old }()
	err := runEmptyTrash(nil, nil)
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("history error")) {
		t.Fatalf("expected history error, got %v", err)
	}
}

func TestRunEmptyTrashCommitErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Trashed", func(p map[string]any) { p[dongxi.FieldTrashed] = true }),
	})
	mock.commitErr = fmt.Errorf("commit error")
	old := flagEmptyTrashConfirm
	flagEmptyTrashConfirm = true
	defer func() { flagEmptyTrashConfirm = old }()
	err := runEmptyTrash(nil, nil)
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("commit error")) {
		t.Fatalf("expected commit error, got %v", err)
	}
}
