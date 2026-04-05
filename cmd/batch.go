package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

var flagBatchDryRun bool

var batchCmd = &cobra.Command{
	Use:   "batch",
	Short: "Execute multiple operations in a single API call",
	Long: `Read a JSON array of operations from stdin and execute them all in one commit.

Instead of running many individual commands (each loading full state from the API),
batch combines everything into a single state load and a single API commit. This is
dramatically faster and avoids rate limiting.

INPUT FORMAT

The input is a JSON array of operation objects. Every object must have "op" and "uuid".
Additional fields depend on the operation. Fields marked with ? are optional.

  [
    {"op": "complete", "uuid": "<task-uuid>"},
    {"op": "reopen",   "uuid": "<task-uuid>"},
    {"op": "cancel",   "uuid": "<task-uuid>"},
    {"op": "trash",    "uuid": "<task-uuid>"},
    {"op": "untrash",  "uuid": "<task-uuid>"},
    {"op": "move",     "uuid": "<task-uuid>", "destination": "today", "area": "<area-uuid>", "project": "<project-uuid>"},
    {"op": "tag",      "uuid": "<task-uuid>", "tag": "<tag-uuid>"},
    {"op": "untag",    "uuid": "<task-uuid>", "tag": "<tag-uuid>"},
    {"op": "edit",     "uuid": "<task-uuid>", "title": "New title", "note": "New note", "scheduled": "2026-04-01", "deadline": "2026-05-01"},
    {"op": "convert",  "uuid": "<task-uuid>", "to": "project"}
  ]

OPERATIONS

  complete   Mark a task as completed.
             Fields: uuid

  reopen     Reopen a completed or cancelled task.
             Fields: uuid

  cancel     Cancel a task (mark as cancelled).
             Fields: uuid

  trash      Move a task to the trash.
             Fields: uuid

  untrash    Restore a task from the trash.
             Fields: uuid

  move       Move a task to a different area, project, heading, or destination.
             Fields: uuid, area?, project?, heading?, destination?
             At least one of area/project/heading/destination is required.
             Set area, project, or heading to "" to clear it.
             Destination must be "inbox", "today", or "someday".

  tag        Add a tag to a task.
             Fields: uuid, tag (the tag's UUID)

  untag      Remove a tag from a task.
             Fields: uuid, tag (the tag's UUID)

  edit       Edit a task's title, note, scheduled date, or deadline.
             Fields: uuid, title?, note?, scheduled?, deadline?
             At least one field is required.
             Set note/scheduled/deadline to "" to clear it.
             Dates use YYYY-MM-DD format.

  convert    Convert a task to a project or vice versa.
             Fields: uuid, to (must be "task" or "project")

MERGING

Multiple operations on the same UUID are merged into one commit entry.
For example, moving a task to today AND tagging it produces a single API
write. Tag/untag operations accumulate correctly — you can add and remove
multiple tags on the same task in one batch.

EXAMPLES

  Move all inbox tasks to today:
    dongxi list --json -f inbox \
      | jq '[.[] | {op: "move", uuid: .uuid, destination: "today"}]' \
      | dongxi batch

  Move inbox to today and assign to an area:
    dongxi list --json -f inbox \
      | jq '[.[] | {op: "move", uuid: .uuid, destination: "today", area: "AREA_UUID"}]' \
      | dongxi batch

  Complete all tasks in a project:
    dongxi list --json -f all --project <project-uuid> \
      | jq '[.[] | {op: "complete", uuid: .uuid}]' \
      | dongxi batch

  Tag all tasks matching a search:
    dongxi search --json "grocery" \
      | jq '[.[] | {op: "tag", uuid: .uuid, tag: "TAG_UUID"}]' \
      | dongxi batch

  Mix different operations:
    echo '[
      {"op": "complete", "uuid": "abc123"},
      {"op": "move", "uuid": "def456", "destination": "today"},
      {"op": "tag", "uuid": "def456", "tag": "ghi789"}
    ]' | dongxi batch

  Preview what would happen without committing:
    dongxi list --json -f inbox \
      | jq '[.[] | {op: "move", uuid: .uuid, destination: "today"}]' \
      | dongxi batch --dry-run

  Read operations from a file:
    dongxi batch < operations.json`,
	RunE: runBatch,
}

