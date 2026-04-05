package cmd

import (
	"fmt"
	"regexp"
	"time"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

var (
	flagQueryField          string
	flagQueryType           string
	flagQueryStatus         string
	flagQueryDestination    string
	flagQueryArea           string
	flagQueryProject        string
	flagQueryTag            string
	flagQueryScheduledBefore string
	flagQueryScheduledAfter  string
	flagQueryDeadlineBefore  string
	flagQueryDeadlineAfter   string
	flagQueryCreatedBefore   string
	flagQueryCreatedAfter    string
	flagQueryEvening        bool
	flagQueryHasNotes       bool
	flagQueryHasChecklist   bool
	flagQueryHasTags        bool
	flagQueryHasDeadline    bool
	flagQueryCount          bool
	flagQueryIncludeTrashed bool
)

var queryCmd = &cobra.Command{
	Use:   "query [pattern]",
	Short: "Query items with regexp matching and powerful filters",
	Long: `Query items from the local cache using regexp patterns and filters.

Examples:
  dongxi query "buy.*milk"                    # Regexp search across title+notes
  dongxi query --field title "^Weekly"        # Search specific field
  dongxi query --field notes "important"      # Search notes only
  dongxi query --type task --status open      # Filter by type and status
  dongxi query --type project --area <uuid>   # Filter by type and area
  dongxi query --destination today            # Filter by destination
  dongxi query --tag <uuid>                   # Filter by tag
  dongxi query --scheduled-before 2025-04-01  # Date range filters
  dongxi query --evening                      # Only evening tasks
  dongxi query --has-notes                    # Only items with notes
  dongxi query --count                        # Just print count`,
	RunE: runQuery,
}

func init() {
	queryCmd.Flags().StringVar(&flagQueryField, "field", "all", "Field to search: title, notes, all")
	queryCmd.Flags().StringVar(&flagQueryType, "type", "all", "Item type: task, project, heading, area, tag, checklist, all")
	queryCmd.Flags().StringVar(&flagQueryStatus, "status", "open", "Status filter: open, completed, cancelled, any")
	queryCmd.Flags().StringVar(&flagQueryDestination, "destination", "any", "Destination: inbox, today, evening, someday, any")
	queryCmd.Flags().StringVar(&flagQueryArea, "area", "", "Filter by area UUID")
	queryCmd.Flags().StringVar(&flagQueryProject, "project", "", "Filter by project UUID")
	queryCmd.Flags().StringVar(&flagQueryTag, "tag", "", "Filter by tag UUID")
	queryCmd.Flags().StringVar(&flagQueryScheduledBefore, "scheduled-before", "", "Scheduled before date (YYYY-MM-DD)")
	queryCmd.Flags().StringVar(&flagQueryScheduledAfter, "scheduled-after", "", "Scheduled after date (YYYY-MM-DD)")
	queryCmd.Flags().StringVar(&flagQueryDeadlineBefore, "deadline-before", "", "Deadline before date (YYYY-MM-DD)")
	queryCmd.Flags().StringVar(&flagQueryDeadlineAfter, "deadline-after", "", "Deadline after date (YYYY-MM-DD)")
	queryCmd.Flags().StringVar(&flagQueryCreatedBefore, "created-before", "", "Created before date (YYYY-MM-DD)")
	queryCmd.Flags().StringVar(&flagQueryCreatedAfter, "created-after", "", "Created after date (YYYY-MM-DD)")
	queryCmd.Flags().BoolVar(&flagQueryEvening, "evening", false, "Only evening tasks")
	queryCmd.Flags().BoolVar(&flagQueryHasNotes, "has-notes", false, "Only items with notes")
	queryCmd.Flags().BoolVar(&flagQueryHasChecklist, "has-checklist", false, "Only items with checklist items")
	queryCmd.Flags().BoolVar(&flagQueryHasTags, "has-tags", false, "Only items with tags")
	queryCmd.Flags().BoolVar(&flagQueryHasDeadline, "has-deadline", false, "Only items with a deadline")
	queryCmd.Flags().BoolVar(&flagQueryCount, "count", false, "Just print the count")
	queryCmd.Flags().BoolVar(&flagQueryIncludeTrashed, "include-trashed", false, "Include trashed items")
}

func runQuery(cmd *cobra.Command, args []string) error {
	// Compile regexp if provided.
	var re *regexp.Regexp
	if len(args) > 0 {
		var err error
		re, err = regexp.Compile(args[0])
		if err != nil {
			return fmt.Errorf("invalid regexp %q: %w", args[0], err)
		}
	}

	// Parse date filters.
	parseDateFlag := func(val, name string) (float64, error) {
		if val == "" {
			return 0, nil
		}
		t, err := time.Parse("2006-01-02", val)
		if err != nil {
			return 0, fmt.Errorf("invalid date for --%s: %q (expected YYYY-MM-DD)", name, val)
		}
		return float64(t.Unix()), nil
	}

	scheduledBefore, err := parseDateFlag(flagQueryScheduledBefore, "scheduled-before")
	if err != nil {
		return err
	}
	scheduledAfter, err := parseDateFlag(flagQueryScheduledAfter, "scheduled-after")
	if err != nil {
		return err
	}
	deadlineBefore, err := parseDateFlag(flagQueryDeadlineBefore, "deadline-before")
	if err != nil {
		return err
	}
	deadlineAfter, err := parseDateFlag(flagQueryDeadlineAfter, "deadline-after")
	if err != nil {
		return err
	}
	createdBefore, err := parseDateFlag(flagQueryCreatedBefore, "created-before")
	if err != nil {
		return err
	}
	createdAfter, err := parseDateFlag(flagQueryCreatedAfter, "created-after")
	if err != nil {
		return err
	}

	s, _, _, err := loadState()
	if err != nil {
		return err
	}

	var filtered []replayedItem
	for _, item := range s.items {
		// Type filter.
		if !queryMatchesType(item, flagQueryType) {
			continue
		}

		// Trashed filter.
		trashed := toBool(item.fields[dongxi.FieldTrashed])
		if item.entity == string(dongxi.EntityTask) {
			trashed = trashed || s.isOrphanedByTrashedParent(&item)
		}
		if trashed && !flagQueryIncludeTrashed {
			continue
		}

		// Status filter (applies to tasks and checklist items).
		if flagQueryStatus != "any" && (item.entity == string(dongxi.EntityTask) || item.entity == string(dongxi.EntityChecklistItem)) {
			status := toInt(item.fields[dongxi.FieldStatus])
			switch flagQueryStatus {
			case "open":
				if status != int(dongxi.TaskStatusOpen) {
					continue
				}
			case "completed":
				if status != int(dongxi.TaskStatusCompleted) {
					continue
				}
			case "cancelled":
				if status != int(dongxi.TaskStatusCancelled) {
					continue
				}
			}
		}

		// Destination filter (tasks/projects only).
		if flagQueryDestination != "any" && item.entity == string(dongxi.EntityTask) {
			dest := toInt(item.fields[dongxi.FieldDestination])
			switch flagQueryDestination {
			case "inbox":
				if dest != int(dongxi.TaskDestinationInbox) {
					continue
				}
			case "today":
				if dest != int(dongxi.TaskDestinationAnytime) {
					continue
				}
			case "evening":
				if dest != int(dongxi.TaskDestinationAnytime) || toInt(item.fields[dongxi.FieldStartBucket]) != 1 {
					continue
				}
			case "someday":
				if dest != int(dongxi.TaskDestinationSomeday) {
					continue
				}
			}
		}

		// Area filter.
		if flagQueryArea != "" && item.entity == string(dongxi.EntityTask) {
			taskArea := firstString(item.fields[dongxi.FieldAreaIDs])
			if taskArea == "" {
				if projID := firstString(item.fields[dongxi.FieldProjectIDs]); projID != "" {
					if proj, ok := s.projects[projID]; ok {
						taskArea = firstString(proj.fields[dongxi.FieldAreaIDs])
					}
				}
			}
			if taskArea != flagQueryArea {
				continue
			}
		}

		// Project filter.
		if flagQueryProject != "" && item.entity == string(dongxi.EntityTask) {
			if firstString(item.fields[dongxi.FieldProjectIDs]) != flagQueryProject {
				continue
			}
		}

		// Tag filter.
		if flagQueryTag != "" {
			if !hasString(item.fields[dongxi.FieldTagIDs], flagQueryTag) {
				continue
			}
		}

		// Date filters.
		if scheduledBefore > 0 || scheduledAfter > 0 {
			sr := toFloat(item.fields[dongxi.FieldScheduledDate])
			if scheduledBefore > 0 && (sr <= 0 || sr >= scheduledBefore) {
				continue
			}
			if scheduledAfter > 0 && (sr <= 0 || sr < scheduledAfter) {
				continue
			}
		}

		if deadlineBefore > 0 || deadlineAfter > 0 {
			dd := toFloat(item.fields[dongxi.FieldDeadline])
			if deadlineBefore > 0 && (dd <= 0 || dd >= deadlineBefore) {
				continue
			}
			if deadlineAfter > 0 && (dd <= 0 || dd < deadlineAfter) {
				continue
			}
		}

		if createdBefore > 0 || createdAfter > 0 {
			cd := toFloat(item.fields[dongxi.FieldCreationDate])
			if createdBefore > 0 && (cd <= 0 || cd >= createdBefore) {
				continue
			}
			if createdAfter > 0 && (cd <= 0 || cd < createdAfter) {
				continue
			}
		}

		// Evening filter.
		if flagQueryEvening {
			if toInt(item.fields[dongxi.FieldStartBucket]) != 1 {
				continue
			}
		}

		// Presence filters.
		if flagQueryHasNotes {
			if dongxi.NoteText(item.fields[dongxi.FieldNote]) == "" {
				continue
			}
		}
		if flagQueryHasChecklist {
			if !queryItemHasChecklist(s, item.uuid) {
				continue
			}
		}
		if flagQueryHasTags {
			tags := toStringSlice(item.fields[dongxi.FieldTagIDs])
			if len(tags) == 0 {
				continue
			}
		}
		if flagQueryHasDeadline {
			if toFloat(item.fields[dongxi.FieldDeadline]) <= 0 {
				continue
			}
		}

		// Regexp filter.
		if re != nil {
			title := toStr(item.fields[dongxi.FieldTitle])
			notes := dongxi.NoteText(item.fields[dongxi.FieldNote])
			matched := false
			switch flagQueryField {
			case "title":
				matched = re.MatchString(title)
			case "notes":
				matched = re.MatchString(notes)
			default: // "all"
				matched = re.MatchString(title) || re.MatchString(notes)
			}
			if !matched {
				continue
			}
		}

		filtered = append(filtered, item)
	}

	// Count mode.
	if flagQueryCount {
		fmt.Println(len(filtered))
		return nil
	}

	// JSON output.
	if flagJSON {
		var out []ItemOutput
		for _, item := range filtered {
			out = append(out, s.itemToOutput(&item))
		}
		return printJSON(out)
	}

	// Human output.
	for _, item := range filtered {
		title := toStr(item.fields[dongxi.FieldTitle])
		if title == "" {
			title = "(untitled)"
		}
		typePrefix := ""
		switch {
		case item.entity == string(dongxi.EntityChecklistItem):
			typePrefix = "(checklist) "
		case item.entity == string(dongxi.EntityArea):
			typePrefix = "(area) "
		case item.entity == string(dongxi.EntityTag):
			typePrefix = "(tag) "
		case item.entity == string(dongxi.EntityTask) && toInt(item.fields[dongxi.FieldType]) == int(dongxi.TaskTypeProject):
			typePrefix = "(project) "
		case item.entity == string(dongxi.EntityTask) && toInt(item.fields[dongxi.FieldType]) == int(dongxi.TaskTypeHeading):
			typePrefix = "(heading) "
		}
		fmt.Printf("  %s%s  [%s]\n", typePrefix, title, item.uuid)
	}

	if len(filtered) == 0 {
		fmt.Println("  (no results)")
	} else {
		fmt.Printf("\n%d result(s)\n", len(filtered))
	}
	return nil
}

// queryMatchesType checks if an item matches the --type filter.
func queryMatchesType(item replayedItem, typeFilter string) bool {
	switch typeFilter {
	case "all":
		return true
	case "task":
		return item.entity == string(dongxi.EntityTask) && toInt(item.fields[dongxi.FieldType]) == int(dongxi.TaskTypeTask)
	case "project":
		return item.entity == string(dongxi.EntityTask) && toInt(item.fields[dongxi.FieldType]) == int(dongxi.TaskTypeProject)
	case "heading":
		return item.entity == string(dongxi.EntityTask) && toInt(item.fields[dongxi.FieldType]) == int(dongxi.TaskTypeHeading)
	case "area":
		return item.entity == string(dongxi.EntityArea)
	case "tag":
		return item.entity == string(dongxi.EntityTag)
	case "checklist":
		return item.entity == string(dongxi.EntityChecklistItem)
	}
	return false
}

// queryItemHasChecklist checks if a task has any checklist items.
func queryItemHasChecklist(s *thingsState, taskUUID string) bool {
	for _, item := range s.items {
		if item.entity == string(dongxi.EntityChecklistItem) {
			if firstString(item.fields[dongxi.FieldTaskIDs]) == taskUUID {
				return true
			}
		}
	}
	return false
}
