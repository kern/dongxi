package cmd

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/kern/dongxi/dongxi"
)

func TestRunUpcoming(t *testing.T) {
	tomorrow := float64(time.Now().Add(24 * time.Hour).Unix())
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Scheduled task", func(p map[string]any) {
			p[dongxi.FieldScheduledDate] = tomorrow
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
		makeTask("task-2", "No date task"),
	})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runUpcoming(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Scheduled task")) {
		t.Error("expected scheduled task in upcoming")
	}
	if bytes.Contains([]byte(output), []byte("No date task")) {
		t.Error("should not show task without date")
	}
}

func TestRunUpcomingEmpty(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "No date"),
	})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runUpcoming(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("(no upcoming tasks)")) {
		t.Error("expected '(no upcoming tasks)' message")
	}
}

func TestRunUpcomingWithDeadline(t *testing.T) {
	tomorrow := float64(time.Now().Add(24 * time.Hour).Unix())
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Deadline task", func(p map[string]any) {
			p[dongxi.FieldDeadline] = tomorrow
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
	})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runUpcoming(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Deadline task")) {
		t.Error("expected task with deadline in upcoming")
	}
	if !bytes.Contains([]byte(output), []byte("deadline:")) {
		t.Error("expected deadline annotation")
	}
}

func TestRunUpcomingExcludesCompleted(t *testing.T) {
	tomorrow := float64(time.Now().Add(24 * time.Hour).Unix())
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Completed scheduled", func(p map[string]any) {
			p[dongxi.FieldScheduledDate] = tomorrow
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
		}),
	})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runUpcoming(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("(no upcoming tasks)")) {
		t.Error("expected no upcoming for completed tasks")
	}
}

func TestRunUpcomingJSON(t *testing.T) {
	tomorrow := float64(time.Now().Add(24 * time.Hour).Unix())
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Scheduled task", func(p map[string]any) {
			p[dongxi.FieldScheduledDate] = tomorrow
		}),
	})

	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runUpcoming(nil, nil)

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

func TestRunUpcomingUntitledTask(t *testing.T) {
	tomorrow := float64(time.Now().Add(24 * time.Hour).Unix())
	setupMockState(t, []map[string]any{
		makeTask("task-1", "", func(p map[string]any) {
			p[dongxi.FieldScheduledDate] = tomorrow
		}),
	})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runUpcoming(nil, nil)

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

func TestRunUpcomingDeadlineOnlySortDate(t *testing.T) {
	// Task with deadline but no scheduled date - tests the sortDate=dd fallback.
	tomorrow := float64(time.Now().Add(24 * time.Hour).Unix())
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Deadline only", func(p map[string]any) {
			p[dongxi.FieldDeadline] = tomorrow
		}),
	})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runUpcoming(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("Deadline only")) {
		t.Error("expected deadline-only task in upcoming")
	}
}

func TestRunUpcomingExcludesTrashed(t *testing.T) {
	tomorrow := float64(time.Now().Add(24 * time.Hour).Unix())
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Trashed scheduled", func(p map[string]any) {
			p[dongxi.FieldScheduledDate] = tomorrow
			p[dongxi.FieldTrashed] = true
		}),
	})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runUpcoming(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("(no upcoming tasks)")) {
		t.Error("expected no upcoming for trashed tasks")
	}
}

func TestRunUpcomingMultipleDates(t *testing.T) {
	day1 := float64(time.Now().Add(24 * time.Hour).Unix())
	day2 := float64(time.Now().Add(48 * time.Hour).Unix())
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Task day 1", func(p map[string]any) {
			p[dongxi.FieldScheduledDate] = day1
		}),
		makeTask("task-2", "Task day 2", func(p map[string]any) {
			p[dongxi.FieldScheduledDate] = day2
		}),
	})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runUpcoming(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("2 task(s)")) {
		t.Error("expected 2 tasks in upcoming")
	}
}
