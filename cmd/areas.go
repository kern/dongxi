package cmd

import (
	"fmt"
	"strings"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

var (
	flagAreasFilter   string
	flagAreasProjects bool
)

var areasCmd = &cobra.Command{
	Use:   "areas",
	Short: "List areas of responsibility",
	Long: `List areas from Things Cloud.

Examples:
  dongxi areas                           # active areas (default)
  dongxi areas -f trash                  # trashed areas
  dongxi areas -f all                    # all areas
  dongxi areas --projects                # show projects under each area`,
	RunE: runAreas,
}

func init() {
	areasCmd.Flags().StringVarP(&flagAreasFilter, "filter", "f", "active", "Filter: active, trash, all")
	areasCmd.Flags().BoolVar(&flagAreasProjects, "projects", false, "Show open projects under each area")
}

func runAreas(cmd *cobra.Command, args []string) error {
	s, _, _, err := loadState()
	if err != nil {
		return err
	}

	showActive := false
	showTrashed := false
	switch strings.ToLower(flagAreasFilter) {
	case "active":
		showActive = true
	case "trash":
		showTrashed = true
	case "all":
		showActive = true
		showTrashed = true
	default:
		return fmt.Errorf("unknown filter %q: must be active, trash, or all", flagAreasFilter)
	}

	var filtered []replayedItem
	for _, item := range s.items {
		if item.entity != string(dongxi.EntityArea) {
			continue
		}
		trashed := toBool(item.fields[dongxi.FieldTrashed])
		if showActive && !showTrashed && trashed {
			continue
		}
		if showTrashed && !showActive && !trashed {
			continue
		}
		filtered = append(filtered, item)
	}

	sortByIndex(filtered)

	if flagJSON {
		var out []ItemOutput
		for _, item := range filtered {
			out = append(out, s.itemToOutput(&item))
		}
		return printJSON(out)
	}

	// Collect open projects per area if --projects is set.
	var areaProjects map[string][]replayedItem
	if flagAreasProjects {
		areaProjects = make(map[string][]replayedItem)
		for _, item := range s.items {
			if item.entity != string(dongxi.EntityTask) {
				continue
			}
			if toInt(item.fields[dongxi.FieldType]) != int(dongxi.TaskTypeProject) {
				continue
			}
			if toInt(item.fields[dongxi.FieldStatus]) != int(dongxi.TaskStatusOpen) {
				continue
			}
			if toBool(item.fields[dongxi.FieldTrashed]) {
				continue
			}
			areaID := firstString(item.fields[dongxi.FieldAreaIDs])
			areaProjects[areaID] = append(areaProjects[areaID], item)
		}
		for k := range areaProjects {
			sortByIndex(areaProjects[k])
		}
	}

	for _, item := range filtered {
		title := toStr(item.fields[dongxi.FieldTitle])
		if title == "" {
			title = "(untitled)"
		}
		fmt.Printf("  %s  [%s]\n", title, item.uuid)
		if flagAreasProjects {
			for _, proj := range areaProjects[item.uuid] {
				pTitle := toStr(proj.fields[dongxi.FieldTitle])
				if pTitle == "" {
					pTitle = "(untitled)"
				}
				total, completed := s.projectProgress(proj.uuid)
				progress := ""
				if total > 0 {
					progress = fmt.Sprintf("  (%d/%d)", completed, total)
				}
				fmt.Printf("    - %s%s\n", pTitle, progress)
			}
		}
	}

	if len(filtered) == 0 {
		fmt.Println("  (no areas)")
	} else {
		fmt.Printf("\n%d area(s)\n", len(filtered))
	}
	return nil
}
