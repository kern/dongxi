package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

const base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

func TestNewUUIDLength(t *testing.T) {
	for i := 0; i < 100; i++ {
		uuid := newUUID()
		if len(uuid) < 20 || len(uuid) > 22 {
			t.Errorf("newUUID() length = %d (%q), want 20-22", len(uuid), uuid)
		}
	}
}

func TestNewUUIDBase58Only(t *testing.T) {
	valid := map[byte]bool{}
	for i := 0; i < len(base58Alphabet); i++ {
		valid[base58Alphabet[i]] = true
	}

	// Specifically verify excluded chars are never present.
	excluded := []byte{'0', 'I', 'O', 'l'}

	for i := 0; i < 1000; i++ {
		uuid := newUUID()
		for _, c := range []byte(uuid) {
			if !valid[c] {
				t.Fatalf("newUUID() contains invalid char %q in %q", string(c), uuid)
			}
			for _, ex := range excluded {
				if c == ex {
					t.Fatalf("newUUID() contains excluded char %q in %q", string(c), uuid)
				}
			}
		}
	}
}

func TestNewUUIDUniqueness(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		uuid := newUUID()
		if seen[uuid] {
			t.Fatalf("duplicate UUID: %q", uuid)
		}
		seen[uuid] = true
	}
}

// ---------------------------------------------------------------------------
// runCreate integration tests
// ---------------------------------------------------------------------------

// commitCapturingMock wraps a mockClient and records every commit.
type commitCapturingMock struct {
	inner   *mockClient
	commits *[]map[string]dongxi.CommitItem
}

func (m *commitCapturingMock) GetHistory(historyKey string) (*dongxi.HistoryInfo, error) {
	return m.inner.GetHistory(historyKey)
}

func (m *commitCapturingMock) Commit(historyKey string, ancestorIndex int, items map[string]dongxi.CommitItem) (*dongxi.CommitResponse, error) {
	*m.commits = append(*m.commits, items)
	return m.inner.Commit(historyKey, ancestorIndex, items)
}

func (m *commitCapturingMock) ResetHistory(email string) (*dongxi.ResetResponse, error) {
	return m.inner.ResetHistory(email)
}

func (m *commitCapturingMock) GetAccount(email string) (*dongxi.Account, error) {
	return m.inner.GetAccount(email)
}

func (m *commitCapturingMock) GetHistoryItems(historyKey string) ([]map[string]any, error) {
	return m.inner.GetHistoryItems(historyKey)
}

// mockStateLoaderWithClient uses a specific CloudClient implementation.
type mockStateLoaderWithClient struct {
	client     CloudClient
	historyKey string
}

func (m *mockStateLoaderWithClient) LoadState() (*thingsState, CloudClient, string, error) {
	s := buildState(nil)
	return s, m.client, m.historyKey, nil
}

// makeCreateCmd builds a cobra.Command with the same flags as createCmd
// and resets all flag variables to defaults.
func makeCreateCmd() *cobra.Command {
	flagCreateTitle = ""
	flagCreateDestination = "inbox"
	flagCreateNote = ""
	flagCreateChecklist = ""
	flagCreateScheduled = ""
	flagCreateDeadline = ""
	flagCreateArea = ""
	flagCreateProject = ""
	flagCreateType = "task"
	flagCreateHeading = ""
	flagCreateTags = ""
	flagCreateEvening = false

	cmd := &cobra.Command{}
	cmd.Flags().StringVarP(&flagCreateTitle, "title", "t", "", "")
	cmd.Flags().StringVarP(&flagCreateDestination, "destination", "d", "inbox", "")
	cmd.Flags().StringVarP(&flagCreateNote, "note", "n", "", "")
	cmd.Flags().StringVar(&flagCreateChecklist, "checklist", "", "")
	cmd.Flags().StringVar(&flagCreateScheduled, "scheduled", "", "")
	cmd.Flags().StringVar(&flagCreateDeadline, "deadline", "", "")
	cmd.Flags().StringVar(&flagCreateArea, "area", "", "")
	cmd.Flags().StringVar(&flagCreateProject, "project", "", "")
	cmd.Flags().StringVar(&flagCreateType, "type", "task", "")
	cmd.Flags().StringVar(&flagCreateHeading, "heading", "", "")
	cmd.Flags().StringVar(&flagCreateTags, "tags", "", "")
	cmd.Flags().BoolVar(&flagCreateEvening, "evening", false, "")
	return cmd
}

