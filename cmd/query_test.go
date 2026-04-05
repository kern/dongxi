package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/kern/dongxi/dongxi"
)

// helper to capture stdout from runQuery.
func runQueryCapture(t *testing.T, args []string) (string, error) {
	t.Helper()
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runQuery(nil, args)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String(), err
}

// resetQueryFlags resets all query flags to defaults and restores them after the test.
func resetQueryFlags(t *testing.T) {
	t.Helper()
	origField := flagQueryField
	origType := flagQueryType
	origStatus := flagQueryStatus
	origDest := flagQueryDestination
	origArea := flagQueryArea
	origProject := flagQueryProject
	origTag := flagQueryTag
	origSchedBefore := flagQueryScheduledBefore
	origSchedAfter := flagQueryScheduledAfter
	origDeadBefore := flagQueryDeadlineBefore
	origDeadAfter := flagQueryDeadlineAfter
	origCreatedBefore := flagQueryCreatedBefore
	origCreatedAfter := flagQueryCreatedAfter
	origEvening := flagQueryEvening
	origHasNotes := flagQueryHasNotes
	origHasChecklist := flagQueryHasChecklist
	origHasTags := flagQueryHasTags
	origHasDeadline := flagQueryHasDeadline
	origCount := flagQueryCount
	origIncludeTrashed := flagQueryIncludeTrashed
	origJSON := flagJSON

	flagQueryField = "all"
	flagQueryType = "all"
	flagQueryStatus = "open"
	flagQueryDestination = "any"
	flagQueryArea = ""
	flagQueryProject = ""
	flagQueryTag = ""
	flagQueryScheduledBefore = ""
	flagQueryScheduledAfter = ""
	flagQueryDeadlineBefore = ""
	flagQueryDeadlineAfter = ""
	flagQueryCreatedBefore = ""
	flagQueryCreatedAfter = ""
	flagQueryEvening = false
	flagQueryHasNotes = false
	flagQueryHasChecklist = false
	flagQueryHasTags = false
	flagQueryHasDeadline = false
	flagQueryCount = false
	flagQueryIncludeTrashed = false
	flagJSON = false

	t.Cleanup(func() {
		flagQueryField = origField
		flagQueryType = origType
		flagQueryStatus = origStatus
		flagQueryDestination = origDest
		flagQueryArea = origArea
		flagQueryProject = origProject
		flagQueryTag = origTag
		flagQueryScheduledBefore = origSchedBefore
		flagQueryScheduledAfter = origSchedAfter
		flagQueryDeadlineBefore = origDeadBefore
		flagQueryDeadlineAfter = origDeadAfter
		flagQueryCreatedBefore = origCreatedBefore
		flagQueryCreatedAfter = origCreatedAfter
		flagQueryEvening = origEvening
		flagQueryHasNotes = origHasNotes
		flagQueryHasChecklist = origHasChecklist
		flagQueryHasTags = origHasTags
		flagQueryHasDeadline = origHasDeadline
		flagQueryCount = origCount
		flagQueryIncludeTrashed = origIncludeTrashed
		flagJSON = origJSON
	})
}

func TestQueryRegexpTitle(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "Buy milk today"),
		makeTask("t2", "Call dentist"),
	})
	resetQueryFlags(t)

	out, err := runQueryCapture(t, []string{"Buy.*today"})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("Buy milk today")) {
		t.Error("expected to find 'Buy milk today'")
	}
	if bytes.Contains([]byte(out), []byte("Call dentist")) {
		t.Error("should not find 'Call dentist'")
	}
}

func TestQueryRegexpNotes(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "Task one", func(p map[string]any) {
			p[dongxi.FieldNote] = map[string]any{"_t": "tx", "ch": float64(9), "v": "important", "t": float64(1)}
		}),
		makeTask("t2", "Task two"),
	})
	resetQueryFlags(t)

	out, err := runQueryCapture(t, []string{"important"})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("Task one")) {
		t.Error("expected to match notes content")
	}
	if bytes.Contains([]byte(out), []byte("Task two")) {
		t.Error("should not match task without matching notes")
	}
}

