package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/kern/dongxi/dongxi"
)

func TestRunShowTask(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runShow(nil, []string{"task-1"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Buy milk")) {
		t.Error("expected task title in output")
	}
	if !bytes.Contains([]byte(output), []byte("task-1")) {
		t.Error("expected UUID in output")
	}
}

func TestRunShowTaskWithAllFields(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
		makeProject("proj-1", "My Project"),
		makeTag("tag-1", "Urgent"),
		makeTask("task-1", "Full task", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusOpen)
			p[dongxi.FieldAreaIDs] = []any{"area-1"}
			p[dongxi.FieldProjectIDs] = []any{"proj-1"}
			p[dongxi.FieldTagIDs] = []any{"tag-1"}
			p[dongxi.FieldCreationDate] = float64(1700000000)
			p[dongxi.FieldModificationDate] = float64(1700001000)
			p[dongxi.FieldScheduledDate] = float64(1700000000)
			p[dongxi.FieldDeadline] = float64(1700100000)
			p[dongxi.FieldNote] = dongxi.NewNote("test note")
			p[dongxi.FieldStartBucket] = float64(1)
			p[dongxi.FieldTrashed] = true
		}),
	})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runShow(nil, []string{"task-1"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Work")) {
		t.Error("expected area name")
	}
	if !bytes.Contains([]byte(output), []byte("My Project")) {
		t.Error("expected project name")
	}
	if !bytes.Contains([]byte(output), []byte("Urgent")) {
		t.Error("expected tag name")
	}
	if !bytes.Contains([]byte(output), []byte("Evening")) {
		t.Error("expected Evening marker")
	}
	if !bytes.Contains([]byte(output), []byte("Trashed")) {
		t.Error("expected Trashed marker")
	}
	if !bytes.Contains([]byte(output), []byte("test note")) {
		t.Error("expected notes in output")
	}
}

func TestRunShowProjectWithProgress(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeProject("proj-1", "My Project"),
		makeTask("task-1", "Task 1", func(p map[string]any) {
			p[dongxi.FieldProjectIDs] = []any{"proj-1"}
		}),
		makeTask("task-2", "Task 2", func(p map[string]any) {
			p[dongxi.FieldProjectIDs] = []any{"proj-1"}
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
		}),
	})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runShow(nil, []string{"proj-1"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Project")) {
		t.Error("expected Project kind")
	}
	if !bytes.Contains([]byte(output), []byte("1/2")) {
		t.Error("expected progress 1/2")
	}
}

func TestRunShowArea(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
	})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runShow(nil, []string{"area-1"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("Work")) {
		t.Error("expected area title in output")
	}
}

func TestRunShowWithChecklist(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeChecklistItem("ci-1", "Step 1", "task-1"),
		{
			"ci-2": map[string]any{
				dongxi.CommitKeyType:   float64(dongxi.ItemTypeCreate),
				dongxi.CommitKeyEntity: string(dongxi.EntityChecklistItem),
				dongxi.CommitKeyPayload: map[string]any{
					dongxi.FieldTitle:   "Step 2",
					dongxi.FieldStatus:  float64(dongxi.TaskStatusCompleted),
					dongxi.FieldTaskIDs: []any{"task-1"},
				},
			},
		},
	})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runShow(nil, []string{"task-1"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Checklist")) {
		t.Error("expected Checklist section")
	}
	if !bytes.Contains([]byte(output), []byte("[ ] Step 1")) {
		t.Error("expected open checklist item")
	}
	if !bytes.Contains([]byte(output), []byte("[x] Step 2")) {
		t.Error("expected completed checklist item")
	}
}

func TestRunShowNotFound(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	err := runShow(nil, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent UUID")
	}
}

func TestRunShowJSON(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runShow(nil, []string{"task-1"})

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

func TestRunShowJSONProject(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeProject("proj-1", "My Project"),
		makeTask("task-1", "Task 1", func(p map[string]any) {
			p[dongxi.FieldProjectIDs] = []any{"proj-1"}
		}),
	})

	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runShow(nil, []string{"proj-1"})

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

