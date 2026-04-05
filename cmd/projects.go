package cmd

import (
	"fmt"
	"strings"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

var (
	flagProjectsFilter string
	flagProjectsArea   string
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List projects",
	Long: `List projects from Things Cloud.

Examples:
  dongxi projects                        # open projects (default)
  dongxi projects -f all                 # all projects
  dongxi projects -f completed           # completed projects
  dongxi projects -f trash               # trashed projects
  dongxi projects --area <uuid>          # projects in an area`,
	RunE: runProjects,
}

func init() {
	projectsCmd.Flags().StringVarP(&flagProjectsFilter, "filter", "f", "open", "Filter: open, completed, trash, all")
	projectsCmd.Flags().StringVar(&flagProjectsArea, "area", "", "Filter by area UUID")
}

func runProjects(cmd *cobra.Command, args []string) error {
	s, _, _, err := loadState()
	if err != nil {
		return err
	}

	showOpen := false
	showCompleted := false
	showTrashed := false
	switch strings.ToLower(flagProjectsFilter) {
	case "open":
		showOpen = true
	case "completed":
		showCompleted = true
	case "trash":
		showTrashed = true
	case "all":
		showOpen = true
		showCompleted = true
	default:
		return fmt.Errorf("unknown filter %q: must be open, completed, trash, or all", flagProjectsFilter)
	}

	var areaUUID string
	if flagProjectsArea != "" {
		area, err := s.resolveUUID(flagProjectsArea)
		if err != nil {
			return fmt.Errorf("resolve area: %w", err)
		}
		areaUUID = area.uuid
	}

	var filtered []replayedItem
	for _, item := range s.items {
		if item.entity != string(dongxi.EntityTask) {
			continue
		}
		if toInt(item.fields[dongxi.FieldType]) != int(dongxi.TaskTypeProject) {
			continue
		}

		trashed := toBool(item.fields[dongxi.FieldTrashed])
		if showTrashed {
			if !trashed {
				continue
			}
		} else {
			if trashed {
				continue
			}
		}

		if !showTrashed {
			status := toInt(item.fields[dongxi.FieldStatus])
			if showOpen && !showCompleted && status != int(dongxi.TaskStatusOpen) {
				continue
			}
			if showCompleted && !showOpen && status != int(dongxi.TaskStatusCompleted) {
				continue
			}
		}

		if areaUUID != "" {
			if firstString(item.fields[dongxi.FieldAreaIDs]) != areaUUID {
				continue
			}
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
		area := ""
		if areaUUID := firstString(item.fields[dongxi.FieldAreaIDs]); areaUUID != "" {
			if a := s.areaTitle(areaUUID); a != "" {
				area = fmt.Sprintf("  [%s]", a)
			}
		}
		progress := ""
		total, completed := s.projectProgress(item.uuid)
		if total > 0 {
			progress = fmt.Sprintf("  (%d/%d)", completed, total)
		}
		fmt.Printf("  %s%s%s\n", title, progress, area)
	}

	if len(filtered) == 0 {
		fmt.Println("  (no projects)")
	} else {
		fmt.Printf("\n%d project(s)\n", len(filtered))
	}
	return nil
}