func TestQueryBadRegexp(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "Test"),
	})
	resetQueryFlags(t)

	_, err := runQueryCapture(t, []string{"[invalid"})
	if err == nil {
		t.Error("expected error for bad regexp")
	}
}

func TestQueryFieldTitle(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "Buy milk", func(p map[string]any) {
			p[dongxi.FieldNote] = map[string]any{"_t": "tx", "ch": float64(9), "v": "something", "t": float64(1)}
		}),
		makeTask("t2", "Other task", func(p map[string]any) {
			p[dongxi.FieldNote] = map[string]any{"_t": "tx", "ch": float64(8), "v": "Buy milk", "t": float64(1)}
		}),
	})
	resetQueryFlags(t)
	flagQueryField = "title"

	out, err := runQueryCapture(t, []string{"Buy milk"})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("Buy milk")) {
		t.Error("expected title match")
	}
	// "Other task" has "Buy milk" in notes but not title; should not appear with --field title.
	if bytes.Contains([]byte(out), []byte("Other task")) {
		t.Error("should not match notes when --field=title")
	}
}

func TestQueryFieldNotes(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "Buy milk"),
		makeTask("t2", "Task", func(p map[string]any) {
			p[dongxi.FieldNote] = map[string]any{"_t": "tx", "ch": float64(4), "v": "milk", "t": float64(1)}
		}),
	})
	resetQueryFlags(t)
	flagQueryField = "notes"

	out, err := runQueryCapture(t, []string{"milk"})
	if err != nil {
		t.Fatal(err)
	}
	// "Buy milk" has milk in title only, not notes.
	if bytes.Contains([]byte(out), []byte("Buy milk")) {
		t.Error("should not match title when --field=notes")
	}
	if !bytes.Contains([]byte(out), []byte("Task")) {
		t.Error("expected to match notes")
	}
}

func TestQueryFieldAll(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "Buy milk"),
		makeTask("t2", "Task", func(p map[string]any) {
			p[dongxi.FieldNote] = map[string]any{"_t": "tx", "ch": float64(4), "v": "milk", "t": float64(1)}
		}),
	})
	resetQueryFlags(t)
	flagQueryField = "all"

	out, err := runQueryCapture(t, []string{"milk"})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("Buy milk")) {
		t.Error("expected title match with --field=all")
	}
	if !bytes.Contains([]byte(out), []byte("Task")) {
		t.Error("expected notes match with --field=all")
	}
}

func TestQueryTypeTask(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "My Task"),
		makeProject("p1", "My Project"),
	})
	resetQueryFlags(t)
	flagQueryType = "task"
	flagQueryStatus = "any"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("My Task")) {
		t.Error("expected task")
	}
	if bytes.Contains([]byte(out), []byte("My Project")) {
		t.Error("should not include project with --type=task")
	}
}

func TestQueryTypeProject(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "My Task"),
		makeProject("p1", "My Project"),
	})
	resetQueryFlags(t)
	flagQueryType = "project"
	flagQueryStatus = "any"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains([]byte(out), []byte("My Task")) {
		t.Error("should not include task with --type=project")
	}
	if !bytes.Contains([]byte(out), []byte("My Project")) {
		t.Error("expected project")
	}
}

func TestQueryTypeHeading(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeProject("p1", "Proj"),
		makeHeading("h1", "My Heading", "p1"),
	})
	resetQueryFlags(t)
	flagQueryType = "heading"
	flagQueryStatus = "any"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("My Heading")) {
		t.Error("expected heading")
	}
}

func TestQueryTypeArea(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("a1", "Work"),
		makeTask("t1", "My Task"),
	})
	resetQueryFlags(t)
	flagQueryType = "area"
	flagQueryStatus = "any"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("Work")) {
		t.Error("expected area")
	}
	if bytes.Contains([]byte(out), []byte("My Task")) {
		t.Error("should not include task with --type=area")
	}
}

