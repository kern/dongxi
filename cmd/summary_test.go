package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kern/dongxi/dongxi"
)

func captureSummaryOutput(t *testing.T, jsonMode bool) string {
	t.Helper()
	oldJSON := flagJSON
	flagJSON = jsonMode
	t.Cleanup(func() { flagJSON = oldJSON })

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runSummary(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func TestRunSummaryEmpty(t *testing.T) {
	setupMockState(t, []map[string]any{})
	output := captureSummaryOutput(t, false)

	if !strings.Contains(output, "Tasks: 0 open, 0 completed, 0 cancelled (0 total)") {
		t.Errorf("expected zero counts, got:\n%s", output)
	}
	if !strings.Contains(output, "Projects: 0 open, 0 completed (0 total)") {
		t.Error("expected zero project counts")
	}
	if !strings.Contains(output, "Areas: 0") {
		t.Error("expected zero areas")
	}
}

func TestRunSummaryEmptyJSON(t *testing.T) {
	setupMockState(t, []map[string]any{})
	output := captureSummaryOutput(t, true)

	var result summaryOutput
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result.Overview.TotalTasks != 0 {
		t.Errorf("expected 0 total tasks, got %d", result.Overview.TotalTasks)
	}
}

func TestRunSummaryBasic(t *testing.T) {
	yesterday := float64(time.Now().Add(-24 * time.Hour).Unix())

	setupMockState(t, []map[string]any{
		// Open inbox task
		makeTask("task-1", "Inbox task"),
		// Open today task (morning)
		makeTask("task-2", "Today morning", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
			p[dongxi.FieldStartBucket] = float64(0)
			withToday(p)
		}),
		// Open today task (evening)
		makeTask("task-3", "Today evening", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
			p[dongxi.FieldStartBucket] = float64(1)
			withToday(p)
		}),
		// Someday task
		makeTask("task-4", "Someday task", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationSomeday)
		}),
		// Completed task
		makeTask("task-5", "Completed task", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
			p[dongxi.FieldStopDate] = yesterday
		}),
		// Cancelled task
		makeTask("task-6", "Cancelled task", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCancelled)
		}),
	})

	output := captureSummaryOutput(t, false)

	if !strings.Contains(output, "Tasks: 4 open, 1 completed, 1 cancelled (6 total)") {
		t.Errorf("wrong task counts, got:\n%s", output)
	}
	if !strings.Contains(output, "Inbox: 1 | Today: 2 (1 evening) | Someday: 1") {
		t.Errorf("wrong destination counts, got:\n%s", output)
	}
}

