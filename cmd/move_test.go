package cmd

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

func makeMoveCmd(changedFlags ...string) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().StringVar(&flagMoveArea, "area", "", "")
	cmd.Flags().StringVar(&flagMoveProject, "project", "", "")
	cmd.Flags().StringVar(&flagMoveDestination, "destination", "", "")
	cmd.Flags().StringVar(&flagMoveHeading, "heading", "", "")
	for _, name := range changedFlags {
		_ = cmd.Flags().Set(name, cmd.Flags().Lookup(name).DefValue)
	}
	return cmd
}

func TestRunMoveDestinationToday(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	cmd := makeMoveCmd()
	flagMoveDestination = "today"

	err := runMove(cmd, []string{"task-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldDestination] != int(dongxi.TaskDestinationAnytime) {
		t.Errorf("expected anytime destination, got %v", commit.P[dongxi.FieldDestination])
	}
}

func TestRunMoveDestinationInbox(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	cmd := makeMoveCmd()
	flagMoveDestination = "inbox"

	err := runMove(cmd, []string{"task-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldDestination] != int(dongxi.TaskDestinationInbox) {
		t.Errorf("expected inbox destination, got %v", commit.P[dongxi.FieldDestination])
	}
}

func TestRunMoveDestinationSomeday(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	cmd := makeMoveCmd()
	flagMoveDestination = "someday"

	err := runMove(cmd, []string{"task-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldDestination] != int(dongxi.TaskDestinationSomeday) {
		t.Errorf("expected someday destination, got %v", commit.P[dongxi.FieldDestination])
	}
}

func TestRunMoveDestinationEvening(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	cmd := makeMoveCmd()
	flagMoveDestination = "evening"

	err := runMove(cmd, []string{"task-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldStartBucket] != 1 {
		t.Errorf("expected start bucket 1 for evening, got %v", commit.P[dongxi.FieldStartBucket])
	}
}

func TestRunMoveDestinationBad(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	cmd := makeMoveCmd()
	flagMoveDestination = "unknown"

	err := runMove(cmd, []string{"task-1"})
	if err == nil {
		t.Fatal("expected error for unknown destination")
	}
}

func TestRunMoveArea(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeArea("area-1", "Work"),
	})

	flagMoveArea = "area-1"
	cmd := makeMoveCmd("area")
	_ = cmd.Flags().Set("area", "area-1")

	err := runMove(cmd, []string{"task-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	areaIDs, ok := commit.P[dongxi.FieldAreaIDs].([]string)
	if !ok || len(areaIDs) != 1 || areaIDs[0] != "area-1" {
		t.Errorf("expected area-1, got %v", commit.P[dongxi.FieldAreaIDs])
	}
}

func TestRunMoveClearArea(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	flagMoveArea = ""
	cmd := makeMoveCmd("area")

	err := runMove(cmd, []string{"task-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	areaIDs, ok := commit.P[dongxi.FieldAreaIDs].([]string)
	if !ok || len(areaIDs) != 0 {
		t.Errorf("expected empty area IDs, got %v", commit.P[dongxi.FieldAreaIDs])
	}
}

func TestRunMoveAreaNotAnArea(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeTask("task-2", "Not an area"),
	})

	flagMoveArea = "task-2"
	cmd := makeMoveCmd("area")
	_ = cmd.Flags().Set("area", "task-2")

	err := runMove(cmd, []string{"task-1"})
	if err == nil {
		t.Fatal("expected error for non-area entity")
	}
}

func TestRunMoveProject(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeProject("proj-1", "My Project"),
	})

	flagMoveProject = "proj-1"
	cmd := makeMoveCmd("project")
	_ = cmd.Flags().Set("project", "proj-1")

	err := runMove(cmd, []string{"task-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	projIDs, ok := commit.P[dongxi.FieldProjectIDs].([]string)
	if !ok || len(projIDs) != 1 || projIDs[0] != "proj-1" {
		t.Errorf("expected proj-1, got %v", commit.P[dongxi.FieldProjectIDs])
	}
}

func TestRunMoveClearProject(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	flagMoveProject = ""
	cmd := makeMoveCmd("project")

	err := runMove(cmd, []string{"task-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	projIDs, ok := commit.P[dongxi.FieldProjectIDs].([]string)
	if !ok || len(projIDs) != 0 {
		t.Errorf("expected empty project IDs, got %v", commit.P[dongxi.FieldProjectIDs])
	}
}

func TestRunMoveProjectNotAProject(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeTask("task-2", "Not a project"),
	})

	flagMoveProject = "task-2"
	cmd := makeMoveCmd("project")
	_ = cmd.Flags().Set("project", "task-2")

	err := runMove(cmd, []string{"task-1"})
	if err == nil {
		t.Fatal("expected error for non-project entity")
	}
}

func TestRunMoveHeading(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeProject("proj-1", "My Project"),
		makeHeading("heading-1", "Design", "proj-1"),
	})

	flagMoveHeading = "heading-1"
	cmd := makeMoveCmd("heading")
	_ = cmd.Flags().Set("heading", "heading-1")

	err := runMove(cmd, []string{"task-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	headingIDs, ok := commit.P[dongxi.FieldHeadingIDs].([]string)
	if !ok || len(headingIDs) != 1 || headingIDs[0] != "heading-1" {
		t.Errorf("expected heading-1, got %v", commit.P[dongxi.FieldHeadingIDs])
	}
}

func TestRunMoveClearHeading(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	flagMoveHeading = ""
	cmd := makeMoveCmd("heading")

	err := runMove(cmd, []string{"task-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldHeadingIDs] == nil {
		t.Error("expected heading IDs to be set (empty array)")
	}
}

func TestRunMoveHeadingNotAHeading(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeTask("task-2", "Not a heading"),
	})

	flagMoveHeading = "task-2"
	cmd := makeMoveCmd("heading")
	_ = cmd.Flags().Set("heading", "task-2")

	err := runMove(cmd, []string{"task-1"})
	if err == nil {
		t.Fatal("expected error for non-heading entity")
	}
}

func TestRunMoveNoChanges(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	cmd := makeMoveCmd()
	flagMoveDestination = ""

	err := runMove(cmd, []string{"task-1"})
	if err == nil {
		t.Fatal("expected error for no changes")
	}
}

func TestRunMoveNotATask(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
	})

	cmd := makeMoveCmd()
	flagMoveDestination = "today"

	err := runMove(cmd, []string{"area-1"})
	if err == nil {
		t.Fatal("expected error for non-task entity")
	}
}

func TestRunMoveNotFound(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	cmd := makeMoveCmd()
	flagMoveDestination = "today"

	err := runMove(cmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent UUID")
	}
}

func TestRunMoveJSON(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	cmd := makeMoveCmd()
	flagMoveDestination = "today"

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runMove(cmd, []string{"task-1"})

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

func TestRunMoveUntitledTask(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", ""),
	})

	cmd := makeMoveCmd()
	flagMoveDestination = "today"

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runMove(cmd, []string{"task-1"})

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

func TestRunMoveResolveAreaErr(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	flagMoveArea = "nonexistent"
	cmd := makeMoveCmd("area")
	_ = cmd.Flags().Set("area", "nonexistent")

	err := runMove(cmd, []string{"task-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("resolve area")) {
		t.Fatalf("expected resolve area error, got %v", err)
	}
}

func TestRunMoveResolveProjectErr(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	flagMoveProject = "nonexistent"
	cmd := makeMoveCmd("project")
	_ = cmd.Flags().Set("project", "nonexistent")

	err := runMove(cmd, []string{"task-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("resolve project")) {
		t.Fatalf("expected resolve project error, got %v", err)
	}
}

func TestRunMoveResolveHeadingErr(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	flagMoveHeading = "nonexistent"
	cmd := makeMoveCmd("heading")
	_ = cmd.Flags().Set("heading", "nonexistent")

	err := runMove(cmd, []string{"task-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("resolve heading")) {
		t.Fatalf("expected resolve heading error, got %v", err)
	}
}

func TestRunMoveCommitErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})
	mock.commitErr = fmt.Errorf("commit error")

	cmd := makeMoveCmd()
	flagMoveDestination = "today"

	err := runMove(cmd, []string{"task-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("commit")) {
		t.Fatalf("expected commit error, got %v", err)
	}
}

func TestRunMoveGetHistoryErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})
	mock.getHistoryErr = fmt.Errorf("history error")

	cmd := makeMoveCmd()
	flagMoveDestination = "today"

	err := runMove(cmd, []string{"task-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("history error")) {
		t.Fatalf("expected history error, got %v", err)
	}
}

func TestRunMoveLoadStateErr(t *testing.T) {
	setupMockStateErr(t, fmt.Errorf("mock load error"))

	cmd := makeMoveCmd()
	flagMoveDestination = "today"

	err := runMove(cmd, []string{"task-1"})
	if err == nil || err.Error() != "mock load error" {
		t.Fatalf("expected load error, got %v", err)
	}
}