// captureStdout runs fn and returns whatever was written to os.Stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

// setupCapturingMock installs a mock that records every commit.
// Returns the capturing mock so the caller can inspect allCommits.
func setupCapturingMock(t *testing.T) *[]map[string]dongxi.CommitItem {
	t.Helper()
	mock := setupMockState(t, nil)
	var allCommits []map[string]dongxi.CommitItem
	captureMock := &commitCapturingMock{inner: mock, commits: &allCommits}
	orig := stateLoader
	stateLoader = &mockStateLoaderWithClient{client: captureMock, historyKey: "test-key"}
	t.Cleanup(func() { stateLoader = orig })
	return &allCommits
}

// findTaskUUID returns the UUID of the first task entity in a commit map.
func findTaskUUID(items map[string]dongxi.CommitItem) string {
	for uuid, item := range items {
		if item.E == dongxi.EntityTask {
			return uuid
		}
	}
	return ""
}

// --- 1. Basic task creation (default flags) ---

func TestRunCreateBasicTask(t *testing.T) {
	mock := setupMockState(t, nil)
	cmd := makeCreateCmd()
	flagCreateTitle = "Buy groceries"
	_ = cmd.Flags().Set("title", "Buy groceries")

	out := captureStdout(t, func() {
		if err := runCreate(cmd, nil); err != nil {
			t.Fatal(err)
		}
	})

	if !strings.Contains(out, "Created task") {
		t.Errorf("expected 'Created task' in output, got %q", out)
	}
	if !strings.Contains(out, "Buy groceries") {
		t.Errorf("expected title in output, got %q", out)
	}

	taskUUID := findTaskUUID(mock.lastCommit)
	if taskUUID == "" {
		t.Fatal("expected task in last commit")
	}
	if mock.lastCommit[taskUUID].P[dongxi.FieldTitle] != "Buy groceries" {
		t.Errorf("expected title 'Buy groceries', got %v", mock.lastCommit[taskUUID].P[dongxi.FieldTitle])
	}
}

// --- 2. Project creation (--type project) ---

func TestRunCreateProjectType(t *testing.T) {
	mock := setupMockState(t, nil)
	cmd := makeCreateCmd()
	flagCreateTitle = "Q3 Planning"
	flagCreateType = "project"
	_ = cmd.Flags().Set("title", "Q3 Planning")
	_ = cmd.Flags().Set("type", "project")

	out := captureStdout(t, func() {
		if err := runCreate(cmd, nil); err != nil {
			t.Fatal(err)
		}
	})

	if !strings.Contains(out, "Created project") {
		t.Errorf("expected 'Created project' in output, got %q", out)
	}
	taskUUID := findTaskUUID(mock.lastCommit)
	if mock.lastCommit[taskUUID].P[dongxi.FieldTitle] != "Q3 Planning" {
		t.Errorf("expected title 'Q3 Planning', got %v", mock.lastCommit[taskUUID].P[dongxi.FieldTitle])
	}
}

// --- 3. Heading creation (--type heading) ---

func TestRunCreateHeadingType(t *testing.T) {
	setupMockState(t, nil)
	cmd := makeCreateCmd()
	flagCreateTitle = "Research"
	flagCreateType = "heading"
	_ = cmd.Flags().Set("title", "Research")
	_ = cmd.Flags().Set("type", "heading")

	out := captureStdout(t, func() {
		if err := runCreate(cmd, nil); err != nil {
			t.Fatal(err)
		}
	})

	if !strings.Contains(out, "Created heading") {
		t.Errorf("expected 'Created heading' in output, got %q", out)
	}
}

// --- 4. Bad type error ---

func TestRunCreateBadType(t *testing.T) {
	setupMockState(t, nil)
	cmd := makeCreateCmd()
	flagCreateTitle = "Something"
	flagCreateType = "widget"
	_ = cmd.Flags().Set("title", "Something")
	_ = cmd.Flags().Set("type", "widget")

	err := runCreate(cmd, nil)
	if err == nil {
		t.Fatal("expected error for bad type")
	}
	if !strings.Contains(err.Error(), "unknown type") {
		t.Errorf("expected 'unknown type' error, got %v", err)
	}
}

