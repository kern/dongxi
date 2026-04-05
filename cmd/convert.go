package cmd

import (
	"fmt"
	"time"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

var convertCmd = &cobra.Command{
	Use:   "convert <uuid>",
	Short: "Convert a task to a project or vice versa",
	Long: `Convert between task and project types.

Examples:
  dongxi convert <uuid> --to project     # convert task to project
  dongxi convert <uuid> --to task        # convert project to task`,
	Args: cobra.ExactArgs(1),
	RunE: runConvert,
}

var flagConvertTo string

func init() {
	convertCmd.Flags().StringVar(&flagConvertTo, "to", "", "Target type: task or project (required)")
	_ = convertCmd.MarkFlagRequired("to")
}

func runConvert(cmd *cobra.Command, args []string) error {
	s, client, historyKey, err := loadState()
	if err != nil {
		return err
	}

	item, err := s.resolveUUID(args[0])
	if err != nil {
		return err
	}
	if item.entity != string(dongxi.EntityTask) {
		return fmt.Errorf("%s is a %s, not a task or project", args[0], item.entity)
	}

	var targetType dongxi.TaskType
	switch flagConvertTo {
	case "task":
		targetType = dongxi.TaskTypeTask
	case "project":
		targetType = dongxi.TaskTypeProject
	default:
		return fmt.Errorf("unknown target type %q: must be task or project", flagConvertTo)
	}

	currentType := dongxi.TaskType(toInt(item.fields[dongxi.FieldType]))
	if currentType == targetType {
		return fmt.Errorf("item is already a %s", flagConvertTo)
	}

	histInfo, err := client.GetHistory(historyKey)
	if err != nil {
		return fmt.Errorf("fetch history info: %w", err)
	}

	now := float64(time.Now().UnixNano()) / 1e9
	commit := map[string]dongxi.CommitItem{
		item.uuid: {
			T: dongxi.ItemTypeModify,
			E: dongxi.EntityTask,
			P: map[string]any{
				dongxi.FieldType:             int(targetType),
				dongxi.FieldModificationDate: now,
			},
		},
	}

	resp, err := client.Commit(historyKey, histInfo.LatestServerIndex, commit)
	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	title := toStr(item.fields[dongxi.FieldTitle])
	if flagJSON {
		return printJSON(EditOutput{
			UUID:        item.uuid,
			Title:       title,
			Changes:     []string{fmt.Sprintf("type -> %s", flagConvertTo)},
			ServerIndex: resp.ServerHeadIndex,
		})
	}

	if title == "" {
		title = "(untitled)"
	}
	fmt.Printf("  Converted %q to %s\n", title, flagConvertTo)
	fmt.Printf("Server index: %d\n", resp.ServerHeadIndex)
	return nil
}
