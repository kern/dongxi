package cmd

import (
	"fmt"
	"sort"
	"time"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

var flagLogbookLimit int

var logbookCmd = &cobra.Command{
	Use:   "logbook",
	Short: "Show completed tasks",
	RunE:  runLogbook,
}

func init() {
	logbookCmd.Flags().IntVarP(&flagLogbookLimit, "limit", "n", 20, "Number of tasks to show")
}

func runLogbook(cmd *cobra.Command, args []string) error {
	s, _, _, err := loadState()
	if err != nil {
		return err
	}

	type completedTask struct {
		title       string
		uuid        string
		completedAt float64
	}

	var tasks []completedTask
	for _, item := range s.items {
		if item.entity != string(dongxi.EntityTask) {
			continue
		}
		if toInt(item.fields[dongxi.FieldType]) != int(dongxi.TaskTypeTask) {
			continue
		}
		if toInt(item.fields[dongxi.FieldStatus]) != int(dongxi.TaskStatusCompleted) {
			continue
		}
		if toBool(item.fields[dongxi.FieldTrashed]) {
			continue
		}

		title := toStr(item.fields[dongxi.FieldTitle])
		if title == "" {
			title = "(untitled)"
		}
		sp := toFloat(item.fields[dongxi.FieldStopDate])
		tasks = append(tasks, completedTask{title: title, uuid: item.uuid, completedAt: sp})
	}

	// Sort by completion date, most recent first.
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].completedAt > tasks[j].completedAt
	})

	if flagLogbookLimit > 0 && len(tasks) > flagLogbookLimit {
		tasks = tasks[:flagLogbookLimit]
	}

	if flagJSON {
		var out []ItemOutput
		for _, t := range tasks {
			item := s.byUUID[t.uuid]
			if item != nil {
				out = append(out, s.itemToOutput(item))
			}
		}
		return printJSON(out)
	}

	for _, t := range tasks {
		date := ""
		if t.completedAt > 0 {
			date = time.Unix(int64(t.completedAt), 0).Local().Format("2006-01-02")
		}
		fmt.Printf("  %s  %s  [%s]\n", date, t.title, t.uuid)
	}

	if len(tasks) == 0 {
		fmt.Println("  (no completed tasks)")
	} else {
		fmt.Printf("\n%d task(s)\n", len(tasks))
	}
	return nil
}