// --- 5. Destination: inbox (default) ---

func TestRunCreateDestinationInbox(t *testing.T) {
	mock := setupMockState(t, nil)
	cmd := makeCreateCmd()
	flagCreateTitle = "Inbox task"
	_ = cmd.Flags().Set("title", "Inbox task")

	captureStdout(t, func() {
		if err := runCreate(cmd, nil); err != nil {
			t.Fatal(err)
		}
	})

	// Two commits: create + modify => index goes from 42 to 44.
	if mock.historyIndex != 44 {
		t.Errorf("expected history index 44, got %d", mock.historyIndex)
	}
}

// --- 5b. Destination: today ---

func TestRunCreateDestinationToday(t *testing.T) {
	mock := setupMockState(t, nil)
	cmd := makeCreateCmd()
	flagCreateTitle = "Today task"
	flagCreateDestination = "today"
	_ = cmd.Flags().Set("title", "Today task")
	_ = cmd.Flags().Set("destination", "today")

	captureStdout(t, func() {
		if err := runCreate(cmd, nil); err != nil {
			t.Fatal(err)
		}
	})

	if mock.historyIndex != 44 {
		t.Errorf("expected history index 44, got %d", mock.historyIndex)
	}
}

// --- 5c. Destination: someday ---

func TestRunCreateDestinationSomeday(t *testing.T) {
	mock := setupMockState(t, nil)
	cmd := makeCreateCmd()
	flagCreateTitle = "Someday task"
	flagCreateDestination = "someday"
	_ = cmd.Flags().Set("title", "Someday task")
	_ = cmd.Flags().Set("destination", "someday")

	captureStdout(t, func() {
		if err := runCreate(cmd, nil); err != nil {
			t.Fatal(err)
		}
	})

	if mock.historyIndex != 44 {
		t.Errorf("expected history index 44, got %d", mock.historyIndex)
	}
}

// --- 6. Bad destination error ---

func TestRunCreateBadDestination(t *testing.T) {
	setupMockState(t, nil)
	cmd := makeCreateCmd()
	flagCreateTitle = "Something"
	flagCreateDestination = "nowhere"
	_ = cmd.Flags().Set("title", "Something")
	_ = cmd.Flags().Set("destination", "nowhere")

	err := runCreate(cmd, nil)
	if err == nil {
		t.Fatal("expected error for bad destination")
	}
	if !strings.Contains(err.Error(), "unknown destination") {
		t.Errorf("expected 'unknown destination' error, got %v", err)
	}
}

// --- 7. With --note ---

func TestRunCreateWithNote(t *testing.T) {
	mock := setupMockState(t, nil)
	cmd := makeCreateCmd()
	flagCreateTitle = "Task with note"
	flagCreateNote = "Some important notes"
	_ = cmd.Flags().Set("title", "Task with note")
	_ = cmd.Flags().Set("note", "Some important notes")

	captureStdout(t, func() {
		if err := runCreate(cmd, nil); err != nil {
			t.Fatal(err)
		}
	})

	taskUUID := findTaskUUID(mock.lastCommit)
	if mock.lastCommit[taskUUID].P[dongxi.FieldNote] == nil {
		t.Error("expected note to be set in modify commit")
	}
}

// --- 8. With --checklist (comma-separated items) ---

