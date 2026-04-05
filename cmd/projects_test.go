package cmd

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/kern/dongxi/dongxi"
)

func TestRunProjectsOpen(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeProject("proj-1", "Project A"),
		makeProject("proj-2", "Project B", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
		}),
	})

	old := flagProjectsFilter
	flagProjectsFilter = "open"
	defer func() { flagProjectsFilter = old }()

	oldArea := flagProjectsArea
	flagProjectsArea = ""
	defer func() { flagProjectsArea = oldArea }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runProjects(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Project A")) {
		t.Error("expected Project A in output")
	}
	if bytes.Contains([]byte(output), []byte("Project B")) {
		t.Error("should not show completed project in 'open' filter")
	}
}

func TestRunProjectsCompleted(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeProject("proj-1", "Project A"),
		makeProject("proj-2", "Project B", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
		}),
	})

	old := flagProjectsFilter
	flagProjectsFilter = "completed"
	defer func() { flagProjectsFilter = old }()

	oldArea := flagProjectsArea
	flagProjectsArea = ""
	defer func() { flagProjectsArea = oldArea }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runProjects(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("Project B")) {
		t.Error("expected Project B in completed filter")
	}
}

func TestRunProjectsTrash(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeProject("proj-1", "Project A"),
		makeProject("proj-2", "Trashed Project", func(p map[string]any) {
			p[dongxi.FieldTrashed] = true
		}),
	})

	old := flagProjectsFilter
	flagProjectsFilter = "trash"
	defer func() { flagProjectsFilter = old }()

	oldArea := flagProjectsArea
	flagProjectsArea = ""
	defer func() { flagProjectsArea = oldArea }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runProjects(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("Trashed Project")) {
		t.Error("expected Trashed Project in trash filter")
	}
}

func TestRunProjectsAll(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeProject("proj-1", "Project A"),
		makeProject("proj-2", "Project B", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
		}),
	})

	old := flagProjectsFilter
	flagProjectsFilter = "all"
	defer func() { flagProjectsFilter = old }()

	oldArea := flagProjectsArea
	flagProjectsArea = ""
	defer func() { flagProjectsArea = oldArea }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runProjects(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("2 project(s)")) {
		t.Error("expected 2 projects in 'all' filter")
	}
}

func TestRunProjectsBadFilter(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeProject("proj-1", "Project A"),
	})

	old := flagProjectsFilter
	flagProjectsFilter = "bad"
	defer func() { flagProjectsFilter = old }()

	oldArea := flagProjectsArea
	flagProjectsArea = ""
	defer func() { flagProjectsArea = oldArea }()

	err := runProjects(nil, nil)
	if err == nil {
		t.Fatal("expected error for bad filter")
	}
}

func TestRunProjectsFilterByArea(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
		makeProject("proj-1", "Project A", func(p map[string]any) {
			p[dongxi.FieldAreaIDs] = []any{"area-1"}
		}),
		makeProject("proj-2", "Project B"),
	})

	old := flagProjectsFilter
	flagProjectsFilter = "open"
	defer func() { flagProjectsFilter = old }()

	oldArea := flagProjectsArea
	flagProjectsArea = "area-1"
	defer func() { flagProjectsArea = oldArea }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runProjects(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Project A")) {
		t.Error("expected Project A in area-filtered output")
	}
	if bytes.Contains([]byte(output), []byte("Project B")) {
		t.Error("should not show Project B outside area")
	}
}

func TestRunProjectsEmpty(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagProjectsFilter
	flagProjectsFilter = "open"
	defer func() { flagProjectsFilter = old }()

	oldArea := flagProjectsArea
	flagProjectsArea = ""
	defer func() { flagProjectsArea = oldArea }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runProjects(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("(no projects)")) {
		t.Error("expected '(no projects)' message")
	}
}

func TestRunProjectsJSON(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeProject("proj-1", "Project A"),
	})

	oldJSON := flagJSON
	flagJSON = true
	defer func() { flagJSON = oldJSON }()

	old := flagProjectsFilter
	flagProjectsFilter = "open"
	defer func() { flagProjectsFilter = old }()

	oldArea := flagProjectsArea
	flagProjectsArea = ""
	defer func() { flagProjectsArea = oldArea }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runProjects(nil, nil)

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

func TestRunProjectsWithProgress(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeProject("proj-1", "Project A"),
		makeTask("task-1", "Task 1", func(p map[string]any) {
			p[dongxi.FieldProjectIDs] = []any{"proj-1"}
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
		makeTask("task-2", "Task 2", func(p map[string]any) {
			p[dongxi.FieldProjectIDs] = []any{"proj-1"}
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
	})

	old := flagProjectsFilter
	flagProjectsFilter = "open"
	defer func() { flagProjectsFilter = old }()

	oldArea := flagProjectsArea
	flagProjectsArea = ""
	defer func() { flagProjectsArea = oldArea }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runProjects(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("(1/2)")) {
		t.Error("expected progress (1/2) in output")
	}
}

func TestRunProjectsResolveAreaErr(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeProject("proj-1", "Project A"),
	})

	old := flagProjectsFilter
	flagProjectsFilter = "open"
	defer func() { flagProjectsFilter = old }()

	oldArea := flagProjectsArea
	flagProjectsArea = "nonexistent"
	defer func() { flagProjectsArea = oldArea }()

	err := runProjects(nil, nil)
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("resolve area")) {
		t.Fatalf("expected resolve area error, got %v", err)
	}
}

func TestRunProjectsCancelledFilter(t *testing.T) {
	// A cancelled project (status != open) should be excluded in "open" filter
	// This covers the showOpen && !showCompleted && status != Open branch
	setupMockState(t, []map[string]any{
		makeProject("proj-1", "Open Project"),
		makeProject("proj-2", "Cancelled Project", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCancelled)
		}),
	})

	old := flagProjectsFilter
	flagProjectsFilter = "open"
	defer func() { flagProjectsFilter = old }()

	oldArea := flagProjectsArea
	flagProjectsArea = ""
	defer func() { flagProjectsArea = oldArea }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runProjects(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Open Project")) {
		t.Error("expected Open Project")
	}
	if bytes.Contains([]byte(output), []byte("Cancelled Project")) {
		t.Error("should not show Cancelled Project in 'open' filter")
	}
}

func TestRunProjectsUntitledText(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeProject("proj-1", ""),
	})

	old := flagProjectsFilter
	flagProjectsFilter = "open"
	defer func() { flagProjectsFilter = old }()

	oldArea := flagProjectsArea
	flagProjectsArea = ""
	defer func() { flagProjectsArea = oldArea }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runProjects(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("(untitled)")) {
		t.Error("expected (untitled) for project with empty title")
	}
}

func TestRunProjectsLoadStateErr(t *testing.T) {
	setupMockStateErr(t, fmt.Errorf("mock load error"))

	old := flagProjectsFilter
	flagProjectsFilter = "open"
	defer func() { flagProjectsFilter = old }()

	oldArea := flagProjectsArea
	flagProjectsArea = ""
	defer func() { flagProjectsArea = oldArea }()

	err := runProjects(nil, nil)
	if err == nil || err.Error() != "mock load error" {
		t.Fatalf("expected load error, got %v", err)
	}
}
