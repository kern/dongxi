package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/kern/dongxi/dongxi"
)

// mockSyncer implements Syncer for testing.
type mockSyncer struct {
	result *syncResult
	err    error
}

func (m *mockSyncer) Sync() (*syncResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func setupMockSyncer(t *testing.T, result *syncResult) {
	t.Helper()
	orig := syncer
	syncer = &mockSyncer{result: result}
	t.Cleanup(func() { syncer = orig })
}

func setupMockSyncerErr(t *testing.T, err error) {
	t.Helper()
	orig := syncer
	syncer = &mockSyncer{err: err}
	t.Cleanup(func() { syncer = orig })
}

func TestRunSyncJSON(t *testing.T) {
	newItems := []map[string]any{
		{
			"uuid1": map[string]any{
				dongxi.CommitKeyType:    float64(dongxi.ItemTypeCreate),
				dongxi.CommitKeyEntity:  string(dongxi.EntityTask),
				dongxi.CommitKeyPayload: map[string]any{},
			},
		},
		{
			"uuid2": map[string]any{
				dongxi.CommitKeyType:    float64(dongxi.ItemTypeModify),
				dongxi.CommitKeyEntity:  string(dongxi.EntityArea),
				dongxi.CommitKeyPayload: map[string]any{},
			},
		},
	}

	setupMockSyncer(t, &syncResult{
		cachedBefore: 10,
		newItems:     newItems,
		totalItems:   12,
	})

	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runSync(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("sync: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var out SyncOutput
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v\noutput: %s", err, buf.String())
	}
	if out.CachedItems != 10 {
		t.Errorf("CachedItems = %d, want 10", out.CachedItems)
	}
	if out.NewItems != 2 {
		t.Errorf("NewItems = %d, want 2", out.NewItems)
	}
	if out.TotalItems != 12 {
		t.Errorf("TotalItems = %d, want 12", out.TotalItems)
	}
	if out.Summary["task created"] != 1 {
		t.Errorf("task created = %d, want 1", out.Summary["task created"])
	}
}

func TestRunSyncTable(t *testing.T) {
	setupMockSyncer(t, &syncResult{
		cachedBefore: 5,
		newItems:     nil,
		totalItems:   5,
	})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runSync(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("sync: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !bytes.Contains([]byte(output), []byte("Already up to date")) {
		t.Errorf("expected 'Already up to date' in output, got: %s", output)
	}
}

func TestRunSyncWithChanges(t *testing.T) {
	newItems := []map[string]any{
		{
			"uuid1": map[string]any{
				dongxi.CommitKeyType:    float64(dongxi.ItemTypeCreate),
				dongxi.CommitKeyEntity:  string(dongxi.EntityTask),
				dongxi.CommitKeyPayload: map[string]any{},
			},
		},
	}

	setupMockSyncer(t, &syncResult{
		cachedBefore: 3,
		newItems:     newItems,
		totalItems:   4,
	})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runSync(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("sync: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !bytes.Contains([]byte(output), []byte("Changes received")) {
		t.Errorf("expected 'Changes received' in output, got: %s", output)
	}
}

func TestRunSyncError(t *testing.T) {
	setupMockSyncerErr(t, fmt.Errorf("connection refused"))

	err := runSync(nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSummariseCommits(t *testing.T) {
	commits := []map[string]any{
		{
			"uuid1": map[string]any{
				dongxi.CommitKeyType:    float64(dongxi.ItemTypeCreate),
				dongxi.CommitKeyEntity:  string(dongxi.EntityTask),
				dongxi.CommitKeyPayload: map[string]any{},
			},
		},
		{
			"uuid2": map[string]any{
				dongxi.CommitKeyType:    float64(dongxi.ItemTypeModify),
				dongxi.CommitKeyEntity:  string(dongxi.EntityTask),
				dongxi.CommitKeyPayload: map[string]any{},
			},
			"uuid3": map[string]any{
				dongxi.CommitKeyType:    float64(dongxi.ItemTypeCreate),
				dongxi.CommitKeyEntity:  string(dongxi.EntityArea),
				dongxi.CommitKeyPayload: map[string]any{},
			},
		},
		{
			"uuid4": map[string]any{
				dongxi.CommitKeyType:    float64(dongxi.ItemTypeDelete),
				dongxi.CommitKeyEntity:  string(dongxi.EntityTag),
				dongxi.CommitKeyPayload: map[string]any{},
			},
		},
	}

	summary := summariseCommits(commits)

	if summary["task created"] != 1 {
		t.Errorf("task created = %d, want 1", summary["task created"])
	}
	if summary["task modified"] != 1 {
		t.Errorf("task modified = %d, want 1", summary["task modified"])
	}
	if summary["area created"] != 1 {
		t.Errorf("area created = %d, want 1", summary["area created"])
	}
	if summary["tag deleted"] != 1 {
		t.Errorf("tag deleted = %d, want 1", summary["tag deleted"])
	}
}

func TestSummariseCommitsEmpty(t *testing.T) {
	summary := summariseCommits(nil)
	if len(summary) != 0 {
		t.Errorf("expected empty summary, got %v", summary)
	}
}

func TestSummariseCommitsSkipsBadValues(t *testing.T) {
	commits := []map[string]any{
		{"uuid1": "not a map"},
	}
	summary := summariseCommits(commits)
	if len(summary) != 0 {
		t.Errorf("expected empty summary, got %v", summary)
	}
}

func TestSummariseCommitsUnknownOp(t *testing.T) {
	commits := []map[string]any{
		{
			"uuid1": map[string]any{
				dongxi.CommitKeyType:    float64(99),
				dongxi.CommitKeyEntity:  string(dongxi.EntityTask),
				dongxi.CommitKeyPayload: map[string]any{},
			},
		},
	}
	summary := summariseCommits(commits)
	if summary["task unknown"] != 1 {
		t.Errorf("expected 'task unknown' = 1, got %v", summary)
	}
}

func TestEntityLabel(t *testing.T) {
	tests := []struct {
		entity string
		want   string
	}{
		{string(dongxi.EntityTask), "task"},
		{string(dongxi.EntityChecklistItem), "checklist item"},
		{string(dongxi.EntityArea), "area"},
		{string(dongxi.EntityTag), "tag"},
		{"UnknownEntity", "UnknownEntity"},
	}
	for _, tt := range tests {
		got := entityLabel(tt.entity)
		if got != tt.want {
			t.Errorf("entityLabel(%q) = %q, want %q", tt.entity, got, tt.want)
		}
	}
}
