package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync with Things Cloud and report changes",
	Long:  `Fetch new history from Things Cloud, update the local cache, and report what was received.`,
	RunE:  runSync,
}

// SyncOutput is the JSON output for the sync command.
type SyncOutput struct {
	CachedItems int            `json:"cached_items"`
	NewItems    int            `json:"new_items"`
	TotalItems  int            `json:"total_items"`
	Summary     map[string]int `json:"summary"`
}

// syncResult holds the result of a sync operation for reporting.
type syncResult struct {
	cachedBefore int
	newItems     []map[string]any
	totalItems   int
}

// Syncer abstracts the sync operation. Tests can replace this.
type Syncer interface {
	Sync() (*syncResult, error)
}

type realSyncer struct{}

func (realSyncer) Sync() (*syncResult, error) {
	cfg, err := dongxi.LoadConfig()
	if err != nil {
		return nil, err
	}

	client := dongxi.NewClient(cfg.Email, cfg.Password)
	acct, err := client.GetAccount(cfg.Email)
	if err != nil {
		return nil, fmt.Errorf("fetch account: %w", err)
	}

	cache, err := dongxi.LoadCache()
	if err != nil {
		return nil, fmt.Errorf("load cache: %w", err)
	}

	if cache.HistoryKey != acct.HistoryKey {
		cache = &dongxi.Cache{}
	}

	cachedBefore := cache.ItemCount

	newItems, err := client.GetHistoryItemsFrom(acct.HistoryKey, cache.ItemCount)
	if err != nil {
		return nil, fmt.Errorf("fetch history: %w", err)
	}

	cache.HistoryKey = acct.HistoryKey
	cache.LastSyncUnix = time.Now().Unix()
	if len(newItems) > 0 {
		cache.Items = append(cache.Items, newItems...)
		cache.ItemCount = len(cache.Items)
	}
	_ = dongxi.SaveCache(cache)

	return &syncResult{
		cachedBefore: cachedBefore,
		newItems:     newItems,
		totalItems:   cache.ItemCount,
	}, nil
}

var syncer Syncer = realSyncer{}

func runSync(cmd *cobra.Command, args []string) error {
	result, err := syncer.Sync()
	if err != nil {
		return err
	}

	summary := summariseCommits(result.newItems)

	if flagJSON {
		return printJSON(SyncOutput{
			CachedItems: result.cachedBefore,
			NewItems:    len(result.newItems),
			TotalItems:  result.totalItems,
			Summary:     summary,
		})
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Cached records:\t%d\n", result.cachedBefore)
	fmt.Fprintf(w, "New records:\t%d\n", len(result.newItems))
	fmt.Fprintf(w, "Total records:\t%d\n", result.totalItems)

	if len(summary) > 0 {
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "Changes received:")
		for label, count := range summary {
			fmt.Fprintf(w, "  %s:\t%d\n", label, count)
		}
	} else {
		fmt.Fprintln(w, "\nAlready up to date.")
	}

	return w.Flush()
}

// summariseCommits counts operations by "entity/operation" across commits.
func summariseCommits(commits []map[string]any) map[string]int {
	counts := map[string]int{}
	for _, commit := range commits {
		for _, rawVal := range commit {
			val, ok := rawVal.(map[string]any)
			if !ok {
				continue
			}
			entity, _ := val[dongxi.CommitKeyEntity].(string)
			opType := dongxi.ItemType(toInt(val[dongxi.CommitKeyType]))

			var opLabel string
			switch opType {
			case dongxi.ItemTypeCreate:
				opLabel = "created"
			case dongxi.ItemTypeModify:
				opLabel = "modified"
			case dongxi.ItemTypeDelete:
				opLabel = "deleted"
			default:
				opLabel = "unknown"
			}

			key := entityLabel(entity) + " " + opLabel
			counts[key]++
		}
	}
	return counts
}

func entityLabel(entity string) string {
	switch dongxi.EntityType(entity) {
	case dongxi.EntityTask:
		return "task"
	case dongxi.EntityChecklistItem:
		return "checklist item"
	case dongxi.EntityArea:
		return "area"
	case dongxi.EntityTag:
		return "tag"
	default:
		return entity
	}
}