func TestRunSummaryJSON(t *testing.T) {
	tomorrow := float64(time.Now().Add(48 * time.Hour).Unix())
	yesterday := float64(time.Now().Add(-48 * time.Hour).Unix())

	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
		makeTag("tag-1", "urgent"),
		makeProject("proj-1", "Q3 Planning", func(p map[string]any) {
			p[dongxi.FieldAreaIDs] = []any{"area-1"}
			p[dongxi.FieldScheduledDate] = tomorrow
			p[dongxi.FieldDeadline] = tomorrow
		}),
		makeTask("task-1", "Task in project", func(p map[string]any) {
			p[dongxi.FieldProjectIDs] = []any{"proj-1"}
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
			p[dongxi.FieldTagIDs] = []any{"tag-1"}
			withToday(p)
		}),
		makeTask("task-2", "Inbox task", func(p map[string]any) {
			p[dongxi.FieldCreationDate] = float64(time.Now().Unix())
		}),
		makeTask("task-3", "Overdue task", func(p map[string]any) {
			p[dongxi.FieldDeadline] = yesterday
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
			withToday(p)
		}),
		makeTask("task-4", "Upcoming task", func(p map[string]any) {
			p[dongxi.FieldScheduledDate] = tomorrow
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
			withToday(p)
		}),
	})

	output := captureSummaryOutput(t, true)

	var result summaryOutput
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if result.Overview.TotalTasks != 4 {
		t.Errorf("expected 4 total tasks, got %d", result.Overview.TotalTasks)
	}
	if result.Overview.OpenTasks != 4 {
		t.Errorf("expected 4 open tasks, got %d", result.Overview.OpenTasks)
	}
	if result.Overview.InboxCount != 1 {
		t.Errorf("expected 1 inbox, got %d", result.Overview.InboxCount)
	}
	if result.Overview.TodayCount != 3 {
		t.Errorf("expected 3 today, got %d", result.Overview.TodayCount)
	}
	if result.Overview.UpcomingCount != 1 {
		t.Errorf("expected 1 upcoming, got %d", result.Overview.UpcomingCount)
	}
	if result.Overview.OverdueCount != 1 {
		t.Errorf("expected 1 overdue, got %d", result.Overview.OverdueCount)
	}
	if result.Overview.TotalProjects != 1 {
		t.Errorf("expected 1 project, got %d", result.Overview.TotalProjects)
	}
	if result.Overview.TotalAreas != 1 {
		t.Errorf("expected 1 area, got %d", result.Overview.TotalAreas)
	}
	if result.Overview.TotalTags != 1 {
		t.Errorf("expected 1 tag, got %d", result.Overview.TotalTags)
	}
	if len(result.Areas) != 1 {
		t.Fatalf("expected 1 area, got %d", len(result.Areas))
	}
	if result.Areas[0].Title != "Work" {
		t.Errorf("expected area title Work, got %s", result.Areas[0].Title)
	}
	if len(result.Areas[0].Projects) != 1 {
		t.Fatalf("expected 1 project in area, got %d", len(result.Areas[0].Projects))
	}
	if result.Areas[0].Projects[0].Scheduled == "" {
		t.Error("expected project to have scheduled date")
	}
	if result.Areas[0].Projects[0].Deadline == "" {
		t.Error("expected project to have deadline")
	}
	if len(result.Tags) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(result.Tags))
	}
	if result.Tags[0].Title != "urgent" {
		t.Errorf("expected tag title urgent, got %s", result.Tags[0].Title)
	}
	if result.Tags[0].TaskCount != 1 {
		t.Errorf("expected tag task count 1, got %d", result.Tags[0].TaskCount)
	}
	if len(result.Inbox) != 1 {
		t.Fatalf("expected 1 inbox item, got %d", len(result.Inbox))
	}
	if result.Inbox[0].Title != "Inbox task" {
		t.Errorf("expected inbox title 'Inbox task', got %s", result.Inbox[0].Title)
	}
	if result.Inbox[0].Created == "" {
		t.Error("expected inbox item to have created date")
	}
}

func TestRunSummaryWithAreas(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
		makeArea("area-2", "Personal"),
		makeProject("proj-1", "Project A", func(p map[string]any) {
			p[dongxi.FieldAreaIDs] = []any{"area-1"}
		}),
		makeProject("proj-2", "Project B", func(p map[string]any) {
			p[dongxi.FieldAreaIDs] = []any{"area-2"}
		}),
		makeTask("task-1", "Task in A", func(p map[string]any) {
			p[dongxi.FieldProjectIDs] = []any{"proj-1"}
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
	})

	output := captureSummaryOutput(t, false)

	if !strings.Contains(output, "Work") {
		t.Error("expected Work area")
	}
	if !strings.Contains(output, "Personal") {
		t.Error("expected Personal area")
	}
	if !strings.Contains(output, "Project A") {
		t.Error("expected Project A")
	}
	if !strings.Contains(output, "Project B") {
		t.Error("expected Project B")
	}
}

