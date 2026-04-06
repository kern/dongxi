package cmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/kern/dongxi/dongxi"
)

func TestRunTagsUntitled(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTag("tag-1", ""),
	})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runTags(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("(untitled)")) {
		t.Error("expected (untitled) for tag with empty title")
	}
}

func TestRunTagNotFound(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	err := runTag(nil, []string{"task-1", "nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent tag")
	}
}

func TestRunUntagTaskNotFound(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTag("tag-1", "Urgent"),
	})

	err := runUntag(nil, []string{"nonexistent", "tag-1"})
	if err == nil {
		t.Fatal("expected error for nonexistent task")
	}
}

func TestRunUntagTagNotFound(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk", func(p map[string]any) {
			p[dongxi.FieldTagIDs] = []any{"tag-1"}
		}),
	})

	err := runUntag(nil, []string{"task-1", "nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent tag")
	}
}

func TestRunUntagNotOnTask(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeTag("tag-1", "Urgent"),
	})

	err := runUntag(nil, []string{"task-1", "tag-1"})
	if err == nil {
		t.Fatal("expected error when task does not have that tag")
	}
}

func TestRunTagsLoadStateErr(t *testing.T) {
	setupMockStateErr(t, fmt.Errorf("mock load error"))
	err := runTags(nil, nil)
	if err == nil || err.Error() != "mock load error" {
		t.Fatalf("expected load error, got %v", err)
	}
}

func TestRunTagLoadStateErr(t *testing.T) {
	setupMockStateErr(t, fmt.Errorf("mock load error"))
	err := runTag(nil, []string{"task-1", "tag-1"})
	if err == nil || err.Error() != "mock load error" {
		t.Fatalf("expected load error, got %v", err)
	}
}

func TestRunTagGetHistoryErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeTag("tag-1", "Urgent"),
	})
	mock.getHistoryErr = fmt.Errorf("history error")
	err := runTag(nil, []string{"task-1", "tag-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("history error")) {
		t.Fatalf("expected history error, got %v", err)
	}
}

func TestRunTagCommitErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeTag("tag-1", "Urgent"),
	})
	mock.commitErr = fmt.Errorf("commit error")
	err := runTag(nil, []string{"task-1", "tag-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("commit error")) {
		t.Fatalf("expected commit error, got %v", err)
	}
}

func TestRunUntagLoadStateErr(t *testing.T) {
	setupMockStateErr(t, fmt.Errorf("mock load error"))
	err := runUntag(nil, []string{"task-1", "tag-1"})
	if err == nil || err.Error() != "mock load error" {
		t.Fatalf("expected load error, got %v", err)
	}
}

func TestRunUntagGetHistoryErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk", func(p map[string]any) {
			p[dongxi.FieldTagIDs] = []any{"tag-1"}
		}),
		makeTag("tag-1", "Urgent"),
	})
	mock.getHistoryErr = fmt.Errorf("history error")
	err := runUntag(nil, []string{"task-1", "tag-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("history error")) {
		t.Fatalf("expected history error, got %v", err)
	}
}

func TestRunUntagCommitErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk", func(p map[string]any) {
			p[dongxi.FieldTagIDs] = []any{"tag-1"}
		}),
		makeTag("tag-1", "Urgent"),
	})
	mock.commitErr = fmt.Errorf("commit error")
	err := runUntag(nil, []string{"task-1", "tag-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("commit error")) {
		t.Fatalf("expected commit error, got %v", err)
	}
}

func TestToStringSlice(t *testing.T) {
	tests := []struct {
		name string
		v    any
		want []string
	}{
		{"normal", []any{"a", "b", "c"}, []string{"a", "b", "c"}},
		{"mixed", []any{"a", 42, "b"}, []string{"a", "b"}},
		{"nil", nil, nil},
		{"not array", "hello", nil},
		{"empty", []any{}, nil},
		{"single", []any{"only"}, []string{"only"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toStringSlice(tt.v)
			if tt.want == nil {
				if got != nil {
					t.Errorf("toStringSlice(%v) = %v, want nil", tt.v, got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("toStringSlice(%v) length = %d, want %d", tt.v, len(got), len(tt.want))
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("toStringSlice(%v)[%d] = %q, want %q", tt.v, i, got[i], tt.want[i])
				}
			}
		})
	}
}

// Covers line 81-83: runTag where first arg resolves but is not a task entity
func TestRunTagFirstArgNotTask(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
		makeTag("tag-1", "Urgent"),
	})

	err := runTag(nil, []string{"area-1", "tag-1"})
	if err == nil || !strings.Contains(err.Error(), "is not a task") {
		t.Fatalf("expected 'is not a task' error, got %v", err)
	}
}

// Covers line 167-169: runUntag where tag entity is resolved but tag not on task
// (The "task does not have that tag" error from runUntag)
func TestRunUntagTagNotOnTaskCoverage(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "A task"),
		makeTag("tag-1", "Urgent"),
		makeTag("tag-2", "Other"),
	})

	// task-1 has no tags, so untag should fail
	err := runUntag(nil, []string{"task-1", "tag-1"})
	if err == nil || !strings.Contains(err.Error(), "does not have that tag") {
		t.Fatalf("expected 'does not have that tag' error, got %v", err)
	}
}

// tags.go:81 — runTag where first arg UUID not found
func TestRunTagFirstArgNotFound(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTag("tag-1", "Urgent"),
	})
	err := runTag(nil, []string{"nonexistent", "tag-1"})
	if err == nil {
		t.Fatal("expected error for nonexistent first arg")
	}
}

// tags.go:167 — untag keeps other tags in the else branch
func TestRunUntagKeepsOtherTags(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Multi-tagged", func(p map[string]any) {
			p[dongxi.FieldTagIDs] = []any{"tag-1", "tag-2"}
		}),
		makeTag("tag-1", "Tag A"),
		makeTag("tag-2", "Tag B"),
	})

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runUntag(nil, []string{"task-1", "tag-1"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}
}
