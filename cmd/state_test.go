package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/kern/dongxi/dongxi"
)

func TestRealStateLoaderImplementsInterface(t *testing.T) {
	// Compile-time check that realStateLoader satisfies StateLoader.
	var _ StateLoader = realStateLoader{}
}

func TestFirstString(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  string
	}{
		{"normal", []any{"abc", "def"}, "abc"},
		{"first not string", []any{42}, ""},
		{"empty array", []any{}, ""},
		{"nil", nil, ""},
		{"not array", "hello", ""},
		{"mixed", []any{42, "abc"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := firstString(tt.input)
			if got != tt.want {
				t.Errorf("firstString(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func buildTestState() *thingsState {
	items := []replayedItem{
		{uuid: "area-1", entity: string(dongxi.EntityArea), fields: map[string]any{dongxi.FieldTitle: "Work"}},
		{uuid: "area-2", entity: string(dongxi.EntityArea), fields: map[string]any{dongxi.FieldTitle: "Personal"}},
		{uuid: "proj-1", entity: string(dongxi.EntityTask), fields: map[string]any{dongxi.FieldTitle: "My Project", dongxi.FieldType: float64(dongxi.TaskTypeProject)}},
		{uuid: "task-1", entity: string(dongxi.EntityTask), fields: map[string]any{dongxi.FieldTitle: "Buy milk", dongxi.FieldType: float64(dongxi.TaskTypeTask)}},
		{uuid: "task-2", entity: string(dongxi.EntityTask), fields: map[string]any{dongxi.FieldTitle: "Call dentist", dongxi.FieldType: float64(dongxi.TaskTypeTask)}},
		{uuid: "tag-1", entity: string(dongxi.EntityTag), fields: map[string]any{dongxi.FieldTitle: "Urgent"}},
		{uuid: "ci-1", entity: string(dongxi.EntityChecklistItem), fields: map[string]any{dongxi.FieldTitle: "Step 1", dongxi.FieldTaskIDs: []any{"task-1"}, dongxi.FieldStatus: float64(dongxi.TaskStatusOpen)}},
		{uuid: "ci-2", entity: string(dongxi.EntityChecklistItem), fields: map[string]any{dongxi.FieldTitle: "Step 2", dongxi.FieldTaskIDs: []any{"task-1"}, dongxi.FieldStatus: float64(dongxi.TaskStatusCompleted)}},
		{uuid: "ci-3", entity: string(dongxi.EntityChecklistItem), fields: map[string]any{dongxi.FieldTitle: "Other", dongxi.FieldTaskIDs: []any{"task-2"}, dongxi.FieldStatus: float64(dongxi.TaskStatusOpen)}},
	}

	s := &thingsState{
		items:    items,
		byUUID:   make(map[string]*replayedItem),
		areas:    make(map[string]*replayedItem),
		projects: make(map[string]*replayedItem),
	}
	for i := range s.items {
		item := &s.items[i]
		s.byUUID[item.uuid] = item
		if item.entity == string(dongxi.EntityArea) {
			s.areas[item.uuid] = item
		}
		if item.entity == string(dongxi.EntityTask) && toInt(item.fields[dongxi.FieldType]) == int(dongxi.TaskTypeProject) {
			s.projects[item.uuid] = item
		}
	}
	return s
}

func TestResolveUUIDExact(t *testing.T) {
	s := buildTestState()
	item, err := s.resolveUUID("task-1")
	if err != nil {
		t.Fatal(err)
	}
	if item.uuid != "task-1" {
		t.Errorf("uuid = %q, want %q", item.uuid, "task-1")
	}
}

func TestResolveUUIDPrefix(t *testing.T) {
	s := buildTestState()
	item, err := s.resolveUUID("proj")
	if err != nil {
		t.Fatal(err)
	}
	if item.uuid != "proj-1" {
		t.Errorf("uuid = %q, want %q", item.uuid, "proj-1")
	}
}

func TestResolveUUIDAmbiguous(t *testing.T) {
	s := buildTestState()
	_, err := s.resolveUUID("task")
	if err == nil {
		t.Fatal("expected error for ambiguous prefix")
	}
}

func TestResolveUUIDNotFound(t *testing.T) {
	s := buildTestState()
	_, err := s.resolveUUID("nonexistent")
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestAreaTitle(t *testing.T) {
	s := buildTestState()
	if got := s.areaTitle("area-1"); got != "Work" {
		t.Errorf("areaTitle(area-1) = %q, want %q", got, "Work")
	}
	if got := s.areaTitle("nonexistent"); got != "" {
		t.Errorf("areaTitle(nonexistent) = %q, want %q", got, "")
	}
}

func TestProjectTitle(t *testing.T) {
	s := buildTestState()
	if got := s.projectTitle("proj-1"); got != "My Project" {
		t.Errorf("projectTitle(proj-1) = %q, want %q", got, "My Project")
	}
	if got := s.projectTitle("nonexistent"); got != "" {
		t.Errorf("projectTitle(nonexistent) = %q, want %q", got, "")
	}
}

func TestChecklistForTask(t *testing.T) {
	s := buildTestState()

	items := s.checklistForTask("task-1")
	if len(items) != 2 {
		t.Fatalf("checklistForTask(task-1) returned %d items, want 2", len(items))
	}

	items = s.checklistForTask("task-2")
	if len(items) != 1 {
		t.Fatalf("checklistForTask(task-2) returned %d items, want 1", len(items))
	}

	items = s.checklistForTask("nonexistent")
	if len(items) != 0 {
		t.Fatalf("checklistForTask(nonexistent) returned %d items, want 0", len(items))
	}
}

func buildProjectState() *thingsState {
	items := []replayedItem{
		{uuid: "area-1", entity: string(dongxi.EntityArea), fields: map[string]any{dongxi.FieldTitle: "Work", dongxi.FieldTrashed: false}},
		{uuid: "proj-1", entity: string(dongxi.EntityTask), fields: map[string]any{
			dongxi.FieldTitle:   "My Project",
			dongxi.FieldType:    float64(dongxi.TaskTypeProject),
			dongxi.FieldTrashed: false,
			dongxi.FieldAreaIDs: []any{"area-1"},
		}},
		{uuid: "proj-trashed", entity: string(dongxi.EntityTask), fields: map[string]any{
			dongxi.FieldTitle:   "Trashed Project",
			dongxi.FieldType:    float64(dongxi.TaskTypeProject),
			dongxi.FieldTrashed: true,
		}},
		{uuid: "heading-1", entity: string(dongxi.EntityTask), fields: map[string]any{
			dongxi.FieldTitle:      "Design Phase",
			dongxi.FieldType:       float64(dongxi.TaskTypeHeading),
			dongxi.FieldProjectIDs: []any{"proj-1"},
		}},
		{uuid: "heading-2", entity: string(dongxi.EntityTask), fields: map[string]any{
			dongxi.FieldTitle:      "Dev Phase",
			dongxi.FieldType:       float64(dongxi.TaskTypeHeading),
			dongxi.FieldProjectIDs: []any{"proj-1"},
		}},
		{uuid: "heading-trashed-proj", entity: string(dongxi.EntityTask), fields: map[string]any{
			dongxi.FieldTitle:      "Trashed Heading",
			dongxi.FieldType:       float64(dongxi.TaskTypeHeading),
			dongxi.FieldProjectIDs: []any{"proj-trashed"},
		}},
		// Tasks in proj-1: 2 open, 1 completed, 1 trashed (excluded from count).
		{uuid: "t-open1", entity: string(dongxi.EntityTask), fields: map[string]any{
			dongxi.FieldTitle:      "Open 1",
			dongxi.FieldType:       float64(dongxi.TaskTypeTask),
			dongxi.FieldStatus:     float64(dongxi.TaskStatusOpen),
			dongxi.FieldProjectIDs: []any{"proj-1"},
			dongxi.FieldTrashed:    false,
		}},
		{uuid: "t-open2", entity: string(dongxi.EntityTask), fields: map[string]any{
			dongxi.FieldTitle:      "Open 2",
			dongxi.FieldType:       float64(dongxi.TaskTypeTask),
			dongxi.FieldStatus:     float64(dongxi.TaskStatusOpen),
			dongxi.FieldProjectIDs: []any{"proj-1"},
			dongxi.FieldTrashed:    false,
		}},
		{uuid: "t-done", entity: string(dongxi.EntityTask), fields: map[string]any{
			dongxi.FieldTitle:      "Done",
			dongxi.FieldType:       float64(dongxi.TaskTypeTask),
			dongxi.FieldStatus:     float64(dongxi.TaskStatusCompleted),
			dongxi.FieldProjectIDs: []any{"proj-1"},
			dongxi.FieldTrashed:    false,
		}},
		{uuid: "t-trashed", entity: string(dongxi.EntityTask), fields: map[string]any{
			dongxi.FieldTitle:      "Trashed Task",
			dongxi.FieldType:       float64(dongxi.TaskTypeTask),
			dongxi.FieldStatus:     float64(dongxi.TaskStatusOpen),
			dongxi.FieldProjectIDs: []any{"proj-1"},
			dongxi.FieldTrashed:    true,
		}},
		// Task orphaned via action group pointing into trashed project.
		{uuid: "t-orphan-agr", entity: string(dongxi.EntityTask), fields: map[string]any{
			dongxi.FieldTitle:           "Orphaned by AGR",
			dongxi.FieldType:            float64(dongxi.TaskTypeTask),
			dongxi.FieldStatus:          float64(dongxi.TaskStatusOpen),
			dongxi.FieldTrashed:         false,
			dongxi.FieldActionGroupIDs:  []any{"heading-trashed-proj"},
		}},
		// Task orphaned via direct project being trashed.
		{uuid: "t-orphan-proj", entity: string(dongxi.EntityTask), fields: map[string]any{
			dongxi.FieldTitle:      "Orphaned by Project",
			dongxi.FieldType:       float64(dongxi.TaskTypeTask),
			dongxi.FieldStatus:     float64(dongxi.TaskStatusOpen),
			dongxi.FieldTrashed:    false,
			dongxi.FieldProjectIDs: []any{"proj-trashed"},
		}},
		// Non-orphaned task (no project).
		{uuid: "t-free", entity: string(dongxi.EntityTask), fields: map[string]any{
			dongxi.FieldTitle:   "Free Task",
			dongxi.FieldType:    float64(dongxi.TaskTypeTask),
			dongxi.FieldStatus:  float64(dongxi.TaskStatusOpen),
			dongxi.FieldTrashed: false,
		}},
		// Non-task entity (area) — should be skipped by project-related methods.
		{uuid: "area-2", entity: string(dongxi.EntityArea), fields: map[string]any{dongxi.FieldTitle: "Personal", dongxi.FieldTrashed: false}},
	}

	s := &thingsState{
		items:    items,
		byUUID:   make(map[string]*replayedItem),
		areas:    make(map[string]*replayedItem),
		projects: make(map[string]*replayedItem),
	}
	for i := range s.items {
		item := &s.items[i]
		s.byUUID[item.uuid] = item
		if item.entity == string(dongxi.EntityArea) {
			s.areas[item.uuid] = item
		}
		if item.entity == string(dongxi.EntityTask) && toInt(item.fields[dongxi.FieldType]) == int(dongxi.TaskTypeProject) {
			s.projects[item.uuid] = item
		}
	}
	return s
}

func TestProjectProgress(t *testing.T) {
	s := buildProjectState()

	total, completed := s.projectProgress("proj-1")
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if completed != 1 {
		t.Errorf("completed = %d, want 1", completed)
	}
}

func TestProjectProgressEmpty(t *testing.T) {
	s := buildProjectState()

	total, completed := s.projectProgress("nonexistent")
	if total != 0 || completed != 0 {
		t.Errorf("got (%d, %d), want (0, 0)", total, completed)
	}
}

func TestHeadingsForProject(t *testing.T) {
	s := buildProjectState()

	headings := s.headingsForProject("proj-1")
	if len(headings) != 2 {
		t.Fatalf("headingsForProject(proj-1) = %d, want 2", len(headings))
	}
	uuids := map[string]bool{}
	for _, h := range headings {
		uuids[h.uuid] = true
	}
	if !uuids["heading-1"] || !uuids["heading-2"] {
		t.Errorf("missing expected headings, got %v", uuids)
	}
}

func TestHeadingsForProjectEmpty(t *testing.T) {
	s := buildProjectState()

	headings := s.headingsForProject("nonexistent")
	if len(headings) != 0 {
		t.Errorf("headingsForProject(nonexistent) = %d, want 0", len(headings))
	}
}

func TestIsOrphanedByTrashedParentViaAGR(t *testing.T) {
	s := buildProjectState()

	item := s.byUUID["t-orphan-agr"]
	if !s.isOrphanedByTrashedParent(item) {
		t.Error("expected orphan via AGR to return true")
	}
}

func TestIsOrphanedByTrashedParentViaDirctProject(t *testing.T) {
	s := buildProjectState()

	item := s.byUUID["t-orphan-proj"]
	if !s.isOrphanedByTrashedParent(item) {
		t.Error("expected orphan via direct project to return true")
	}
}

func TestIsOrphanedByTrashedParentFreeTask(t *testing.T) {
	s := buildProjectState()

	item := s.byUUID["t-free"]
	if s.isOrphanedByTrashedParent(item) {
		t.Error("free task should not be orphaned")
	}
}

func TestIsOrphanedByTrashedParentNotTrashed(t *testing.T) {
	s := buildProjectState()

	item := s.byUUID["t-open1"]
	if s.isOrphanedByTrashedParent(item) {
		t.Error("task in non-trashed project should not be orphaned")
	}
}

func TestIsOrphanedByTrashedParentAGRNotFound(t *testing.T) {
	s := buildProjectState()

	item := &replayedItem{
		uuid:   "t-agr-missing",
		entity: string(dongxi.EntityTask),
		fields: map[string]any{
			dongxi.FieldActionGroupIDs: []any{"nonexistent-heading"},
			dongxi.FieldTrashed:        false,
		},
	}
	if s.isOrphanedByTrashedParent(item) {
		t.Error("missing AGR heading should not cause orphan")
	}
}

func TestIsOrphanedByTrashedParentAGRHeadingNoProject(t *testing.T) {
	s := buildProjectState()

	// Heading with no project assigned.
	headingNoProj := &replayedItem{
		uuid:   "heading-no-proj",
		entity: string(dongxi.EntityTask),
		fields: map[string]any{
			dongxi.FieldType: float64(dongxi.TaskTypeHeading),
		},
	}
	s.byUUID["heading-no-proj"] = headingNoProj

	item := &replayedItem{
		uuid:   "t-agr-no-proj",
		entity: string(dongxi.EntityTask),
		fields: map[string]any{
			dongxi.FieldActionGroupIDs: []any{"heading-no-proj"},
			dongxi.FieldTrashed:        false,
		},
	}
	if s.isOrphanedByTrashedParent(item) {
		t.Error("heading with no project should not cause orphan")
	}
}

func TestIsOrphanedByTrashedParentAGRProjectNotFound(t *testing.T) {
	s := buildProjectState()

	// Heading whose project doesn't exist in state.
	headingMissingProj := &replayedItem{
		uuid:   "heading-missing-proj",
		entity: string(dongxi.EntityTask),
		fields: map[string]any{
			dongxi.FieldType:       float64(dongxi.TaskTypeHeading),
			dongxi.FieldProjectIDs: []any{"nonexistent-proj"},
		},
	}
	s.byUUID["heading-missing-proj"] = headingMissingProj

	item := &replayedItem{
		uuid:   "t-agr-missing-proj",
		entity: string(dongxi.EntityTask),
		fields: map[string]any{
			dongxi.FieldActionGroupIDs: []any{"heading-missing-proj"},
			dongxi.FieldTrashed:        false,
		},
	}
	if s.isOrphanedByTrashedParent(item) {
		t.Error("heading with missing project should not cause orphan")
	}
}

func TestIsOrphanedDirectProjectNotFound(t *testing.T) {
	s := buildProjectState()

	item := &replayedItem{
		uuid:   "t-proj-missing",
		entity: string(dongxi.EntityTask),
		fields: map[string]any{
			dongxi.FieldProjectIDs: []any{"nonexistent-proj"},
			dongxi.FieldTrashed:    false,
		},
	}
	if s.isOrphanedByTrashedParent(item) {
		t.Error("task with missing project should not be orphaned")
	}
}

func TestBoolToInt(t *testing.T) {
	if boolToInt(true) != 1 {
		t.Error("boolToInt(true) should be 1")
	}
	if boolToInt(false) != 0 {
		t.Error("boolToInt(false) should be 0")
	}
}

func TestPrintJSON(t *testing.T) {
	// Capture stdout.
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := printJSON(map[string]string{"hello": "world"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	var result map[string]string
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if result["hello"] != "world" {
		t.Errorf("got %v, want {hello: world}", result)
	}
}

func TestIsTodayWithTodayIndex(t *testing.T) {
	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	fields := map[string]any{
		dongxi.FieldTodayIndex: float64(-485),
	}
	if !isToday(fields, now) {
		t.Error("task with todayIndex should be today")
	}
}

func TestIsTodayWithScheduledToday(t *testing.T) {
	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	// Scheduled for today at midnight UTC.
	scheduled := time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC)
	fields := map[string]any{
		dongxi.FieldScheduledDate: float64(scheduled.Unix()),
	}
	if !isToday(fields, now) {
		t.Error("task scheduled for today should be today")
	}
}

func TestIsTodayWithScheduledPast(t *testing.T) {
	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	scheduled := time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)
	fields := map[string]any{
		dongxi.FieldScheduledDate: float64(scheduled.Unix()),
	}
	if !isToday(fields, now) {
		t.Error("task scheduled in the past should be today")
	}
}

func TestIsTodayWithScheduledFuture(t *testing.T) {
	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	scheduled := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	fields := map[string]any{
		dongxi.FieldScheduledDate: float64(scheduled.Unix()),
	}
	if isToday(fields, now) {
		t.Error("task scheduled in the future should not be today")
	}
}

func TestIsTodayNoIndicators(t *testing.T) {
	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	fields := map[string]any{}
	if isToday(fields, now) {
		t.Error("task with no todayIndex or scheduledDate should not be today")
	}
}
