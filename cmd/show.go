package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <uuid>",
	Short: "Show details of a task, project, or area",
	Args:  cobra.ExactArgs(1),
	RunE:  runShow,
}

func runShow(cmd *cobra.Command, args []string) error {
	s, _, _, err := loadState()
	if err != nil {
		return err
	}

	item, err := s.resolveUUID(args[0])
	if err != nil {
		return err
	}

	if flagJSON {
		out := s.itemToOutput(item)
		if item.entity == string(dongxi.EntityTask) {
			if toInt(item.fields[dongxi.FieldType]) == int(dongxi.TaskTypeProject) {
				total, completed := s.projectProgress(item.uuid)
				out.TasksTotal = &total
				out.TasksCompleted = &completed
			}
			cis := s.checklistForTask(item.uuid)
			for _, ci := range cis {
				out.Checklist = append(out.Checklist, s.itemToOutput(&ci))
			}
		}
		return printJSON(out)
	}

	f := item.fields
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintf(w, "UUID:\t%s\n", item.uuid)
	fmt.Fprintf(w, "Type:\t%s\n", item.entity)

	title := toStr(f[dongxi.FieldTitle])
	if title == "" {
		title = "(untitled)"
	}
	fmt.Fprintf(w, "Title:\t%s\n", title)

	if item.entity == string(dongxi.EntityTask) {
		switch dongxi.TaskStatus(toInt(f[dongxi.FieldStatus])) {
		case dongxi.TaskStatusOpen:
			fmt.Fprintf(w, "Status:\tOpen\n")
		case dongxi.TaskStatusCancelled:
			fmt.Fprintf(w, "Status:\tCancelled\n")
		case dongxi.TaskStatusCompleted:
			fmt.Fprintf(w, "Status:\tCompleted\n")
		}

		switch dongxi.TaskType(toInt(f[dongxi.FieldType])) {
		case dongxi.TaskTypeTask:
			fmt.Fprintf(w, "Kind:\tTask\n")
		case dongxi.TaskTypeProject:
			fmt.Fprintf(w, "Kind:\tProject\n")
		case dongxi.TaskTypeHeading:
			fmt.Fprintf(w, "Kind:\tHeading\n")
		}

		switch dongxi.TaskDestination(toInt(f[dongxi.FieldDestination])) {
		case dongxi.TaskDestinationInbox:
			fmt.Fprintf(w, "Destination:\tInbox\n")
		case dongxi.TaskDestinationAnytime:
			fmt.Fprintf(w, "Destination:\tToday\n")
		case dongxi.TaskDestinationSomeday:
			fmt.Fprintf(w, "Destination:\tSomeday\n")
		}

		if toBool(f[dongxi.FieldTrashed]) {
			fmt.Fprintf(w, "Trashed:\tYes\n")
		}

		if areaUUID := firstString(f[dongxi.FieldAreaIDs]); areaUUID != "" {
			areaName := s.areaTitle(areaUUID)
			if areaName != "" {
				fmt.Fprintf(w, "Area:\t%s\n", areaName)
			} else {
				fmt.Fprintf(w, "Area:\t%s\n", areaUUID)
			}
		}

		if projUUID := firstString(f[dongxi.FieldProjectIDs]); projUUID != "" {
			projName := s.projectTitle(projUUID)
			if projName != "" {
				fmt.Fprintf(w, "Project:\t%s\n", projName)
			} else {
				fmt.Fprintf(w, "Project:\t%s\n", projUUID)
			}
		}

		if cd := toFloat(f[dongxi.FieldCreationDate]); cd > 0 {
			fmt.Fprintf(w, "Created:\t%s\n", formatTS(cd))
		}
		if md := toFloat(f[dongxi.FieldModificationDate]); md > 0 {
			fmt.Fprintf(w, "Modified:\t%s\n", formatTS(md))
		}
		if sr := toFloat(f[dongxi.FieldScheduledDate]); sr > 0 {
			fmt.Fprintf(w, "Scheduled:\t%s\n", time.Unix(int64(sr), 0).UTC().Format("2006-01-02"))
		}
		if dd := toFloat(f[dongxi.FieldDeadline]); dd > 0 {
			fmt.Fprintf(w, "Deadline:\t%s\n", time.Unix(int64(dd), 0).UTC().Format("2006-01-02"))
		}

		if v := dongxi.NoteText(f[dongxi.FieldNote]); v != "" {
			fmt.Fprintf(w, "Notes:\t%s\n", v)
		}

		// Tags.
		if tg, ok := f[dongxi.FieldTagIDs].([]any); ok && len(tg) > 0 {
			for _, t := range tg {
				if tagUUID, ok := t.(string); ok {
					if tag, ok := s.byUUID[tagUUID]; ok {
						fmt.Fprintf(w, "Tag:\t%s\n", toStr(tag.fields[dongxi.FieldTitle]))
					} else {
						fmt.Fprintf(w, "Tag:\t%s\n", tagUUID)
					}
				}
			}
		}

		if toInt(f[dongxi.FieldStartBucket]) == 1 {
			fmt.Fprintf(w, "Evening:\tYes\n")
		}

		// Project progress.
		if toInt(f[dongxi.FieldType]) == int(dongxi.TaskTypeProject) {
			total, completed := s.projectProgress(item.uuid)
			if total > 0 {
				fmt.Fprintf(w, "Progress:\t%d/%d tasks\n", completed, total)
			}
		}

		// Checklist items.
		checklistItems := s.checklistForTask(item.uuid)
		if len(checklistItems) > 0 {
			fmt.Fprintln(w, "")
			fmt.Fprintln(w, "Checklist:")
			for _, ci := range checklistItems {
				status := "[ ]"
				if dongxi.TaskStatus(toInt(ci.fields[dongxi.FieldStatus])) == dongxi.TaskStatusCompleted {
					status = "[x]"
				}
				fmt.Fprintf(w, "  %s %s\n", status, toStr(ci.fields[dongxi.FieldTitle]))
			}
		}
	}

	return w.Flush()
}

func (s *thingsState) checklistForTask(taskUUID string) []replayedItem {
	var result []replayedItem
	for _, item := range s.items {
		if item.entity != string(dongxi.EntityChecklistItem) {
			continue
		}
		if firstString(item.fields[dongxi.FieldTaskIDs]) == taskUUID {
			result = append(result, item)
		}
	}
	return result
}

func toFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	}
	return 0
}

func formatTS(f float64) string {
	sec := int64(f)
	return time.Unix(sec, 0).Local().Format("2006-01-02 15:04")
}
