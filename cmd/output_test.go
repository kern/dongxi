package cmd

import (
	"testing"

	"github.com/kern/dongxi/dongxi"
)

func TestItemToOutputTask(t *testing.T) {
	s := buildTestState()
	item := s.byUUID["task-1"]
	item.fields[dongxi.FieldStatus] = float64(dongxi.TaskStatusOpen)
	item.fields[dongxi.FieldDestination] = float64(dongxi.TaskDestinationInbox)
	item.fields[dongxi.FieldTrashed] = false
	item.fields[dongxi.FieldCreationDate] = float64(1700000000)
	item.fields[dongxi.FieldModificationDate] = float64(1700001000)

	out := s.itemToOutput(item)

	if out.UUID != "task-1" {
		t.Errorf("UUID = %q, want %q", out.UUID, "task-1")
	}
	if out.Type != "task" {
		t.Errorf("Type = %q, want %q", out.Type, "task")
	}
	if out.Status != "open" {
		t.Errorf("Status = %q, want %q", out.Status, "open")
	}
	if out.Destination != "inbox" {
		t.Errorf("Destination = %q, want %q", out.Destination, "inbox")
	}
	if out.Trashed == nil || *out.Trashed != false {
		t.Errorf("Trashed = %v, want false", out.Trashed)
	}
	if out.Created == "" {
		t.Error("Created should not be empty")
	}
	if out.Modified == "" {
		t.Error("Modified should not be empty")
	}
	if out.Evening == nil {
		t.Error("Evening should not be nil for tasks")
	}
}

func TestItemToOutputTaskStatuses(t *testing.T) {
	s := buildTestState()

	tests := []struct {
		name   string
		status float64
		want   string
	}{
		{"open", float64(dongxi.TaskStatusOpen), "open"},
		{"cancelled", float64(dongxi.TaskStatusCancelled), "cancelled"},
		{"completed", float64(dongxi.TaskStatusCompleted), "completed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &replayedItem{
				uuid:   "test-status",
				entity: string(dongxi.EntityTask),
				fields: map[string]any{
					dongxi.FieldType:   float64(dongxi.TaskTypeTask),
					dongxi.FieldStatus: tt.status,
				},
			}
			s.byUUID["test-status"] = item
			out := s.itemToOutput(item)
			if out.Status != tt.want {
				t.Errorf("Status = %q, want %q", out.Status, tt.want)
			}
		})
	}
}

func TestItemToOutputTaskDestinations(t *testing.T) {
	s := buildTestState()

	tests := []struct {
		name string
		dest float64
		want string
	}{
		{"inbox", float64(dongxi.TaskDestinationInbox), "inbox"},
		{"today", float64(dongxi.TaskDestinationAnytime), "today"},
		{"someday", float64(dongxi.TaskDestinationSomeday), "someday"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &replayedItem{
				uuid:   "test-dest",
				entity: string(dongxi.EntityTask),
				fields: map[string]any{
					dongxi.FieldType:        float64(dongxi.TaskTypeTask),
					dongxi.FieldDestination: tt.dest,
				},
			}
			s.byUUID["test-dest"] = item
			out := s.itemToOutput(item)
			if out.Destination != tt.want {
				t.Errorf("Destination = %q, want %q", out.Destination, tt.want)
			}
		})
	}
}

func TestItemToOutputTaskTypes(t *testing.T) {
	s := buildTestState()

	tests := []struct {
		name string
		tp   float64
		want string
	}{
		{"task", float64(dongxi.TaskTypeTask), "task"},
		{"project", float64(dongxi.TaskTypeProject), "project"},
		{"heading", float64(dongxi.TaskTypeHeading), "heading"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &replayedItem{
				uuid:   "test-type",
				entity: string(dongxi.EntityTask),
				fields: map[string]any{dongxi.FieldType: tt.tp},
			}
			s.byUUID["test-type"] = item
			out := s.itemToOutput(item)
			if out.Type != tt.want {
				t.Errorf("Type = %q, want %q", out.Type, tt.want)
			}
		})
	}
}

