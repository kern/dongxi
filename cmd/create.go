package cmd

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

var (
	flagCreateTitle       string
	flagCreateDestination string
	flagCreateNote        string
	flagCreateChecklist   string
	flagCreateScheduled   string
	flagCreateDeadline    string
	flagCreateArea        string
	flagCreateProject     string
	flagCreateType        string
	flagCreateHeading     string
	flagCreateTags    string
	flagCreateEvening bool
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new task, project, or heading",
	Long: `Create a new item in Things via the Things Cloud API.

Examples:
  dongxi create -t "Buy groceries"
  dongxi create -t "Buy groceries" -d today
  dongxi create -t "Buy groceries" --checklist "Milk,Eggs,Bread"
  dongxi create -t "Q3 Planning" --type project --area <area-uuid>
  dongxi create -t "Research" --type heading --project <project-uuid>
  dongxi create -t "Call dentist" --scheduled 2025-04-01 --deadline 2025-04-15
  dongxi create -t "Weekly review" --tags <tag-uuid>,<tag-uuid>`,
	RunE: runCreate,
}

func init() {
	createCmd.Flags().StringVarP(&flagCreateTitle, "title", "t", "", "Title (required)")
	createCmd.Flags().StringVarP(&flagCreateDestination, "destination", "d", "inbox", "Destination: today, inbox, or someday")
	createCmd.Flags().StringVarP(&flagCreateNote, "note", "n", "", "Note")
	createCmd.Flags().StringVar(&flagCreateChecklist, "checklist", "", "Comma-separated checklist items")
	createCmd.Flags().StringVar(&flagCreateScheduled, "scheduled", "", "Scheduled date (YYYY-MM-DD)")
	createCmd.Flags().StringVar(&flagCreateDeadline, "deadline", "", "Deadline date (YYYY-MM-DD)")
	createCmd.Flags().StringVar(&flagCreateArea, "area", "", "Area UUID")
	createCmd.Flags().StringVar(&flagCreateProject, "project", "", "Project UUID (assign task to a project)")
	createCmd.Flags().StringVar(&flagCreateType, "type", "task", "Type: task, project, or heading")
	createCmd.Flags().StringVar(&flagCreateHeading, "heading", "", "Heading UUID (assign task under a heading)")
	createCmd.Flags().StringVar(&flagCreateTags, "tags", "", "Comma-separated tag UUIDs")
	createCmd.Flags().BoolVar(&flagCreateEvening, "evening", false, "Add to This Evening (Today view only)")

	_ = createCmd.MarkFlagRequired("title")
}

