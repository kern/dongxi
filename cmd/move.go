package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

var (
	flagMoveArea        string
	flagMoveProject     string
	flagMoveDestination string
	flagMoveHeading     string
)

var moveCmd = &cobra.Command{
	Use:   "move <uuid>",
	Short: "Move a task to a different area, project, or destination",
	Long: `Move a task to a different area, project, or destination (inbox/today/someday).

Examples:
  dongxi move <uuid> --area <area-uuid>
  dongxi move <uuid> --project <project-uuid>
  dongxi move <uuid> --destination today
  dongxi move <uuid> --destination evening
  dongxi move <uuid> --destination inbox --area ""   # clear area`,
	Args: cobra.ExactArgs(1),
	RunE: runMove,
}

func init() {
	moveCmd.Flags().StringVar(&flagMoveArea, "area", "", "Area UUID (use \"\" to clear)")
	moveCmd.Flags().StringVar(&flagMoveProject, "project", "", "Project UUID (use \"\" to clear)")
	moveCmd.Flags().StringVar(&flagMoveDestination, "destination", "", "Destination: inbox, today, evening, or someday")
	moveCmd.Flags().StringVar(&flagMoveHeading, "heading", "", "Heading UUID (use \"\" to clear)")
}

func runMove(cmd *cobra.Command, args []string) error {
	s, client, historyKey, err := loadState()
	if err != nil {
		return err
	}

	item, err := s.resolveUUID(args[0])
	if err != nil {
		return err
	}
	if item.entity != string(dongxi.EntityTask) {
		return fmt.Errorf("%s is a %s, not a task", args[0], item.entity)
	}

	histInfo, err := client.GetHistory(historyKey)
	if err != nil {
		return fmt.Errorf("fetch history info: %w", err)
	}

	now := float64(time.Now().UnixNano()) / 1e9
	payload := map[string]any{dongxi.FieldModificationDate: now}
	changes := []string{}

	if cmd.Flags().Changed("area") {
		if flagMoveArea == "" {
			payload[dongxi.FieldAreaIDs] = []string{}
			changes = append(changes, "cleared area")
		} else {
			area, err := s.resolveUUID(flagMoveArea)
			if err != nil {
				return fmt.Errorf("resolve area: %w", err)
			}
			if area.entity != string(dongxi.EntityArea) {
				return fmt.Errorf("%s is not an area", flagMoveArea)
			}
			payload[dongxi.FieldAreaIDs] = []string{area.uuid}
			changes = append(changes, fmt.Sprintf("area -> %s", toStr(area.fields[dongxi.FieldTitle])))
		}
	}

	if cmd.Flags().Changed("project") {
		if flagMoveProject == "" {
			payload[dongxi.FieldProjectIDs] = []string{}
			payload[dongxi.FieldActionGroupIDs] = []any{}
			payload[dongxi.FieldHeadingIDs] = []any{}
			changes = append(changes, "cleared project")
		} else {
			proj, err := s.resolveUUID(flagMoveProject)
			if err != nil {
				return fmt.Errorf("resolve project: %w", err)
			}
			if toInt(proj.fields[dongxi.FieldType]) != int(dongxi.TaskTypeProject) {
				return fmt.Errorf("%s is not a project", flagMoveProject)
			}
			payload[dongxi.FieldProjectIDs] = []string{proj.uuid}
			changes = append(changes, fmt.Sprintf("project -> %s", toStr(proj.fields[dongxi.FieldTitle])))
		}
	}

	if cmd.Flags().Changed("heading") {
		if flagMoveHeading == "" {
			payload[dongxi.FieldHeadingIDs] = []any{}
			payload[dongxi.FieldActionGroupIDs] = []any{}
			changes = append(changes, "cleared heading")
		} else {
			heading, err := s.resolveUUID(flagMoveHeading)
			if err != nil {
				return fmt.Errorf("resolve heading: %w", err)
			}
			if toInt(heading.fields[dongxi.FieldType]) != int(dongxi.TaskTypeHeading) {
				return fmt.Errorf("%s is not a heading", flagMoveHeading)
			}
			payload[dongxi.FieldHeadingIDs] = []string{heading.uuid}
			payload[dongxi.FieldActionGroupIDs] = []string{heading.uuid}
			changes = append(changes, fmt.Sprintf("heading -> %s", toStr(heading.fields[dongxi.FieldTitle])))
		}
	}

	if flagMoveDestination != "" {
		switch strings.ToLower(flagMoveDestination) {
		case "inbox":
			payload[dongxi.FieldDestination] = int(dongxi.TaskDestinationInbox)
			payload[dongxi.FieldScheduledDate] = nil
			payload[dongxi.FieldTodayIndexRef] = nil
			changes = append(changes, "destination -> inbox")
		case "today", "anytime":
			payload[dongxi.FieldDestination] = int(dongxi.TaskDestinationAnytime)
			t := time.Now()
			midnight := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).Unix()
			payload[dongxi.FieldScheduledDate] = midnight
			payload[dongxi.FieldTodayIndexRef] = midnight
			payload[dongxi.FieldStartBucket] = 0
			changes = append(changes, "destination -> today")
		case "evening":
			payload[dongxi.FieldDestination] = int(dongxi.TaskDestinationAnytime)
			t := time.Now()
			midnight := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).Unix()
			payload[dongxi.FieldScheduledDate] = midnight
			payload[dongxi.FieldTodayIndexRef] = midnight
			payload[dongxi.FieldStartBucket] = 1
			changes = append(changes, "destination -> evening")
		case "someday":
			payload[dongxi.FieldDestination] = int(dongxi.TaskDestinationSomeday)
			payload[dongxi.FieldScheduledDate] = nil
			payload[dongxi.FieldTodayIndexRef] = nil
			changes = append(changes, "destination -> someday")
		default:
			return fmt.Errorf("unknown destination %q: must be inbox, today, evening, or someday", flagMoveDestination)
		}
	}

	if len(changes) == 0 {
		return fmt.Errorf("no changes specified (use --area, --project, or --destination)")
	}

	commit := map[string]dongxi.CommitItem{
		item.uuid: {
			T: dongxi.ItemTypeModify,
			E: dongxi.EntityTask,
			P: payload,
		},
	}

	resp, err := client.Commit(historyKey, histInfo.LatestServerIndex, commit)
	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	if flagJSON {
		return printJSON(EditOutput{
			UUID:        item.uuid,
			Title:       toStr(item.fields[dongxi.FieldTitle]),
			Changes:     changes,
			ServerIndex: resp.ServerHeadIndex,
		})
	}

	title := toStr(item.fields[dongxi.FieldTitle])
	if title == "" {
		title = "(untitled)"
	}
	fmt.Printf("  %s: %s\n", title, strings.Join(changes, ", "))
	fmt.Printf("Server index: %d\n", resp.ServerHeadIndex)
	return nil
}
