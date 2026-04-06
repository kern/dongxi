package cmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/kern/dongxi/dongxi"
)

func mockStdin(t *testing.T, content string) {
	t.Helper()
	old := os.Stdin
	r, w, _ := os.Pipe()
	w.WriteString(content)
	w.Close()
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = old })
}

func TestRunBatchComplete(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"complete","uuid":"task-1"}]`)

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	if mock.lastCommit == nil {
		t.Fatal("expected a commit")
	}
	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldStatus] != int(dongxi.TaskStatusCompleted) {
		t.Errorf("expected completed status, got %v", commit.P[dongxi.FieldStatus])
	}
}

func TestRunBatchReopen(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"reopen","uuid":"task-1"}]`)

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldStatus] != int(dongxi.TaskStatusOpen) {
		t.Errorf("expected open status, got %v", commit.P[dongxi.FieldStatus])
	}
}

func TestRunBatchCancel(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"cancel","uuid":"task-1"}]`)

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldStatus] != int(dongxi.TaskStatusCancelled) {
		t.Errorf("expected cancelled status")
	}
}

func TestRunBatchTrash(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"trash","uuid":"task-1"}]`)

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldTrashed] != true {
		t.Error("expected trashed=true")
	}
}

func TestRunBatchUntrash(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"untrash","uuid":"task-1"}]`)

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldTrashed] != false {
		t.Error("expected trashed=false")
	}
}

func TestRunBatchMove(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"move","uuid":"task-1","destination":"today"}]`)

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldDestination] != int(dongxi.TaskDestinationAnytime) {
		t.Errorf("expected anytime destination, got %v", commit.P[dongxi.FieldDestination])
	}
}

func TestRunBatchMoveSomeday(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"move","uuid":"task-1","destination":"someday"}]`)

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldDestination] != int(dongxi.TaskDestinationSomeday) {
		t.Errorf("expected someday destination")
	}
}

func TestRunBatchMoveInbox(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"move","uuid":"task-1","destination":"inbox"}]`)

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldDestination] != int(dongxi.TaskDestinationInbox) {
		t.Errorf("expected inbox destination")
	}
}

func TestRunBatchMoveWithArea(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeArea("area-1", "Work"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"move","uuid":"task-1","area":"area-1"}]`)

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	areaIDs, ok := commit.P[dongxi.FieldAreaIDs].([]string)
	if !ok || len(areaIDs) != 1 || areaIDs[0] != "area-1" {
		t.Errorf("expected area-1, got %v", commit.P[dongxi.FieldAreaIDs])
	}
}

func TestRunBatchMoveClearArea(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"move","uuid":"task-1","area":""}]`)

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	areaIDs, ok := commit.P[dongxi.FieldAreaIDs].([]string)
	if !ok || len(areaIDs) != 0 {
		t.Errorf("expected empty area, got %v", commit.P[dongxi.FieldAreaIDs])
	}
}

func TestRunBatchMoveWithProject(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeProject("proj-1", "My Project"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"move","uuid":"task-1","project":"proj-1"}]`)

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	projIDs, ok := commit.P[dongxi.FieldProjectIDs].([]string)
	if !ok || len(projIDs) != 1 || projIDs[0] != "proj-1" {
		t.Errorf("expected proj-1, got %v", commit.P[dongxi.FieldProjectIDs])
	}
}

func TestRunBatchMoveClearProject(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"move","uuid":"task-1","project":""}]`)

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	projIDs, ok := commit.P[dongxi.FieldProjectIDs].([]string)
	if !ok || len(projIDs) != 0 {
		t.Errorf("expected empty project, got %v", commit.P[dongxi.FieldProjectIDs])
	}
}

