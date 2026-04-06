package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kern/dongxi/dongxi"
)

// CloudClient is the interface for Things Cloud API operations.
// *dongxi.Client satisfies this interface.
type CloudClient interface {
	GetHistory(historyKey string) (*dongxi.HistoryInfo, error)
	Commit(historyKey string, ancestorIndex int, items map[string]dongxi.CommitItem) (*dongxi.CommitResponse, error)
	ResetHistory(email string) (*dongxi.ResetResponse, error)
	GetAccount(email string) (*dongxi.Account, error)
	GetHistoryItems(historyKey string) ([]map[string]any, error)
	Email() string
}

// StateLoader loads the full Things Cloud state. Tests can replace this
// with a mock implementation.
type StateLoader interface {
	LoadState() (*thingsState, CloudClient, string, error)
}

// realStateLoader is the production implementation that talks to the Things Cloud API.
type realStateLoader struct{}

func (realStateLoader) LoadState() (*thingsState, CloudClient, string, error) {
	cfg, err := dongxi.LoadConfig()
	if err != nil {
		return nil, nil, "", err
	}

	client := dongxi.NewClient(cfg.Email, cfg.Password)

	// Load local cache.
	cache, err := dongxi.LoadCache()
	if err != nil {
		return nil, nil, "", fmt.Errorf("load cache: %w", err)
	}

	// Decide whether to sync: skip-sync, throttle, or force.
	shouldSync := !flagSkipSync
	if shouldSync && !flagSync && cache.ItemCount > 0 {
		elapsed := time.Since(time.Unix(cache.LastSyncUnix, 0))
		if elapsed < time.Duration(cfg.SyncInterval())*time.Second {
			shouldSync = false
		}
	}

	if !shouldSync {
		if cache.ItemCount == 0 {
			return nil, nil, "", fmt.Errorf("no cached data — run 'dongxi sync' first")
		}
		items := replayHistory(cache.Items)
		s := buildState(items)
		return s, client, cache.HistoryKey, nil
	}

	acct, err := client.GetAccount(cfg.Email)
	if err != nil {
		return nil, nil, "", fmt.Errorf("fetch account: %w", err)
	}

	// Invalidate cache if history key changed (e.g. after reset).
	if cache.HistoryKey != acct.HistoryKey {
		cache = &dongxi.Cache{}
	}

	// Fetch only new items from where the cache left off.
	newItems, err := client.GetHistoryItemsFrom(acct.HistoryKey, cache.ItemCount)
	if err != nil {
		return nil, nil, "", fmt.Errorf("fetch history: %w", err)
	}

	// Merge and persist.
	cache.HistoryKey = acct.HistoryKey
	cache.LastSyncUnix = time.Now().Unix()
	if len(newItems) > 0 {
		cache.Items = append(cache.Items, newItems...)
		cache.ItemCount = len(cache.Items)
	}
	_ = dongxi.SaveCache(cache)

	items := replayHistory(cache.Items)
	s := buildState(items)
	return s, client, acct.HistoryKey, nil
}

// stateLoader is the active StateLoader. Tests replace this.
var stateLoader StateLoader = realStateLoader{}

// loadState is the convenience function used by all commands.
func loadState() (*thingsState, CloudClient, string, error) {
	return stateLoader.LoadState()
}

// thingsState holds the fully replayed state from Things Cloud history.
type thingsState struct {
	items  []replayedItem
	byUUID map[string]*replayedItem

	// Convenience lookups.
	areas    map[string]*replayedItem // uuid -> Area3
	projects map[string]*replayedItem // uuid -> Task6 with tp=1
}

// buildState creates a thingsState from replayed items.
func buildState(items []replayedItem) *thingsState {
	s := &thingsState{
		items:    items,
		byUUID:   make(map[string]*replayedItem, len(items)),
		areas:    make(map[string]*replayedItem),
		projects: make(map[string]*replayedItem),
	}
	for i := range s.items {
		item := &s.items[i]
		s.byUUID[item.uuid] = item
		switch {
		case item.entity == string(dongxi.EntityArea):
			s.areas[item.uuid] = item
		case item.entity == string(dongxi.EntityTask) && toInt(item.fields[dongxi.FieldType]) == int(dongxi.TaskTypeProject):
			s.projects[item.uuid] = item
		}
	}
	return s
}

// areaTitle returns the title for an area UUID, or "".
func (s *thingsState) areaTitle(uuid string) string {
	if a, ok := s.areas[uuid]; ok {
		return toStr(a.fields[dongxi.FieldTitle])
	}
	return ""
}

