package cmd

import (
	"testing"
	"time"

	"github.com/kern/dongxi/dongxi"
)

// mockClient implements CloudClient for testing.
type mockClient struct {
	historyIndex   int
	lastCommit     map[string]dongxi.CommitItem
	getHistoryErr  error
	commitErr      error
	getAccountErr  error
	resetHistErr   error
	commitCount    int   // how many commits before erroring (0 = error on first)
	commitCallNum  int   // tracks number of Commit calls
}

func (m *mockClient) GetHistory(historyKey string) (*dongxi.HistoryInfo, error) {
	if m.getHistoryErr != nil {
		return nil, m.getHistoryErr
	}
	return &dongxi.HistoryInfo{LatestServerIndex: m.historyIndex}, nil
}

func (m *mockClient) Commit(historyKey string, ancestorIndex int, items map[string]dongxi.CommitItem) (*dongxi.CommitResponse, error) {
	m.commitCallNum++
	if m.commitErr != nil && m.commitCallNum > m.commitCount {
		return nil, m.commitErr
	}
	m.lastCommit = items
	m.historyIndex++
	return &dongxi.CommitResponse{ServerHeadIndex: m.historyIndex}, nil
}

func (m *mockClient) ResetHistory(email string) (*dongxi.ResetResponse, error) {
	if m.resetHistErr != nil {
		return nil, m.resetHistErr
	}
	return &dongxi.ResetResponse{NewHistoryKey: "new-key"}, nil
}

func (m *mockClient) GetAccount(email string) (*dongxi.Account, error) {
	if m.getAccountErr != nil {
		return nil, m.getAccountErr
	}
	return &dongxi.Account{HistoryKey: "test-key", Status: "SYCDAccountStatusActive", Email: "test@test.com"}, nil
}

func (m *mockClient) GetHistoryItems(historyKey string) ([]map[string]any, error) {
	return nil, nil
}

func (m *mockClient) Email() string {
	return "test@test.com"
}

// mockErrorStateLoader returns an error from LoadState.
type mockErrorStateLoader struct {
	err error
}

func (m *mockErrorStateLoader) LoadState() (*thingsState, CloudClient, string, error) {
	return nil, nil, "", m.err
}

// setupMockStateErr installs a mock state loader that returns an error.
func setupMockStateErr(t *testing.T, err error) {
	t.Helper()
	orig := stateLoader
	stateLoader = &mockErrorStateLoader{err: err}
	t.Cleanup(func() { stateLoader = orig })
}

// mockStateLoader implements StateLoader for testing.
type mockStateLoader struct {
	state      *thingsState
	client     *mockClient
	historyKey string
}

func (m *mockStateLoader) LoadState() (*thingsState, CloudClient, string, error) {
	return m.state, m.client, m.historyKey, nil
}

// setupMockState installs a mock state loader with the given history items.
// Returns the mock client for commit verification.
func setupMockState(t *testing.T, historyItems []map[string]any) *mockClient {
	t.Helper()

	items := replayHistory(historyItems)
	s := buildState(items)
	client := &mockClient{historyIndex: 42}

	orig := stateLoader
	stateLoader = &mockStateLoader{state: s, client: client, historyKey: "test-key"}
	t.Cleanup(func() { stateLoader = orig })

	return client
}

// makeTask creates a minimal task history entry.
func makeTask(uuid, title string, opts ...func(map[string]any)) map[string]any {
	p := map[string]any{
		dongxi.FieldTitle:       title,
		dongxi.FieldType:        float64(dongxi.TaskTypeTask),
		dongxi.FieldStatus:      float64(dongxi.TaskStatusOpen),
		dongxi.FieldDestination: float64(dongxi.TaskDestinationInbox),
		dongxi.FieldTrashed:     false,
	}
	for _, opt := range opts {
		opt(p)
	}
	return map[string]any{
		uuid: map[string]any{
			dongxi.CommitKeyType:    float64(dongxi.ItemTypeCreate),
			dongxi.CommitKeyEntity:  string(dongxi.EntityTask),
			dongxi.CommitKeyPayload: p,
		},
	}
}

func makeProject(uuid, title string, opts ...func(map[string]any)) map[string]any {
	p := map[string]any{
		dongxi.FieldTitle:       title,
		dongxi.FieldType:        float64(dongxi.TaskTypeProject),
		dongxi.FieldStatus:      float64(dongxi.TaskStatusOpen),
		dongxi.FieldDestination: float64(dongxi.TaskDestinationAnytime),
		dongxi.FieldTrashed:     false,
	}
	for _, opt := range opts {
		opt(p)
	}
	return map[string]any{
		uuid: map[string]any{
			dongxi.CommitKeyType:    float64(dongxi.ItemTypeCreate),
			dongxi.CommitKeyEntity:  string(dongxi.EntityTask),
			dongxi.CommitKeyPayload: p,
		},
	}
}

func makeArea(uuid, title string, opts ...func(map[string]any)) map[string]any {
	p := map[string]any{
		dongxi.FieldTitle:   title,
		dongxi.FieldTrashed: false,
	}
	for _, opt := range opts {
		opt(p)
	}
	return map[string]any{
		uuid: map[string]any{
			dongxi.CommitKeyType:    float64(dongxi.ItemTypeCreate),
			dongxi.CommitKeyEntity:  string(dongxi.EntityArea),
			dongxi.CommitKeyPayload: p,
		},
	}
}

func makeTag(uuid, title string) map[string]any {
	return map[string]any{
		uuid: map[string]any{
			dongxi.CommitKeyType:   float64(dongxi.ItemTypeCreate),
			dongxi.CommitKeyEntity: string(dongxi.EntityTag),
			dongxi.CommitKeyPayload: map[string]any{
				dongxi.FieldTitle: title,
			},
		},
	}
}

func makeHeading(uuid, title, projectUUID string) map[string]any {
	return map[string]any{
		uuid: map[string]any{
			dongxi.CommitKeyType:   float64(dongxi.ItemTypeCreate),
			dongxi.CommitKeyEntity: string(dongxi.EntityTask),
			dongxi.CommitKeyPayload: map[string]any{
				dongxi.FieldTitle:      title,
				dongxi.FieldType:       float64(dongxi.TaskTypeHeading),
				dongxi.FieldProjectIDs: []any{projectUUID},
			},
		},
	}
}

// withToday marks a task as being in the Today view by setting todayIndex and todayIndexRef.
func withToday(p map[string]any) {
	now := nowFunc()
	todayMidnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	p[dongxi.FieldTodayIndex] = float64(dongxi.DefaultTodayIndex)
	p[dongxi.FieldTodayIndexRef] = float64(todayMidnight.Unix())
}

func makeChecklistItem(uuid, title, taskUUID string) map[string]any {
	return map[string]any{
		uuid: map[string]any{
			dongxi.CommitKeyType:   float64(dongxi.ItemTypeCreate),
			dongxi.CommitKeyEntity: string(dongxi.EntityChecklistItem),
			dongxi.CommitKeyPayload: map[string]any{
				dongxi.FieldTitle:   title,
				dongxi.FieldStatus:  float64(dongxi.TaskStatusOpen),
				dongxi.FieldTaskIDs: []any{taskUUID},
			},
		},
	}
}