func TestRunCreateWithChecklist(t *testing.T) {
	mock := setupMockState(t, nil)
	cmd := makeCreateCmd()
	flagCreateTitle = "Groceries"
	flagCreateChecklist = "Milk,Eggs,Bread"
	_ = cmd.Flags().Set("title", "Groceries")
	_ = cmd.Flags().Set("checklist", "Milk,Eggs,Bread")

	out := captureStdout(t, func() {
		if err := runCreate(cmd, nil); err != nil {
			t.Fatal(err)
		}
	})

	if !strings.Contains(out, "3 checklist item(s)") {
		t.Errorf("expected checklist count in output, got %q", out)
	}

	// Modify commit: 1 task modify + 3 checklist creates = 4 entries.
	if len(mock.lastCommit) != 4 {
		t.Errorf("expected 4 items in modify commit, got %d", len(mock.lastCommit))
	}

	checklistCount := 0
	for _, item := range mock.lastCommit {
		if item.E == dongxi.EntityChecklistItem {
			checklistCount++
			if item.T != dongxi.ItemTypeCreate {
				t.Error("expected checklist item to be a create")
			}
		}
	}
	if checklistCount != 3 {
		t.Errorf("expected 3 checklist items, got %d", checklistCount)
	}
}

// --- 9. With --scheduled date ---

func TestRunCreateWithScheduled(t *testing.T) {
	mock := setupMockState(t, nil)
	cmd := makeCreateCmd()
	flagCreateTitle = "Scheduled task"
	flagCreateScheduled = "2025-04-01"
	_ = cmd.Flags().Set("title", "Scheduled task")
	_ = cmd.Flags().Set("scheduled", "2025-04-01")

	captureStdout(t, func() {
		if err := runCreate(cmd, nil); err != nil {
			t.Fatal(err)
		}
	})

	if mock.historyIndex != 44 {
		t.Errorf("expected history index 44, got %d", mock.historyIndex)
	}
}

// --- 10. Bad --scheduled date error ---

func TestRunCreateBadScheduled(t *testing.T) {
	setupMockState(t, nil)
	cmd := makeCreateCmd()
	flagCreateTitle = "Bad scheduled"
	flagCreateScheduled = "not-a-date"
	_ = cmd.Flags().Set("title", "Bad scheduled")
	_ = cmd.Flags().Set("scheduled", "not-a-date")

	err := runCreate(cmd, nil)
	if err == nil {
		t.Fatal("expected error for bad scheduled date")
	}
	if !strings.Contains(err.Error(), "parse --scheduled date") {
		t.Errorf("expected parse error, got %v", err)
	}
}

// --- 11. With --deadline date ---

func TestRunCreateWithDeadline(t *testing.T) {
	mock := setupMockState(t, nil)
	cmd := makeCreateCmd()
	flagCreateTitle = "Deadline task"
	flagCreateDeadline = "2025-04-15"
	_ = cmd.Flags().Set("title", "Deadline task")
	_ = cmd.Flags().Set("deadline", "2025-04-15")

	captureStdout(t, func() {
		if err := runCreate(cmd, nil); err != nil {
			t.Fatal(err)
		}
	})

	taskUUID := findTaskUUID(mock.lastCommit)
	commit := mock.lastCommit[taskUUID]
	if commit.P[dongxi.FieldDeadline] == nil {
		t.Error("expected deadline to be set")
	}
}

// --- 12. Bad --deadline date error ---

func TestRunCreateBadDeadline(t *testing.T) {
	setupMockState(t, nil)
	cmd := makeCreateCmd()
	flagCreateTitle = "Bad deadline"
	flagCreateDeadline = "bad-date"
	_ = cmd.Flags().Set("title", "Bad deadline")
	_ = cmd.Flags().Set("deadline", "bad-date")

	err := runCreate(cmd, nil)
	if err == nil {
		t.Fatal("expected error for bad deadline date")
	}
	if !strings.Contains(err.Error(), "parse --deadline date") {
		t.Errorf("expected parse error, got %v", err)
	}
}

// --- 13. With --area ---

func TestRunCreateWithArea(t *testing.T) {
	mock := setupMockState(t, nil)
	cmd := makeCreateCmd()
	flagCreateTitle = "Area task"
	flagCreateArea = "area-uuid-123"
	_ = cmd.Flags().Set("title", "Area task")
	_ = cmd.Flags().Set("area", "area-uuid-123")

	captureStdout(t, func() {
		if err := runCreate(cmd, nil); err != nil {
			t.Fatal(err)
		}
	})

	taskUUID := findTaskUUID(mock.lastCommit)
	areaIDs, ok := mock.lastCommit[taskUUID].P[dongxi.FieldAreaIDs].([]string)
	if !ok || len(areaIDs) != 1 || areaIDs[0] != "area-uuid-123" {
		t.Errorf("expected area IDs [area-uuid-123], got %v", mock.lastCommit[taskUUID].P[dongxi.FieldAreaIDs])
	}
}

