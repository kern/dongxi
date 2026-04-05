package cmd

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

func TestRunEditArea(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
	})

	cmd := &cobra.Command{}
	cmd.Flags().StringVar(&flagEditAreaTitle, "title", "", "")
	flagEditAreaTitle = "Office"
	_ = cmd.Flags().Set("title", "Office")

	err := runEditArea(cmd, []string{"area-1"})
	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["area-1"]
	if commit.P[dongxi.FieldTitle] != "Office" {
		t.Errorf("expected title 'Office', got %v", commit.P[dongxi.FieldTitle])
	}
}

func TestRunEditAreaNoChanges(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
	})

	cmd := &cobra.Command{}
	cmd.Flags().StringVar(&flagEditAreaTitle, "title", "", "")

	err := runEditArea(cmd, []string{"area-1"})
	if err == nil {
		t.Fatal("expected error for no changes")
	}
}

func TestRunEditAreaNotAnArea(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	cmd := &cobra.Command{}
	cmd.Flags().StringVar(&flagEditAreaTitle, "title", "", "")
	_ = cmd.Flags().Set("title", "x")

	err := runEditArea(cmd, []string{"task-1"})
	if err == nil {
		t.Fatal("expected error for non-area entity")
	}
}

func TestRunEditAreaNotFound(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
	})

	cmd := &cobra.Command{}
	cmd.Flags().StringVar(&flagEditAreaTitle, "title", "", "")
	_ = cmd.Flags().Set("title", "x")

	err := runEditArea(cmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent UUID")
	}
}

func TestRunEditAreaJSON(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
	})

	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	cmd := &cobra.Command{}
	cmd.Flags().StringVar(&flagEditAreaTitle, "title", "", "")
	flagEditAreaTitle = "Office"
	_ = cmd.Flags().Set("title", "Office")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runEditArea(cmd, []string{"area-1"})

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

func TestRunCreateArea(t *testing.T) {
	mock := setupMockState(t, nil)

	flagCreateAreaTitle = "Personal"

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runCreateArea(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	// Verify two commits happened (create + modify), mock index starts at 42
	// and increments per commit, so after two commits it should be 44.
	if mock.historyIndex != 44 {
		t.Errorf("expected 2 commits (index 44), got index %d", mock.historyIndex)
	}

	// The last commit should be the modify with the title set.
	for _, item := range mock.lastCommit {
		if item.P[dongxi.FieldTitle] != "Personal" {
			t.Errorf("expected title 'Personal', got %v", item.P[dongxi.FieldTitle])
		}
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()
	if !bytes.Contains([]byte(out), []byte("Created area")) {
		t.Error("expected 'Created area' in output")
	}
	if !bytes.Contains([]byte(out), []byte("Personal")) {
		t.Error("expected area title in output")
	}
}

func TestRunCreateAreaJSON(t *testing.T) {
	setupMockState(t, nil)

	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	flagCreateAreaTitle = "Work"

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runCreateArea(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()
	if !bytes.Contains([]byte(out), []byte(`"type": "area"`)) {
		t.Error("expected type 'area' in JSON output")
	}
	if !bytes.Contains([]byte(out), []byte(`"title": "Work"`)) {
		t.Error("expected title 'Work' in JSON output")
	}
}

func TestRunEditAreaUntitled(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", ""),
	})

	cmd := &cobra.Command{}
	cmd.Flags().StringVar(&flagEditAreaTitle, "title", "", "")
	flagEditAreaTitle = "Named"
	_ = cmd.Flags().Set("title", "Named")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runEditArea(cmd, []string{"area-1"})

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

func TestRunCreateAreaLoadStateErr(t *testing.T) {
	setupMockStateErr(t, fmt.Errorf("mock load error"))
	err := runCreateArea(nil, nil)
	if err == nil || err.Error() != "mock load error" {
		t.Fatalf("expected load error, got %v", err)
	}
}

func TestRunCreateAreaGetHistoryErr(t *testing.T) {
	mock := setupMockState(t, nil)
	mock.getHistoryErr = fmt.Errorf("history error")
	err := runCreateArea(nil, nil)
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("history error")) {
		t.Fatalf("expected history error, got %v", err)
	}
}

func TestRunCreateAreaCommitCreateErr(t *testing.T) {
	mock := setupMockState(t, nil)
	mock.commitErr = fmt.Errorf("commit error")
	err := runCreateArea(nil, nil)
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("commit error")) {
		t.Fatalf("expected commit error, got %v", err)
	}
}

func TestRunCreateAreaCommitModifyErr(t *testing.T) {
	mock := setupMockState(t, nil)
	mock.commitErr = fmt.Errorf("commit modify error")
	mock.commitCount = 1 // first commit succeeds, second fails
	err := runCreateArea(nil, nil)
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("commit modify error")) {
		t.Fatalf("expected commit modify error, got %v", err)
	}
}

func TestRunEditAreaLoadStateErr(t *testing.T) {
	setupMockStateErr(t, fmt.Errorf("mock load error"))
	cmd := &cobra.Command{}
	cmd.Flags().StringVar(&flagEditAreaTitle, "title", "", "")
	_ = cmd.Flags().Set("title", "x")
	err := runEditArea(cmd, []string{"area-1"})
	if err == nil || err.Error() != "mock load error" {
		t.Fatalf("expected load error, got %v", err)
	}
}

func TestRunEditAreaGetHistoryErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{makeArea("area-1", "Work")})
	mock.getHistoryErr = fmt.Errorf("history error")
	cmd := &cobra.Command{}
	cmd.Flags().StringVar(&flagEditAreaTitle, "title", "", "")
	_ = cmd.Flags().Set("title", "x")
	err := runEditArea(cmd, []string{"area-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("history error")) {
		t.Fatalf("expected history error, got %v", err)
	}
}

func TestRunEditAreaCommitErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{makeArea("area-1", "Work")})
	mock.commitErr = fmt.Errorf("commit error")
	cmd := &cobra.Command{}
	cmd.Flags().StringVar(&flagEditAreaTitle, "title", "", "")
	_ = cmd.Flags().Set("title", "x")
	err := runEditArea(cmd, []string{"area-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("commit error")) {
		t.Fatalf("expected commit error, got %v", err)
	}
}