func init() {
	batchCmd.Flags().BoolVar(&flagBatchDryRun, "dry-run", false, "Preview operations without committing")
}

type batchOp struct {
	Op          string  `json:"op"`
	UUID        string  `json:"uuid"`
	Area        *string `json:"area,omitempty"`
	Project     *string `json:"project,omitempty"`
	Heading     *string `json:"heading,omitempty"`
	Destination string  `json:"destination,omitempty"`
	Tag         string  `json:"tag,omitempty"`
	Title       *string `json:"title,omitempty"`
	Note        *string `json:"note,omitempty"`
	Scheduled   *string `json:"scheduled,omitempty"`
	Deadline    *string `json:"deadline,omitempty"`
	To          string  `json:"to,omitempty"` // for convert op
}

// batchAccum accumulates commit fields for a single UUID.
type batchAccum struct {
	uuid    string
	entity  dongxi.EntityType
	payload map[string]any
	// Tag tracking: we need to build the final tag list from current state + adds - removes.
	tagsToAdd    []string
	tagsToRemove []string
	hasTags      bool
	descriptions []string
}

func runBatch(cmd *cobra.Command, args []string) error {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}

	data = trimBOM(data)
	if len(strings.TrimSpace(string(data))) == 0 {
		fmt.Println("Nothing to do (empty input).")
		return nil
	}

	var ops []batchOp
	if err := json.Unmarshal(data, &ops); err != nil {
		return fmt.Errorf("parse JSON: expected a JSON array of operations (e.g. [{\"op\":\"complete\",\"uuid\":\"...\"}]): %w", err)
	}

	if len(ops) == 0 {
		fmt.Println("Nothing to do (empty array).")
		return nil
	}

	s, client, historyKey, err := loadState()
	if err != nil {
		return err
	}

	histInfo, err := client.GetHistory(historyKey)
	if err != nil {
		return fmt.Errorf("fetch history info: %w", err)
	}

	now := float64(time.Now().UnixNano()) / 1e9

	// Accumulate operations per UUID.
	accums := map[string]*batchAccum{}
	getAccum := func(uuid string, entity dongxi.EntityType) *batchAccum {
		a, ok := accums[uuid]
		if !ok {
			a = &batchAccum{
				uuid:    uuid,
				entity:  entity,
				payload: map[string]any{dongxi.FieldModificationDate: now},
			}
			accums[uuid] = a
		}
		return a
	}

	validOps := map[string]bool{
		"complete": true, "reopen": true, "cancel": true, "trash": true, "untrash": true,
		"move": true, "tag": true, "untag": true, "edit": true, "convert": true,
	}

	for i, op := range ops {
		if op.Op == "" {
			return fmt.Errorf("operation %d: missing op field", i)
		}
		if !validOps[op.Op] {
			return fmt.Errorf("operation %d: unknown op %q (valid: complete, reopen, cancel, trash, move, tag, untag, edit)", i, op.Op)
		}
		if op.UUID == "" {
			return fmt.Errorf("operation %d (%s): missing uuid", i, op.Op)
		}

		item, err := s.resolveUUID(op.UUID)
		if err != nil {
			return fmt.Errorf("operation %d (%s): %w", i, op.Op, err)
		}

		title := toStr(item.fields[dongxi.FieldTitle])
		if title == "" {
			title = "(untitled)"
		}

		switch op.Op {
		case "complete":
			if item.entity != string(dongxi.EntityTask) {
				return fmt.Errorf("operation %d: %s is not a task", i, op.UUID)
			}
			a := getAccum(item.uuid, dongxi.EntityTask)
			a.payload[dongxi.FieldStatus] = int(dongxi.TaskStatusCompleted)
			a.payload[dongxi.FieldStopDate] = now
			a.descriptions = append(a.descriptions, fmt.Sprintf("complete %q", title))

		case "reopen":
			if item.entity != string(dongxi.EntityTask) {
				return fmt.Errorf("operation %d: %s is not a task", i, op.UUID)
			}
			a := getAccum(item.uuid, dongxi.EntityTask)
			a.payload[dongxi.FieldStatus] = int(dongxi.TaskStatusOpen)
			a.payload[dongxi.FieldStopDate] = nil
			a.descriptions = append(a.descriptions, fmt.Sprintf("reopen %q", title))

		case "cancel":
			if item.entity != string(dongxi.EntityTask) {
				return fmt.Errorf("operation %d: %s is not a task", i, op.UUID)
			}
			a := getAccum(item.uuid, dongxi.EntityTask)
			a.payload[dongxi.FieldStatus] = int(dongxi.TaskStatusCancelled)
			a.payload[dongxi.FieldStopDate] = now
			a.descriptions = append(a.descriptions, fmt.Sprintf("cancel %q", title))

		case "trash":
			if item.entity != string(dongxi.EntityTask) {
				return fmt.Errorf("operation %d: %s is not a task", i, op.UUID)
			}
			a := getAccum(item.uuid, dongxi.EntityTask)
			a.payload[dongxi.FieldTrashed] = true
			a.descriptions = append(a.descriptions, fmt.Sprintf("trash %q", title))

		case "untrash":
			if item.entity != string(dongxi.EntityTask) && item.entity != string(dongxi.EntityArea) {
				return fmt.Errorf("operation %d: %s is not a task or area", i, op.UUID)
			}
			a := getAccum(item.uuid, dongxi.EntityType(item.entity))
			a.payload[dongxi.FieldTrashed] = false
			a.descriptions = append(a.descriptions, fmt.Sprintf("untrash %q", title))

		case "move":
			if item.entity != string(dongxi.EntityTask) {
				return fmt.Errorf("operation %d: %s is not a task", i, op.UUID)
			}
			a := getAccum(item.uuid, dongxi.EntityTask)
			changes := []string{}

			if op.Area != nil {
				if *op.Area == "" {
					a.payload[dongxi.FieldAreaIDs] = []string{}
					changes = append(changes, "clear area")
				} else {
					area, err := s.resolveUUID(*op.Area)
					if err != nil {
						return fmt.Errorf("operation %d: resolve area: %w", i, err)
					}
					if area.entity != string(dongxi.EntityArea) {
						return fmt.Errorf("operation %d: %s is not an area", i, *op.Area)
					}
					a.payload[dongxi.FieldAreaIDs] = []string{area.uuid}
					changes = append(changes, fmt.Sprintf("area -> %s", toStr(area.fields[dongxi.FieldTitle])))
				}
			}

			if op.Project != nil {
				if *op.Project == "" {
					a.payload[dongxi.FieldProjectIDs] = []string{}
					a.payload[dongxi.FieldActionGroupIDs] = []any{}
					a.payload[dongxi.FieldHeadingIDs] = []any{}
					changes = append(changes, "clear project")
				} else {
					proj, err := s.resolveUUID(*op.Project)
					if err != nil {
						return fmt.Errorf("operation %d: resolve project: %w", i, err)
					}
					if toInt(proj.fields[dongxi.FieldType]) != int(dongxi.TaskTypeProject) {
						return fmt.Errorf("operation %d: %s is not a project", i, *op.Project)
					}
					a.payload[dongxi.FieldProjectIDs] = []string{proj.uuid}
					changes = append(changes, fmt.Sprintf("project -> %s", toStr(proj.fields[dongxi.FieldTitle])))
				}
			}

			if op.Destination != "" {
				switch strings.ToLower(op.Destination) {
				case "inbox":
					a.payload[dongxi.FieldDestination] = int(dongxi.TaskDestinationInbox)
					a.payload[dongxi.FieldScheduledDate] = nil
					a.payload[dongxi.FieldTodayIndexRef] = nil
					changes = append(changes, "destination -> inbox")
				case "today", "anytime":
					a.payload[dongxi.FieldDestination] = int(dongxi.TaskDestinationAnytime)
					t := time.Now()
					midnight := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).Unix()
					a.payload[dongxi.FieldScheduledDate] = midnight
					a.payload[dongxi.FieldTodayIndexRef] = midnight
					changes = append(changes, "destination -> today")
				case "someday":
					a.payload[dongxi.FieldDestination] = int(dongxi.TaskDestinationSomeday)
					a.payload[dongxi.FieldScheduledDate] = nil
					a.payload[dongxi.FieldTodayIndexRef] = nil
					changes = append(changes, "destination -> someday")
				default:
					return fmt.Errorf("operation %d: unknown destination %q", i, op.Destination)
				}
			}

			if op.Heading != nil {
				if *op.Heading == "" {
					a.payload[dongxi.FieldHeadingIDs] = []any{}
					a.payload[dongxi.FieldActionGroupIDs] = []any{}
					changes = append(changes, "clear heading")
				} else {
					heading, err := s.resolveUUID(*op.Heading)
					if err != nil {
						return fmt.Errorf("operation %d: resolve heading: %w", i, err)
					}
					if toInt(heading.fields[dongxi.FieldType]) != int(dongxi.TaskTypeHeading) {
						return fmt.Errorf("operation %d: %s is not a heading", i, *op.Heading)
					}
					a.payload[dongxi.FieldHeadingIDs] = []string{heading.uuid}
					a.payload[dongxi.FieldActionGroupIDs] = []string{heading.uuid}
					changes = append(changes, fmt.Sprintf("heading -> %s", toStr(heading.fields[dongxi.FieldTitle])))
				}
			}

			if len(changes) == 0 {
				return fmt.Errorf("operation %d: move requires at least one of area, project, heading, or destination", i)
			}
			a.descriptions = append(a.descriptions, fmt.Sprintf("move %q: %s", title, strings.Join(changes, ", ")))

		case "tag":
			if item.entity != string(dongxi.EntityTask) {
				return fmt.Errorf("operation %d: %s is not a task", i, op.UUID)
			}
			if op.Tag == "" {
				return fmt.Errorf("operation %d: tag op requires a tag UUID", i)
			}
			tag, err := s.resolveUUID(op.Tag)
			if err != nil {
				return fmt.Errorf("operation %d: resolve tag: %w", i, err)
			}
			if tag.entity != string(dongxi.EntityTag) {
				return fmt.Errorf("operation %d: %s is not a tag", i, op.Tag)
			}
			a := getAccum(item.uuid, dongxi.EntityTask)
			a.tagsToAdd = append(a.tagsToAdd, tag.uuid)
			a.hasTags = true
			a.descriptions = append(a.descriptions, fmt.Sprintf("tag %q + %s", title, toStr(tag.fields[dongxi.FieldTitle])))

		case "untag":
			if item.entity != string(dongxi.EntityTask) {
				return fmt.Errorf("operation %d: %s is not a task", i, op.UUID)
			}
			if op.Tag == "" {
				return fmt.Errorf("operation %d: untag op requires a tag UUID", i)
			}
			tag, err := s.resolveUUID(op.Tag)
			if err != nil {
				return fmt.Errorf("operation %d: resolve tag: %w", i, err)
			}
			a := getAccum(item.uuid, dongxi.EntityTask)
			a.tagsToRemove = append(a.tagsToRemove, tag.uuid)
			a.hasTags = true
			a.descriptions = append(a.descriptions, fmt.Sprintf("untag %q - %s", title, toStr(tag.fields[dongxi.FieldTitle])))

		case "edit":
			if item.entity != string(dongxi.EntityTask) {
				return fmt.Errorf("operation %d: %s is not a task", i, op.UUID)
			}
			a := getAccum(item.uuid, dongxi.EntityTask)
			changes := []string{}

			if op.Title != nil {
				a.payload[dongxi.FieldTitle] = *op.Title
				changes = append(changes, fmt.Sprintf("title -> %q", *op.Title))
			}

			if op.Note != nil {
				if *op.Note == "" {
					a.payload[dongxi.FieldNote] = dongxi.EmptyNote
					changes = append(changes, "clear note")
				} else {
					a.payload[dongxi.FieldNote] = dongxi.NewNote(*op.Note)
					changes = append(changes, "update note")
				}
			}

			if op.Scheduled != nil {
				if *op.Scheduled == "" {
					a.payload[dongxi.FieldScheduledDate] = nil
					a.payload[dongxi.FieldTodayIndexRef] = nil
					changes = append(changes, "clear scheduled")
				} else {
					t, err := time.Parse("2006-01-02", *op.Scheduled)
					if err != nil {
						return fmt.Errorf("operation %d: parse scheduled date %q: %w", i, *op.Scheduled, err)
					}
					ts := t.Unix()
					a.payload[dongxi.FieldScheduledDate] = ts
					a.payload[dongxi.FieldTodayIndexRef] = ts
					changes = append(changes, fmt.Sprintf("scheduled -> %s", *op.Scheduled))
				}
			}

			if op.Deadline != nil {
				if *op.Deadline == "" {
					a.payload[dongxi.FieldDeadline] = nil
					changes = append(changes, "clear deadline")
				} else {
					t, err := time.Parse("2006-01-02", *op.Deadline)
					if err != nil {
						return fmt.Errorf("operation %d: parse deadline date %q: %w", i, *op.Deadline, err)
					}
					a.payload[dongxi.FieldDeadline] = t.Unix()
					changes = append(changes, fmt.Sprintf("deadline -> %s", *op.Deadline))
				}
			}

			if len(changes) == 0 {
				return fmt.Errorf("operation %d: edit requires at least one of title, note, scheduled, or deadline", i)
			}
			a.descriptions = append(a.descriptions, fmt.Sprintf("edit %q: %s", title, strings.Join(changes, ", ")))

		case "convert":
			if item.entity != string(dongxi.EntityTask) {
				return fmt.Errorf("operation %d: %s is not a task or project", i, op.UUID)
			}
			if op.To == "" {
				return fmt.Errorf("operation %d: convert requires a 'to' field (task or project)", i)
			}
			var targetType dongxi.TaskType
			switch op.To {
			case "task":
				targetType = dongxi.TaskTypeTask
			case "project":
				targetType = dongxi.TaskTypeProject
			default:
				return fmt.Errorf("operation %d: unknown convert target %q", i, op.To)
			}
			a := getAccum(item.uuid, dongxi.EntityTask)
			a.payload[dongxi.FieldType] = int(targetType)
			a.descriptions = append(a.descriptions, fmt.Sprintf("convert %q -> %s", title, op.To))

		default:
			return fmt.Errorf("operation %d: unknown op %q (valid: complete, reopen, cancel, trash, untrash, move, tag, untag, edit, convert)", i, op.Op)
		}
	}

	// Resolve tag accumulations against current state.
	for uuid, a := range accums {
		if !a.hasTags {
			continue
		}
		item := s.byUUID[uuid]
		if item == nil {
			continue
		}
		currentTags := toStringSlice(item.fields[dongxi.FieldTagIDs])

		// Build tag set.
		tagSet := map[string]bool{}
		for _, t := range currentTags {
			tagSet[t] = true
		}
		for _, t := range a.tagsToAdd {
			tagSet[t] = true
		}
		for _, t := range a.tagsToRemove {
			delete(tagSet, t)
		}

		var finalTags []string
		for t := range tagSet {
			finalTags = append(finalTags, t)
		}
		a.payload[dongxi.FieldTagIDs] = finalTags
	}

	// Dry run.
	if flagBatchDryRun {
		if flagJSON {
			var ops []BatchOpOutput
			for _, a := range accums {
				ops = append(ops, BatchOpOutput{
					UUID:         a.uuid,
					Descriptions: a.descriptions,
					Payload:      a.payload,
				})
			}
			return printJSON(BatchOutput{DryRun: true, Operations: ops, Count: len(accums)})
		}

		fmt.Printf("Dry run: %d item(s) would be modified\n\n", len(accums))
		for _, a := range accums {
			for _, d := range a.descriptions {
				fmt.Printf("  %s\n", d)
			}
		}
		return nil
	}

	// Build commit.
	commit := map[string]dongxi.CommitItem{}
	for uuid, a := range accums {
		commit[uuid] = dongxi.CommitItem{
			T: dongxi.ItemTypeModify,
			E: a.entity,
			P: a.payload,
		}
	}

	resp, err := client.Commit(historyKey, histInfo.LatestServerIndex, commit)
	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	if flagJSON {
		var ops []BatchOpOutput
		for _, a := range accums {
			ops = append(ops, BatchOpOutput{
				UUID:         a.uuid,
				Descriptions: a.descriptions,
			})
		}
		return printJSON(BatchOutput{
			Committed:   len(accums),
			Operations:  ops,
			ServerIndex: resp.ServerHeadIndex,
		})
	}

	for _, a := range accums {
		for _, d := range a.descriptions {
			fmt.Printf("  %s\n", d)
		}
	}
	fmt.Printf("\n%d item(s) committed. Server index: %d\n", len(accums), resp.ServerHeadIndex)
	return nil
}

// trimBOM removes a UTF-8 BOM if present.
func trimBOM(data []byte) []byte {
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return data[3:]
	}
	return data
}
