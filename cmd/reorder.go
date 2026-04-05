package cmd

import (
	"fmt"
	"time"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

var (
	flagReorderAfter  string
	flagReorderBefore string
	flagReorderTop    bool
	flagReorderBottom bool
	flagReorderToday  bool
)

var reorderCmd = &cobra.Command{
	Use:   "reorder <uuid>",
	Short: "Reorder a task within its list",
	Long: `Reorder a task by placing it relative to another task.

Examples:
  dongxi reorder <uuid> --top              # move to top of list
  dongxi reorder <uuid> --bottom           # move to bottom of list
  dongxi reorder <uuid> --after <uuid>     # place after another task
  dongxi reorder <uuid> --before <uuid>    # place before another task
  dongxi reorder <uuid> --top --today      # move to top of Today list`,
	Args: cobra.ExactArgs(1),
	RunE: runReorder,
}

func init() {
	reorderCmd.Flags().StringVar(&flagReorderAfter, "after", "", "Place after this task UUID")
	reorderCmd.Flags().StringVar(&flagReorderBefore, "before", "", "Place before this task UUID")
	reorderCmd.Flags().BoolVar(&flagReorderTop, "top", false, "Move to top of list")
	reorderCmd.Flags().BoolVar(&flagReorderBottom, "bottom", false, "Move to bottom of list")
	reorderCmd.Flags().BoolVar(&flagReorderToday, "today", false, "Reorder within the Today list (uses today index)")
}

func runReorder(cmd *cobra.Command, args []string) error {
	opts := 0
	if flagReorderAfter != "" {
		opts++
	}
	if flagReorderBefore != "" {
		opts++
	}
	if flagReorderTop {
		opts++
	}
	if flagReorderBottom {
		opts++
	}
	if opts != 1 {
		return fmt.Errorf("specify exactly one of --top, --bottom, --after, or --before")
	}

	s, client, historyKey, err := loadState()
	if err != nil {
		return err
	}

	item, err := s.resolveUUID(args[0])
	if err != nil {
		return err
	}

	histInfo, err := client.GetHistory(historyKey)
	if err != nil {
		return fmt.Errorf("fetch history info: %w", err)
	}

	// Choose index field based on --today flag.
	indexField := dongxi.FieldIndex
	if flagReorderToday {
		indexField = dongxi.FieldTodayIndex
	}

	// Collect all sibling tasks to determine index range.
	// Siblings share the same destination (st) and project (pr).
	itemDest := toInt(item.fields[dongxi.FieldDestination])
	itemProj := firstString(item.fields[dongxi.FieldProjectIDs])
	var siblings []replayedItem
	for _, t := range s.items {
		if t.entity != string(dongxi.EntityTask) {
			continue
		}
		if toInt(t.fields[dongxi.FieldType]) != int(dongxi.TaskTypeTask) {
			continue
		}
		if toInt(t.fields[dongxi.FieldStatus]) != int(dongxi.TaskStatusOpen) {
			continue
		}
		if toBool(t.fields[dongxi.FieldTrashed]) {
			continue
		}
		if flagReorderToday {
			// For today reordering, siblings are all tasks in the Today list.
			if toInt(t.fields[dongxi.FieldDestination]) != int(dongxi.TaskDestinationAnytime) {
				continue
			}
		} else {
			if toInt(t.fields[dongxi.FieldDestination]) != itemDest {
				continue
			}
			if firstString(t.fields[dongxi.FieldProjectIDs]) != itemProj {
				continue
			}
		}
		siblings = append(siblings, t)
	}

	var newIx int
	switch {
	case flagReorderTop:
		minIx := toInt(item.fields[indexField])
		for _, sib := range siblings {
			if ix := toInt(sib.fields[indexField]); ix < minIx {
				minIx = ix
			}
		}
		newIx = minIx - 1000
	case flagReorderBottom:
		maxIx := toInt(item.fields[indexField])
		for _, sib := range siblings {
			if ix := toInt(sib.fields[indexField]); ix > maxIx {
				maxIx = ix
			}
		}
		newIx = maxIx + 1000
	case flagReorderAfter != "":
		ref, err := s.resolveUUID(flagReorderAfter)
		if err != nil {
			return fmt.Errorf("resolve --after: %w", err)
		}
		newIx = toInt(ref.fields[indexField]) + 1
	case flagReorderBefore != "":
		ref, err := s.resolveUUID(flagReorderBefore)
		if err != nil {
			return fmt.Errorf("resolve --before: %w", err)
		}
		newIx = toInt(ref.fields[indexField]) - 1
	}

	now := float64(time.Now().UnixNano()) / 1e9
	commit := map[string]dongxi.CommitItem{
		item.uuid: {
			T: dongxi.ItemTypeModify,
			E: dongxi.EntityTask,
			P: map[string]any{
				indexField:                   newIx,
				dongxi.FieldModificationDate: now,
			},
		},
	}

	resp, err := client.Commit(historyKey, histInfo.LatestServerIndex, commit)
	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	if flagJSON {
		return printJSON(ReorderOutput{
			UUID:        item.uuid,
			Title:       toStr(item.fields[dongxi.FieldTitle]),
			Index:       newIx,
			ServerIndex: resp.ServerHeadIndex,
		})
	}

	title := toStr(item.fields[dongxi.FieldTitle])
	if title == "" {
		title = "(untitled)"
	}
	fmt.Printf("  %s: reordered (ix=%d)\n", title, newIx)
	fmt.Printf("Server index: %d\n", resp.ServerHeadIndex)
	return nil
}
