package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/kern/dongxi/dongxi"
)

func TestRunAreasActive(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
		makeArea("area-2", "Personal"),
	})

	old := flagAreasFilter
	flagAreasFilter = "active"
	defer func() { flagAreasFilter = old }()

	oldProj := flagAreasProjects
	flagAreasProjects = false
	defer func() { flagAreasProjects = oldProj }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runAreas(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Work")) {
		t.Error("expected Work in output")
	}
	if !bytes.Contains([]byte(output), []byte("2 area(s)")) {
		t.Error("expected '2 area(s)' count")
	}
}

func TestRunAreasTrash(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
		{
			"area-2": map[string]any{
				dongxi.CommitKeyType:   float64(dongxi.ItemTypeCreate),
				dongxi.CommitKeyEntity: string(dongxi.EntityArea),
				dongxi.CommitKeyPayload: map[string]any{
					dongxi.FieldTitle:   "Trashed Area",
					dongxi.FieldTrashed: true,
				},
			},
		},
	})

	old := flagAreasFilter
	flagAreasFilter = "trash"
	defer func() { flagAreasFilter = old }()

	oldProj := flagAreasProjects
	flagAreasProjects = false
	defer func() { flagAreasProjects = oldProj }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runAreas(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Trashed Area")) {
		t.Error("expected 'Trashed Area' in trash output")
	}
	if bytes.Contains([]byte(output), []byte("Work")) {
		t.Error("should not show active areas in trash filter")
	}
}

func TestRunAreasAll(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
		{
			"area-2": map[string]any{
				dongxi.CommitKeyType:   float64(dongxi.ItemTypeCreate),
				dongxi.CommitKeyEntity: string(dongxi.EntityArea),
				dongxi.CommitKeyPayload: map[string]any{
					dongxi.FieldTitle:   "Trashed Area",
					dongxi.FieldTrashed: true,
				},
			},
		},
	})

	old := flagAreasFilter
	flagAreasFilter = "all"
	defer func() { flagAreasFilter = old }()

	oldProj := flagAreasProjects
	flagAreasProjects = false
	defer func() { flagAreasProjects = oldProj }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runAreas(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("2 area(s)")) {
		t.Error("expected 2 areas in 'all' filter")
	}
}

func TestRunAreasBadFilter(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
	})

	old := flagAreasFilter
	flagAreasFilter = "bad"
	defer func() { flagAreasFilter = old }()

	err := runAreas(nil, nil)
	if err == nil {
		t.Fatal("expected error for bad filter")
	}
}

func TestRunAreasEmpty(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagAreasFilter
	flagAreasFilter = "active"
	defer func() { flagAreasFilter = old }()

	oldProj := flagAreasProjects
	flagAreasProjects = false
	defer func() { flagAreasProjects = oldProj }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runAreas(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("(no areas)")) {
		t.Error("expected '(no areas)' message")
	}
}

func TestRunAreasWithProjects(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
		makeProject("proj-1", "Project A", func(p map[string]any) {
			p[dongxi.FieldAreaIDs] = []any{"area-1"}
		}),
	})

	old := flagAreasFilter
	flagAreasFilter = "active"
	defer func() { flagAreasFilter = old }()

	oldProj := flagAreasProjects
	flagAreasProjects = true
	defer func() { flagAreasProjects = oldProj }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runAreas(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Project A")) {
		t.Error("expected 'Project A' in output when --projects is set")
	}
}

func TestRunAreasJSON(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
	})

	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	oldFilter := flagAreasFilter
	flagAreasFilter = "active"
	defer func() { flagAreasFilter = oldFilter }()

	oldProj := flagAreasProjects
	flagAreasProjects = false
	defer func() { flagAreasProjects = oldProj }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runAreas(nil, nil)

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

func TestRunAreasUntitledArea(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", ""),
	})

	old := flagAreasFilter
	flagAreasFilter = "active"
	defer func() { flagAreasFilter = old }()

	oldProj := flagAreasProjects
	flagAreasProjects = false
	defer func() { flagAreasProjects = oldProj }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runAreas(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("(untitled)")) {
		t.Error("expected (untitled) for area with empty title")
	}
}

func TestRunAreasWithProjectsProgress(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
		makeProject("proj-1", "Project A", func(p map[string]any) {
			p[dongxi.FieldAreaIDs] = []any{"area-1"}
		}),
		makeTask("task-1", "Task 1", func(p map[string]any) {
			p[dongxi.FieldProjectIDs] = []any{"proj-1"}
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
		makeTask("task-2", "Task 2", func(p map[string]any) {
			p[dongxi.FieldProjectIDs] = []any{"proj-1"}
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
	})

	old := flagAreasFilter
	flagAreasFilter = "active"
	defer func() { flagAreasFilter = old }()

	oldProj := flagAreasProjects
	flagAreasProjects = true
	defer func() { flagAreasProjects = oldProj }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runAreas(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("(1/2)")) {
		t.Error("expected progress (1/2) in project listing under area")
	}
}

func TestRunAreasWithUntitledProject(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
		makeProject("proj-1", "", func(p map[string]any) {
			p[dongxi.FieldAreaIDs] = []any{"area-1"}
		}),
	})

	old := flagAreasFilter
	flagAreasFilter = "active"
	defer func() { flagAreasFilter = old }()

	oldProj := flagAreasProjects
	flagAreasProjects = true
	defer func() { flagAreasProjects = oldProj }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runAreas(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("(untitled)")) {
		t.Error("expected (untitled) for project with empty title under area")
	}
}

func TestRunAreasWithProjectsExcludesTrashed(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
		makeProject("proj-1", "Trashed Project", func(p map[string]any) {
			p[dongxi.FieldAreaIDs] = []any{"area-1"}
			p[dongxi.FieldTrashed] = true
		}),
	})

	old := flagAreasFilter
	flagAreasFilter = "active"
	defer func() { flagAreasFilter = old }()

	oldProj := flagAreasProjects
	flagAreasProjects = true
	defer func() { flagAreasProjects = oldProj }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runAreas(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if bytes.Contains(buf.Bytes(), []byte("Trashed Project")) {
		t.Error("should not show trashed projects under area")
	}
}

func TestRunAreasWithProjectsExcludesCompleted(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
		makeProject("proj-1", "Done Project", func(p map[string]any) {
			p[dongxi.FieldAreaIDs] = []any{"area-1"}
			p[dongxi.FieldStatus] = float64(dongxi.TaskStatusCompleted)
		}),
	})

	old := flagAreasFilter
	flagAreasFilter = "active"
	defer func() { flagAreasFilter = old }()

	oldProj := flagAreasProjects
	flagAreasProjects = true
	defer func() { flagAreasProjects = oldProj }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runAreas(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if bytes.Contains(buf.Bytes(), []byte("Done Project")) {
		t.Error("should not show completed projects under area")
	}
}

func TestRunAreasWithProjectsExcludesTasks(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
		makeTask("task-1", "Just a task", func(p map[string]any) {
			p[dongxi.FieldAreaIDs] = []any{"area-1"}
			p[dongxi.FieldDestination] = float64(dongxi.TaskDestinationAnytime)
		}),
	})

	old := flagAreasFilter
	flagAreasFilter = "active"
	defer func() { flagAreasFilter = old }()

	oldProj := flagAreasProjects
	flagAreasProjects = true
	defer func() { flagAreasProjects = oldProj }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runAreas(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if bytes.Contains(buf.Bytes(), []byte("Just a task")) {
		t.Error("should not show regular tasks under area projects listing")
	}
}
