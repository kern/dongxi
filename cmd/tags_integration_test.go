package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/kern/dongxi/dongxi"
)

func TestRunTags(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTag("tag-1", "Urgent"),
		makeTag("tag-2", "Later"),
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
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Urgent")) {
		t.Error("expected Urgent in output")
	}
	if !bytes.Contains([]byte(output), []byte("2 tag(s)")) {
		t.Error("expected '2 tag(s)' count")
	}
}

func TestRunTagsEmpty(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
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
	if !bytes.Contains(buf.Bytes(), []byte("(no tags)")) {
		t.Error("expected '(no tags)' message")
	}
}

func TestRunTagsWithShortcut(t *testing.T) {
	setupMockState(t, []map[string]any{
		{
			"tag-1": map[string]any{
				dongxi.CommitKeyType:   float64(dongxi.ItemTypeCreate),
				dongxi.CommitKeyEntity: string(dongxi.EntityTag),
				dongxi.CommitKeyPayload: map[string]any{
					dongxi.FieldTitle:    "Urgent",
					dongxi.FieldShortcut: "u",
				},
			},
		},
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
	if !bytes.Contains(buf.Bytes(), []byte("(u)")) {
		t.Error("expected shortcut (u) in output")
	}
}

func TestRunTagsJSON(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTag("tag-1", "Urgent"),
	})

	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

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
	if buf.Len() == 0 {
		t.Error("expected JSON output")
	}
}

func TestRunTag(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeTag("tag-1", "Urgent"),
	})

	err := runTag(nil, []string{"task-1", "tag-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	tags, ok := commit.P[dongxi.FieldTagIDs].([]string)
	if !ok {
		t.Fatal("expected tag IDs to be []string")
	}
	if len(tags) != 1 || tags[0] != "tag-1" {
		t.Errorf("expected [tag-1], got %v", tags)
	}
}

func TestRunTagAlreadyTagged(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk", func(p map[string]any) {
			p[dongxi.FieldTagIDs] = []any{"tag-1"}
		}),
		makeTag("tag-1", "Urgent"),
	})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runTag(nil, []string{"task-1", "tag-1"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("already has tag")) {
		t.Error("expected 'already has tag' message")
	}
}

func TestRunTagNotATask(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
		makeTag("tag-1", "Urgent"),
	})

	err := runTag(nil, []string{"area-1", "tag-1"})
	if err == nil {
		t.Fatal("expected error for non-task entity")
	}
}

func TestRunTagNotATag(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeTask("task-2", "Not a tag"),
	})

	err := runTag(nil, []string{"task-1", "task-2"})
	if err == nil {
		t.Fatal("expected error for non-tag entity")
	}
}

func TestRunTagJSON(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeTag("tag-1", "Urgent"),
	})

	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runTag(nil, []string{"task-1", "tag-1"})

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

func TestRunUntag(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk", func(p map[string]any) {
			p[dongxi.FieldTagIDs] = []any{"tag-1"}
		}),
		makeTag("tag-1", "Urgent"),
	})

	err := runUntag(nil, []string{"task-1", "tag-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	tags := commit.P[dongxi.FieldTagIDs]
	if tags == nil {
		t.Fatal("expected tag IDs to be set")
	}
}

func TestRunUntagNotTagged(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
		makeTag("tag-1", "Urgent"),
	})

	err := runUntag(nil, []string{"task-1", "tag-1"})
	if err == nil {
		t.Fatal("expected error when tag not present")
	}
}

func TestRunUntagNotATask(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
		makeTag("tag-1", "Urgent"),
	})

	err := runUntag(nil, []string{"area-1", "tag-1"})
	if err == nil {
		t.Fatal("expected error for non-task entity")
	}
}

func TestRunUntagJSON(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk", func(p map[string]any) {
			p[dongxi.FieldTagIDs] = []any{"tag-1"}
		}),
		makeTag("tag-1", "Urgent"),
	})

	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runUntag(nil, []string{"task-1", "tag-1"})

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