func TestQueryTypeTag(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTag("tg1", "Urgent"),
		makeTask("t1", "My Task"),
	})
	resetQueryFlags(t)
	flagQueryType = "tag"
	flagQueryStatus = "any"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("Urgent")) {
		t.Error("expected tag")
	}
	if bytes.Contains([]byte(out), []byte("My Task")) {
		t.Error("should not include task with --type=tag")
	}
}

func TestQueryTypeChecklist(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "My Task"),
		makeChecklistItem("ci1", "Check item", "t1"),
	})
	resetQueryFlags(t)
	flagQueryType = "checklist"
	flagQueryStatus = "any"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("Check item")) {
		t.Error("expected checklist item")
	}
	if bytes.Contains([]byte(out), []byte("My Task")) {
		t.Error("should not include task with --type=checklist")
	}
}

func TestQueryTypeAll(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "My Task"),
		makeProject("p1", "My Project"),
		makeArea("a1", "Work Area"),
		makeTag("tg1", "Urgent Tag"),
	})
	resetQueryFlags(t)
	flagQueryType = "all"
	flagQueryStatus = "any"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("My Task")) {
		t.Error("expected task")
	}
	if !bytes.Contains([]byte(out), []byte("My Project")) {
		t.Error("expected project")
	}
	if !bytes.Contains([]byte(out), []byte("Work Area")) {
		t.Error("expected area")
	}
	if !bytes.Contains([]byte(out), []byte("Urgent Tag")) {
		t.Error("expected tag")
	}
}

func TestQueryStatusOpen(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "Open Task"),
		makeTask("t2", "Done Task", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
		}),
	})
	resetQueryFlags(t)
	flagQueryStatus = "open"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("Open Task")) {
		t.Error("expected open task")
	}
	if bytes.Contains([]byte(out), []byte("Done Task")) {
		t.Error("should not include completed task")
	}
}

func TestQueryStatusCompleted(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "Open Task"),
		makeTask("t2", "Done Task", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
		}),
	})
	resetQueryFlags(t)
	flagQueryStatus = "completed"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains([]byte(out), []byte("Open Task")) {
		t.Error("should not include open task")
	}
	if !bytes.Contains([]byte(out), []byte("Done Task")) {
		t.Error("expected completed task")
	}
}

func TestQueryStatusCancelled(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "Open Task"),
		makeTask("t2", "Cancelled Task", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCancelled)
		}),
	})
	resetQueryFlags(t)
	flagQueryStatus = "cancelled"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains([]byte(out), []byte("Open Task")) {
		t.Error("should not include open task")
	}
	if !bytes.Contains([]byte(out), []byte("Cancelled Task")) {
		t.Error("expected cancelled task")
	}
}

func TestQueryStatusAny(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "Open Task"),
		makeTask("t2", "Done Task", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
		}),
		makeTask("t3", "Cancelled Task", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCancelled)
		}),
	})
	resetQueryFlags(t)
	flagQueryStatus = "any"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("Open Task")) {
		t.Error("expected open task")
	}
	if !bytes.Contains([]byte(out), []byte("Done Task")) {
		t.Error("expected completed task")
	}
	if !bytes.Contains([]byte(out), []byte("Cancelled Task")) {
		t.Error("expected cancelled task")
	}
}

func TestQueryDestinationInbox(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "Inbox Task", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationInbox)
		}),
		makeTask("t2", "Today Task", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
	})
	resetQueryFlags(t)
	flagQueryDestination = "inbox"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("Inbox Task")) {
		t.Error("expected inbox task")
	}
	if bytes.Contains([]byte(out), []byte("Today Task")) {
		t.Error("should not include today task")
	}
}

func TestQueryDestinationToday(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "Inbox Task", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationInbox)
		}),
		makeTask("t2", "Today Task", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
	})
	resetQueryFlags(t)
	flagQueryDestination = "today"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains([]byte(out), []byte("Inbox Task")) {
		t.Error("should not include inbox task")
	}
	if !bytes.Contains([]byte(out), []byte("Today Task")) {
		t.Error("expected today task")
	}
}

