package cmd

import (
	"fmt"
	"time"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

// nowFunc is the time source; tests can override it.
var nowFunc = time.Now

var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show a comprehensive overview of your Things database",
	Long: `Display a full summary of areas, projects, tags, and task counts.

Examples:
  dongxi summary         # Human-readable summary
  dongxi summary --json  # JSON summary for AI consumption`,
	RunE: runSummary,
}

// --- JSON output types ---

type summaryOverview struct {
	TotalTasks        int `json:"total_tasks"`
	OpenTasks         int `json:"open_tasks"`
	CompletedTasks    int `json:"completed_tasks"`
	CancelledTasks    int `json:"cancelled_tasks"`
	InboxCount        int `json:"inbox_count"`
	TodayCount        int `json:"today_count"`
	EveningCount      int `json:"evening_count"`
	SomedayCount      int `json:"someday_count"`
	UpcomingCount     int `json:"upcoming_count"`
	OverdueCount      int `json:"overdue_count"`
	TotalProjects     int `json:"total_projects"`
	OpenProjects      int `json:"open_projects"`
	CompletedProjects int `json:"completed_projects"`
	TotalAreas        int `json:"total_areas"`
	TotalTags         int `json:"total_tags"`
}

type summaryProject struct {
	UUID           string   `json:"uuid"`
	Title          string   `json:"title"`
	Status         string   `json:"status"`
	TasksTotal     int      `json:"tasks_total"`
	TasksCompleted int      `json:"tasks_completed"`
	TasksOpen      int      `json:"tasks_open"`
	HasNotes       bool     `json:"has_notes"`
	Headings       []string `json:"headings,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	Scheduled      string   `json:"scheduled,omitempty"`
	Deadline       string   `json:"deadline,omitempty"`
}

type summaryArea struct {
	UUID          string           `json:"uuid"`
	Title         string           `json:"title"`
	ProjectCount  int              `json:"project_count"`
	OpenTaskCount int              `json:"open_task_count"`
	Projects      []summaryProject `json:"projects"`
}

type summaryTag struct {
	UUID      string `json:"uuid"`
	Title     string `json:"title"`
	TaskCount int    `json:"task_count"`
}

type summaryInboxItem struct {
	UUID    string `json:"uuid"`
	Title   string `json:"title"`
	Created string `json:"created,omitempty"`
}

type summaryTodayItem struct {
	UUID    string   `json:"uuid"`
	Title   string   `json:"title"`
	Project string   `json:"project,omitempty"`
	Evening bool     `json:"evening"`
	Tags    []string `json:"tags,omitempty"`
}

type summaryOutput struct {
	Overview             summaryOverview    `json:"overview"`
	Areas                []summaryArea      `json:"areas"`
	UnassignedProjects   []summaryProject   `json:"unassigned_projects"`
	Tags                 []summaryTag       `json:"tags"`
	Inbox                []summaryInboxItem `json:"inbox"`
	Today                []summaryTodayItem `json:"today"`
}

func runSummary(cmd *cobra.Command, args []string) error {
	s, _, _, err := loadState()
	if err != nil {
		return err
	}

	now := nowFunc()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// Collect all non-trashed tasks (type=task).
	var overview summaryOverview
	var inboxItems []summaryInboxItem
	var todayItems []summaryTodayItem

	// Tag usage: tagUUID -> count of open tasks
	tagCounts := map[string]int{}

	for i := range s.items {
		item := &s.items[i]
		if item.entity != string(dongxi.EntityTask) {
			continue
		}
		if toInt(item.fields[dongxi.FieldType]) != int(dongxi.TaskTypeTask) {
			continue
		}
		if toBool(item.fields[dongxi.FieldTrashed]) || s.isOrphanedByTrashedParent(item) {
			continue
		}

		overview.TotalTasks++
		status := dongxi.TaskStatus(toInt(item.fields[dongxi.FieldStatus]))
		dest := dongxi.TaskDestination(toInt(item.fields[dongxi.FieldDestination]))

		switch status {
		case dongxi.TaskStatusOpen:
			overview.OpenTasks++
		case dongxi.TaskStatusCompleted:
			overview.CompletedTasks++
		case dongxi.TaskStatusCancelled:
			overview.CancelledTasks++
		}

		if status == dongxi.TaskStatusOpen {
			switch dest {
			case dongxi.TaskDestinationInbox:
				overview.InboxCount++
				title := toStr(item.fields[dongxi.FieldTitle])
				if title == "" {
					title = "(untitled)"
				}
				inboxItem := summaryInboxItem{
					UUID:  item.uuid,
					Title: title,
				}
				if cd := toFloat(item.fields[dongxi.FieldCreationDate]); cd > 0 {
					inboxItem.Created = time.Unix(int64(cd), 0).UTC().Format(time.RFC3339)
				}
				inboxItems = append(inboxItems, inboxItem)
			case dongxi.TaskDestinationAnytime:
				if !isToday(item.fields, now) {
					continue
				}
				overview.TodayCount++
				evening := toInt(item.fields[dongxi.FieldStartBucket]) == 1
				if evening {
					overview.EveningCount++
				}
				title := toStr(item.fields[dongxi.FieldTitle])
				if title == "" {
					title = "(untitled)"
				}
				todayItem := summaryTodayItem{
					UUID:    item.uuid,
					Title:   title,
					Evening: evening,
				}
				if projUUID := firstString(item.fields[dongxi.FieldProjectIDs]); projUUID != "" {
					todayItem.Project = s.projectTitle(projUUID)
				}
				if tg := toStringSlice(item.fields[dongxi.FieldTagIDs]); len(tg) > 0 {
					var tagNames []string
					for _, tagID := range tg {
						if tag, ok := s.byUUID[tagID]; ok {
							tagNames = append(tagNames, toStr(tag.fields[dongxi.FieldTitle]))
						}
					}
					todayItem.Tags = tagNames
				}
				todayItems = append(todayItems, todayItem)
			case dongxi.TaskDestinationSomeday:
				overview.SomedayCount++
			}

			// Upcoming: has scheduled date in the future.
			if sr := toFloat(item.fields[dongxi.FieldScheduledDate]); sr > 0 {
				scheduledTime := time.Unix(int64(sr), 0).UTC()
				if scheduledTime.After(todayStart) {
					overview.UpcomingCount++
				}
			}

			// Overdue: has deadline before today and still open.
			if dd := toFloat(item.fields[dongxi.FieldDeadline]); dd > 0 {
				deadlineTime := time.Unix(int64(dd), 0).UTC()
				if deadlineTime.Before(todayStart) {
					overview.OverdueCount++
				}
			}

			// Count tags.
			for _, tagID := range toStringSlice(item.fields[dongxi.FieldTagIDs]) {
				tagCounts[tagID]++
			}
		}
	}

	// Count areas.
	var areaOrder []string
	areaMap := map[string]*replayedItem{}
	for i := range s.items {
		item := &s.items[i]
		if item.entity != string(dongxi.EntityArea) {
			continue
		}
		if toBool(item.fields[dongxi.FieldTrashed]) {
			continue
		}
		overview.TotalAreas++
		areaOrder = append(areaOrder, item.uuid)
		areaMap[item.uuid] = item
	}

	// Collect projects grouped by area.
	type projectInfo struct {
		item    *replayedItem
		areaID  string
		project summaryProject
	}
	var allProjects []projectInfo
	for i := range s.items {
		item := &s.items[i]
		if item.entity != string(dongxi.EntityTask) {
			continue
		}
		if toInt(item.fields[dongxi.FieldType]) != int(dongxi.TaskTypeProject) {
			continue
		}
		if toBool(item.fields[dongxi.FieldTrashed]) {
			continue
		}

		overview.TotalProjects++
		status := dongxi.TaskStatus(toInt(item.fields[dongxi.FieldStatus]))
		if status == dongxi.TaskStatusCompleted {
			overview.CompletedProjects++
			continue
		}
		if status != dongxi.TaskStatusOpen {
			continue
		}
		overview.OpenProjects++

		// Skip someday projects.
		if dongxi.TaskDestination(toInt(item.fields[dongxi.FieldDestination])) == dongxi.TaskDestinationSomeday {
			continue
		}

		total, completed := s.projectProgress(item.uuid)
		title := toStr(item.fields[dongxi.FieldTitle])
		if title == "" {
			title = "(untitled)"
		}

		sp := summaryProject{
			UUID:           item.uuid,
			Title:          title,
			Status:         "open",
			TasksTotal:     total,
			TasksCompleted: completed,
			TasksOpen:      total - completed,
			HasNotes:       dongxi.NoteText(item.fields[dongxi.FieldNote]) != "",
		}

		// Headings.
		headings := s.headingsForProject(item.uuid)
		for _, h := range headings {
			sp.Headings = append(sp.Headings, toStr(h.fields[dongxi.FieldTitle]))
		}

		// Tags.
		if tg := toStringSlice(item.fields[dongxi.FieldTagIDs]); len(tg) > 0 {
			var tagNames []string
			for _, tagID := range tg {
				if tag, ok := s.byUUID[tagID]; ok {
					tagNames = append(tagNames, toStr(tag.fields[dongxi.FieldTitle]))
				}
			}
			sp.Tags = tagNames
		}

		if sr := toFloat(item.fields[dongxi.FieldScheduledDate]); sr > 0 {
			sp.Scheduled = time.Unix(int64(sr), 0).UTC().Format("2006-01-02")
		}
		if dd := toFloat(item.fields[dongxi.FieldDeadline]); dd > 0 {
			sp.Deadline = time.Unix(int64(dd), 0).UTC().Format("2006-01-02")
		}

		areaID := firstString(item.fields[dongxi.FieldAreaIDs])
		allProjects = append(allProjects, projectInfo{item: item, areaID: areaID, project: sp})
	}

	// Count tags.
	var tags []summaryTag
	for i := range s.items {
		item := &s.items[i]
		if item.entity != string(dongxi.EntityTag) {
			continue
		}
		overview.TotalTags++
		title := toStr(item.fields[dongxi.FieldTitle])
		if title == "" {
			title = "(untitled)"
		}
		tags = append(tags, summaryTag{
			UUID:      item.uuid,
			Title:     title,
			TaskCount: tagCounts[item.uuid],
		})
	}

	// Build areas output with projects and open task counts.
	// Count open tasks per area.
	areaOpenTasks := map[string]int{}
	for i := range s.items {
		item := &s.items[i]
		if item.entity != string(dongxi.EntityTask) {
			continue
		}
		if toInt(item.fields[dongxi.FieldType]) != int(dongxi.TaskTypeTask) {
			continue
		}
		if toBool(item.fields[dongxi.FieldTrashed]) || s.isOrphanedByTrashedParent(item) {
			continue
		}
		if dongxi.TaskStatus(toInt(item.fields[dongxi.FieldStatus])) != dongxi.TaskStatusOpen {
			continue
		}
		areaID := firstString(item.fields[dongxi.FieldAreaIDs])
		if areaID == "" {
			// Inherit from project.
			if projUUID := firstString(item.fields[dongxi.FieldProjectIDs]); projUUID != "" {
				if proj, ok := s.projects[projUUID]; ok {
					areaID = firstString(proj.fields[dongxi.FieldAreaIDs])
				}
			}
		}
		if areaID != "" {
			areaOpenTasks[areaID]++
		}
	}

	var areas []summaryArea
	for _, areaUUID := range areaOrder {
		aItem := areaMap[areaUUID]
		title := toStr(aItem.fields[dongxi.FieldTitle])
		if title == "" {
			title = "(untitled)"
		}
		sa := summaryArea{
			UUID:          areaUUID,
			Title:         title,
			OpenTaskCount: areaOpenTasks[areaUUID],
		}
		for _, pi := range allProjects {
			if pi.areaID == areaUUID {
				sa.Projects = append(sa.Projects, pi.project)
				sa.ProjectCount++
			}
		}
		areas = append(areas, sa)
	}

	// Unassigned projects.
	var unassigned []summaryProject
	for _, pi := range allProjects {
		if pi.areaID == "" {
			unassigned = append(unassigned, pi.project)
		}
	}

	out := summaryOutput{
		Overview:           overview,
		Areas:              areas,
		UnassignedProjects: unassigned,
		Tags:               tags,
		Inbox:              inboxItems,
		Today:              todayItems,
	}

	if flagJSON {
		return printJSON(out)
	}

	// Human-readable output.
	printSummaryHuman(out)
	return nil
}

func printSummaryHuman(out summaryOutput) {
	o := out.Overview

	fmt.Println("THINGS SUMMARY")
	fmt.Println("==============")
	fmt.Println()
	fmt.Println("Overview:")
	fmt.Printf("  Tasks: %d open, %d completed, %d cancelled (%d total)\n",
		o.OpenTasks, o.CompletedTasks, o.CancelledTasks, o.TotalTasks)
	fmt.Printf("  Inbox: %d | Today: %d (%d evening) | Someday: %d\n",
		o.InboxCount, o.TodayCount, o.EveningCount, o.SomedayCount)
	fmt.Printf("  Upcoming: %d | Overdue: %d\n", o.UpcomingCount, o.OverdueCount)
	fmt.Printf("  Projects: %d open, %d completed (%d total)\n",
		o.OpenProjects, o.CompletedProjects, o.TotalProjects)
	fmt.Printf("  Areas: %d | Tags: %d\n", o.TotalAreas, o.TotalTags)

	if len(out.Areas) > 0 || len(out.UnassignedProjects) > 0 {
		fmt.Println()
		fmt.Println("Areas & Projects:")
		for _, a := range out.Areas {
			fmt.Printf("  %s\n", a.Title)
			for _, p := range a.Projects {
				fmt.Printf("    - %s (%d/%d tasks)\n", p.Title, p.TasksCompleted, p.TasksTotal)
			}
		}
		if len(out.UnassignedProjects) > 0 {
			fmt.Println()
			fmt.Println("  (no area)")
			for _, p := range out.UnassignedProjects {
				fmt.Printf("    - %s (%d/%d tasks)\n", p.Title, p.TasksCompleted, p.TasksTotal)
			}
		}
	}

	if len(out.Tags) > 0 {
		fmt.Println()
		fmt.Print("Tags:\n  ")
		for i, t := range out.Tags {
			if i > 0 {
				fmt.Print(" | ")
			}
			fmt.Printf("%s (%d tasks)", t.Title, t.TaskCount)
		}
		fmt.Println()
	}

	if len(out.Inbox) > 0 {
		fmt.Println()
		fmt.Printf("Inbox (%d):\n", len(out.Inbox))
		for _, item := range out.Inbox {
			fmt.Printf("  %s\n", item.Title)
		}
	}

	if len(out.Today) > 0 {
		fmt.Println()
		fmt.Printf("Today (%d):\n", len(out.Today))

		// Split morning/evening.
		var morning, evening []summaryTodayItem
		for _, item := range out.Today {
			if item.Evening {
				evening = append(evening, item)
			} else {
				morning = append(morning, item)
			}
		}

		if len(morning) > 0 {
			fmt.Println("  Morning:")
			for _, item := range morning {
				fmt.Printf("    %s\n", item.Title)
			}
		}
		if len(evening) > 0 {
			fmt.Println("  Evening:")
			for _, item := range evening {
				fmt.Printf("    %s\n", item.Title)
			}
		}
	}
}