func TestRunSummaryUnassignedProjects(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
		makeProject("proj-1", "Assigned", func(p map[string]any) {
			p[dongxi.FieldAreaIDs] = []any{"area-1"}
		}),
		makeProject("proj-2", "Unassigned"),
	})

	output := captureSummaryOutput(t, false)

	if !strings.Contains(output, "(no area)") {
		t.Errorf("expected '(no area)' section, got:\n%s", output)
	}
	if !strings.Contains(output, "Unassigned") {
		t.Error("expected Unassigned project")
	}

	// Also test JSON.
	jsonOutput := captureSummaryOutput(t, true)
	var result summaryOutput
	if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result.UnassignedProjects) != 1 {
		t.Errorf("expected 1 unassigned project, got %d", len(result.UnassignedProjects))
	}
	if result.UnassignedProjects[0].Title != "Unassigned" {
		t.Errorf("expected title 'Unassigned', got %s", result.UnassignedProjects[0].Title)
	}
}

func TestRunSummaryTags(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTag("tag-1", "urgent"),
		makeTag("tag-2", "waiting"),
		makeTask("task-1", "Tagged task 1", func(p map[string]any) {
			p[dongxi.FieldTagIDs] = []any{"tag-1"}
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
			withToday(p)
		}),
		makeTask("task-2", "Tagged task 2", func(p map[string]any) {
			p[dongxi.FieldTagIDs] = []any{"tag-1"}
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
			withToday(p)
		}),
		makeTask("task-3", "Waiting task", func(p map[string]any) {
			p[dongxi.FieldTagIDs] = []any{"tag-2"}
		}),
	})

	// Tags are not shown in human output, only JSON.
	jsonOutput := captureSummaryOutput(t, true)
	var result summaryOutput
	if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(result.Tags))
	}
}

func TestRunSummaryInbox(t *testing.T) {
	createdTS := float64(time.Date(2025, 3, 28, 10, 0, 0, 0, time.UTC).Unix())
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy groceries", func(p map[string]any) {
			p[dongxi.FieldCreationDate] = createdTS
		}),
		makeTask("task-2", "Call dentist"),
		makeTask("task-3", "", func(p map[string]any) {
			// untitled inbox task
		}),
	})

	output := captureSummaryOutput(t, false)

	if !strings.Contains(output, "Inbox (3):") {
		t.Errorf("expected 'Inbox (3):', got:\n%s", output)
	}
	if !strings.Contains(output, "Buy groceries") {
		t.Error("expected Buy groceries")
	}
	if !strings.Contains(output, "Call dentist") {
		t.Error("expected Call dentist")
	}
	if !strings.Contains(output, "(untitled)") {
		t.Error("expected (untitled) for untitled task")
	}

	// JSON
	jsonOutput := captureSummaryOutput(t, true)
	var result summaryOutput
	if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result.Inbox) != 3 {
		t.Errorf("expected 3 inbox items, got %d", len(result.Inbox))
	}
	if result.Inbox[0].Created == "" {
		t.Error("expected created date on first inbox item")
	}
}