func TestQueryDestinationEvening(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "Evening Task", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
			p[dongxi.FieldStartBucket] = float64(1)
		}),
		makeTask("t2", "Morning Task", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
			p[dongxi.FieldStartBucket] = float64(0)
		}),
	})
	resetQueryFlags(t)
	flagQueryDestination = "evening"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("Evening Task")) {
		t.Error("expected evening task")
	}
	if bytes.Contains([]byte(out), []byte("Morning Task")) {
		t.Error("should not include morning task")
	}
}

func TestQueryDestinationSomeday(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "Someday Task", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationSomeday)
		}),
		makeTask("t2", "Today Task", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
	})
	resetQueryFlags(t)
	flagQueryDestination = "someday"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("Someday Task")) {
		t.Error("expected someday task")
	}
	if bytes.Contains([]byte(out), []byte("Today Task")) {
		t.Error("should not include today task")
	}
}

func TestQueryDestinationAny(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "Inbox Task", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationInbox)
		}),
		makeTask("t2", "Today Task", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
	})
	resetQueryFlags(t)
	flagQueryDestination = "any"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("Inbox Task")) {
		t.Error("expected inbox task")
	}
	if !bytes.Contains([]byte(out), []byte("Today Task")) {
		t.Error("expected today task")
	}
}

func TestQueryAreaFilter(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area1", "Work"),
		makeTask("t1", "Work Task", func(p map[string]any) {
			p[dongxi.FieldAreaIDs] = []any{"area1"}
		}),
		makeTask("t2", "Personal Task"),
	})
	resetQueryFlags(t)
	flagQueryArea = "area1"
	flagQueryType = "task"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("Work Task")) {
		t.Error("expected work task")
	}
	if bytes.Contains([]byte(out), []byte("Personal Task")) {
		t.Error("should not include personal task")
	}
}

func TestQueryAreaFilterInheritedFromProject(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area1", "Work"),
		makeProject("proj1", "Work Project", func(p map[string]any) {
			p[dongxi.FieldAreaIDs] = []any{"area1"}
		}),
		makeTask("t1", "Task in project", func(p map[string]any) {
			p[dongxi.FieldProjectIDs] = []any{"proj1"}
		}),
		makeTask("t2", "Unrelated task"),
	})
	resetQueryFlags(t)
	flagQueryArea = "area1"
	flagQueryType = "task"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("Task in project")) {
		t.Error("expected task inheriting area from project")
	}
}

func TestQueryProjectFilter(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeProject("proj1", "Project A"),
		makeTask("t1", "Task in A", func(p map[string]any) {
			p[dongxi.FieldProjectIDs] = []any{"proj1"}
		}),
		makeTask("t2", "Task elsewhere"),
	})
	resetQueryFlags(t)
	flagQueryProject = "proj1"
	flagQueryType = "task"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("Task in A")) {
		t.Error("expected task in project")
	}
	if bytes.Contains([]byte(out), []byte("Task elsewhere")) {
		t.Error("should not include task from other project")
	}
}

func TestQueryTagFilter(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTag("tag1", "Urgent"),
		makeTask("t1", "Tagged Task", func(p map[string]any) {
			p[dongxi.FieldTagIDs] = []any{"tag1"}
		}),
		makeTask("t2", "Untagged Task"),
	})
	resetQueryFlags(t)
	flagQueryTag = "tag1"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("Tagged Task")) {
		t.Error("expected tagged task")
	}
	if bytes.Contains([]byte(out), []byte("Untagged Task")) {
		t.Error("should not include untagged task")
	}
}

func TestQueryScheduledBefore(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "Early Task", func(p map[string]any) {
			p[dongxi.FieldScheduledDate] = float64(1711929600) // 2024-04-01
		}),
		makeTask("t2", "Late Task", func(p map[string]any) {
			p[dongxi.FieldScheduledDate] = float64(1743465600) // 2025-04-01
		}),
	})
	resetQueryFlags(t)
	flagQueryScheduledBefore = "2025-01-01"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("Early Task")) {
		t.Error("expected early task")
	}
	if bytes.Contains([]byte(out), []byte("Late Task")) {
		t.Error("should not include late task")
	}
}