func TestItemToOutputTaskWithProject(t *testing.T) {
	s := buildTestState()
	// task with project assignment
	item := &replayedItem{
		uuid:   "task-in-proj",
		entity: string(dongxi.EntityTask),
		fields: map[string]any{
			dongxi.FieldType:       float64(dongxi.TaskTypeTask),
			dongxi.FieldProjectIDs: []any{"proj-1"},
		},
	}
	s.byUUID["task-in-proj"] = item

	out := s.itemToOutput(item)
	if out.ProjectUUID != "proj-1" {
		t.Errorf("ProjectUUID = %q, want %q", out.ProjectUUID, "proj-1")
	}
	if out.Project != "My Project" {
		t.Errorf("Project = %q, want %q", out.Project, "My Project")
	}
}

func TestItemToOutputTaskInheritsAreaFromProject(t *testing.T) {
	// Task has no area, but its project has an area — should inherit.
	s := buildTestState()
	s.projects["proj-1"].fields[dongxi.FieldAreaIDs] = []any{"area-1"}

	item := &replayedItem{
		uuid:   "task-inherit-area",
		entity: string(dongxi.EntityTask),
		fields: map[string]any{
			dongxi.FieldType:       float64(dongxi.TaskTypeTask),
			dongxi.FieldProjectIDs: []any{"proj-1"},
		},
	}
	s.byUUID["task-inherit-area"] = item

	out := s.itemToOutput(item)
	if out.AreaUUID != "area-1" {
		t.Errorf("AreaUUID = %q, want %q", out.AreaUUID, "area-1")
	}
	if out.Area != "Work" {
		t.Errorf("Area = %q, want %q", out.Area, "Work")
	}
}

func TestItemToOutputTaskWithArea(t *testing.T) {
	s := buildTestState()
	item := &replayedItem{
		uuid:   "task-with-area",
		entity: string(dongxi.EntityTask),
		fields: map[string]any{
			dongxi.FieldType:    float64(dongxi.TaskTypeTask),
			dongxi.FieldAreaIDs: []any{"area-2"},
		},
	}
	s.byUUID["task-with-area"] = item

	out := s.itemToOutput(item)
	if out.AreaUUID != "area-2" {
		t.Errorf("AreaUUID = %q, want %q", out.AreaUUID, "area-2")
	}
	if out.Area != "Personal" {
		t.Errorf("Area = %q, want %q", out.Area, "Personal")
	}
}

func TestItemToOutputTaskWithHeading(t *testing.T) {
	s := buildTestState()
	heading := &replayedItem{
		uuid:   "heading-1",
		entity: string(dongxi.EntityTask),
		fields: map[string]any{
			dongxi.FieldType:  float64(dongxi.TaskTypeHeading),
			dongxi.FieldTitle: "Design",
		},
	}
	s.byUUID["heading-1"] = heading

	item := &replayedItem{
		uuid:   "task-heading",
		entity: string(dongxi.EntityTask),
		fields: map[string]any{
			dongxi.FieldType:            float64(dongxi.TaskTypeTask),
			dongxi.FieldActionGroupIDs:  []any{"heading-1"},
		},
	}
	s.byUUID["task-heading"] = item

	out := s.itemToOutput(item)
	if out.HeadingUUID != "heading-1" {
		t.Errorf("HeadingUUID = %q, want %q", out.HeadingUUID, "heading-1")
	}
	if out.Heading != "Design" {
		t.Errorf("Heading = %q, want %q", out.Heading, "Design")
	}
}

func TestItemToOutputTaskWithAgrNotHeading(t *testing.T) {
	// Action group points to a non-heading item — should not set heading.
	s := buildTestState()
	item := &replayedItem{
		uuid:   "task-agr-task",
		entity: string(dongxi.EntityTask),
		fields: map[string]any{
			dongxi.FieldType:           float64(dongxi.TaskTypeTask),
			dongxi.FieldActionGroupIDs: []any{"task-1"}, // task-1 is a task, not heading
		},
	}
	s.byUUID["task-agr-task"] = item

	out := s.itemToOutput(item)
	if out.HeadingUUID != "" {
		t.Errorf("HeadingUUID = %q, want empty", out.HeadingUUID)
	}
}

