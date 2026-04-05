package cmd

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

func TestRunCreateTag(t *testing.T) {
	mock := setupMockState(t, nil)

	flagCreateTagTitle = "Errand"
	flagCreateTagShortcut = ""

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runCreateTag(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	// Two commits: create + modify.
	if mock.historyIndex != 44 {
		t.Errorf("expected 2 commits (index 44), got index %d", mock.historyIndex)
	}

	for _, item := range mock.lastCommit {
		if item.P[dongxi.FieldTitle] != "Errand" {
			t.Errorf("expected title 'Errand', got %v", item.P[dongxi.FieldTitle])
		}
		if _, ok := item.P[dongxi.FieldShortcut]; ok {
			t.Error("expected no shortcut in modify payload when shortcut is empty")
		}
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()
	if !bytes.Contains([]byte(out), []byte("Created tag")) {
		t.Error("expected 'Created tag' in output")
	}
	if !bytes.Contains([]byte(out), []byte("Errand")) {
		t.Error("expected tag title in output")
	}
}

func TestRunCreateTagWithShortcut(t *testing.T) {
	mock := setupMockState(t, nil)

	flagCreateTagTitle = "Urgent"
	flagCreateTagShortcut = "u"

	err := runCreateTag(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, item := range mock.lastCommit {
		if item.P[dongxi.FieldShortcut] != "u" {
			t.Errorf("expected shortcut 'u', got %v", item.P[dongxi.FieldShortcut])
		}
	}
}

func TestRunCreateTagJSON(t *testing.T) {
	setupMockState(t, nil)

	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	flagCreateTagTitle = "Important"
	flagCreateTagShortcut = ""

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runCreateTag(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()
	if !bytes.Contains([]byte(out), []byte(`"type": "tag"`)) {
		t.Error("expected type 'tag' in JSON output")
	}
	if !bytes.Contains([]byte(out), []byte(`"title": "Important"`)) {
		t.Error("expected title 'Important' in JSON output")
	}
}

func TestRunEditTag(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTag("tag-1", "Urgent"),
	})

	cmd := &cobra.Command{}
	cmd.Flags().StringVar(&flagEditTagTitle, "title", "", "")
	cmd.Flags().StringVar(&flagEditTagShortcut, "shortcut", "", "")
	flagEditTagTitle = "Important"
	_ = cmd.Flags().Set("title", "Important")

	err := runEditTag(cmd, []string{"tag-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["tag-1"]
	if commit.P[dongxi.FieldTitle] != "Important" {
		t.Errorf("expected title 'Important', got %v", commit.P[dongxi.FieldTitle])
	}
}

func TestRunEditTagShortcut(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTag("tag-1", "Urgent"),
	})

	cmd := &cobra.Command{}
	cmd.Flags().StringVar(&flagEditTagTitle, "title", "", "")
	cmd.Flags().StringVar(&flagEditTagShortcut, "shortcut", "", "")
	flagEditTagShortcut = "u"
	_ = cmd.Flags().Set("shortcut", "u")

	err := runEditTag(cmd, []string{"tag-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["tag-1"]
	if commit.P[dongxi.FieldShortcut] != "u" {
		t.Errorf("expected shortcut 'u', got %v", commit.P[dongxi.FieldShortcut])
	}
}

func TestRunEditTagClearShortcut(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTag("tag-1", "Urgent"),
	})

	cmd := &cobra.Command{}
	cmd.Flags().StringVar(&flagEditTagTitle, "title", "", "")
	cmd.Flags().StringVar(&flagEditTagShortcut, "shortcut", "", "")
	flagEditTagShortcut = ""
	_ = cmd.Flags().Set("shortcut", "")

	err := runEditTag(cmd, []string{"tag-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["tag-1"]
	if commit.P[dongxi.FieldShortcut] != nil {
		t.Errorf("expected shortcut nil, got %v", commit.P[dongxi.FieldShortcut])
	}
}

func TestRunEditTagNoChanges(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTag("tag-1", "Urgent"),
	})

	cmd := &cobra.Command{}
	cmd.Flags().StringVar(&flagEditTagTitle, "title", "", "")
	cmd.Flags().StringVar(&flagEditTagShortcut, "shortcut", "", "")

	err := runEditTag(cmd, []string{"tag-1"})
	if err == nil {
		t.Fatal("expected error for no changes")
	}
}

func TestRunEditTagNotATag(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	cmd := &cobra.Command{}
	cmd.Flags().StringVar(&flagEditTagTitle, "title", "", "")
	cmd.Flags().StringVar(&flagEditTagShortcut, "shortcut", "", "")
	_ = cmd.Flags().Set("title", "x")

	err := runEditTag(cmd, []string{"task-1"})
	if err == nil {
		t.Fatal("expected error for non-tag entity")
	}
}

func TestRunEditTagJSON(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTag("tag-1", "Urgent"),
	})

	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	cmd := &cobra.Command{}
	cmd.Flags().StringVar(&flagEditTagTitle, "title", "", "")
	cmd.Flags().StringVar(&flagEditTagShortcut, "shortcut", "", "")
	flagEditTagTitle = "Important"
	_ = cmd.Flags().Set("title", "Important")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runEditTag(cmd, []string{"tag-1"})

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

func TestRunEditTagUntitled(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTag("tag-1", ""),
	})

	cmd := &cobra.Command{}
	cmd.Flags().StringVar(&flagEditTagTitle, "title", "", "")
	cmd.Flags().StringVar(&flagEditTagShortcut, "shortcut", "", "")
	flagEditTagTitle = "Named"
	_ = cmd.Flags().Set("title", "Named")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runEditTag(cmd, []string{"tag-1"})

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