func TestQueryScheduledAfter(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "Early Task", func(p map[string]any) {
			p[dongxi.FieldScheduledDate] = float64(1711929600) // 2024-04-01
		}),
		makeTask("t2", "Late Task", func(p map[string]any) {
			p[dongxi.FieldScheduledDate] = float64(1743465600) // 2025-04-01
		}),
	})
	resetQueryFlags(t)
	flagQueryScheduledAfter = "2025-01-01"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains([]byte(out), []byte("Early Task")) {
		t.Error("should not include early task")
	}
	if !bytes.Contains([]byte(out), []byte("Late Task")) {
		t.Error("expected late task")
	}
}

func TestQueryDeadlineBefore(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "Early Deadline", func(p map[string]any) {
			p[dongxi.FieldDeadline] = float64(1711929600) // 2024-04-01
		}),
		makeTask("t2", "Late Deadline", func(p map[string]any) {
			p[dongxi.FieldDeadline] = float64(1743465600) // 2025-04-01
		}),
	})
	resetQueryFlags(t)
	flagQueryDeadlineBefore = "2025-01-01"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("Early Deadline")) {
		t.Error("expected early deadline task")
	}
	if bytes.Contains([]byte(out), []byte("Late Deadline")) {
		t.Error("should not include late deadline task")
	}
}

func TestQueryDeadlineAfter(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "Early Deadline", func(p map[string]any) {
			p[dongxi.FieldDeadline] = float64(1711929600) // 2024-04-01
		}),
		makeTask("t2", "Late Deadline", func(p map[string]any) {
			p[dongxi.FieldDeadline] = float64(1743465600) // 2025-04-01
		}),
	})
	resetQueryFlags(t)
	flagQueryDeadlineAfter = "2025-01-01"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains([]byte(out), []byte("Early Deadline")) {
		t.Error("should not include early deadline task")
	}
	if !bytes.Contains([]byte(out), []byte("Late Deadline")) {
		t.Error("expected late deadline task")
	}
}

func TestQueryCreatedAfter(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "Old Task", func(p map[string]any) {
			p[dongxi.FieldCreationDate] = float64(1704067200) // 2024-01-01
		}),
		makeTask("t2", "New Task", func(p map[string]any) {
			p[dongxi.FieldCreationDate] = float64(1743465600) // 2025-04-01
		}),
	})
	resetQueryFlags(t)
	flagQueryCreatedAfter = "2025-01-01"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains([]byte(out), []byte("Old Task")) {
		t.Error("should not include old task")
	}
	if !bytes.Contains([]byte(out), []byte("New Task")) {
		t.Error("expected new task")
	}
}

func TestQueryCreatedBefore(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "Old Task", func(p map[string]any) {
			p[dongxi.FieldCreationDate] = float64(1704067200) // 2024-01-01
		}),
		makeTask("t2", "New Task", func(p map[string]any) {
			p[dongxi.FieldCreationDate] = float64(1743465600) // 2025-04-01
		}),
	})
	resetQueryFlags(t)
	flagQueryCreatedBefore = "2025-01-01"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("Old Task")) {
		t.Error("expected old task")
	}
	if bytes.Contains([]byte(out), []byte("New Task")) {
		t.Error("should not include new task")
	}
}

func TestQueryBadDateFormat(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "Test"),
	})
	resetQueryFlags(t)

	// Test each date flag with bad format.
	badDates := []struct {
		name string
		set  func()
	}{
		{"scheduled-before", func() { flagQueryScheduledBefore = "not-a-date" }},
		{"scheduled-after", func() { flagQueryScheduledAfter = "not-a-date" }},
		{"deadline-before", func() { flagQueryDeadlineBefore = "not-a-date" }},
		{"deadline-after", func() { flagQueryDeadlineAfter = "not-a-date" }},
		{"created-before", func() { flagQueryCreatedBefore = "not-a-date" }},
		{"created-after", func() { flagQueryCreatedAfter = "not-a-date" }},
	}
	for _, tt := range badDates {
		t.Run(tt.name, func(t *testing.T) {
			resetQueryFlags(t)
			tt.set()
			_, err := runQueryCapture(t, nil)
			if err == nil {
				t.Errorf("expected error for bad %s date", tt.name)
			}
		})
	}
}