// --- 14. With --project (assign to project) ---

func TestRunCreateWithProject(t *testing.T) {
	mock := setupMockState(t, nil)
	cmd := makeCreateCmd()
	flagCreateTitle = "Project task"
	flagCreateProject = "proj-uuid-456"
	_ = cmd.Flags().Set("title", "Project task")
	_ = cmd.Flags().Set("project", "proj-uuid-456")

	captureStdout(t, func() {
		if err := runCreate(cmd, nil); err != nil {
			t.Fatal(err)
		}
	})

	taskUUID := findTaskUUID(mock.lastCommit)
	projectIDs, ok := mock.lastCommit[taskUUID].P[dongxi.FieldProjectIDs].([]string)
	if !ok || len(projectIDs) != 1 || projectIDs[0] != "proj-uuid-456" {
		t.Errorf("expected project IDs [proj-uuid-456], got %v", mock.lastCommit[taskUUID].P[dongxi.FieldProjectIDs])
	}
}

// --- 15. With --heading ---

func TestRunCreateWithHeading(t *testing.T) {
	mock := setupMockState(t, nil)
	cmd := makeCreateCmd()
	flagCreateTitle = "Heading task"
	flagCreateHeading = "heading-uuid-789"
	_ = cmd.Flags().Set("title", "Heading task")
	_ = cmd.Flags().Set("heading", "heading-uuid-789")

	captureStdout(t, func() {
		if err := runCreate(cmd, nil); err != nil {
			t.Fatal(err)
		}
	})

	taskUUID := findTaskUUID(mock.lastCommit)
	headingIDs, ok := mock.lastCommit[taskUUID].P[dongxi.FieldHeadingIDs].([]string)
	if !ok || len(headingIDs) != 1 || headingIDs[0] != "heading-uuid-789" {
		t.Errorf("expected heading IDs [heading-uuid-789], got %v", mock.lastCommit[taskUUID].P[dongxi.FieldHeadingIDs])
	}
}

// --- 16. With --tags (comma-separated) ---

func TestRunCreateWithTags(t *testing.T) {
	mock := setupMockState(t, nil)
	cmd := makeCreateCmd()
	flagCreateTitle = "Tagged task"
	flagCreateTags = "tag-1,tag-2,tag-3"
	_ = cmd.Flags().Set("title", "Tagged task")
	_ = cmd.Flags().Set("tags", "tag-1,tag-2,tag-3")

	captureStdout(t, func() {
		if err := runCreate(cmd, nil); err != nil {
			t.Fatal(err)
		}
	})

	taskUUID := findTaskUUID(mock.lastCommit)
	tagIDs, ok := mock.lastCommit[taskUUID].P[dongxi.FieldTagIDs].([]string)
	if !ok || len(tagIDs) != 3 {
		t.Fatalf("expected 3 tag IDs, got %v", mock.lastCommit[taskUUID].P[dongxi.FieldTagIDs])
	}
	if tagIDs[0] != "tag-1" || tagIDs[1] != "tag-2" || tagIDs[2] != "tag-3" {
		t.Errorf("expected tag IDs [tag-1 tag-2 tag-3], got %v", tagIDs)
	}
}

// --- 17. With --evening flag ---

func TestRunCreateWithEvening(t *testing.T) {
	allCommits := setupCapturingMock(t)
	cmd := makeCreateCmd()
	flagCreateTitle = "Evening task"
	flagCreateEvening = true
	_ = cmd.Flags().Set("title", "Evening task")
	_ = cmd.Flags().Set("evening", "true")

	captureStdout(t, func() {
		if err := runCreate(cmd, nil); err != nil {
			t.Fatal(err)
		}
	})

	if len(*allCommits) < 1 {
		t.Fatal("expected at least 1 commit")
	}
	createCommit := (*allCommits)[0]
	for _, item := range createCommit {
		if item.E == dongxi.EntityTask && item.T == dongxi.ItemTypeCreate {
			sb, ok := item.P[dongxi.FieldStartBucket].(int)
			if !ok || sb != 1 {
				t.Errorf("expected startBucket=1 for evening, got %v", item.P[dongxi.FieldStartBucket])
			}
		}
	}
}