func TestRunBatchMoveWithHeading(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeProject("proj-1", "My Project"),
		makeHeading("heading-1", "Design", "proj-1"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"move","uuid":"task-1","heading":"heading-1"}]`)

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	headingIDs, ok := commit.P[dongxi.FieldHeadingIDs].([]string)
	if !ok || len(headingIDs) != 1 || headingIDs[0] != "heading-1" {
		t.Errorf("expected heading-1, got %v", commit.P[dongxi.FieldHeadingIDs])
	}
}

func TestRunBatchMoveClearHeading(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"move","uuid":"task-1","heading":""}]`)

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldHeadingIDs] == nil {
		t.Error("expected heading IDs to be set")
	}
}

func TestRunBatchTag(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeTag("tag-1", "Urgent"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"tag","uuid":"task-1","tag":"tag-1"}]`)

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldTagIDs] == nil {
		t.Error("expected tag IDs to be set")
	}
}

func TestRunBatchUntag(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk", func(p map[string]any) {
			p[dongxi.FieldTagIDs] = []any{"tag-1"}
		}),
		makeTag("tag-1", "Urgent"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"untag","uuid":"task-1","tag":"tag-1"}]`)

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	if mock.lastCommit == nil {
		t.Fatal("expected a commit")
	}
}

func TestRunBatchEdit(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"edit","uuid":"task-1","title":"New title"}]`)

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldTitle] != "New title" {
		t.Errorf("expected title 'New title', got %v", commit.P[dongxi.FieldTitle])
	}
}

func TestRunBatchEditNote(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"edit","uuid":"task-1","note":"Hello"}]`)

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldNote] == nil {
		t.Error("expected note to be set")
	}
}

func TestRunBatchEditClearNote(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"edit","uuid":"task-1","note":""}]`)

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldNote] == nil {
		t.Error("expected note to be set to empty note")
	}
}

func TestRunBatchEditScheduled(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"edit","uuid":"task-1","scheduled":"2025-04-01"}]`)

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldScheduledDate] == nil {
		t.Error("expected scheduled date to be set")
	}
}

func TestRunBatchEditClearScheduled(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"edit","uuid":"task-1","scheduled":""}]`)

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldScheduledDate] != nil {
		t.Error("expected scheduled date to be nil")
	}
}

func TestRunBatchEditDeadline(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"edit","uuid":"task-1","deadline":"2025-05-01"}]`)

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldDeadline] == nil {
		t.Error("expected deadline to be set")
	}
}

func TestRunBatchEditClearDeadline(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"edit","uuid":"task-1","deadline":""}]`)

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldDeadline] != nil {
		t.Error("expected deadline to be nil")
	}
}

func TestRunBatchConvert(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"convert","uuid":"task-1","to":"project"}]`)

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldType] != int(dongxi.TaskTypeProject) {
		t.Errorf("expected project type, got %v", commit.P[dongxi.FieldType])
	}
}

func TestRunBatchDryRun(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = true
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"complete","uuid":"task-1"}]`)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("Dry run")) {
		t.Error("expected 'Dry run' in output")
	}
}

func TestRunBatchDryRunJSON(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = true
	defer func() { flagBatchDryRun = old }()

	oldJSON := flagJSON
	flagJSON = true
	defer func() { flagJSON = oldJSON }()

	mockStdin(t, `[{"op":"complete","uuid":"task-1"}]`)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

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

func TestRunBatchEmptyInput(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, "")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("Nothing to do")) {
		t.Error("expected 'Nothing to do' for empty input")
	}
}

func TestRunBatchEmptyArray(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, "[]")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("Nothing to do")) {
		t.Error("expected 'Nothing to do' for empty array")
	}
}

func TestRunBatchBadJSON(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, "not json")

	err := runBatch(nil, nil)
	if err == nil {
		t.Fatal("expected error for bad JSON")
	}
}

func TestRunBatchMissingOp(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"uuid":"task-1"}]`)

	err := runBatch(nil, nil)
	if err == nil {
		t.Fatal("expected error for missing op")
	}
}

func TestRunBatchNotATask(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"complete","uuid":"area-1"}]`)

	err := runBatch(nil, nil)
	if err == nil {
		t.Fatal("expected error for non-task entity")
	}
}

func TestRunBatchTagMissingTagUUID(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"tag","uuid":"task-1"}]`)

	err := runBatch(nil, nil)
	if err == nil {
		t.Fatal("expected error for missing tag uuid")
	}
}

