package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/kern/dongxi/dongxi"
)

func TestRunSearchFindsTask(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeTask("task-2", "Call dentist"),
	})

	old := flagSearchAll
	flagSearchAll = false
	defer func() { flagSearchAll = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runSearch(nil, []string{"milk"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Buy milk")) {
		t.Error("expected to find 'Buy milk' in output")
	}
	if bytes.Contains([]byte(output), []byte("Call dentist")) {
		t.Error("should not find 'Call dentist' in output")
	}
}

func TestRunSearchNoResults(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagSearchAll
	flagSearchAll = false
	defer func() { flagSearchAll = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runSearch(nil, []string{"nonexistent"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("no results")) {
		t.Error("expected 'no results' message")
	}
}

func TestRunSearchAllIncludesCompleted(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
		}),
	})

	old := flagSearchAll
	flagSearchAll = true
	defer func() { flagSearchAll = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runSearch(nil, []string{"milk"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Buy milk")) {
		t.Error("expected to find completed task with --all")
	}
	if !bytes.Contains([]byte(output), []byte("[completed]")) {
		t.Error("expected [completed] annotation")
	}
}

func TestRunSearchAllIncludesTrashed(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk", func(p map[string]any) {
			p[dongxi.FieldTrashed] = true
		}),
	})

	old := flagSearchAll
	flagSearchAll = true
	defer func() { flagSearchAll = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runSearch(nil, []string{"milk"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("[trashed]")) {
		t.Error("expected [trashed] annotation")
	}
}

func TestRunSearchAllCancelledStatus(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCancelled)
		}),
	})

	old := flagSearchAll
	flagSearchAll = true
	defer func() { flagSearchAll = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runSearch(nil, []string{"milk"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("[cancelled]")) {
		t.Error("expected [cancelled] annotation")
	}
}

func TestRunSearchExcludesNonOpen(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
		}),
	})

	old := flagSearchAll
	flagSearchAll = false
	defer func() { flagSearchAll = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runSearch(nil, []string{"milk"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("no results")) {
		t.Error("expected no results for completed task without --all")
	}
}

func TestRunSearchFindsProject(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeProject("proj-1", "My Project"),
	})

	old := flagSearchAll
	flagSearchAll = false
	defer func() { flagSearchAll = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runSearch(nil, []string{"project"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("(project)")) {
		t.Error("expected (project) prefix in output")
	}
}

func TestRunSearchFindsChecklistItem(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeChecklistItem("ci-1", "Get whole milk", "task-1"),
	})

	old := flagSearchAll
	flagSearchAll = false
	defer func() { flagSearchAll = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runSearch(nil, []string{"whole"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("(checklist)")) {
		t.Error("expected (checklist) prefix in output")
	}
}

func TestRunSearchJSON(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	oldAll := flagSearchAll
	flagSearchAll = false
	defer func() { flagSearchAll = oldAll }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runSearch(nil, []string{"milk"})

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

func TestRunSearchUntitledTask(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "", func(p map[string]any) {
			p[dongxi.FieldNote] = dongxi.NewNote("find me in notes")
		}),
	})

	old := flagSearchAll
	flagSearchAll = false
	defer func() { flagSearchAll = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runSearch(nil, []string{"find me"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("(untitled)")) {
		t.Error("expected (untitled) in output for task with empty title")
	}
}

func TestRunSearchExcludesAreas(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work stuff"),
	})

	old := flagSearchAll
	flagSearchAll = true
	defer func() { flagSearchAll = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runSearch(nil, []string{"work"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("no results")) {
		t.Error("expected no results since areas should be excluded from search")
	}
}

func TestRunSearchOrphanedByTrashedParent(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeProject("proj-1", "Trashed Project", func(p map[string]any) {
			p[dongxi.FieldTrashed] = true
		}),
		makeTask("task-1", "Orphaned task", func(p map[string]any) {
			p[dongxi.FieldProjectIDs] = []any{"proj-1"}
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
	})

	old := flagSearchAll
	flagSearchAll = false
	defer func() { flagSearchAll = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runSearch(nil, []string{"orphaned"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("no results")) {
		t.Error("expected no results since task is orphaned by trashed parent")
	}
}