// --- 18. Empty title error ---

func TestRunCreateEmptyTitle(t *testing.T) {
	setupMockState(t, nil)
	cmd := makeCreateCmd()
	flagCreateTitle = "   "
	_ = cmd.Flags().Set("title", "   ")

	err := runCreate(cmd, nil)
	if err == nil {
		t.Fatal("expected error for empty title")
	}
	if !strings.Contains(err.Error(), "--title must not be empty") {
		t.Errorf("expected '--title must not be empty', got %v", err)
	}
}

func TestRunCreateBlankTitle(t *testing.T) {
	setupMockState(t, nil)
	cmd := makeCreateCmd()
	// flagCreateTitle stays ""

	err := runCreate(cmd, nil)
	if err == nil {
		t.Fatal("expected error for blank title")
	}
}

// --- 19. JSON output mode ---

func TestRunCreateJSON(t *testing.T) {
	setupMockState(t, nil)
	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	cmd := makeCreateCmd()
	flagCreateTitle = "JSON task"
	_ = cmd.Flags().Set("title", "JSON task")

	out := captureStdout(t, func() {
		if err := runCreate(cmd, nil); err != nil {
			t.Fatal(err)
		}
	})

	var result CreateOutput
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("expected valid JSON output, parse error: %v\noutput: %s", err, out)
	}
	if result.Type != "task" {
		t.Errorf("expected type 'task', got %q", result.Type)
	}
	if result.Title != "JSON task" {
		t.Errorf("expected title 'JSON task', got %q", result.Title)
	}
	if result.UUID == "" {
		t.Error("expected non-empty UUID")
	}
	if result.ServerIndex == 0 {
		t.Error("expected non-zero server index")
	}
}

func TestRunCreateJSONProject(t *testing.T) {
	setupMockState(t, nil)
	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	cmd := makeCreateCmd()
	flagCreateTitle = "JSON project"
	flagCreateType = "project"
	_ = cmd.Flags().Set("title", "JSON project")
	_ = cmd.Flags().Set("type", "project")

	out := captureStdout(t, func() {
		if err := runCreate(cmd, nil); err != nil {
			t.Fatal(err)
		}
	})

	var result CreateOutput
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %v", err)
	}
	if result.Type != "project" {
		t.Errorf("expected type 'project', got %q", result.Type)
	}
}

func TestRunCreateJSONHeading(t *testing.T) {
	setupMockState(t, nil)
	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	cmd := makeCreateCmd()
	flagCreateTitle = "JSON heading"
	flagCreateType = "heading"
	_ = cmd.Flags().Set("title", "JSON heading")
	_ = cmd.Flags().Set("type", "heading")

	out := captureStdout(t, func() {
		if err := runCreate(cmd, nil); err != nil {
			t.Fatal(err)
		}
	})

	var result CreateOutput
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %v", err)
	}
	if result.Type != "heading" {
		t.Errorf("expected type 'heading', got %q", result.Type)
	}
}

func TestRunCreateJSONWithChecklist(t *testing.T) {
	setupMockState(t, nil)
	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	cmd := makeCreateCmd()
	flagCreateTitle = "Checklist JSON"
	flagCreateChecklist = "A,B"
	_ = cmd.Flags().Set("title", "Checklist JSON")
	_ = cmd.Flags().Set("checklist", "A,B")

	out := captureStdout(t, func() {
		if err := runCreate(cmd, nil); err != nil {
			t.Fatal(err)
		}
	})

	var result CreateOutput
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %v", err)
	}
	if result.ChecklistCount != 2 {
		t.Errorf("expected checklist_count 2, got %d", result.ChecklistCount)
	}
}

// --- 20. "today" destination sets sr/scheduledDate ---

