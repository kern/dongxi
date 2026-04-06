package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/kern/dongxi/dongxi"
)

func TestRunListInbox(t *testing.T) {
	resetListFlags(t)
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Inbox task"),
		makeTask("task-2", "Today task", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
	})

	flagListFilter = "inbox"

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runList(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Inbox task")) {
		t.Error("expected Inbox task in output")
	}
	if bytes.Contains([]byte(output), []byte("Today task")) {
		t.Error("should not show Today task in inbox filter")
	}
}

func TestRunListAnytime(t *testing.T) {
	resetListFlags(t)
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Anytime task", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
		makeTask("task-2", "Today task", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
			withToday(p)
		}),
	})

	flagListFilter = "anytime"

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runList(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Anytime task")) {
		t.Error("expected Anytime task in output")
	}
	if !bytes.Contains([]byte(output), []byte("Today task")) {
		t.Error("expected Today task in anytime filter (anytime includes all)")
	}
}

func TestRunListTodayFiltersAnytime(t *testing.T) {
	resetListFlags(t)
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Anytime only", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
		makeTask("task-2", "Today task", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
			withToday(p)
		}),
	})

	flagListFilter = "today"

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runList(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if bytes.Contains([]byte(output), []byte("Anytime only")) {
		t.Error("should not show anytime-only task in today filter")
	}
	if !bytes.Contains([]byte(output), []byte("Today task")) {
		t.Error("expected Today task in today filter")
	}
}

func TestRunListToday(t *testing.T) {
	resetListFlags(t)
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Inbox task"),
		makeTask("task-2", "Today task", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
			withToday(p)
		}),
	})

	flagListFilter = "today"

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runList(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Today task")) {
		t.Error("expected Today task in output")
	}
}

func TestRunListEvening(t *testing.T) {
	resetListFlags(t)
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Evening task", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
			p[dongxi.FieldStartBucket] = float64(1)
			withToday(p)
		}),
		makeTask("task-2", "Today task", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
			p[dongxi.FieldStartBucket] = float64(0)
			withToday(p)
		}),
	})

	flagListFilter = "evening"

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runList(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Evening task")) {
		t.Error("expected Evening task in output")
	}
	if bytes.Contains([]byte(output), []byte("Today task")) {
		t.Error("should not show non-evening task")
	}
}

func TestRunListSomeday(t *testing.T) {
	resetListFlags(t)
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Someday task", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationSomeday)
		}),
	})

	flagListFilter = "someday"

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runList(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("Someday task")) {
		t.Error("expected Someday task in output")
	}
}

func TestRunListCompleted(t *testing.T) {
	resetListFlags(t)
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Done task", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
		}),
		makeTask("task-2", "Open task"),
	})

	flagListFilter = "completed"

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runList(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("Done task")) {
		t.Error("expected completed task")
	}
}

func TestRunListTrash(t *testing.T) {
	resetListFlags(t)
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Trashed task", func(p map[string]any) {
			p[dongxi.FieldTrashed] = true
		}),
		makeTask("task-2", "Open task"),
	})

	flagListFilter = "trash"

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runList(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("Trashed task")) {
		t.Error("expected trashed task")
	}
}

func TestRunListAll(t *testing.T) {
	resetListFlags(t)
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Inbox task"),
		makeTask("task-2", "Today task", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
	})

	flagListFilter = "all"

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runList(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("2 task(s)")) {
		t.Error("expected 2 tasks in 'all' filter")
	}
}

func TestRunListBadFilter(t *testing.T) {
	resetListFlags(t)
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	flagListFilter = "bad"

	err := runList(nil, nil)
	if err == nil {
		t.Fatal("expected error for bad filter")
	}
}

func TestRunListFilterByProject(t *testing.T) {
	resetListFlags(t)
	setupMockState(t, []map[string]any{
		makeProject("proj-1", "My Project"),
		makeTask("task-1", "In project", func(p map[string]any) {
			p[dongxi.FieldProjectIDs] = []any{"proj-1"}
		}),
		makeTask("task-2", "Not in project"),
	})

	flagListFilter = "all"
	flagListProject = "proj-1"

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runList(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("In project")) {
		t.Error("expected 'In project' task")
	}
	if bytes.Contains([]byte(output), []byte("Not in project")) {
		t.Error("should not show task outside project")
	}
}

func TestRunListFilterByArea(t *testing.T) {
	resetListFlags(t)
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
		makeTask("task-1", "In area", func(p map[string]any) {
			p[dongxi.FieldAreaIDs] = []any{"area-1"}
		}),
		makeTask("task-2", "Not in area"),
	})

	flagListFilter = "all"
	flagListArea = "area-1"

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runList(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("In area")) {
		t.Error("expected 'In area' task")
	}
	if bytes.Contains([]byte(output), []byte("Not in area")) {
		t.Error("should not show task outside area")
	}
}

func TestRunListFilterByAreaInheritedFromProject(t *testing.T) {
	resetListFlags(t)
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
		makeProject("proj-1", "My Project", func(p map[string]any) {
			p[dongxi.FieldAreaIDs] = []any{"area-1"}
		}),
		makeTask("task-1", "Inherited area", func(p map[string]any) {
			p[dongxi.FieldProjectIDs] = []any{"proj-1"}
		}),
	})

	flagListFilter = "all"
	flagListArea = "area-1"

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runList(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("Inherited area")) {
		t.Error("expected task with inherited area from project")
	}
}

func TestRunListFilterByTag(t *testing.T) {
	resetListFlags(t)
	setupMockState(t, []map[string]any{
		makeTag("tag-1", "Urgent"),
		makeTask("task-1", "Tagged", func(p map[string]any) {
			p[dongxi.FieldTagIDs] = []any{"tag-1"}
		}),
		makeTask("task-2", "Untagged"),
	})

	flagListFilter = "all"
	flagListTag = "tag-1"

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runList(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Tagged")) {
		t.Error("expected tagged task")
	}
	if bytes.Contains([]byte(output), []byte("Untagged")) {
		t.Error("should not show untagged task")
	}
}

func TestRunListEmpty(t *testing.T) {
	resetListFlags(t)
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
	})

	flagListFilter = "inbox"

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runList(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("(no tasks)")) {
		t.Error("expected '(no tasks)' message")
	}
}

func TestRunListJSON(t *testing.T) {
	resetListFlags(t)
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	flagListFilter = "inbox"

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runList(nil, nil)

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

func TestRunListProjectWithHeadings(t *testing.T) {
	resetListFlags(t)
	setupMockState(t, []map[string]any{
		makeProject("proj-1", "My Project"),
		makeHeading("heading-1", "Design", "proj-1"),
		makeTask("task-1", "Under heading", func(p map[string]any) {
			p[dongxi.FieldProjectIDs] = []any{"proj-1"}
			p[dongxi.FieldActionGroupIDs] = []any{"heading-1"}
		}),
		makeTask("task-2", "No heading", func(p map[string]any) {
			p[dongxi.FieldProjectIDs] = []any{"proj-1"}
		}),
	})

	flagListFilter = "all"
	flagListProject = "proj-1"

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runList(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Design")) {
		t.Error("expected heading name in output")
	}
	if !bytes.Contains([]byte(output), []byte("Under heading")) {
		t.Error("expected task under heading")
	}
}