func TestQueryEvening(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "Evening Task", func(p map[string]any) {
			p[dongxi.FieldStartBucket] = float64(1)
		}),
		makeTask("t2", "Morning Task", func(p map[string]any) {
			p[dongxi.FieldStartBucket] = float64(0)
		}),
	})
	resetQueryFlags(t)
	flagQueryEvening = true

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("Evening Task")) {
		t.Error("expected evening task")
	}
	if bytes.Contains([]byte(out), []byte("Morning Task")) {
		t.Error("should not include morning task")
	}
}

func TestQueryHasNotes(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "With Notes", func(p map[string]any) {
			p[dongxi.FieldNote] = map[string]any{"_t": "tx", "ch": float64(5), "v": "hello", "t": float64(1)}
		}),
		makeTask("t2", "No Notes"),
	})
	resetQueryFlags(t)
	flagQueryHasNotes = true

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("With Notes")) {
		t.Error("expected task with notes")
	}
	if bytes.Contains([]byte(out), []byte("No Notes")) {
		t.Error("should not include task without notes")
	}
}

func TestQueryHasChecklist(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "With Checklist"),
		makeChecklistItem("ci1", "Item 1", "t1"),
		makeTask("t2", "Without Checklist"),
	})
	resetQueryFlags(t)
	flagQueryHasChecklist = true
	flagQueryType = "task"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("With Checklist")) {
		t.Error("expected task with checklist")
	}
	if bytes.Contains([]byte(out), []byte("Without Checklist")) {
		t.Error("should not include task without checklist")
	}
}

func TestQueryHasTags(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTag("tag1", "Urgent"),
		makeTask("t1", "Tagged", func(p map[string]any) {
			p[dongxi.FieldTagIDs] = []any{"tag1"}
		}),
		makeTask("t2", "Untagged"),
	})
	resetQueryFlags(t)
	flagQueryHasTags = true

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("Tagged")) {
		t.Error("expected tagged task")
	}
	if bytes.Contains([]byte(out), []byte("Untagged")) {
		t.Error("should not include untagged task")
	}
}

func TestQueryHasDeadline(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "With Deadline", func(p map[string]any) {
			p[dongxi.FieldDeadline] = float64(1743465600)
		}),
		makeTask("t2", "No Deadline"),
	})
	resetQueryFlags(t)
	flagQueryHasDeadline = true

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("With Deadline")) {
		t.Error("expected task with deadline")
	}
	if bytes.Contains([]byte(out), []byte("No Deadline")) {
		t.Error("should not include task without deadline")
	}
}

func TestQueryCount(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "Task A"),
		makeTask("t2", "Task B"),
		makeTask("t3", "Done", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
		}),
	})
	resetQueryFlags(t)
	flagQueryCount = true

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "2\n" {
		t.Errorf("expected count '2', got %q", out)
	}
}

func TestQueryIncludeTrashed(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "Normal Task"),
		makeTask("t2", "Trashed Task", func(p map[string]any) {
			p[dongxi.FieldTrashed] = true
		}),
	})
	resetQueryFlags(t)
	flagQueryStatus = "any"
	flagQueryIncludeTrashed = true

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("Normal Task")) {
		t.Error("expected normal task")
	}
	if !bytes.Contains([]byte(out), []byte("Trashed Task")) {
		t.Error("expected trashed task with --include-trashed")
	}
}