func runCreate(cmd *cobra.Command, args []string) error {
	if strings.TrimSpace(flagCreateTitle) == "" {
		return fmt.Errorf("--title must not be empty")
	}

	// Resolve type.
	var taskType dongxi.TaskType
	switch strings.ToLower(flagCreateType) {
	case "task", "":
		taskType = dongxi.TaskTypeTask
	case "project":
		taskType = dongxi.TaskTypeProject
	case "heading":
		taskType = dongxi.TaskTypeHeading
	default:
		return fmt.Errorf("unknown type %q: must be task, project, or heading", flagCreateType)
	}

	// Resolve destination.
	var destination dongxi.TaskDestination
	switch strings.ToLower(flagCreateDestination) {
	case "inbox", "":
		destination = dongxi.TaskDestinationInbox
	case "today", "anytime":
		destination = dongxi.TaskDestinationAnytime
	case "someday":
		destination = dongxi.TaskDestinationSomeday
	default:
		return fmt.Errorf("unknown destination %q: must be today, inbox, or someday", flagCreateDestination)
	}

	// Parse optional dates.
	var scheduledTS *int64
	if flagCreateScheduled != "" {
		t, err := time.Parse("2006-01-02", flagCreateScheduled)
		if err != nil {
			return fmt.Errorf("parse --scheduled date %q: %w", flagCreateScheduled, err)
		}
		ts := t.Unix()
		scheduledTS = &ts
	}

	var deadlineTS *int64
	if flagCreateDeadline != "" {
		t, err := time.Parse("2006-01-02", flagCreateDeadline)
		if err != nil {
			return fmt.Errorf("parse --deadline date %q: %w", flagCreateDeadline, err)
		}
		ts := t.Unix()
		deadlineTS = &ts
	}

	// Parse checklist items.
	var checklistItems []string
	if flagCreateChecklist != "" {
		for _, item := range strings.Split(flagCreateChecklist, ",") {
			item = strings.TrimSpace(item)
			if item != "" {
				checklistItems = append(checklistItems, item)
			}
		}
	}

	// Parse tags.
	var tagUUIDs []string
	if flagCreateTags != "" {
		for _, t := range strings.Split(flagCreateTags, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tagUUIDs = append(tagUUIDs, t)
			}
		}
	}

	_, client, historyKey, err := loadState()
	if err != nil {
		return err
	}

	histInfo, err := client.GetHistory(historyKey)
	if err != nil {
		return fmt.Errorf("fetch history info: %w", err)
	}

	now := float64(time.Now().UnixNano()) / 1e9

	// For "today" destination, sr = UTC midnight of today.
	var srTS *int64
	if destination == dongxi.TaskDestinationAnytime {
		t := time.Now()
		midnight := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).Unix()
		srTS = &midnight
	} else if scheduledTS != nil {
		srTS = scheduledTS
	}

	taskUUID := newUUID()

	// md is null for inbox at creation, set for today/someday.
	var md any
	if destination == dongxi.TaskDestinationAnytime || destination == dongxi.TaskDestinationSomeday {
		md = now
	}

	// Build create payload.
	createPayload := map[string]any{
		dongxi.FieldTitle:              "",
		dongxi.FieldStatus:             int(dongxi.TaskStatusOpen),
		dongxi.FieldDestination:        int(destination),
		dongxi.FieldType:               int(taskType),
		dongxi.FieldCreationDate:       now,
		dongxi.FieldModificationDate:   md,
		dongxi.FieldTrashed:            false,
		dongxi.FieldIndex:              dongxi.DefaultIndex,
		dongxi.FieldTodayIndex:         dongxi.DefaultTodayIndex,
		dongxi.FieldStartBucket:        boolToInt(flagCreateEvening),
		dongxi.FieldDueOrder:           0,
		dongxi.FieldChecklistCount:     0,
		dongxi.FieldChecklistComplete:  false,
		dongxi.FieldLateTask:           false,
		dongxi.FieldAreaIDs:            []string{},
		dongxi.FieldProjectIDs:         []string{},
		dongxi.FieldTagIDs:             []string{},
		dongxi.FieldHeadingIDs:         []string{},
		dongxi.FieldActionGroupIDs:     []string{},
		dongxi.FieldReminders:          []any{},
		dongxi.FieldStopDate:           nil,
		dongxi.FieldDeadline:           nil,
		dongxi.FieldScheduledDate:      nil,
		dongxi.FieldTodayIndexRef:      nil,
		dongxi.FieldAutoTimeOffer:      nil,
		dongxi.FieldNote:               dongxi.EmptyNote,
		dongxi.FieldRepeatRule:         nil,
		dongxi.FieldRepeatPaused:       nil,
		dongxi.FieldRepeatMethodDate:   nil,
		dongxi.FieldDueDate:            nil,
		dongxi.FieldLastAlarmInteract:  nil,
		dongxi.FieldInstanceCreatedSrc: nil,
		dongxi.FieldAutoCompRepeatDate: nil,
		dongxi.FieldSyncMeta:           dongxi.SyncMeta,
	}

	if srTS != nil {
		createPayload[dongxi.FieldScheduledDate] = *srTS
		createPayload[dongxi.FieldTodayIndexRef] = *srTS
	}

	createCommit := map[string]dongxi.CommitItem{
		taskUUID: {
			T: dongxi.ItemTypeCreate,
			E: dongxi.EntityTask,
			P: createPayload,
		},
	}

	ancestorIndex := histInfo.LatestServerIndex
	resp, err := client.Commit(historyKey, ancestorIndex, createCommit)
	if err != nil {
		return fmt.Errorf("commit create: %w", err)
	}
	ancestorIndex = resp.ServerHeadIndex

	// Step 2: Modify to set title and other fields.
	modifyNow := float64(time.Now().UnixNano()) / 1e9
	modifyPayload := map[string]any{
		dongxi.FieldTitle:            flagCreateTitle,
		dongxi.FieldModificationDate: modifyNow,
	}

	if flagCreateNote != "" {
		modifyPayload[dongxi.FieldNote] = dongxi.NewNote(flagCreateNote)
	}

	if deadlineTS != nil {
		modifyPayload[dongxi.FieldDeadline] = *deadlineTS
	}

	if flagCreateArea != "" {
		modifyPayload[dongxi.FieldAreaIDs] = []string{flagCreateArea}
	}

	if flagCreateProject != "" {
		modifyPayload[dongxi.FieldProjectIDs] = []string{flagCreateProject}
	}

	if flagCreateHeading != "" {
		modifyPayload[dongxi.FieldHeadingIDs] = []string{flagCreateHeading}
	}

	if len(tagUUIDs) > 0 {
		modifyPayload[dongxi.FieldTagIDs] = tagUUIDs
	}

	modifyCommit := map[string]dongxi.CommitItem{
		taskUUID: {
			T: dongxi.ItemTypeModify,
			E: dongxi.EntityTask,
			P: modifyPayload,
		},
	}

	// Add checklist items to the modify commit.
	for i, ciTitle := range checklistItems {
		ciUUID := newUUID()
		modifyCommit[ciUUID] = dongxi.CommitItem{
			T: dongxi.ItemTypeCreate,
			E: dongxi.EntityChecklistItem,
			P: map[string]any{
				dongxi.FieldTitle:            ciTitle,
				dongxi.FieldStatus:           int(dongxi.TaskStatusOpen),
				dongxi.FieldTaskIDs:          []string{taskUUID},
				dongxi.FieldCreationDate:     modifyNow,
				dongxi.FieldModificationDate: modifyNow,
				dongxi.FieldIndex:            i,
				dongxi.FieldStopDate:         nil,
				dongxi.FieldLateTask:         false,
				dongxi.FieldSyncMeta:         dongxi.SyncMeta,
			},
		}
	}

	resp, err = client.Commit(historyKey, ancestorIndex, modifyCommit)
	if err != nil {
		return fmt.Errorf("commit modify: %w", err)
	}

	typeName := "task"
	if taskType == dongxi.TaskTypeProject {
		typeName = "project"
	} else if taskType == dongxi.TaskTypeHeading {
		typeName = "heading"
	}

	if flagJSON {
		return printJSON(CreateOutput{
			UUID:           taskUUID,
			Type:           typeName,
			Title:          flagCreateTitle,
			ChecklistCount: len(checklistItems),
			ServerIndex:    resp.ServerHeadIndex,
		})
	}

	fmt.Printf("Created %s: %q  [%s]\n", typeName, flagCreateTitle, taskUUID)
	if len(checklistItems) > 0 {
		fmt.Printf("  with %d checklist item(s)\n", len(checklistItems))
	}
	fmt.Printf("Server index: %d\n", resp.ServerHeadIndex)

	return nil
}

// newUUID generates a Things-style 22-character Base58-encoded random ID.
func newUUID() string {
	const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		panic(err)
	}
	n := new(big.Int).SetBytes(buf[:])
	base := big.NewInt(58)
	rem := new(big.Int)
	var result []byte
	for n.Sign() > 0 {
		n.DivMod(n, base, rem)
		result = append(result, alphabet[rem.Int64()])
	}
	for _, b := range buf {
		if b == 0 {
			result = append(result, '1')
		} else {
			break
		}
	}
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return string(result)
}
