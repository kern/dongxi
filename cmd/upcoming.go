package cmd

import (
	"fmt"
	"sort"
	"time"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

var upcomingCmd = &cobra.Command{
	Use:   "upcoming",
	Short: "Show tasks with a scheduled date, grouped by date",
	RunE:  runUpcoming,
}

func runUpcoming(cmd *cobra.Command, args []string) error {
	s, _, _, err := loadState()
	if err != nil {
		return err
	}

	type scheduledTask struct {
		item      *replayedItem
		title     string
		uuid      string
		scheduled float64
		deadline  float64
		sortDate  float64
	}

	var tasks []scheduledTask
	for i := range s.items {
		item := &s.items[i]
		if item.entity != string(dongxi.EntityTask) {
			continue
		}
		if toInt(item.fields[dongxi.FieldType]) != int(dongxi.TaskTypeTask) {
			continue
		}
		if toInt(item.fields[dongxi.FieldStatus]) != int(dongxi.TaskStatusOpen) {
			continue
		}
		if toBool(item.fields[dongxi.FieldTrashed]) {
			continue
		}

		sr := toFloat(item.fields[dongxi.FieldScheduledDate])
		dd := toFloat(item.fields[dongxi.FieldDeadline])
		if sr == 0 && dd == 0 {
			continue
		}

		title := toStr(item.fields[dongxi.FieldTitle])
		if title == "" {
			title = "(untitled)"
		}
		sortDate := sr
		if sortDate == 0 {
			sortDate = dd
		}
		tasks = append(tasks, scheduledTask{item: item, title: title, uuid: item.uuid, scheduled: sr, deadline: dd, sortDate: sortDate})
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].sortDate < tasks[j].sortDate
	})

	if flagJSON {
		var out []ItemOutput
		for _, t := range tasks {
			out = append(out, s.itemToOutput(t.item))
		}
		return printJSON(out)
	}

	// Group by date for human-readable output.
	lastDate := ""
	for _, t := range tasks {
		dateStr := time.Unix(int64(t.sortDate), 0).UTC().Format("2006-01-02")
		if dateStr != lastDate {
			if lastDate != "" {
				fmt.Println()
			}
			// Format with weekday for readability.
			dt := time.Unix(int64(t.sortDate), 0).UTC()
			fmt.Printf("%s (%s)\n", dateStr, dt.Weekday().String()[:3])
			lastDate = dateStr
		}
		dl := ""
		if t.deadline > 0 {
			dl = fmt.Sprintf(" (deadline: %s)", time.Unix(int64(t.deadline), 0).UTC().Format("2006-01-02"))
		}
		fmt.Printf("  %s%s  [%s]\n", t.title, dl, t.uuid)
	}

	if len(tasks) == 0 {
		fmt.Println("  (no upcoming tasks)")
	} else {
		fmt.Printf("\n%d task(s)\n", len(tasks))
	}
	return nil
}
