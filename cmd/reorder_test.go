package cmd

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/kern/dongxi/dongxi"
)

func resetReorderFlags(t *testing.T) {
	t.Helper()
	oldTop := flagReorderTop
	oldBottom := flagReorderBottom
	oldAfter := flagReorderAfter
	oldBefore := flagReorderBefore
	oldToday := flagReorderToday
	t.Cleanup(func() {
		flagReorderTop = oldTop
		flagReorderBottom = oldBottom
		flagReorderAfter = oldAfter
		flagReorderBefore = oldBefore
		flagReorderToday = oldToday
	})
	flagReorderTop = false
	flagReorderBottom = false
	flagReorderAfter = ""
	flagReorderBefore = ""
	flagReorderToday = false
}

func TestRunReorderTop(t *testing.T) {
	resetReorderFlags(t)
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Task A"),
		makeTask("task-2", "Task B"),
	})

	flagReorderTop = true

	err := runReorder(nil, []string{"task-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldIndex] == nil {
		t.Error("expected index to be set")
	}
}

func TestRunReorderBottom(t *testing.T) {
	resetReorderFlags(t)
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Task A"),
		makeTask("task-2", "Task B"),
	})

	flagReorderBottom = true

	err := runReorder(nil, []string{"task-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldIndex] == nil {
		t.Error("expected index to be set")
	}
}

func TestRunReorderAfter(t *testing.T) {
	resetReorderFlags(t)
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Task A"),
		makeTask("task-2", "Task B"),
	})

	flagReorderAfter = "task-2"

	err := runReorder(nil, []string{"task-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldIndex] == nil {
		t.Error("expected index to be set")
	}
}

func TestRunReorderBefore(t *testing.T) {
	resetReorderFlags(t)
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Task A"),
		makeTask("task-2", "Task B"),
	})

	flagReorderBefore = "task-2"

	err := runReorder(nil, []string{"task-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldIndex] == nil {
		t.Error("expected index to be set")
	}
}

func TestRunReorderToday(t *testing.T) {
	resetReorderFlags(t)
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Task A", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
	})

	flagReorderTop = true
	flagReorderToday = true

	err := runReorder(nil, []string{"task-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldTodayIndex] == nil {
		t.Error("expected today index to be set")
	}
}

func TestRunReorderNoOption(t *testing.T) {
	resetReorderFlags(t)
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Task A"),
	})

	err := runReorder(nil, []string{"task-1"})
	if err == nil {
		t.Fatal("expected error when no option specified")
	}
}

func TestRunReorderMultipleOptions(t *testing.T) {
	resetReorderFlags(t)
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Task A"),
	})

	flagReorderTop = true
	flagReorderBottom = true

	err := runReorder(nil, []string{"task-1"})
	if err == nil {
		t.Fatal("expected error when multiple options specified")
	}
}

func TestRunReorderNotFound(t *testing.T) {
	resetReorderFlags(t)
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Task A"),
	})

	flagReorderTop = true

	err := runReorder(nil, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent UUID")
	}
}

func TestRunReorderAfterNotFound(t *testing.T) {
	resetReorderFlags(t)
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Task A"),
	})

	flagReorderAfter = "nonexistent"

	err := runReorder(nil, []string{"task-1"})
	if err == nil {
		t.Fatal("expected error for nonexistent --after UUID")
	}
}

func TestRunReorderBeforeNotFound(t *testing.T) {
	resetReorderFlags(t)
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Task A"),
	})

	flagReorderBefore = "nonexistent"

	err := runReorder(nil, []string{"task-1"})
	if err == nil {
		t.Fatal("expected error for nonexistent --before UUID")
	}
}

func TestRunReorderJSON(t *testing.T) {
	resetReorderFlags(t)
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Task A"),
	})

	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	flagReorderTop = true

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runReorder(nil, []string{"task-1"})

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