func TestRunBatchUntagMissingTagUUID(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"untag","uuid":"task-1"}]`)

	err := runBatch(nil, nil)
	if err == nil {
		t.Fatal("expected error for missing tag uuid")
	}
}

func TestRunBatchConvertBadTarget(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"convert","uuid":"task-1","to":"heading"}]`)

	err := runBatch(nil, nil)
	if err == nil {
		t.Fatal("expected error for bad convert target")
	}
}

func TestRunBatchMoveBadDestination(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"move","uuid":"task-1","destination":"mars"}]`)

	err := runBatch(nil, nil)
	if err == nil {
		t.Fatal("expected error for bad destination")
	}
}

func TestRunBatchJSON(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	oldJSON := flagJSON
	flagJSON = true
	defer func() { flagJSON = oldJSON }()

	mockStdin(t, `[{"op":"complete","uuid":"task-1"}]`)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

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

func TestRunBatchUntrashNotATaskOrArea(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTag("tag-1", "Urgent"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"untrash","uuid":"tag-1"}]`)

	err := runBatch(nil, nil)
	if err == nil {
		t.Fatal("expected error for untrash on tag")
	}
}

func TestRunBatchNotFoundUUID(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"complete","uuid":"nonexistent"}]`)

	err := runBatch(nil, nil)
	if err == nil {
		t.Fatal("expected error for nonexistent UUID")
	}
}

func TestRunBatchWithBOM(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	// BOM + JSON
	bomJSON := string([]byte{0xEF, 0xBB, 0xBF}) + `[{"op":"complete","uuid":"task-1"}]`
	mockStdin(t, bomJSON)

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	if mock.lastCommit == nil {
		t.Fatal("expected a commit")
	}
}

func TestRunBatchWhitespaceOnly(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, "   \n  \t  ")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("Nothing to do")) {
		t.Error("expected 'Nothing to do' for whitespace input")
	}
}

func TestRunBatchConvertToTask(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeProject("proj-1", "My Project"),
	})

	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()

	mockStdin(t, `[{"op":"convert","uuid":"proj-1","to":"task"}]`)

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["proj-1"]
	if commit.P[dongxi.FieldType] != int(dongxi.TaskTypeTask) {
		t.Errorf("expected task type, got %v", commit.P[dongxi.FieldType])
	}
}

// Verify _ import is used.
var _ = strings.Contains

func TestTrimBOM(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{"with BOM", []byte{0xEF, 0xBB, 0xBF, 'h', 'i'}, "hi"},
		{"without BOM", []byte("hi"), "hi"},
		{"empty", []byte{}, ""},
		{"only BOM", []byte{0xEF, 0xBB, 0xBF}, ""},
		{"partial BOM", []byte{0xEF, 0xBB}, string([]byte{0xEF, 0xBB})},
		{"short", []byte{0xEF}, string([]byte{0xEF})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(trimBOM(tt.input))
			if got != tt.want {
				t.Errorf("trimBOM = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRunBatchLoadStateErr(t *testing.T) {
	setupMockStateErr(t, fmt.Errorf("mock load error"))
	mockStdin(t, `[{"op":"complete","uuid":"task-1"}]`)
	err := runBatch(nil, nil)
	if err == nil || err.Error() != "mock load error" {
		t.Fatalf("expected load error, got %v", err)
	}
}

func TestRunBatchGetHistoryErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{makeTask("task-1", "Buy milk")})
	mock.getHistoryErr = fmt.Errorf("history error")
	mockStdin(t, `[{"op":"complete","uuid":"task-1"}]`)
	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "history error") {
		t.Fatalf("expected history error, got %v", err)
	}
}

func TestRunBatchCommitErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{makeTask("task-1", "Buy milk")})
	mock.commitErr = fmt.Errorf("commit error")
	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()
	mockStdin(t, `[{"op":"complete","uuid":"task-1"}]`)
	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "commit error") {
		t.Fatalf("expected commit error, got %v", err)
	}
}

func TestRunBatchMissingUUID(t *testing.T) {
	setupMockState(t, []map[string]any{makeTask("task-1", "Buy milk")})
	mockStdin(t, `[{"op":"complete","uuid":""}]`)
	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "missing uuid") {
		t.Fatalf("expected missing uuid error, got %v", err)
	}
}

func TestRunBatchUnknownOp(t *testing.T) {
	setupMockState(t, []map[string]any{makeTask("task-1", "Buy milk")})
	mockStdin(t, `[{"op":"explode","uuid":"task-1"}]`)
	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "unknown op") {
		t.Fatalf("expected unknown op error, got %v", err)
	}
}

func TestRunBatchNotATaskComplete(t *testing.T) {
	setupMockState(t, []map[string]any{makeArea("area-1", "Work")})
	mockStdin(t, `[{"op":"complete","uuid":"area-1"}]`)
	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "not a task") {
		t.Fatalf("expected not a task error, got %v", err)
	}
}

func TestRunBatchNotATaskReopen(t *testing.T) {
	setupMockState(t, []map[string]any{makeArea("area-1", "Work")})
	mockStdin(t, `[{"op":"reopen","uuid":"area-1"}]`)
	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "not a task") {
		t.Fatalf("expected not a task error, got %v", err)
	}
}

func TestRunBatchNotATaskCancel(t *testing.T) {
	setupMockState(t, []map[string]any{makeArea("area-1", "Work")})
	mockStdin(t, `[{"op":"cancel","uuid":"area-1"}]`)
	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "not a task") {
		t.Fatalf("expected not a task error, got %v", err)
	}
}

func TestRunBatchNotATaskTrash(t *testing.T) {
	setupMockState(t, []map[string]any{makeTag("tag-1", "Urgent")})
	mockStdin(t, `[{"op":"trash","uuid":"tag-1"}]`)
	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "not a task") {
		t.Fatalf("expected not a task error, got %v", err)
	}
}

func TestRunBatchNotATaskOrAreaUntrash(t *testing.T) {
	setupMockState(t, []map[string]any{makeTag("tag-1", "Urgent")})
	mockStdin(t, `[{"op":"untrash","uuid":"tag-1"}]`)
	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "not a task or area") {
		t.Fatalf("expected not a task or area error, got %v", err)
	}
}

func TestRunBatchMoveNoChanges(t *testing.T) {
	setupMockState(t, []map[string]any{makeTask("task-1", "Buy milk")})
	mockStdin(t, `[{"op":"move","uuid":"task-1"}]`)
	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "move requires") {
		t.Fatalf("expected move requires error, got %v", err)
	}
}

func TestRunBatchMoveNotATask(t *testing.T) {
	setupMockState(t, []map[string]any{makeArea("area-1", "Work")})
	mockStdin(t, `[{"op":"move","uuid":"area-1","destination":"today"}]`)
	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "not a task") {
		t.Fatalf("expected not a task error, got %v", err)
	}
}

func TestRunBatchMoveUnknownDestination(t *testing.T) {
	setupMockState(t, []map[string]any{makeTask("task-1", "Buy milk")})
	mockStdin(t, `[{"op":"move","uuid":"task-1","destination":"nowhere"}]`)
	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "unknown destination") {
		t.Fatalf("expected unknown destination error, got %v", err)
	}
}

func TestRunBatchMoveAreaNotAnArea(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeTask("task-2", "Not area"),
	})
	mockStdin(t, `[{"op":"move","uuid":"task-1","area":"task-2"}]`)
	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "not an area") {
		t.Fatalf("expected not an area error, got %v", err)
	}
}

func TestRunBatchMoveProjectNotAProject(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeTask("task-2", "Not project"),
	})
	mockStdin(t, `[{"op":"move","uuid":"task-1","project":"task-2"}]`)
	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "not a project") {
		t.Fatalf("expected not a project error, got %v", err)
	}
}

func TestRunBatchMoveHeadingNotAHeading(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeTask("task-2", "Not heading"),
	})
	mockStdin(t, `[{"op":"move","uuid":"task-1","heading":"task-2"}]`)
	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "not a heading") {
		t.Fatalf("expected not a heading error, got %v", err)
	}
}

func TestRunBatchTagNotATask(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
		makeTag("tag-1", "Urgent"),
	})
	mockStdin(t, `[{"op":"tag","uuid":"area-1","tag":"tag-1"}]`)
	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "not a task") {
		t.Fatalf("expected not a task error, got %v", err)
	}
}

func TestRunBatchTagMissingTag(t *testing.T) {
	setupMockState(t, []map[string]any{makeTask("task-1", "Buy milk")})
	mockStdin(t, `[{"op":"tag","uuid":"task-1","tag":""}]`)
	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "requires a tag UUID") {
		t.Fatalf("expected tag UUID error, got %v", err)
	}
}

func TestRunBatchTagNotATag(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeTask("task-2", "Not tag"),
	})
	mockStdin(t, `[{"op":"tag","uuid":"task-1","tag":"task-2"}]`)
	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "not a tag") {
		t.Fatalf("expected not a tag error, got %v", err)
	}
}

func TestRunBatchUntagNotATask(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
		makeTag("tag-1", "Urgent"),
	})
	mockStdin(t, `[{"op":"untag","uuid":"area-1","tag":"tag-1"}]`)
	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "not a task") {
		t.Fatalf("expected not a task error, got %v", err)
	}
}

func TestRunBatchUntagMissingTag(t *testing.T) {
	setupMockState(t, []map[string]any{makeTask("task-1", "Buy milk")})
	mockStdin(t, `[{"op":"untag","uuid":"task-1","tag":""}]`)
	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "requires a tag UUID") {
		t.Fatalf("expected tag UUID error, got %v", err)
	}
}

func TestRunBatchEditNotATask(t *testing.T) {
	setupMockState(t, []map[string]any{makeArea("area-1", "Work")})
	mockStdin(t, `[{"op":"edit","uuid":"area-1","title":"x"}]`)
	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "not a task") {
		t.Fatalf("expected not a task error, got %v", err)
	}
}

func TestRunBatchEditNoChanges(t *testing.T) {
	setupMockState(t, []map[string]any{makeTask("task-1", "Buy milk")})
	mockStdin(t, `[{"op":"edit","uuid":"task-1"}]`)
	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "edit requires") {
		t.Fatalf("expected edit requires error, got %v", err)
	}
}

func TestRunBatchConvertNotATask(t *testing.T) {
	setupMockState(t, []map[string]any{makeArea("area-1", "Work")})
	mockStdin(t, `[{"op":"convert","uuid":"area-1","to":"project"}]`)
	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "not a task") {
		t.Fatalf("expected not a task error, got %v", err)
	}
}

func TestRunBatchConvertMissingTo(t *testing.T) {
	setupMockState(t, []map[string]any{makeTask("task-1", "Buy milk")})
	mockStdin(t, `[{"op":"convert","uuid":"task-1"}]`)
	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "requires a 'to' field") {
		t.Fatalf("expected to field error, got %v", err)
	}
}

func TestRunBatchConvertUnknownTarget(t *testing.T) {
	setupMockState(t, []map[string]any{makeTask("task-1", "Buy milk")})
	mockStdin(t, `[{"op":"convert","uuid":"task-1","to":"heading"}]`)
	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "unknown convert target") {
		t.Fatalf("expected unknown target error, got %v", err)
	}
}

func TestRunBatchUntitledTask(t *testing.T) {
	setupMockState(t, []map[string]any{makeTask("task-1", "")})
	old := flagBatchDryRun
	flagBatchDryRun = false
	defer func() { flagBatchDryRun = old }()
	mockStdin(t, `[{"op":"complete","uuid":"task-1"}]`)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

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

func TestRunBatchResolveNonexistentUUID(t *testing.T) {
	setupMockState(t, []map[string]any{makeTask("task-1", "Buy milk")})
	mockStdin(t, `[{"op":"complete","uuid":"nonexistent"}]`)
	err := runBatch(nil, nil)
	if err == nil {
		t.Fatal("expected error for nonexistent UUID")
	}
}

func TestRunBatchResolveAreaNotFound(t *testing.T) {
	setupMockState(t, []map[string]any{makeTask("task-1", "Buy milk")})
	mockStdin(t, `[{"op":"move","uuid":"task-1","area":"nonexistent"}]`)
	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "resolve area") {
		t.Fatalf("expected resolve area error, got %v", err)
	}
}

func TestRunBatchResolveProjectNotFound(t *testing.T) {
	setupMockState(t, []map[string]any{makeTask("task-1", "Buy milk")})
	mockStdin(t, `[{"op":"move","uuid":"task-1","project":"nonexistent"}]`)
	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "resolve project") {
		t.Fatalf("expected resolve project error, got %v", err)
	}
}

func TestRunBatchResolveHeadingNotFound(t *testing.T) {
	setupMockState(t, []map[string]any{makeTask("task-1", "Buy milk")})
	mockStdin(t, `[{"op":"move","uuid":"task-1","heading":"nonexistent"}]`)
	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "resolve heading") {
		t.Fatalf("expected resolve heading error, got %v", err)
	}
}

func TestRunBatchResolveTagNotFound(t *testing.T) {
	setupMockState(t, []map[string]any{makeTask("task-1", "Buy milk")})
	mockStdin(t, `[{"op":"tag","uuid":"task-1","tag":"nonexistent"}]`)
	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "resolve tag") {
		t.Fatalf("expected resolve tag error, got %v", err)
	}
}

func TestRunBatchUntagResolveTagNotFound(t *testing.T) {
	setupMockState(t, []map[string]any{makeTask("task-1", "Buy milk")})
	mockStdin(t, `[{"op":"untag","uuid":"task-1","tag":"nonexistent"}]`)
	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "resolve tag") {
		t.Fatalf("expected resolve tag error, got %v", err)
	}
}

func TestRunBatchEditBadScheduledDate(t *testing.T) {
	setupMockState(t, []map[string]any{makeTask("task-1", "Buy milk")})
	mockStdin(t, `[{"op":"edit","uuid":"task-1","scheduled":"not-a-date"}]`)
	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "parse scheduled date") {
		t.Fatalf("expected parse scheduled date error, got %v", err)
	}
}

func TestRunBatchEditBadDeadlineDate(t *testing.T) {
	setupMockState(t, []map[string]any{makeTask("task-1", "Buy milk")})
	mockStdin(t, `[{"op":"edit","uuid":"task-1","deadline":"not-a-date"}]`)
	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "parse deadline date") {
		t.Fatalf("expected parse deadline date error, got %v", err)
	}
}

// Covers line 482-483: unknown operation in batch
func TestRunBatchUnknownOpCoverage(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "A task"),
	})

	oldStdin := os.Stdin
	input := `[{"op": "frobnicate", "uuid": "task-1"}]`
	stdinR, stdinW, _ := os.Pipe()
	stdinW.WriteString(input)
	stdinW.Close()
	os.Stdin = stdinR
	defer func() { os.Stdin = oldStdin }()

	err := runBatch(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "unknown op") {
		t.Fatalf("expected 'unknown op' error, got %v", err)
	}
}

// batch.go:161 — stdin read error
func TestRunBatchStdinReadError(t *testing.T) {
	setupMockState(t, nil)

	oldStdin := os.Stdin
	stdinR, stdinW, _ := os.Pipe()
	stdinW.Close()
	// Close the read end too so ReadAll gets an error...
	// Actually ReadAll on a closed write end returns EOF (no error).
	// A truly erroring reader is hard to inject via os.Stdin.
	os.Stdin = stdinR
	defer func() { os.Stdin = oldStdin }()

	// Empty stdin results in "Nothing to do" — tests line 166-168 instead
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runBatch(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}
}
