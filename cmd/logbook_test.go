package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kern/dongxi/dongxi"
)

func TestRunLogbook(t *testing.T) {
	now := float64(time.Now().Unix())
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Done task", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
			p[dongxi.FieldStopDate] = now
		}),
		makeTask("task-2", "Open task"),
	})

	old := flagLogbookLimit
	flagLogbookLimit = 20
	defer func() { flagLogbookLimit = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runLogbook(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Done task")) {
		t.Error("expected completed task in logbook")
	}
	if bytes.Contains([]byte(output), []byte("Open task")) {
		t.Error("should not show open task in logbook")
	}
}

func TestRunLogbookEmpty(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Open task"),
	})

	old := flagLogbookLimit
	flagLogbookLimit = 20
	defer func() { flagLogbookLimit = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runLogbook(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("(no completed tasks)")) {
		t.Error("expected '(no completed tasks)' message")
	}
}

func TestRunLogbookLimit(t *testing.T) {
	now := float64(time.Now().Unix())
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Done 1", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
			p[dongxi.FieldStopDate] = now - 100
		}),
		makeTask("task-2", "Done 2", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
			p[dongxi.FieldStopDate] = now
		}),
	})

	old := flagLogbookLimit
	flagLogbookLimit = 1
	defer func() { flagLogbookLimit = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runLogbook(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("1 task(s)")) {
		t.Error("expected exactly 1 task in limited logbook")
	}
}

func TestRunLogbookJSON(t *testing.T) {
	now := float64(time.Now().Unix())
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Done task", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
			p[dongxi.FieldStopDate] = now
		}),
	})

	oldJSON := flagJSON
	flagJSON = true
	defer func() { flagJSON = oldJSON }()

	old := flagLogbookLimit
	flagLogbookLimit = 20
	defer func() { flagLogbookLimit = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runLogbook(nil, nil)

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

func TestRunLogbookExcludesTrashed(t *testing.T) {
	now := float64(time.Now().Unix())
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Trashed done", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
			p[dongxi.FieldStopDate] = now
			p[dongxi.FieldTrashed] = true
		}),
	})

	old := flagLogbookLimit
	flagLogbookLimit = 20
	defer func() { flagLogbookLimit = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runLogbook(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("(no completed tasks)")) {
		t.Error("expected no completed tasks for trashed items")
	}
}

func TestRunLogbookZeroStopDate(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Done task", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
			p[dongxi.FieldStopDate] = float64(0)
		}),
	})

	old := flagLogbookLimit
	flagLogbookLimit = 20
	defer func() { flagLogbookLimit = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runLogbook(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	// Should still show the task, just without a date.
	if !bytes.Contains(buf.Bytes(), []byte("Done task")) {
		t.Error("expected task in logbook even with zero stop date")
	}
}

func TestRunLogbookUntitledTask(t *testing.T) {
	now := float64(time.Now().Unix())
	setupMockState(t, []map[string]any{
		makeTask("task-1", "", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
			p[dongxi.FieldStopDate] = now
		}),
	})

	old := flagLogbookLimit
	flagLogbookLimit = 20
	defer func() { flagLogbookLimit = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runLogbook(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("(untitled)")) {
		t.Error("expected (untitled) for task with empty title in logbook")
	}
}

func TestRunLogbookExcludesProjects(t *testing.T) {
	now := float64(time.Now().Unix())
	setupMockState(t, []map[string]any{
		makeProject("proj-1", "Done Project", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
			p[dongxi.FieldStopDate] = now
		}),
	})

	old := flagLogbookLimit
	flagLogbookLimit = 20
	defer func() { flagLogbookLimit = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runLogbook(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("(no completed tasks)")) {
		t.Error("expected no completed tasks since projects should be excluded")
	}
}

// Covers line 38/47: trashed completed task filtered out of logbook
func TestRunLogbookTrashedCompleted(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Good completed", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
			p[dongxi.FieldStopDate] = float64(1700000000)
		}),
		makeTask("task-2", "Trashed completed", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
			p[dongxi.FieldStopDate] = float64(1700000000)
			p[dongxi.FieldTrashed] = true
		}),
	})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runLogbook(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()
	if strings.Contains(out, "Trashed completed") {
		t.Error("trashed completed task should not appear in logbook")
	}
}

// logbook.go:38 — project entity filtered out (type != TaskTypeTask)
func TestRunLogbookFiltersProjects(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Completed task", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
			p[dongxi.FieldStopDate] = float64(1700000000)
		}),
		makeProject("proj-1", "Completed project", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
			p[dongxi.FieldStopDate] = float64(1700000000)
		}),
		makeArea("area-1", "Work"),
	})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runLogbook(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()
	if strings.Contains(out, "Completed project") {
		t.Error("project should not appear in logbook")
	}
	if !strings.Contains(out, "Completed task") {
		t.Error("completed task should appear in logbook")
	}
}