func TestRunSummaryToday(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work", func(p map[string]any) {
			p[dongxi.FieldIndex] = float64(10)
		}),
		makeArea("area-2", "Personal", func(p map[string]any) {
			p[dongxi.FieldIndex] = float64(-10)
		}),
		makeProject("proj-1", "My Project", func(p map[string]any) {
			p[dongxi.FieldAreaIDs] = []any{"area-1"}
			p[dongxi.FieldIndex] = float64(5)
		}),
		makeProject("proj-2", "Side Project", func(p map[string]any) {
			p[dongxi.FieldAreaIDs] = []any{"area-1"}
			p[dongxi.FieldIndex] = float64(-5)
		}),
		makeTask("task-1", "Standalone task", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
			p[dongxi.FieldStartBucket] = float64(0)
			withToday(p)
		}),
		makeTask("task-2", "Project task", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
			p[dongxi.FieldProjectIDs] = []any{"proj-1"}
			withToday(p)
		}),
		makeTask("task-5", "Side project task", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
			p[dongxi.FieldProjectIDs] = []any{"proj-2"}
			withToday(p)
		}),
		makeTask("task-3", "Evening task", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
			p[dongxi.FieldStartBucket] = float64(1)
			p[dongxi.FieldAreaIDs] = []any{"area-2"}
			withToday(p)
		}),
		makeTag("tag-1", "important"),
		makeTask("task-4", "Tagged today", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
			p[dongxi.FieldTagIDs] = []any{"tag-1"}
			withToday(p)
		}),
	})

	output := captureSummaryOutput(t, false)

	if !strings.Contains(output, "Today (5):") {
		t.Errorf("expected 'Today (5):', got:\n%s", output)
	}
	if !strings.Contains(output, "Standalone task") {
		t.Error("expected Standalone task")
	}
	if !strings.Contains(output, "Work:") {
		t.Errorf("expected area heading 'Work:', got:\n%s", output)
	}
	if !strings.Contains(output, "My Project:") {
		t.Errorf("expected project heading 'My Project:', got:\n%s", output)
	}
	if !strings.Contains(output, "Evening task") {
		t.Error("expected Evening task")
	}
	if !strings.Contains(output, "(evening)") {
		t.Error("expected (evening) marker")
	}
	// Personal (ix=-10) should appear before Work (ix=10).
	personalIdx := strings.Index(output, "Personal:")
	workIdx := strings.Index(output, "Work:")
	if personalIdx < 0 || workIdx < 0 || personalIdx > workIdx {
		t.Errorf("expected Personal before Work in output:\n%s", output)
	}

	// JSON: check project and tag resolution.
	jsonOutput := captureSummaryOutput(t, true)
	var result summaryOutput
	if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result.Today) != 5 {
		t.Fatalf("expected 5 today items, got %d", len(result.Today))
	}
	// task-2 should have project and area set.
	var found bool
	for _, item := range result.Today {
		if item.Title == "Project task" {
			found = true
			if item.Project != "My Project" {
				t.Errorf("expected project 'My Project', got %q", item.Project)
			}
			if item.Area != "Work" {
				t.Errorf("expected area 'Work', got %q", item.Area)
			}
		}
		if item.Title == "Tagged today" && len(item.Tags) == 0 {
			t.Error("expected tags on 'Tagged today'")
		}
	}
	if !found {
		t.Error("expected to find 'Project task' in today items")
	}
}

func TestRunSummaryUpcoming(t *testing.T) {
	oldNow := nowFunc
	// Fix time to ensure deterministic results.
	fixedNow := time.Date(2025, 4, 1, 12, 0, 0, 0, time.UTC)
	nowFunc = func() time.Time { return fixedNow }
	t.Cleanup(func() { nowFunc = oldNow })

	tomorrow := float64(time.Date(2025, 4, 2, 0, 0, 0, 0, time.UTC).Unix())
	yesterday := float64(time.Date(2025, 3, 31, 0, 0, 0, 0, time.UTC).Unix())

	setupMockState(t, []map[string]any{
		makeTask("task-1", "Future task", func(p map[string]any) {
			p[dongxi.FieldScheduledDate] = tomorrow
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
			withToday(p)
		}),
		makeTask("task-2", "Past scheduled", func(p map[string]any) {
			p[dongxi.FieldScheduledDate] = yesterday
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
			withToday(p)
		}),
	})

	output := captureSummaryOutput(t, false)

	if !strings.Contains(output, "Upcoming: 1") {
		t.Errorf("expected Upcoming: 1, got:\n%s", output)
	}
}