func TestRunShowTaskWithAreaUUIDOnly(t *testing.T) {
	// Area exists but task references a UUID not in state for project.
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Task", func(p map[string]any) {
			p[dongxi.FieldAreaIDs] = []any{"missing-area"}
			p[dongxi.FieldProjectIDs] = []any{"missing-proj"}
		}),
	})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runShow(nil, []string{"task-1"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	// Should show UUID as fallback when name is not found.
	if !bytes.Contains([]byte(output), []byte("missing-area")) {
		t.Error("expected area UUID fallback")
	}
	if !bytes.Contains([]byte(output), []byte("missing-proj")) {
		t.Error("expected project UUID fallback")
	}
}

func TestRunShowTagWithUnknownUUID(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Task", func(p map[string]any) {
			p[dongxi.FieldTagIDs] = []any{"unknown-tag"}
		}),
	})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runShow(nil, []string{"task-1"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("unknown-tag")) {
		t.Error("expected tag UUID fallback")
	}
}

func TestRunShowTaskStatuses(t *testing.T) {
	tests := []struct {
		name   string
		status float64
		want   string
	}{
		{"open", float64(dongxi.TaskStatusOpen), "Open"},
		{"completed", float64(dongxi.TaskStatusCompleted), "Completed"},
		{"cancelled", float64(dongxi.TaskStatusCancelled), "Cancelled"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupMockState(t, []map[string]any{
				makeTask("task-1", "Task", func(p map[string]any) {
					p[dongxi.FieldStatus] = tt.status
				}),
			})

			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := runShow(nil, []string{"task-1"})

			w.Close()
			os.Stdout = oldStdout

			if err != nil {
				t.Fatal(err)
			}

			var buf bytes.Buffer
			buf.ReadFrom(r)
			if !bytes.Contains(buf.Bytes(), []byte(tt.want)) {
				t.Errorf("expected status %q in output", tt.want)
			}
		})
	}
}

func TestRunShowHeadingKind(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeProject("proj-1", "My Project"),
		makeHeading("heading-1", "Design", "proj-1"),
	})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runShow(nil, []string{"heading-1"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("Heading")) {
		t.Error("expected Heading kind in output")
	}
}

func TestRunShowUntitled(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", ""),
	})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runShow(nil, []string{"task-1"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("(untitled)")) {
		t.Error("expected (untitled) for task with empty title")
	}
}

func TestRunShowDestinations(t *testing.T) {
	tests := []struct {
		name string
		dest float64
		want string
	}{
		{"inbox", float64(dongxi.TaskDestinationInbox), "Inbox"},
		{"today", float64(dongxi.TaskDestinationAnytime), "Today"},
		{"someday", float64(dongxi.TaskDestinationSomeday), "Someday"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupMockState(t, []map[string]any{
				makeTask("task-1", "Task", func(p map[string]any) {
					p[dongxi.FieldDestination] = tt.dest
				}),
			})

			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := runShow(nil, []string{"task-1"})

			w.Close()
			os.Stdout = oldStdout

			if err != nil {
				t.Fatal(err)
			}

			var buf bytes.Buffer
			buf.ReadFrom(r)
			if !bytes.Contains(buf.Bytes(), []byte(tt.want)) {
				t.Errorf("expected destination %q in output", tt.want)
			}
		})
	}
}

// Covers line 40: JSON show with checklist items
func TestRunShowJSONWithChecklist(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Task with checklist"),
		makeChecklistItem("cl-1", "Step 1", "task-1"),
		makeChecklistItem("cl-2", "Step 2", "task-1"),
	})

	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runShow(nil, []string{"task-1"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var out ItemOutput
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if len(out.Checklist) != 2 {
		t.Errorf("expected 2 checklist items, got %d", len(out.Checklist))
	}
}