func TestQueryExcludesTrashedByDefault(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "Normal Task"),
		makeTask("t2", "Trashed Task", func(p map[string]any) {
			p[dongxi.FieldTrashed] = true
		}),
	})
	resetQueryFlags(t)
	flagQueryStatus = "any"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("Normal Task")) {
		t.Error("expected normal task")
	}
	if bytes.Contains([]byte(out), []byte("Trashed Task")) {
		t.Error("should not include trashed task by default")
	}
}

func TestQueryExcludesOrphanedByTrashedParent(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeProject("proj1", "Trashed Project", func(p map[string]any) {
			p[dongxi.FieldTrashed] = true
		}),
		makeTask("t1", "Orphaned Task", func(p map[string]any) {
			p[dongxi.FieldProjectIDs] = []any{"proj1"}
		}),
	})
	resetQueryFlags(t)
	flagQueryType = "task"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains([]byte(out), []byte("Orphaned Task")) {
		t.Error("should not include task orphaned by trashed parent")
	}
}

func TestQueryJSON(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "Buy milk"),
	})
	resetQueryFlags(t)
	flagJSON = true

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte(`"uuid": "t1"`)) {
		t.Error("expected JSON output with uuid")
	}
	if !bytes.Contains([]byte(out), []byte(`"title": "Buy milk"`)) {
		t.Error("expected JSON output with title")
	}
}

func TestQueryNoResults(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "Buy milk"),
	})
	resetQueryFlags(t)

	out, err := runQueryCapture(t, []string{"nonexistent"})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("no results")) {
		t.Error("expected 'no results' message")
	}
}

func TestQueryNoArgument(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "Task A"),
		makeTask("t2", "Task B"),
	})
	resetQueryFlags(t)

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("Task A")) {
		t.Error("expected Task A")
	}
	if !bytes.Contains([]byte(out), []byte("Task B")) {
		t.Error("expected Task B")
	}
	if !bytes.Contains([]byte(out), []byte("2 result(s)")) {
		t.Error("expected result count")
	}
}

func TestQueryMultipleFiltersCombined(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTag("tag1", "Urgent"),
		makeTask("t1", "Urgent evening task", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
			p[dongxi.FieldStartBucket] = float64(1)
			p[dongxi.FieldTagIDs] = []any{"tag1"}
			p[dongxi.FieldDeadline] = float64(1743465600)
		}),
		makeTask("t2", "Urgent morning task", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
			p[dongxi.FieldTagIDs] = []any{"tag1"}
		}),
		makeTask("t3", "Evening untagged", func(p map[string]any) {
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
			p[dongxi.FieldStartBucket] = float64(1)
		}),
	})
	resetQueryFlags(t)
	flagQueryType = "task"
	flagQueryEvening = true
	flagQueryTag = "tag1"
	flagQueryHasDeadline = true

	out, err := runQueryCapture(t, []string{"Urgent"})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("Urgent evening task")) {
		t.Error("expected task matching all filters")
	}
	if bytes.Contains([]byte(out), []byte("Urgent morning task")) {
		t.Error("should not include non-evening task")
	}
	if bytes.Contains([]byte(out), []byte("Evening untagged")) {
		t.Error("should not include untagged task")
	}
}

func TestQueryUntitledItem(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", ""),
	})
	resetQueryFlags(t)

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("(untitled)")) {
		t.Error("expected (untitled) for empty title")
	}
}

func TestQueryTypePrefixes(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeProject("p1", "My Project"),
		makeArea("a1", "My Area"),
		makeTag("tg1", "My Tag"),
		makeTask("t1", "My Task"),
		makeChecklistItem("ci1", "My Check", "t1"),
		makeHeading("h1", "My Heading", "p1"),
	})
	resetQueryFlags(t)
	flagQueryStatus = "any"
	flagQueryType = "all"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("(project)")) {
		t.Error("expected (project) prefix")
	}
	if !bytes.Contains([]byte(out), []byte("(area)")) {
		t.Error("expected (area) prefix")
	}
	if !bytes.Contains([]byte(out), []byte("(tag)")) {
		t.Error("expected (tag) prefix")
	}
	if !bytes.Contains([]byte(out), []byte("(checklist)")) {
		t.Error("expected (checklist) prefix")
	}
	if !bytes.Contains([]byte(out), []byte("(heading)")) {
		t.Error("expected (heading) prefix")
	}
}