func TestRunSummaryOverdue(t *testing.T) {
	oldNow := nowFunc
	fixedNow := time.Date(2025, 4, 1, 12, 0, 0, 0, time.UTC)
	nowFunc = func() time.Time { return fixedNow }
	t.Cleanup(func() { nowFunc = oldNow })

	yesterday := float64(time.Date(2025, 3, 31, 0, 0, 0, 0, time.UTC).Unix())
	tomorrow := float64(time.Date(2025, 4, 2, 0, 0, 0, 0, time.UTC).Unix())

	setupMockState(t, []map[string]any{
		makeTask("task-1", "Overdue task", func(p map[string]any) {
			p[dongxi.FieldDeadline] = yesterday
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
			withToday(p)
		}),
		makeTask("task-2", "Future deadline", func(p map[string]any) {
			p[dongxi.FieldDeadline] = tomorrow
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
			withToday(p)
		}),
	})

	output := captureSummaryOutput(t, false)

	if !strings.Contains(output, "Overdue: 1") {
		t.Errorf("expected Overdue: 1, got:\n%s", output)
	}
}

func TestRunSummaryCompletedProjects(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeProject("proj-1", "Open Project"),
		makeProject("proj-2", "Done Project", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
		}),
		makeProject("proj-3", "Cancelled Project", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCancelled)
		}),
		makeProject("proj-4", "Someday Project", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationSomeday)
		}),
	})

	output := captureSummaryOutput(t, false)

	if !strings.Contains(output, "Projects: 2 open, 1 completed (4 total)") {
		t.Errorf("expected project counts, got:\n%s", output)
	}

	// JSON.
	jsonOutput := captureSummaryOutput(t, true)
	var result summaryOutput
	if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result.Overview.OpenProjects != 2 {
		t.Errorf("expected 2 open projects, got %d", result.Overview.OpenProjects)
	}
	if result.Overview.CompletedProjects != 1 {
		t.Errorf("expected 1 completed project, got %d", result.Overview.CompletedProjects)
	}
	if result.Overview.TotalProjects != 4 {
		t.Errorf("expected 4 total projects, got %d", result.Overview.TotalProjects)
	}
	if len(result.UnassignedProjects) != 1 || result.UnassignedProjects[0].Title != "Open Project" {
		t.Errorf("expected only 'Open Project' in listing, got %v", result.UnassignedProjects)
	}
}

func TestRunSummaryTrashedItemsExcluded(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Normal task"),
		makeTask("task-2", "Trashed task", func(p map[string]any) {
			p[dongxi.FieldTrashed] = true
		}),
		makeProject("proj-1", "Trashed project", func(p map[string]any) {
			p[dongxi.FieldTrashed] = true
		}),
	})

	jsonOutput := captureSummaryOutput(t, true)
	var result summaryOutput
	if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result.Overview.TotalTasks != 1 {
		t.Errorf("expected 1 total task (trashed excluded), got %d", result.Overview.TotalTasks)
	}
	if result.Overview.TotalProjects != 0 {
		t.Errorf("expected 0 projects (trashed excluded), got %d", result.Overview.TotalProjects)
	}
}

func TestRunSummaryProjectWithHeadingsAndNotes(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeProject("proj-1", "Noted Project", func(p map[string]any) {
			p[dongxi.FieldNote] = map[string]any{"_t": "tx", "ch": 5, "v": "hello", "t": 1}
		}),
		makeHeading("head-1", "Phase 1", "proj-1"),
		makeHeading("head-2", "Phase 2", "proj-1"),
		makeTask("task-1", "Task under proj", func(p map[string]any) {
			p[dongxi.FieldProjectIDs] = []any{"proj-1"}
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
		makeTask("task-2", "Completed sub", func(p map[string]any) {
			p[dongxi.FieldProjectIDs] = []any{"proj-1"}
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
	})

	jsonOutput := captureSummaryOutput(t, true)
	var result summaryOutput
	if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(result.UnassignedProjects) != 1 {
		t.Fatalf("expected 1 unassigned project, got %d", len(result.UnassignedProjects))
	}
	p := result.UnassignedProjects[0]
	if !p.HasNotes {
		t.Error("expected has_notes=true")
	}
	if len(p.Headings) != 2 {
		t.Errorf("expected 2 headings, got %d", len(p.Headings))
	}
	if p.TasksTotal != 2 {
		t.Errorf("expected 2 tasks_total, got %d", p.TasksTotal)
	}
	if p.TasksCompleted != 1 {
		t.Errorf("expected 1 tasks_completed, got %d", p.TasksCompleted)
	}
	if p.TasksOpen != 1 {
		t.Errorf("expected 1 tasks_open, got %d", p.TasksOpen)
	}
}

func TestRunSummaryProjectWithTags(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTag("tag-1", "priority"),
		makeProject("proj-1", "Tagged Project", func(p map[string]any) {
			p[dongxi.FieldTagIDs] = []any{"tag-1"}
		}),
	})

	jsonOutput := captureSummaryOutput(t, true)
	var result summaryOutput
	if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(result.UnassignedProjects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(result.UnassignedProjects))
	}
	if len(result.UnassignedProjects[0].Tags) != 1 || result.UnassignedProjects[0].Tags[0] != "priority" {
		t.Errorf("expected tag 'priority', got %v", result.UnassignedProjects[0].Tags)
	}
}

