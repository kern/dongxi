package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

var (
	flagRepeatFrequency  string
	flagRepeatClear      bool
	flagRepeatType       string
	flagRepeatDays       string
	flagRepeatEndDate    string
	flagRepeatEndCount   int
)

var repeatCmd = &cobra.Command{
	Use:   "repeat <uuid>",
	Short: "Set or clear a repeating schedule on a task",
	Long: `Set a repeating schedule on a task.

Frequency format: <number> <unit>
Units: daily, weekly, monthly

Repeat types:
  fixed       Repeat on a fixed schedule (default)
  completion  Repeat N days/weeks/months after completion

Multi-weekday repeats (weekly only):
  --days mon,wed,fri   Repeat on specific weekdays

End conditions:
  --end-date 2026-12-31   Stop repeating after this date
  --end-count 10          Stop after N repetitions

Examples:
  dongxi repeat <uuid> --every "1 daily"
  dongxi repeat <uuid> --every "2 weekly" --days mon,wed,fri
  dongxi repeat <uuid> --every "1 monthly" --type completion
  dongxi repeat <uuid> --every "1 daily" --end-date 2026-12-31
  dongxi repeat <uuid> --every "1 weekly" --end-count 10
  dongxi repeat <uuid> --clear`,
	Args: cobra.ExactArgs(1),
	RunE: runRepeat,
}

func init() {
	repeatCmd.Flags().StringVar(&flagRepeatFrequency, "every", "", "Repeat frequency (e.g. \"1 daily\", \"2 weekly\", \"1 monthly\")")
	repeatCmd.Flags().BoolVar(&flagRepeatClear, "clear", false, "Remove the repeat rule")
	repeatCmd.Flags().StringVar(&flagRepeatType, "type", "fixed", "Repeat type: fixed or completion")
	repeatCmd.Flags().StringVar(&flagRepeatDays, "days", "", "Weekdays for weekly repeat (e.g. \"mon,wed,fri\")")
	repeatCmd.Flags().StringVar(&flagRepeatEndDate, "end-date", "", "End date (YYYY-MM-DD)")
	repeatCmd.Flags().IntVar(&flagRepeatEndCount, "end-count", 0, "Maximum number of repetitions")
}

func runRepeat(cmd *cobra.Command, args []string) error {
	if !flagRepeatClear && flagRepeatFrequency == "" {
		return fmt.Errorf("specify --every or --clear")
	}

	s, client, historyKey, err := loadState()
	if err != nil {
		return err
	}

	item, err := s.resolveUUID(args[0])
	if err != nil {
		return err
	}
	if item.entity != string(dongxi.EntityTask) {
		return fmt.Errorf("%s is not a task", args[0])
	}

	histInfo, err := client.GetHistory(historyKey)
	if err != nil {
		return fmt.Errorf("fetch history info: %w", err)
	}

	now := float64(time.Now().UnixNano()) / 1e9
	payload := map[string]any{dongxi.FieldModificationDate: now}

	if flagRepeatClear {
		payload[dongxi.FieldRepeatRule] = nil
	} else {
		rr, err := parseRepeatFrequency(flagRepeatFrequency)
		if err != nil {
			return err
		}
		payload[dongxi.FieldRepeatRule] = rr
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
		action := "set repeat: " + flagRepeatFrequency
		if flagRepeatClear {
			action = "cleared repeat"
		}
		return printJSON(RepeatOutput{
			UUID:        item.uuid,
			Title:       toStr(item.fields[dongxi.FieldTitle]),
			Action:      action,
			ServerIndex: resp.ServerHeadIndex,
		})
	}

	title := toStr(item.fields[dongxi.FieldTitle])
	if title == "" {
		title = "(untitled)"
	}
	if flagRepeatClear {
		fmt.Printf("  %s: cleared repeat\n", title)
	} else {
		fmt.Printf("  %s: repeats every %s\n", title, flagRepeatFrequency)
	}
	fmt.Printf("Server index: %d\n", resp.ServerHeadIndex)
	return nil
}

var weekdayNames = map[string]int{
	"sun": 0, "sunday": 0,
	"mon": 1, "monday": 1,
	"tue": 2, "tuesday": 2,
	"wed": 3, "wednesday": 3,
	"thu": 4, "thursday": 4,
	"fri": 5, "friday": 5,
	"sat": 6, "saturday": 6,
}

// parseRepeatFrequency parses "N unit" into a Things repeat rule.
func parseRepeatFrequency(freq string) (map[string]any, error) {
	parts := strings.Fields(freq)
	if len(parts) != 2 {
		return nil, fmt.Errorf("frequency must be \"<number> <unit>\" (e.g. \"1 daily\")")
	}

	amount, err := strconv.Atoi(parts[0])
	if err != nil || amount < 1 {
		return nil, fmt.Errorf("frequency amount must be a positive integer")
	}

	var fu int
	var offsets []any
	today := time.Now()
	todayMidnight := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)

	switch strings.ToLower(parts[1]) {
	case "daily", "day", "days":
		fu = dongxi.FreqUnitDaily
		offsets = []any{map[string]any{dongxi.OffsetDay: 0}}
	case "weekly", "week", "weeks":
		fu = dongxi.FreqUnitWeekly
		if flagRepeatDays != "" {
			// Parse multi-weekday specification.
			for _, dayStr := range strings.Split(flagRepeatDays, ",") {
				dayStr = strings.TrimSpace(strings.ToLower(dayStr))
				wd, ok := weekdayNames[dayStr]
				if !ok {
					return nil, fmt.Errorf("unknown weekday %q: use mon, tue, wed, thu, fri, sat, sun", dayStr)
				}
				offsets = append(offsets, map[string]any{dongxi.OffsetWeekday: wd})
			}
		} else {
			offsets = []any{map[string]any{dongxi.OffsetWeekday: int(today.Weekday())}}
		}
	case "monthly", "month", "months":
		fu = dongxi.FreqUnitMonthly
		offsets = []any{map[string]any{dongxi.OffsetDay: today.Day()}}
	default:
		return nil, fmt.Errorf("unknown unit %q: use daily, weekly, or monthly", parts[1])
	}

	// Repeat type.
	repeatType := dongxi.RepeatFixedSchedule
	if flagRepeatType == "completion" {
		repeatType = dongxi.RepeatAfterCompletion
	}

	// End condition.
	endDate := dongxi.RepeatEndNever
	repeatCount := 0
	if flagRepeatEndDate != "" {
		t, err := time.Parse("2006-01-02", flagRepeatEndDate)
		if err != nil {
			return nil, fmt.Errorf("parse --end-date %q: %w", flagRepeatEndDate, err)
		}
		endDate = t.Unix()
	}
	if flagRepeatEndCount > 0 {
		repeatCount = flagRepeatEndCount
	}

	return map[string]any{
		dongxi.RepeatVersion:      4,
		dongxi.RepeatType:         repeatType,
		dongxi.RepeatFreqUnit:     fu,
		dongxi.RepeatFreqAmount:   amount,
		dongxi.RepeatOffset:       offsets,
		dongxi.RepeatAnchor:       todayMidnight.Unix(),
		dongxi.RepeatScheduledRef: todayMidnight.Unix(),
		dongxi.RepeatEndDate:      endDate,
		dongxi.RepeatCount:        repeatCount,
		dongxi.RepeatTimeShift:    0,
	}, nil
}