func TestRunCreateTodaySetsScheduledDate(t *testing.T) {
	allCommits := setupCapturingMock(t)
	cmd := makeCreateCmd()
	flagCreateTitle = "Today task"
	flagCreateDestination = "today"
	_ = cmd.Flags().Set("title", "Today task")
	_ = cmd.Flags().Set("destination", "today")

	captureStdout(t, func() {
		if err := runCreate(cmd, nil); err != nil {
			t.Fatal(err)
		}
	})

	if len(*allCommits) < 2 {
		t.Fatalf("expected 2 commits, got %d", len(*allCommits))
	}

	// First commit is the create; verify destination and scheduledDate.
	createCommit := (*allCommits)[0]
	for _, item := range createCommit {
		if item.E == dongxi.EntityTask && item.T == dongxi.ItemTypeCreate {
			dest, ok := item.P[dongxi.FieldDestination].(int)
			if !ok || dongxi.TaskDestination(dest) != dongxi.TaskDestinationAnytime {
				t.Errorf("expected destination Anytime for 'today', got %v", item.P[dongxi.FieldDestination])
			}
			if item.P[dongxi.FieldScheduledDate] == nil {
				t.Error("expected scheduledDate to be set for 'today' destination")
			}
			if item.P[dongxi.FieldTodayIndexRef] == nil {
				t.Error("expected todayIndexRef to be set for 'today' destination")
			}
			// modificationDate should be non-nil for today/anytime.
			if item.P[dongxi.FieldModificationDate] == nil {
				t.Error("expected modificationDate to be set for 'today' destination")
			}
		}
	}
}

// --- 21. "someday" destination ---

func TestRunCreateSomedayPayload(t *testing.T) {
	allCommits := setupCapturingMock(t)
	cmd := makeCreateCmd()
	flagCreateTitle = "Someday task"
	flagCreateDestination = "someday"
	_ = cmd.Flags().Set("title", "Someday task")
	_ = cmd.Flags().Set("destination", "someday")

	captureStdout(t, func() {
		if err := runCreate(cmd, nil); err != nil {
			t.Fatal(err)
		}
	})

	if len(*allCommits) < 1 {
		t.Fatal("expected at least 1 commit")
	}

	createCommit := (*allCommits)[0]
	for _, item := range createCommit {
		if item.E == dongxi.EntityTask && item.T == dongxi.ItemTypeCreate {
			dest, ok := item.P[dongxi.FieldDestination].(int)
			if !ok || dongxi.TaskDestination(dest) != dongxi.TaskDestinationSomeday {
				t.Errorf("expected destination Someday, got %v", item.P[dongxi.FieldDestination])
			}
			// modificationDate should be set for someday.
			if item.P[dongxi.FieldModificationDate] == nil {
				t.Error("expected modificationDate to be set for someday destination")
			}
			// scheduledDate should be nil for someday (no scheduled date set).
			if item.P[dongxi.FieldScheduledDate] != nil {
				t.Error("expected scheduledDate to be nil for someday destination")
			}
		}
	}
}

// --- Edge cases ---

func TestRunCreateChecklistTrimsWhitespace(t *testing.T) {
	mock := setupMockState(t, nil)
	cmd := makeCreateCmd()
	flagCreateTitle = "Trim test"
	flagCreateChecklist = " Milk , , Eggs , "
	_ = cmd.Flags().Set("title", "Trim test")
	_ = cmd.Flags().Set("checklist", " Milk , , Eggs , ")

	captureStdout(t, func() {
		if err := runCreate(cmd, nil); err != nil {
			t.Fatal(err)
		}
	})

	checklistCount := 0
	for _, item := range mock.lastCommit {
		if item.E == dongxi.EntityChecklistItem {
			checklistCount++
		}
	}
	// Empty items between commas should be skipped.
	if checklistCount != 2 {
		t.Errorf("expected 2 checklist items after trimming empties, got %d", checklistCount)
	}
}

func TestRunCreateTagsTrimsWhitespace(t *testing.T) {
	mock := setupMockState(t, nil)
	cmd := makeCreateCmd()
	flagCreateTitle = "Tag trim"
	flagCreateTags = " tag-1 , , tag-2 "
	_ = cmd.Flags().Set("title", "Tag trim")
	_ = cmd.Flags().Set("tags", " tag-1 , , tag-2 ")

	captureStdout(t, func() {
		if err := runCreate(cmd, nil); err != nil {
			t.Fatal(err)
		}
	})

	taskUUID := findTaskUUID(mock.lastCommit)
	tagIDs, ok := mock.lastCommit[taskUUID].P[dongxi.FieldTagIDs].([]string)
	if !ok || len(tagIDs) != 2 {
		t.Errorf("expected 2 tag IDs after trimming, got %v", mock.lastCommit[taskUUID].P[dongxi.FieldTagIDs])
	}
}