func TestRunSummaryAreaOpenTaskCount(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
		makeProject("proj-1", "Work Project", func(p map[string]any) {
			p[dongxi.FieldAreaIDs] = []any{"area-1"}
		}),
		// Task directly in area.
		makeTask("task-1", "Direct area task", func(p map[string]any) {
			p[dongxi.FieldAreaIDs] = []any{"area-1"}
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
		// Task inheriting area from project.
		makeTask("task-2", "Project task", func(p map[string]any) {
			p[dongxi.FieldProjectIDs] = []any{"proj-1"}
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
		// Completed task should NOT count.
		makeTask("task-3", "Done task", func(p map[string]any) {
			p[dongxi.FieldAreaIDs] = []any{"area-1"}
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
		}),
	})

	jsonOutput := captureSummaryOutput(t, true)
	var result summaryOutput
	if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(result.Areas) != 1 {
		t.Fatalf("expected 1 area, got %d", len(result.Areas))
	}
	if result.Areas[0].OpenTaskCount != 2 {
		t.Errorf("expected 2 open tasks in area, got %d", result.Areas[0].OpenTaskCount)
	}
}

func TestRunSummaryTodayOnlyMorning(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Only morning", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
			p[dongxi.FieldStartBucket] = float64(0)
			withToday(p)
		}),
	})

	output := captureSummaryOutput(t, false)

	if !strings.Contains(output, "Only morning") {
		t.Error("expected morning task in output")
	}
	if strings.Contains(output, "(evening)") {
		t.Error("should not have (evening) marker for morning-only tasks")
	}
}

func TestRunSummaryTodayOnlyEvening(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Only evening", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
			p[dongxi.FieldStartBucket] = float64(1)
			withToday(p)
		}),
	})

	output := captureSummaryOutput(t, false)

	if !strings.Contains(output, "Only evening") {
		t.Error("expected evening task in output")
	}
	if !strings.Contains(output, "(evening)") {
		t.Error("expected (evening) marker")
	}
}

func TestRunSummaryUntitledProject(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeProject("proj-1", ""),
	})

	jsonOutput := captureSummaryOutput(t, true)
	var result summaryOutput
	if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result.UnassignedProjects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(result.UnassignedProjects))
	}
	if result.UnassignedProjects[0].Title != "(untitled)" {
		t.Errorf("expected (untitled), got %s", result.UnassignedProjects[0].Title)
	}
}

func TestRunSummaryUntitledArea(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", ""),
	})

	output := captureSummaryOutput(t, false)
	if !strings.Contains(output, "(untitled)") {
		t.Error("expected (untitled) for untitled area")
	}
}