func TestItemToOutputTaskWithDates(t *testing.T) {
	s := buildTestState()
	item := &replayedItem{
		uuid:   "task-dates",
		entity: string(dongxi.EntityTask),
		fields: map[string]any{
			dongxi.FieldType:          float64(dongxi.TaskTypeTask),
			dongxi.FieldScheduledDate: float64(1700000000),
			dongxi.FieldDeadline:      float64(1700100000),
			dongxi.FieldStopDate:      float64(1700050000),
		},
	}
	s.byUUID["task-dates"] = item

	out := s.itemToOutput(item)
	if out.Scheduled == "" {
		t.Error("Scheduled should not be empty")
	}
	if out.Deadline == "" {
		t.Error("Deadline should not be empty")
	}
	if out.CompletedAt == "" {
		t.Error("CompletedAt should not be empty")
	}
}

func TestItemToOutputTaskWithNotes(t *testing.T) {
	s := buildTestState()
	item := &replayedItem{
		uuid:   "task-notes",
		entity: string(dongxi.EntityTask),
		fields: map[string]any{
			dongxi.FieldType: float64(dongxi.TaskTypeTask),
			dongxi.FieldNote: dongxi.NewNote("hello world"),
		},
	}
	s.byUUID["task-notes"] = item

	out := s.itemToOutput(item)
	if out.Notes != "hello world" {
		t.Errorf("Notes = %q, want %q", out.Notes, "hello world")
	}
}

func TestItemToOutputTaskWithTags(t *testing.T) {
	s := buildTestState()
	item := &replayedItem{
		uuid:   "task-tags",
		entity: string(dongxi.EntityTask),
		fields: map[string]any{
			dongxi.FieldType:   float64(dongxi.TaskTypeTask),
			dongxi.FieldTagIDs: []any{"tag-1", "tag-2"},
		},
	}
	s.byUUID["task-tags"] = item

	out := s.itemToOutput(item)
	if len(out.Tags) != 2 {
		t.Fatalf("Tags len = %d, want 2", len(out.Tags))
	}
}

func TestItemToOutputTaskEvening(t *testing.T) {
	s := buildTestState()
	item := &replayedItem{
		uuid:   "task-evening",
		entity: string(dongxi.EntityTask),
		fields: map[string]any{
			dongxi.FieldType:        float64(dongxi.TaskTypeTask),
			dongxi.FieldStartBucket: float64(1),
		},
	}
	s.byUUID["task-evening"] = item

	out := s.itemToOutput(item)
	if out.Evening == nil || !*out.Evening {
		t.Errorf("Evening = %v, want true", out.Evening)
	}
}

func TestItemToOutputArea(t *testing.T) {
	s := buildTestState()
	item := s.byUUID["area-1"]
	item.fields[dongxi.FieldTrashed] = false

	out := s.itemToOutput(item)
	if out.Type != "area" {
		t.Errorf("Type = %q, want %q", out.Type, "area")
	}
	if out.Trashed == nil || *out.Trashed != false {
		t.Errorf("Trashed = %v, want false", out.Trashed)
	}
}

func TestItemToOutputTag(t *testing.T) {
	s := buildTestState()
	item := s.byUUID["tag-1"]

	out := s.itemToOutput(item)
	if out.Type != "tag" {
		t.Errorf("Type = %q, want %q", out.Type, "tag")
	}
	if out.Title != "Urgent" {
		t.Errorf("Title = %q, want %q", out.Title, "Urgent")
	}
}

func TestItemToOutputChecklistItem(t *testing.T) {
	s := buildTestState()

	// Open checklist item.
	item := s.byUUID["ci-1"]
	out := s.itemToOutput(item)
	if out.Type != "checklist_item" {
		t.Errorf("Type = %q, want %q", out.Type, "checklist_item")
	}
	if out.Status != "open" {
		t.Errorf("Status = %q, want %q", out.Status, "open")
	}
	if out.TaskUUID != "task-1" {
		t.Errorf("TaskUUID = %q, want %q", out.TaskUUID, "task-1")
	}

	// Completed checklist item.
	item2 := s.byUUID["ci-2"]
	out2 := s.itemToOutput(item2)
	if out2.Status != "completed" {
		t.Errorf("Status = %q, want %q", out2.Status, "completed")
	}
}