func TestRunCreateScheduledSetsCreatePayloadSR(t *testing.T) {
	allCommits := setupCapturingMock(t)
	cmd := makeCreateCmd()
	flagCreateTitle = "Scheduled SR"
	flagCreateScheduled = "2025-06-15"
	_ = cmd.Flags().Set("title", "Scheduled SR")
	_ = cmd.Flags().Set("scheduled", "2025-06-15")

	captureStdout(t, func() {
		if err := runCreate(cmd, nil); err != nil {
			t.Fatal(err)
		}
	})

	if len(*allCommits) < 1 {
		t.Fatal("expected at least 1 commit")
	}

	createCommit := (*allCommits)[0]
	for _, item := range createCommit {
		if item.E == dongxi.EntityTask && item.T == dongxi.ItemTypeCreate {
			if item.P[dongxi.FieldScheduledDate] == nil {
				t.Error("expected scheduledDate to be set when --scheduled is provided")
			}
			if item.P[dongxi.FieldTodayIndexRef] == nil {
				t.Error("expected todayIndexRef to be set when --scheduled is provided")
			}
		}
	}
}

// --- Error path tests ---

func TestRunCreateGetHistoryErr(t *testing.T) {
	mock := setupMockState(t, nil)
	mock.getHistoryErr = fmt.Errorf("history error")
	cmd := makeCreateCmd()
	flagCreateTitle = "Test"
	_ = cmd.Flags().Set("title", "Test")

	err := runCreate(cmd, nil)
	if err == nil || !strings.Contains(err.Error(), "history error") {
		t.Fatalf("expected history error, got %v", err)
	}
}

func TestRunCreateCommitCreateErr(t *testing.T) {
	mock := setupMockState(t, nil)
	mock.commitErr = fmt.Errorf("commit create error")
	mock.commitCount = 0 // error on first commit
	cmd := makeCreateCmd()
	flagCreateTitle = "Test"
	_ = cmd.Flags().Set("title", "Test")

	err := runCreate(cmd, nil)
	if err == nil || !strings.Contains(err.Error(), "commit create") {
		t.Fatalf("expected commit create error, got %v", err)
	}
}

func TestRunCreateCommitModifyErr(t *testing.T) {
	mock := setupMockState(t, nil)
	mock.commitErr = fmt.Errorf("commit modify error")
	mock.commitCount = 1 // first commit succeeds, second fails
	cmd := makeCreateCmd()
	flagCreateTitle = "Test"
	_ = cmd.Flags().Set("title", "Test")

	err := runCreate(cmd, nil)
	if err == nil || !strings.Contains(err.Error(), "commit modify") {
		t.Fatalf("expected commit modify error, got %v", err)
	}
}

func TestRunCreateLoadStateErr(t *testing.T) {
	setupMockStateErr(t, fmt.Errorf("mock load error"))
	cmd := makeCreateCmd()
	flagCreateTitle = "Test"
	_ = cmd.Flags().Set("title", "Test")

	err := runCreate(cmd, nil)
	if err == nil || err.Error() != "mock load error" {
		t.Fatalf("expected load error, got %v", err)
	}
}

func TestRunCreateNoNoteByDefault(t *testing.T) {
	mock := setupMockState(t, nil)
	cmd := makeCreateCmd()
	flagCreateTitle = "No note"
	_ = cmd.Flags().Set("title", "No note")

	captureStdout(t, func() {
		if err := runCreate(cmd, nil); err != nil {
			t.Fatal(err)
		}
	})

	taskUUID := findTaskUUID(mock.lastCommit)
	// The modify commit should not set note when flagCreateNote is empty.
	if _, exists := mock.lastCommit[taskUUID].P[dongxi.FieldNote]; exists {
		t.Error("expected note to not be set in modify commit when no note flag")
	}
}
