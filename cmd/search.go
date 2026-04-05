package cmd

import (
	"fmt"
	"strings"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

var flagSearchAll bool

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search tasks, projects, and checklist items by title or notes",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runSearch,
}

func init() {
	searchCmd.Flags().BoolVar(&flagSearchAll, "all", false, "Include completed and trashed items")
}

func runSearch(cmd *cobra.Command, args []string) error {
	s, _, _, err := loadState()
	if err != nil {
		return err
	}

	query := strings.ToLower(strings.Join(args, " "))

	var filtered []replayedItem
	for _, item := range s.items {
		// Search tasks, projects, and checklist items.
		switch item.entity {
		case string(dongxi.EntityTask):
			// Include tasks and projects.
		case string(dongxi.EntityChecklistItem):
			// Include checklist items.
		default:
			continue
		}

		if !flagSearchAll {
			if toInt(item.fields[dongxi.FieldStatus]) != int(dongxi.TaskStatusOpen) {
				continue
			}
			if toBool(item.fields[dongxi.FieldTrashed]) {
				continue
			}
			// Also hide items whose parent project is trashed.
			if item.entity == string(dongxi.EntityTask) && s.isOrphanedByTrashedParent(&item) {
				continue
			}
		}

		title := toStr(item.fields[dongxi.FieldTitle])
		note := dongxi.NoteText(item.fields[dongxi.FieldNote])

		if !strings.Contains(strings.ToLower(title), query) &&
			!strings.Contains(strings.ToLower(note), query) {
			continue
		}

		filtered = append(filtered, item)
	}

	if flagJSON {
		var out []ItemOutput
		for _, item := range filtered {
			out = append(out, s.itemToOutput(&item))
		}
		return printJSON(out)
	}

	for _, item := range filtered {
		title := toStr(item.fields[dongxi.FieldTitle])
		if title == "" {
			title = "(untitled)"
		}
		status := ""
		if flagSearchAll {
			switch toInt(item.fields[dongxi.FieldStatus]) {
			case int(dongxi.TaskStatusCompleted):
				status = " [completed]"
			case int(dongxi.TaskStatusCancelled):
				status = " [cancelled]"
			}
			if toBool(item.fields[dongxi.FieldTrashed]) {
				status = " [trashed]"
			}
		}
		typePrefix := ""
		if item.entity == string(dongxi.EntityChecklistItem) {
			typePrefix = "(checklist) "
		} else if toInt(item.fields[dongxi.FieldType]) == int(dongxi.TaskTypeProject) {
			typePrefix = "(project) "
		}
		fmt.Printf("  %s%s%s  [%s]\n", typePrefix, title, status, item.uuid)
	}

	if len(filtered) == 0 {
		fmt.Println("  (no results)")
	} else {
		fmt.Printf("\n%d result(s)\n", len(filtered))
	}
	return nil
}
