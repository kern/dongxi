package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/kern/dongxi/dongxi"
)

func TestToInt(t *testing.T) {
	tests := []struct {
		input any
		want  int
	}{
		{float64(42), 42},
		{float64(-3.9), -3},
		{float64(0), 0},
		{int(7), 7},
		{int64(99), 99},
		{nil, 0},
		{"hello", 0},
		{true, 0},
	}
	for _, tt := range tests {
		got := toInt(tt.input)
		if got != tt.want {
			t.Errorf("toInt(%v) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestToBool(t *testing.T) {
	tests := []struct {
		input any
		want  bool
	}{
		{true, true},
		{false, false},
		{nil, false},
		{"true", false},
		{1, false},
	}
	for _, tt := range tests {
		got := toBool(tt.input)
		if got != tt.want {
			t.Errorf("toBool(%v) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestToStr(t *testing.T) {
	tests := []struct {
		input any
		want  string
	}{
		{"hello", "hello"},
		{"", ""},
		{nil, ""},
		{42, ""},
		{true, ""},
	}
	for _, tt := range tests {
		got := toStr(tt.input)
		if got != tt.want {
			t.Errorf("toStr(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCopyMap(t *testing.T) {
	original := map[string]any{"a": 1, "b": "hello"}
	copied := copyMap(original)

	if len(copied) != len(original) {
		t.Fatalf("copyMap length %d != %d", len(copied), len(original))
	}
	for k, v := range original {
		if copied[k] != v {
			t.Errorf("copyMap[%s] = %v, want %v", k, copied[k], v)
		}
	}

	// Modifying copy should not affect original.
	copied["a"] = 999
	if original["a"] == 999 {
		t.Error("modifying copy affected original")
	}
}

func TestCopyMapEmpty(t *testing.T) {
	copied := copyMap(map[string]any{})
	if len(copied) != 0 {
		t.Errorf("copyMap(empty) length = %d, want 0", len(copied))
	}
}

func TestHasString(t *testing.T) {
	tests := []struct {
		name   string
		v      any
		target string
		want   bool
	}{
		{"found", []any{"a", "b", "c"}, "b", true},
		{"not found", []any{"a", "b", "c"}, "d", false},
		{"nil", nil, "a", false},
		{"not array", "hello", "h", false},
		{"empty", []any{}, "a", false},
		{"mixed types", []any{"a", 42, "b"}, "b", true},
		{"int target", []any{"a", "42"}, "42", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasString(tt.v, tt.target)
			if got != tt.want {
				t.Errorf("hasString(%v, %q) = %v, want %v", tt.v, tt.target, got, tt.want)
			}
		})
	}
}

func TestReplayHistoryEmpty(t *testing.T) {
	result := replayHistory(nil)
	if len(result) != 0 {
		t.Errorf("replayHistory(nil) returned %d items, want 0", len(result))
	}

	result = replayHistory([]map[string]any{})
	if len(result) != 0 {
		t.Errorf("replayHistory([]) returned %d items, want 0", len(result))
	}
}

func TestReplayHistoryCreate(t *testing.T) {
	commits := []map[string]any{
		{
			"uuid-1": map[string]any{
				dongxi.CommitKeyType:    float64(dongxi.ItemTypeCreate),
				dongxi.CommitKeyEntity:  string(dongxi.EntityTask),
				dongxi.CommitKeyPayload: map[string]any{dongxi.FieldTitle: "Buy milk", dongxi.FieldStatus: float64(dongxi.TaskStatusOpen)},
			},
		},
	}

	result := replayHistory(commits)
	if len(result) != 1 {
		t.Fatalf("got %d items, want 1", len(result))
	}
	if result[0].uuid != "uuid-1" {
		t.Errorf("uuid = %q, want %q", result[0].uuid, "uuid-1")
	}
	if result[0].entity != string(dongxi.EntityTask) {
		t.Errorf("entity = %q, want %q", result[0].entity, string(dongxi.EntityTask))
	}
	if toStr(result[0].fields[dongxi.FieldTitle]) != "Buy milk" {
		t.Errorf("title = %q, want %q", toStr(result[0].fields[dongxi.FieldTitle]), "Buy milk")
	}
}

func TestReplayHistoryCreateThenModify(t *testing.T) {
	commits := []map[string]any{
		{
			"uuid-1": map[string]any{
				dongxi.CommitKeyType:    float64(dongxi.ItemTypeCreate),
				dongxi.CommitKeyEntity:  string(dongxi.EntityTask),
				dongxi.CommitKeyPayload: map[string]any{dongxi.FieldTitle: "", dongxi.FieldStatus: float64(dongxi.TaskStatusOpen), dongxi.FieldDestination: float64(dongxi.TaskDestinationInbox)},
			},
		},
		{
			"uuid-1": map[string]any{
				dongxi.CommitKeyType:    float64(dongxi.ItemTypeModify),
				dongxi.CommitKeyEntity:  string(dongxi.EntityTask),
				dongxi.CommitKeyPayload: map[string]any{dongxi.FieldTitle: "Buy milk", dongxi.FieldModificationDate: float64(12345)},
			},
		},
	}

	result := replayHistory(commits)
	if len(result) != 1 {
		t.Fatalf("got %d items, want 1", len(result))
	}
	if toStr(result[0].fields[dongxi.FieldTitle]) != "Buy milk" {
		t.Errorf("title = %q, want %q", toStr(result[0].fields[dongxi.FieldTitle]), "Buy milk")
	}
	if toInt(result[0].fields[dongxi.FieldStatus]) != int(dongxi.TaskStatusOpen) {
		t.Errorf("ss = %d, want %d", toInt(result[0].fields[dongxi.FieldStatus]), dongxi.TaskStatusOpen)
	}
	if toFloat(result[0].fields[dongxi.FieldModificationDate]) != 12345 {
		t.Errorf("md = %v, want 12345", result[0].fields[dongxi.FieldModificationDate])
	}
}

func TestReplayHistoryCreateThenDelete(t *testing.T) {
	commits := []map[string]any{
		{
			"uuid-1": map[string]any{
				dongxi.CommitKeyType:    float64(dongxi.ItemTypeCreate),
				dongxi.CommitKeyEntity:  string(dongxi.EntityTask),
				dongxi.CommitKeyPayload: map[string]any{dongxi.FieldTitle: "Gone"},
			},
		},
		{
			"uuid-1": map[string]any{
				dongxi.CommitKeyType: float64(dongxi.ItemTypeDelete),
			},
		},
	}

	result := replayHistory(commits)
	if len(result) != 0 {
		t.Errorf("got %d items, want 0 (deleted)", len(result))
	}
}

func TestReplayHistoryModifyNonExistent(t *testing.T) {
	commits := []map[string]any{
		{
			"ghost": map[string]any{
				dongxi.CommitKeyType:    float64(dongxi.ItemTypeModify),
				dongxi.CommitKeyEntity:  string(dongxi.EntityTask),
				dongxi.CommitKeyPayload: map[string]any{dongxi.FieldTitle: "No such item"},
			},
		},
	}

	result := replayHistory(commits)
	if len(result) != 0 {
		t.Errorf("got %d items, want 0", len(result))
	}
}

func TestReplayHistoryOrdering(t *testing.T) {
	commits := []map[string]any{
		{
			"first": map[string]any{
				dongxi.CommitKeyType: float64(dongxi.ItemTypeCreate), dongxi.CommitKeyEntity: string(dongxi.EntityTask),
				dongxi.CommitKeyPayload: map[string]any{dongxi.FieldTitle: "First"},
			},
			"second": map[string]any{
				dongxi.CommitKeyType: float64(dongxi.ItemTypeCreate), dongxi.CommitKeyEntity: string(dongxi.EntityTask),
				dongxi.CommitKeyPayload: map[string]any{dongxi.FieldTitle: "Second"},
			},
		},
		{
			"third": map[string]any{
				dongxi.CommitKeyType: float64(dongxi.ItemTypeCreate), dongxi.CommitKeyEntity: string(dongxi.EntityTask),
				dongxi.CommitKeyPayload: map[string]any{dongxi.FieldTitle: "Third"},
			},
		},
	}

	result := replayHistory(commits)
	if len(result) != 3 {
		t.Fatalf("got %d items, want 3", len(result))
	}
	if result[2].uuid != "third" {
		t.Errorf("last item uuid = %q, want %q", result[2].uuid, "third")
	}
}

func TestReplayHistoryMultipleEntities(t *testing.T) {
	commits := []map[string]any{
		{
			"task-1": map[string]any{
				dongxi.CommitKeyType: float64(dongxi.ItemTypeCreate), dongxi.CommitKeyEntity: string(dongxi.EntityTask),
				dongxi.CommitKeyPayload: map[string]any{dongxi.FieldTitle: "A task", dongxi.FieldType: float64(dongxi.TaskTypeTask)},
			},
			"area-1": map[string]any{
				dongxi.CommitKeyType: float64(dongxi.ItemTypeCreate), dongxi.CommitKeyEntity: string(dongxi.EntityArea),
				dongxi.CommitKeyPayload: map[string]any{dongxi.FieldTitle: "Work"},
			},
			"tag-1": map[string]any{
				dongxi.CommitKeyType: float64(dongxi.ItemTypeCreate), dongxi.CommitKeyEntity: string(dongxi.EntityTag),
				dongxi.CommitKeyPayload: map[string]any{dongxi.FieldTitle: "Important"},
			},
		},
	}

	result := replayHistory(commits)
	if len(result) != 3 {
		t.Fatalf("got %d items, want 3", len(result))
	}

	entities := map[string]bool{}
	for _, item := range result {
		entities[item.entity] = true
	}
	for _, e := range []string{string(dongxi.EntityTask), string(dongxi.EntityArea), string(dongxi.EntityTag)} {
		if !entities[e] {
			t.Errorf("missing entity %s", e)
		}
	}
}

// --- runList tests ---

func resetListFlags(t *testing.T) {
	t.Helper()
	oldFilter := flagListFilter
	oldProject := flagListProject
	oldArea := flagListArea
	oldTag := flagListTag
	oldJSON := flagJSON
	t.Cleanup(func() {
		flagListFilter = oldFilter
		flagListProject = oldProject
		flagListArea = oldArea
		flagListTag = oldTag
		flagJSON = oldJSON
	})
	flagListFilter = "inbox"
	flagListProject = ""
	flagListArea = ""
	flagListTag = ""
	flagJSON = false
}

func TestRunListResolveProjectErr(t *testing.T) {
	resetListFlags(t)
	setupMockState(t, []map[string]any{
		makeTask("task-1", "A task"),
	})
	flagListProject = "nonexistent"
	err := runList(nil, nil)
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("resolve project")) {
		t.Fatalf("expected resolve project error, got %v", err)
	}
}

func TestRunListResolveAreaErr(t *testing.T) {
	resetListFlags(t)
	setupMockState(t, []map[string]any{
		makeTask("task-1", "A task"),
	})
	flagListArea = "nonexistent"
	err := runList(nil, nil)
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("resolve area")) {
		t.Fatalf("expected resolve area error, got %v", err)
	}
}

func TestRunListResolveTagErr(t *testing.T) {
	resetListFlags(t)
	setupMockState(t, []map[string]any{
		makeTask("task-1", "A task"),
	})
	flagListTag = "nonexistent"
	err := runList(nil, nil)
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("resolve tag")) {
		t.Fatalf("expected resolve tag error, got %v", err)
	}
}

func TestRunListProjectGroupedWithHeadings(t *testing.T) {
	resetListFlags(t)
	setupMockState(t, []map[string]any{
		makeProject("proj-1", "My Project"),
		makeHeading("heading-1", "", "proj-1"),
		makeTask("task-1", "", func(p map[string]any) {
			p[dongxi.FieldProjectIDs] = []any{"proj-1"}
			p[dongxi.FieldActionGroupIDs] = []any{"heading-1"}
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
		makeTask("task-2", "Named task", func(p map[string]any) {
			p[dongxi.FieldProjectIDs] = []any{"proj-1"}
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
	})
	flagListFilter = "today"
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
	out := buf.String()
	// Verify untitled task shows (untitled) in project grouped view
	if !bytes.Contains([]byte(out), []byte("(untitled)")) {
		t.Error("expected (untitled) for task with empty title in project grouped view")
	}
}

func TestRunListProjectGroupedMultipleHeadings(t *testing.T) {
	resetListFlags(t)
	setupMockState(t, []map[string]any{
		makeProject("proj-1", "My Project"),
		makeHeading("heading-1", "Design", "proj-1"),
		makeHeading("heading-2", "Development", "proj-1"),
		makeTask("task-1", "No heading task", func(p map[string]any) {
			p[dongxi.FieldProjectIDs] = []any{"proj-1"}
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
		makeTask("task-2", "Design task", func(p map[string]any) {
			p[dongxi.FieldProjectIDs] = []any{"proj-1"}
			p[dongxi.FieldActionGroupIDs] = []any{"heading-1"}
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
		makeTask("task-3", "Dev task", func(p map[string]any) {
			p[dongxi.FieldProjectIDs] = []any{"proj-1"}
			p[dongxi.FieldActionGroupIDs] = []any{"heading-2"}
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
	})
	flagListFilter = "today"
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
	out := buf.String()
	// Multiple headings: the second heading section should have a blank line before it
	if !bytes.Contains([]byte(out), []byte("--- Design ---")) {
		t.Error("expected Design heading in output")
	}
	if !bytes.Contains([]byte(out), []byte("--- Development ---")) {
		t.Error("expected Development heading in output")
	}
}

func TestRunListUntitledTaskNonProjectView(t *testing.T) {
	resetListFlags(t)
	setupMockState(t, []map[string]any{
		makeTask("task-1", ""),
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
	if !bytes.Contains(buf.Bytes(), []byte("(untitled)")) {
		t.Error("expected (untitled) for task with empty title in non-project view")
	}
}

func TestRunListProjectGroupedUnknownHeading(t *testing.T) {
	// Task with an ActionGroupID that doesn't match any known heading -> falls back to ""
	resetListFlags(t)
	setupMockState(t, []map[string]any{
		makeProject("proj-1", "My Project"),
		makeTask("task-1", "Unknown heading task", func(p map[string]any) {
			p[dongxi.FieldProjectIDs] = []any{"proj-1"}
			p[dongxi.FieldActionGroupIDs] = []any{"unknown-heading"}
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
	})
	flagListFilter = "today"
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
	if !bytes.Contains(buf.Bytes(), []byte("Unknown heading task")) {
		t.Error("expected task with unknown heading to still appear")
	}
}

func TestReplayHistoryNonMapValue(t *testing.T) {
	commits := []map[string]any{
		{
			"valid": map[string]any{
				dongxi.CommitKeyType: float64(dongxi.ItemTypeCreate), dongxi.CommitKeyEntity: string(dongxi.EntityTask),
				dongxi.CommitKeyPayload: map[string]any{dongxi.FieldTitle: "Valid"},
			},
			"invalid":      "not a map",
			"also-invalid": float64(42),
		},
	}

	result := replayHistory(commits)
	if len(result) != 1 {
		t.Fatalf("got %d items, want 1", len(result))
	}
	if result[0].uuid != "valid" {
		t.Errorf("uuid = %q, want %q", result[0].uuid, "valid")
	}
}

// Covers line 116: trashed task skipped when filter is NOT "trash"
func TestRunListTrashedTaskSkippedInInbox(t *testing.T) {
	resetListFlags(t)
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Normal task"),
		makeTask("task-2", "Trashed task", func(p map[string]any) {
			p[dongxi.FieldTrashed] = true
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
	out := buf.String()
	if strings.Contains(out, "Trashed task") {
		t.Error("trashed task should be hidden in inbox filter")
	}
	if !strings.Contains(out, "Normal task") {
		t.Error("normal task should appear")
	}
}

// Covers line 127: completed task skipped when filter is not "completed" or "trash"
func TestRunListCompletedTaskSkippedInInbox(t *testing.T) {
	resetListFlags(t)
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Open task"),
		makeTask("task-2", "Completed task", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
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
	out := buf.String()
	if strings.Contains(out, "Completed task") {
		t.Error("completed task should be hidden in inbox filter")
	}
}

// list.go:193 — empty heading group skipped
func TestRunListProjectEmptyHeading(t *testing.T) {
	resetListFlags(t)
	setupMockState(t, []map[string]any{
		makeProject("proj-1", "My Project"),
		makeHeading("heading-1", "Empty heading", "proj-1"),
		makeTask("task-1", "Task without heading", func(p map[string]any) {
			p[dongxi.FieldProjectIDs] = []any{"proj-1"}
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
	})
	flagListFilter = "today"
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
	out := buf.String()
	if strings.Contains(out, "Empty heading") {
		t.Error("empty heading should not appear")
	}
	if !strings.Contains(out, "Task without heading") {
		t.Error("task should appear")
	}
}