func TestRunDeleteTag(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTag("tag-1", "Urgent"),
	})

	err := runDeleteTag(nil, []string{"tag-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["tag-1"]
	if commit.T != dongxi.ItemTypeDelete {
		t.Errorf("expected delete type, got %v", commit.T)
	}
}

func TestRunDeleteTagNotATag(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	err := runDeleteTag(nil, []string{"task-1"})
	if err == nil {
		t.Fatal("expected error for non-tag entity")
	}
}

func TestRunDeleteTagNotFound(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTag("tag-1", "Urgent"),
	})

	err := runDeleteTag(nil, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent UUID")
	}
}

func TestRunDeleteTagJSON(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTag("tag-1", "Urgent"),
	})

	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDeleteTag(nil, []string{"tag-1"})

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

func TestRunDeleteTagUntitled(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTag("tag-1", ""),
	})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDeleteTag(nil, []string{"tag-1"})

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

func TestRunCreateTagLoadStateErr(t *testing.T) {
	setupMockStateErr(t, fmt.Errorf("mock load error"))
	err := runCreateTag(nil, nil)
	if err == nil || err.Error() != "mock load error" {
		t.Fatalf("expected load error, got %v", err)
	}
}

func TestRunCreateTagGetHistoryErr(t *testing.T) {
	mock := setupMockState(t, nil)
	mock.getHistoryErr = fmt.Errorf("history error")
	err := runCreateTag(nil, nil)
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("history error")) {
		t.Fatalf("expected history error, got %v", err)
	}
}

func TestRunCreateTagCommitCreateErr(t *testing.T) {
	mock := setupMockState(t, nil)
	mock.commitErr = fmt.Errorf("commit error")
	flagCreateTagTitle = "Test"
	err := runCreateTag(nil, nil)
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("commit error")) {
		t.Fatalf("expected commit error, got %v", err)
	}
}

func TestRunCreateTagCommitModifyErr(t *testing.T) {
	mock := setupMockState(t, nil)
	mock.commitErr = fmt.Errorf("commit modify error")
	mock.commitCount = 1
	flagCreateTagTitle = "Test"
	err := runCreateTag(nil, nil)
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("commit modify error")) {
		t.Fatalf("expected commit modify error, got %v", err)
	}
}

func TestRunEditTagLoadStateErr(t *testing.T) {
	setupMockStateErr(t, fmt.Errorf("mock load error"))
	cmd := &cobra.Command{}
	cmd.Flags().StringVar(&flagEditTagTitle, "title", "", "")
	cmd.Flags().StringVar(&flagEditTagShortcut, "shortcut", "", "")
	_ = cmd.Flags().Set("title", "x")
	err := runEditTag(cmd, []string{"tag-1"})
	if err == nil || err.Error() != "mock load error" {
		t.Fatalf("expected load error, got %v", err)
	}
}

func TestRunEditTagGetHistoryErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{makeTag("tag-1", "Urgent")})
	mock.getHistoryErr = fmt.Errorf("history error")
	cmd := &cobra.Command{}
	cmd.Flags().StringVar(&flagEditTagTitle, "title", "", "")
	cmd.Flags().StringVar(&flagEditTagShortcut, "shortcut", "", "")
	_ = cmd.Flags().Set("title", "x")
	err := runEditTag(cmd, []string{"tag-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("history error")) {
		t.Fatalf("expected history error, got %v", err)
	}
}

func TestRunEditTagCommitErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{makeTag("tag-1", "Urgent")})
	mock.commitErr = fmt.Errorf("commit error")
	cmd := &cobra.Command{}
	cmd.Flags().StringVar(&flagEditTagTitle, "title", "", "")
	cmd.Flags().StringVar(&flagEditTagShortcut, "shortcut", "", "")
	_ = cmd.Flags().Set("title", "x")
	err := runEditTag(cmd, []string{"tag-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("commit error")) {
		t.Fatalf("expected commit error, got %v", err)
	}
}

func TestRunEditTagNotFound(t *testing.T) {
	setupMockState(t, []map[string]any{makeTag("tag-1", "Urgent")})
	cmd := &cobra.Command{}
	cmd.Flags().StringVar(&flagEditTagTitle, "title", "", "")
	cmd.Flags().StringVar(&flagEditTagShortcut, "shortcut", "", "")
	_ = cmd.Flags().Set("title", "x")
	err := runEditTag(cmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent UUID")
	}
}

func TestRunDeleteTagLoadStateErr(t *testing.T) {
	setupMockStateErr(t, fmt.Errorf("mock load error"))
	err := runDeleteTag(nil, []string{"tag-1"})
	if err == nil || err.Error() != "mock load error" {
		t.Fatalf("expected load error, got %v", err)
	}
}

func TestRunDeleteTagGetHistoryErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{makeTag("tag-1", "Urgent")})
	mock.getHistoryErr = fmt.Errorf("history error")
	err := runDeleteTag(nil, []string{"tag-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("history error")) {
		t.Fatalf("expected history error, got %v", err)
	}
}

func TestRunDeleteTagCommitErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{makeTag("tag-1", "Urgent")})
	mock.commitErr = fmt.Errorf("commit error")
	err := runDeleteTag(nil, []string{"tag-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("commit error")) {
		t.Fatalf("expected commit error, got %v", err)
	}
}