// projectTitle returns the title for a project UUID, or "".
func (s *thingsState) projectTitle(uuid string) string {
	if p, ok := s.projects[uuid]; ok {
		return toStr(p.fields[dongxi.FieldTitle])
	}
	return ""
}

// projectProgress returns (total, completed) task counts for a project.
func (s *thingsState) projectProgress(projectUUID string) (int, int) {
	total, completed := 0, 0
	for _, item := range s.items {
		if item.entity != string(dongxi.EntityTask) {
			continue
		}
		if toInt(item.fields[dongxi.FieldType]) != int(dongxi.TaskTypeTask) {
			continue
		}
		if firstString(item.fields[dongxi.FieldProjectIDs]) != projectUUID {
			continue
		}
		if toBool(item.fields[dongxi.FieldTrashed]) {
			continue
		}
		total++
		if toInt(item.fields[dongxi.FieldStatus]) == int(dongxi.TaskStatusCompleted) {
			completed++
		}
	}
	return total, completed
}

// headingsForProject returns all headings within a project, in order.
func (s *thingsState) headingsForProject(projectUUID string) []replayedItem {
	var result []replayedItem
	for _, item := range s.items {
		if item.entity != string(dongxi.EntityTask) {
			continue
		}
		if toInt(item.fields[dongxi.FieldType]) != int(dongxi.TaskTypeHeading) {
			continue
		}
		if firstString(item.fields[dongxi.FieldProjectIDs]) != projectUUID {
			continue
		}
		result = append(result, item)
	}
	return result
}

// isOrphanedByTrashedParent returns true if a task's action group or heading
// belongs to a trashed or deleted project, which causes Things to hide the task.
func (s *thingsState) isOrphanedByTrashedParent(item *replayedItem) bool {
	// Check action group: if the agr heading's parent project is trashed, the task is effectively trashed.
	for _, agrUUID := range toStringSlice(item.fields[dongxi.FieldActionGroupIDs]) {
		if heading, ok := s.byUUID[agrUUID]; ok {
			if projUUID := firstString(heading.fields[dongxi.FieldProjectIDs]); projUUID != "" {
				if proj, ok := s.byUUID[projUUID]; ok {
					if toBool(proj.fields[dongxi.FieldTrashed]) {
						return true
					}
				}
			}
		}
	}
	// Check direct project: if the task's project is trashed, the task is effectively trashed.
	if projUUID := firstString(item.fields[dongxi.FieldProjectIDs]); projUUID != "" {
		if proj, ok := s.byUUID[projUUID]; ok {
			if toBool(proj.fields[dongxi.FieldTrashed]) {
				return true
			}
		}
	}
	return false
}

// firstString returns the first string from a []any field, or "".
func firstString(v any) string {
	if arr, ok := v.([]any); ok && len(arr) > 0 {
		if s, ok := arr[0].(string); ok {
			return s
		}
	}
	return ""
}

// printJSON encodes v as indented JSON to stdout.
func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// isToday returns true if an open Anytime task belongs in the Today view.
// Things shows a task in Today when it has a todayIndex set, or when its
// scheduled date is today or earlier.
func isToday(fields map[string]any, now time.Time) bool {
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	tomorrow := todayStart.AddDate(0, 0, 1)

	// Has todayIndex set — but only counts if todayIndexRef is today.
	if ti := toFloat(fields[dongxi.FieldTodayIndex]); ti != 0 {
		if tir := toFloat(fields[dongxi.FieldTodayIndexRef]); tir > 0 {
			ref := time.Unix(int64(tir), 0).UTC()
			if !ref.Before(todayStart) && ref.Before(tomorrow) {
				return true
			}
		}
	}
	// Scheduled date is today or earlier.
	if sr := toFloat(fields[dongxi.FieldScheduledDate]); sr > 0 {
		scheduled := time.Unix(int64(sr), 0).UTC()
		if scheduled.Before(tomorrow) {
			return true
		}
	}
	return false
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// resolveUUID finds an item by full or partial UUID prefix.
func (s *thingsState) resolveUUID(query string) (*replayedItem, error) {
	if item, ok := s.byUUID[query]; ok {
		return item, nil
	}
	// Try prefix match.
	var matches []*replayedItem
	for uuid, item := range s.byUUID {
		if strings.HasPrefix(uuid, query) {
			matches = append(matches, item)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no item found matching %q", query)
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("ambiguous UUID prefix %q: matches %d items", query, len(matches))
	}
}