func TestRunReorderUntitledTask(t *testing.T) {
	resetReorderFlags(t)
	setupMockState(t, []map[string]any{
		makeTask("task-1", ""),
	})

	flagReorderTop = true

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runReorder(nil, []string{"task-1"})

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

func TestRunReorderLoadStateErr(t *testing.T) {
	resetReorderFlags(t)
	setupMockStateErr(t, fmt.Errorf("mock load error"))
	flagReorderTop = true
	err := runReorder(nil, []string{"task-1"})
	if err == nil || err.Error() != "mock load error" {
		t.Fatalf("expected load error, got %v", err)
	}
}

func TestRunReorderGetHistoryErr(t *testing.T) {
	resetReorderFlags(t)
	mock := setupMockState(t, []map[string]any{makeTask("task-1", "A")})
	mock.getHistoryErr = fmt.Errorf("history error")
	flagReorderTop = true
	err := runReorder(nil, []string{"task-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("history error")) {
		t.Fatalf("expected history error, got %v", err)
	}
}

func TestRunReorderCommitErr(t *testing.T) {
	resetReorderFlags(t)
	mock := setupMockState(t, []map[string]any{makeTask("task-1", "A")})
	mock.commitErr = fmt.Errorf("commit error")
	flagReorderTop = true
	err := runReorder(nil, []string{"task-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("commit error")) {
		t.Fatalf("expected commit error, got %v", err)
	}
}

func TestRunReorderSiblingFiltering(t *testing.T) {
	// Test that sibling filtering skips non-tasks, projects, completed, trashed,
	// and different destinations/projects.
	resetReorderFlags(t)
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Target"),
		makeProject("proj-1", "A Project"), // not a task type
		makeTask("task-2", "Completed", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
		}),
		makeTask("task-3", "Trashed", func(p map[string]any) {
			p[dongxi.FieldTrashed] = true
		}),
		makeTask("task-4", "Different dest", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationSomeday)
		}),
		makeTask("task-5", "Different project", func(p map[string]any) {
			p[dongxi.FieldProjectIDs] = []any{"some-project"}
		}),
		makeArea("area-1", "Work"), // not a task entity
	})

	flagReorderTop = true

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runReorder(nil, []string{"task-1"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}
	if mock.lastCommit == nil {
		t.Fatal("expected a commit")
	}
}

func TestRunReorderTodayFiltering(t *testing.T) {
	// Test today reordering with siblings that have different destinations.
	resetReorderFlags(t)
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Today task", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
		makeTask("task-2", "Inbox task"), // inbox destination, filtered out for today
		makeTask("task-3", "Today sibling", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
	})

	flagReorderTop = true
	flagReorderToday = true

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runReorder(nil, []string{"task-1"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}
}

// Covers lines 125-132: flagReorderBottom path with sibling comparison
func TestRunReorderBottomWithSiblings(t *testing.T) {
	resetReorderFlags(t)
	setupMockState(t, []map[string]any{
		makeTask("task-1", "First", func(p map[string]any) {
			p[dongxi.FieldIndex] = float64(100)
		}),
		makeTask("task-2", "Second", func(p map[string]any) {
			p[dongxi.FieldIndex] = float64(200)
		}),
	})
	flagReorderBottom = true

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runReorder(nil, []string{"task-1"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}
}

// Covers lines 118-124: flagReorderTop path with multiple siblings
func TestRunReorderTopWithSiblings(t *testing.T) {
	resetReorderFlags(t)
	setupMockState(t, []map[string]any{
		makeTask("task-1", "First", func(p map[string]any) {
			p[dongxi.FieldIndex] = float64(500)
		}),
		makeTask("task-2", "Second", func(p map[string]any) {
			p[dongxi.FieldIndex] = float64(100)
		}),
		makeTask("task-3", "Third", func(p map[string]any) {
			p[dongxi.FieldIndex] = float64(300)
		}),
	})
	flagReorderTop = true

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runReorder(nil, []string{"task-1"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}
}