func TestRunSummaryTrashedAreaExcluded(t *testing.T) {
	setupMockState(t, []map[string]any{
		{
			"area-1": map[string]any{
				dongxi.CommitKeyType:   float64(dongxi.ItemTypeCreate),
				dongxi.CommitKeyEntity: string(dongxi.EntityArea),
				dongxi.CommitKeyPayload: map[string]any{
					dongxi.FieldTitle:   "Trashed Area",
					dongxi.FieldTrashed: true,
				},
			},
		},
		makeArea("area-2", "Active Area"),
	})

	jsonOutput := captureSummaryOutput(t, true)
	var result summaryOutput
	if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result.Overview.TotalAreas != 1 {
		t.Errorf("expected 1 area (trashed excluded), got %d", result.Overview.TotalAreas)
	}
	if len(result.Areas) != 1 || result.Areas[0].Title != "Active Area" {
		t.Errorf("expected only Active Area, got %v", result.Areas)
	}
}

func TestRunSummaryOrphanedTask(t *testing.T) {
	// Task under a trashed project should be excluded.
	setupMockState(t, []map[string]any{
		makeProject("proj-1", "Trashed Proj", func(p map[string]any) {
			p[dongxi.FieldTrashed] = true
		}),
		makeTask("task-1", "Orphaned task", func(p map[string]any) {
			p[dongxi.FieldProjectIDs] = []any{"proj-1"}
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
	})

	jsonOutput := captureSummaryOutput(t, true)
	var result summaryOutput
	if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result.Overview.TotalTasks != 0 {
		t.Errorf("expected 0 tasks (orphaned excluded), got %d", result.Overview.TotalTasks)
	}
}

func TestRunSummaryNoAreasOrProjects(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Just a task"),
	})

	output := captureSummaryOutput(t, false)
	// Should not have Areas & Projects section.
	if strings.Contains(output, "Areas & Projects:") {
		t.Error("should not show Areas & Projects when there are none")
	}
}

func TestRunSummaryNoTags(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Just a task"),
	})

	output := captureSummaryOutput(t, false)
	if strings.Contains(output, "Tags:\n") {
		t.Error("should not show Tags section when there are none")
	}
}

func TestRunSummaryNoInbox(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Today task", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
	})

	output := captureSummaryOutput(t, false)
	if strings.Contains(output, "Inbox (") {
		t.Error("should not show Inbox section when there are no inbox items")
	}
}

func TestRunSummaryNoToday(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Inbox task"),
	})

	output := captureSummaryOutput(t, false)
	if strings.Contains(output, "Today (") {
		t.Error("should not show Today section when there are no today items")
	}
}

func TestRunSummaryProjectStatusStrings(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeProject("proj-1", "Open"),
		makeProject("proj-2", "Done", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
		}),
		makeProject("proj-3", "Cancelled", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCancelled)
		}),
	})

	jsonOutput := captureSummaryOutput(t, true)
	var result summaryOutput
	if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	statusMap := map[string]string{}
	for _, p := range result.UnassignedProjects {
		statusMap[p.Title] = p.Status
	}
	if statusMap["Open"] != "open" {
		t.Errorf("expected status 'open', got %s", statusMap["Open"])
	}
	if _, ok := statusMap["Done"]; ok {
		t.Error("completed projects should not appear in listing")
	}
	if _, ok := statusMap["Cancelled"]; ok {
		t.Error("cancelled projects should not appear in listing")
	}
}

func TestRunSummaryLoadStateErr(t *testing.T) {
	setupMockStateErr(t, fmt.Errorf("mock load error"))
	err := runSummary(nil, nil)
	if err == nil || err.Error() != "mock load error" {
		t.Fatalf("expected load error, got %v", err)
	}
}

func TestRunSummaryUntitledTodayTask(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
			withToday(p)
		}),
	})
	output := captureSummaryOutput(t, false)
	if !strings.Contains(output, "(untitled)") {
		t.Error("expected (untitled) in output for today task with empty title")
	}
}

func TestRunSummaryUntitledTag(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTag("tag-1", ""),
	})
	jsonOutput := captureSummaryOutput(t, true)
	var result summaryOutput
	if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result.Tags) != 1 || result.Tags[0].Title != "(untitled)" {
		t.Error("expected (untitled) tag in JSON output")
	}
}
