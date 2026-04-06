package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

var (
	flagListFilter  string
	flagListProject string
	flagListArea    string
	flagListTag     string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks from Things Cloud",
	Long: `List tasks by replaying the Things Cloud history.

Examples:
  dongxi list                          # inbox tasks (default)
  dongxi list -f today                 # today tasks
  dongxi list -f evening               # this evening tasks
  dongxi list -f all                   # all open tasks
  dongxi list -f all --area <uuid>     # all tasks in an area
  dongxi list -f all --project <uuid>  # all tasks in a project`,
	RunE: runList,
}

func init() {
	listCmd.Flags().StringVarP(&flagListFilter, "filter", "f", "inbox", "Filter: inbox, today, evening, someday, completed, trash, all")
	listCmd.Flags().StringVar(&flagListProject, "project", "", "Filter by project UUID")
	listCmd.Flags().StringVar(&flagListArea, "area", "", "Filter by area UUID")
	listCmd.Flags().StringVar(&flagListTag, "tag", "", "Filter by tag UUID")
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	s, _, _, err := loadState()
	if err != nil {
		return err
	}

	// Parse filter.
	var destFilter *int
	showCompleted := false
	showTrashed := false
	showEvening := false
	showToday := false
	switch strings.ToLower(flagListFilter) {
	case "inbox":
		d := int(dongxi.TaskDestinationInbox)
		destFilter = &d
	case "today":
		d := int(dongxi.TaskDestinationAnytime)
		destFilter = &d
		showToday = true
	case "anytime":
		d := int(dongxi.TaskDestinationAnytime)
		destFilter = &d
	case "evening":
		d := int(dongxi.TaskDestinationAnytime)
		destFilter = &d
		showEvening = true
		showToday = true
	case "someday":
		d := int(dongxi.TaskDestinationSomeday)
		destFilter = &d
	case "completed":
		showCompleted = true
	case "trash":
		showTrashed = true
	case "all":
		// no filter
	default:
		return fmt.Errorf("unknown filter %q: must be inbox, today, evening, someday, completed, trash, or all", flagListFilter)
	}

	// Resolve optional project/area/tag filters.
	var projectUUID, areaUUID, tagUUID string
	if flagListProject != "" {
		proj, err := s.resolveUUID(flagListProject)
		if err != nil {
			return fmt.Errorf("resolve project: %w", err)
		}
		projectUUID = proj.uuid
	}
	if flagListArea != "" {
		area, err := s.resolveUUID(flagListArea)
		if err != nil {
			return fmt.Errorf("resolve area: %w", err)
		}
		areaUUID = area.uuid
	}
	if flagListTag != "" {
		tag, err := s.resolveUUID(flagListTag)
		if err != nil {
			return fmt.Errorf("resolve tag: %w", err)
		}
		tagUUID = tag.uuid
	}

	// Filter.
	var filtered []replayedItem
	for _, t := range s.items {
		if t.entity != string(dongxi.EntityTask) {
			continue
		}
		if toInt(t.fields[dongxi.FieldType]) != int(dongxi.TaskTypeTask) {
			continue
		}

		trashed := toBool(t.fields[dongxi.FieldTrashed]) || s.isOrphanedByTrashedParent(&t)
		if showTrashed {
			if !trashed {
				continue
			}
		} else {
			if trashed {
				continue
			}
		}

		status := toInt(t.fields[dongxi.FieldStatus])
		if showCompleted {
			if status != int(dongxi.TaskStatusCompleted) {
				continue
			}
		} else if !showTrashed {
			if status != int(dongxi.TaskStatusOpen) {
				continue
			}
		}

		if destFilter != nil && toInt(t.fields[dongxi.FieldDestination]) != *destFilter {
			continue
		}
		if showToday && !isToday(t.fields, nowFunc()) {
			continue
		}
		if showEvening && toInt(t.fields[dongxi.FieldStartBucket]) != 1 {
			continue
		}
		if projectUUID != "" && firstString(t.fields[dongxi.FieldProjectIDs]) != projectUUID {
			continue
		}
		if areaUUID != "" {
			taskArea := firstString(t.fields[dongxi.FieldAreaIDs])
			// Inherit area from parent project if task has none.
			if taskArea == "" {
				if projID := firstString(t.fields[dongxi.FieldProjectIDs]); projID != "" {
					if proj, ok := s.projects[projID]; ok {
						taskArea = firstString(proj.fields[dongxi.FieldAreaIDs])
					}
				}
			}
			if taskArea != areaUUID {
				continue
			}
		}
		if tagUUID != "" && !hasString(t.fields[dongxi.FieldTagIDs], tagUUID) {
			continue
		}
		filtered = append(filtered, t)
	}

	sortByIndex(filtered)

	if flagJSON {
		var out []ItemOutput
		for _, t := range filtered {
			out = append(out, s.itemToOutput(&t))
		}
		return printJSON(out)
	}

	// Group by heading when listing tasks within a project.
	if projectUUID != "" && !flagJSON {
		headings := s.headingsForProject(projectUUID)
		headingOrder := []string{""} // empty string = no heading
		for _, h := range headings {
			headingOrder = append(headingOrder, h.uuid)
		}
		headingTitles := map[string]string{}
		for _, h := range headings {
			headingTitles[h.uuid] = toStr(h.fields[dongxi.FieldTitle])
		}

		grouped := map[string][]replayedItem{}
		for _, t := range filtered {
			hID := firstString(t.fields[dongxi.FieldActionGroupIDs])
			if _, ok := headingTitles[hID]; !ok {
				hID = "" // not under a known heading
			}
			grouped[hID] = append(grouped[hID], t)
		}

		first := true
		for _, hID := range headingOrder {
			tasks, ok := grouped[hID]
			if !ok || len(tasks) == 0 {
				continue
			}
			if hID != "" {
				if !first {
					fmt.Println()
				}
				fmt.Printf("  --- %s ---\n", headingTitles[hID])
			}
			for _, t := range tasks {
				title := toStr(t.fields[dongxi.FieldTitle])
				if title == "" {
					title = "(untitled)"
				}
				fmt.Printf("  %s  [%s]\n", title, t.uuid)
			}
			first = false
		}
	} else {
		for _, t := range filtered {
			title := toStr(t.fields[dongxi.FieldTitle])
			if title == "" {
				title = "(untitled)"
			}
			fmt.Printf("  %s  [%s]\n", title, t.uuid)
		}
	}

	if len(filtered) == 0 {
		fmt.Println("  (no tasks)")
	} else {
		fmt.Printf("\n%d task(s)\n", len(filtered))
	}

	return nil
}

