package cmd

import (
	"fmt"
	"time"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

var (
	flagEditTitle        string
	flagEditNote         string
	flagEditScheduled    string
	flagEditDeadline     string
	flagEditEvening string
)

var editCmd = &cobra.Command{
	Use:   "edit <uuid>",
	Short: "Edit a task or project's properties",
	Long: `Edit task or project properties.

Examples:
  dongxi edit <uuid> --title "New title"
  dongxi edit <uuid> --note "Some notes"
  dongxi edit <uuid> --scheduled 2025-04-01
  dongxi edit <uuid> --deadline 2025-04-15
  dongxi edit <uuid> --scheduled ""          # clear scheduled date
  dongxi edit <uuid> --evening true          # move to This Evening
  dongxi edit <uuid> --evening false         # move back to Today`,
	Args: cobra.ExactArgs(1),
	RunE: runEdit,
}

func init() {
	editCmd.Flags().StringVar(&flagEditTitle, "title", "", "New title")
	editCmd.Flags().StringVar(&flagEditNote, "note", "", "New note content")
	editCmd.Flags().StringVar(&flagEditScheduled, "scheduled", "", "Scheduled date (YYYY-MM-DD, or \"\" to clear)")
	editCmd.Flags().StringVar(&flagEditDeadline, "deadline", "", "Deadline date (YYYY-MM-DD, or \"\" to clear)")
	editCmd.Flags().StringVar(&flagEditEvening, "evening", "", "Move to This Evening (true/false)")
}

func runEdit(cmd *cobra.Command, args []string) error {
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

	if cmd.Flags().Changed("title") {
		payload[dongxi.FieldTitle] = flagEditTitle
		changes = append(changes, fmt.Sprintf("title -> %q", flagEditTitle))
	}

	if cmd.Flags().Changed("note") {
		if flagEditNote == "" {
			payload[dongxi.FieldNote] = dongxi.EmptyNote
			changes = append(changes, "cleared note")
		} else {
			payload[dongxi.FieldNote] = dongxi.NewNote(flagEditNote)
			changes = append(changes, "updated note")
		}
	}

	if cmd.Flags().Changed("scheduled") {
		if flagEditScheduled == "" {
			payload[dongxi.FieldScheduledDate] = nil
			payload[dongxi.FieldTodayIndexRef] = nil
			changes = append(changes, "cleared scheduled date")
		} else {
			t, err := time.Parse("2006-01-02", flagEditScheduled)
			if err != nil {
				return fmt.Errorf("parse --scheduled date %q: %w", flagEditScheduled, err)
			}
			ts := t.Unix()
			payload[dongxi.FieldScheduledDate] = ts
			payload[dongxi.FieldTodayIndexRef] = ts
			changes = append(changes, fmt.Sprintf("scheduled -> %s", flagEditScheduled))
		}
	}

	if cmd.Flags().Changed("deadline") {
		if flagEditDeadline == "" {
			payload[dongxi.FieldDeadline] = nil
			changes = append(changes, "cleared deadline")
		} else {
			t, err := time.Parse("2006-01-02", flagEditDeadline)
			if err != nil {
				return fmt.Errorf("parse --deadline date %q: %w", flagEditDeadline, err)
			}
			payload[dongxi.FieldDeadline] = t.Unix()
			changes = append(changes, fmt.Sprintf("deadline -> %s", flagEditDeadline))
		}
	}

	if cmd.Flags().Changed("evening") {
		switch flagEditEvening {
		case "true", "1", "yes":
			payload[dongxi.FieldStartBucket] = 1
			changes = append(changes, "evening -> on")
		case "false", "0", "no":
			payload[dongxi.FieldStartBucket] = 0
			changes = append(changes, "evening -> off")
		default:
			return fmt.Errorf("--evening must be true or false")
		}
	}

	if len(changes) == 0 {
		return fmt.Errorf("no changes specified (use --title, --note, --scheduled, --deadline, or --evening)")
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
	for _, c := range changes {
		fmt.Printf("  %s: %s\n", title, c)
	}
	fmt.Printf("Server index: %d\n", resp.ServerHeadIndex)
	return nil
}