func TestQueryMatchesTypeUnknown(t *testing.T) {
	item := replayedItem{entity: string(dongxi.EntityTask), fields: map[string]any{}}
	if queryMatchesType(item, "unknown_type") {
		t.Error("unknown type should not match")
	}
}

func TestQueryItemHasChecklistFalse(t *testing.T) {
	items := replayHistory([]map[string]any{
		makeTask("t1", "No checklist"),
	})
	s := buildState(items)
	if queryItemHasChecklist(s, "t1") {
		t.Error("task without checklist items should return false")
	}
}

func TestQueryScheduledNoDate(t *testing.T) {
	// Task with no scheduled date should be excluded when scheduled-after is set.
	setupMockState(t, []map[string]any{
		makeTask("t1", "No Schedule"),
		makeTask("t2", "Has Schedule", func(p map[string]any) {
			p[dongxi.FieldScheduledDate] = float64(1743465600)
		}),
	})
	resetQueryFlags(t)
	flagQueryScheduledAfter = "2025-01-01"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains([]byte(out), []byte("No Schedule")) {
		t.Error("should not include task with no scheduled date")
	}
	if !bytes.Contains([]byte(out), []byte("Has Schedule")) {
		t.Error("expected task with schedule")
	}
}

func TestQueryDeadlineNoDate(t *testing.T) {
	// Task with no deadline should be excluded when deadline-after is set.
	setupMockState(t, []map[string]any{
		makeTask("t1", "No Deadline"),
		makeTask("t2", "Has Deadline", func(p map[string]any) {
			p[dongxi.FieldDeadline] = float64(1743465600)
		}),
	})
	resetQueryFlags(t)
	flagQueryDeadlineAfter = "2025-01-01"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains([]byte(out), []byte("No Deadline")) {
		t.Error("should not include task with no deadline")
	}
}

func TestQueryCreatedNoDate(t *testing.T) {
	// Task with no creation date should be excluded when created-after is set.
	setupMockState(t, []map[string]any{
		makeTask("t1", "No Created"),
		makeTask("t2", "Has Created", func(p map[string]any) {
			p[dongxi.FieldCreationDate] = float64(1743465600)
		}),
	})
	resetQueryFlags(t)
	flagQueryCreatedAfter = "2025-01-01"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains([]byte(out), []byte("No Created")) {
		t.Error("should not include task with no creation date")
	}
}

func TestQueryStatusFilterOnChecklistItem(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "Parent"),
		makeChecklistItem("ci1", "Open Check", "t1"),
	})
	resetQueryFlags(t)
	flagQueryType = "checklist"
	flagQueryStatus = "completed"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	// The checklist item is open, so it should not appear when filtering for completed.
	if bytes.Contains([]byte(out), []byte("Open Check")) {
		t.Error("should not include open checklist item with --status=completed")
	}
}

func TestQueryDestinationNotAppliedToNonTask(t *testing.T) {
	// Destination filter should not apply to areas/tags.
	setupMockState(t, []map[string]any{
		makeArea("a1", "Work Area"),
	})
	resetQueryFlags(t)
	flagQueryType = "area"
	flagQueryStatus = "any"
	flagQueryDestination = "today"

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Area should still show because destination filter only applies to tasks.
	if !bytes.Contains([]byte(out), []byte("Work Area")) {
		t.Error("expected area (destination filter should not apply to areas)")
	}
}

func TestQueryCountZero(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "Task", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
		}),
	})
	resetQueryFlags(t)
	flagQueryCount = true

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "0\n" {
		t.Errorf("expected count '0', got %q", out)
	}
}

func TestQueryJSONNoResults(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("t1", "Task", func(p map[string]any) {
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
		}),
	})
	resetQueryFlags(t)
	flagJSON = true

	out, err := runQueryCapture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("null")) {
		t.Error("expected null JSON for no results")
	}
}