func hasString(v any, target string) bool {
	arr, ok := v.([]any)
	if !ok {
		return false
	}
	for _, item := range arr {
		if s, ok := item.(string); ok && s == target {
			return true
		}
	}
	return false
}

type replayedItem struct {
	uuid   string
	entity string
	fields map[string]any
}

// replayHistory replays all commits to get the current state of all items.
func replayHistory(commits []map[string]any) []replayedItem {
	state := map[string]*replayedItem{}
	var order []string

	for _, commit := range commits {
		for uuid, rawVal := range commit {
			val, ok := rawVal.(map[string]any)
			if !ok {
				continue
			}
			switch dongxi.ItemType(toInt(val[dongxi.CommitKeyType])) {
			case dongxi.ItemTypeCreate:
				p, _ := val[dongxi.CommitKeyPayload].(map[string]any)
				entity, _ := val[dongxi.CommitKeyEntity].(string)
				state[uuid] = &replayedItem{
					uuid:   uuid,
					entity: entity,
					fields: copyMap(p),
				}
				order = append(order, uuid)
			case dongxi.ItemTypeModify:
				if item, ok := state[uuid]; ok {
					p, _ := val[dongxi.CommitKeyPayload].(map[string]any)
					for k, v := range p {
						item.fields[k] = v
					}
				}
			case dongxi.ItemTypeDelete:
				delete(state, uuid)
			}
		}
	}

	var result []replayedItem
	for _, uuid := range order {
		if item, ok := state[uuid]; ok {
			result = append(result, *item)
		}
	}
	return result
}

func copyMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func toInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	}
	return 0
}

func toBool(v any) bool {
	b, _ := v.(bool)
	return b
}

func toStr(v any) string {
	s, _ := v.(string)
	return s
}

// sortByIndex sorts replayed items by their ix field (ascending).
func sortByIndex(items []replayedItem) {
	sort.SliceStable(items, func(i, j int) bool {
		return toInt(items[i].fields[dongxi.FieldIndex]) < toInt(items[j].fields[dongxi.FieldIndex])
	})
}
