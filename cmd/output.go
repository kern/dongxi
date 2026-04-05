package cmd

import (
	"time"

	"github.com/kern/dongxi/dongxi"
)

// ItemOutput is the JSON representation of a Things item (task, project, area, tag, checklist item).
type ItemOutput struct {
	UUID   string `json:"uuid"`
	Entity string `json:"entity"`
	Title  string `json:"title,omitempty"`

	// Task/project/heading fields.
	Type        string   `json:"type,omitempty"`
	Status      string   `json:"status,omitempty"`
	Destination string   `json:"destination,omitempty"`
	Trashed     *bool    `json:"trashed,omitempty"`
	AreaUUID    string   `json:"area_uuid,omitempty"`
	Area        string   `json:"area,omitempty"`
	ProjectUUID string   `json:"project_uuid,omitempty"`
	Project     string   `json:"project,omitempty"`
	Created     string   `json:"created,omitempty"`
	Modified    string   `json:"modified,omitempty"`
	Scheduled   string   `json:"scheduled,omitempty"`
	Deadline    string   `json:"deadline,omitempty"`
	CompletedAt string   `json:"completed_at,omitempty"`
	Notes       string   `json:"notes,omitempty"`
	Tags        []string `json:"tags,omitempty"`

	Evening *bool `json:"evening,omitempty"`

	// Project progress.
	TasksTotal     *int `json:"tasks_total,omitempty"`
	TasksCompleted *int `json:"tasks_completed,omitempty"`

	// Heading fields.
	HeadingUUID string `json:"heading_uuid,omitempty"`
	Heading     string `json:"heading,omitempty"`

	// Checklist item fields.
	TaskUUID string `json:"task_uuid,omitempty"`

	// Nested checklist (used by show command).
	Checklist []ItemOutput `json:"checklist,omitempty"`
}

// itemToOutput converts a replayedItem to a typed ItemOutput.
func (s *thingsState) itemToOutput(item *replayedItem) ItemOutput {
	out := ItemOutput{
		UUID:   item.uuid,
		Entity: item.entity,
		Title:  toStr(item.fields[dongxi.FieldTitle]),
	}

	f := item.fields

	if item.entity == string(dongxi.EntityTask) {
		switch dongxi.TaskType(toInt(f[dongxi.FieldType])) {
		case dongxi.TaskTypeTask:
			out.Type = "task"
		case dongxi.TaskTypeProject:
			out.Type = "project"
		case dongxi.TaskTypeHeading:
			out.Type = "heading"
		}

		switch dongxi.TaskStatus(toInt(f[dongxi.FieldStatus])) {
		case dongxi.TaskStatusOpen:
			out.Status = "open"
		case dongxi.TaskStatusCancelled:
			out.Status = "cancelled"
		case dongxi.TaskStatusCompleted:
			out.Status = "completed"
		}

		switch dongxi.TaskDestination(toInt(f[dongxi.FieldDestination])) {
		case dongxi.TaskDestinationInbox:
			out.Destination = "inbox"
		case dongxi.TaskDestinationAnytime:
			out.Destination = "today"
		case dongxi.TaskDestinationSomeday:
			out.Destination = "someday"
		}

		trashed := toBool(f[dongxi.FieldTrashed])
		out.Trashed = &trashed

		areaUUID := firstString(f[dongxi.FieldAreaIDs])
		if projUUID := firstString(f[dongxi.FieldProjectIDs]); projUUID != "" {
			out.ProjectUUID = projUUID
			out.Project = s.projectTitle(projUUID)
			// Inherit area from project if task has none.
			if areaUUID == "" {
				if proj, ok := s.projects[projUUID]; ok {
					areaUUID = firstString(proj.fields[dongxi.FieldAreaIDs])
				}
			}
		}
		if areaUUID != "" {
			out.AreaUUID = areaUUID
			out.Area = s.areaTitle(areaUUID)
		}

		if headingUUID := firstString(f[dongxi.FieldActionGroupIDs]); headingUUID != "" {
			if heading, ok := s.byUUID[headingUUID]; ok {
				if toInt(heading.fields[dongxi.FieldType]) == int(dongxi.TaskTypeHeading) {
					out.HeadingUUID = headingUUID
					out.Heading = toStr(heading.fields[dongxi.FieldTitle])
				}
			}
		}

		if cd := toFloat(f[dongxi.FieldCreationDate]); cd > 0 {
			out.Created = time.Unix(int64(cd), 0).UTC().Format(time.RFC3339)
		}
		if md := toFloat(f[dongxi.FieldModificationDate]); md > 0 {
			out.Modified = time.Unix(int64(md), 0).UTC().Format(time.RFC3339)
		}
		if sr := toFloat(f[dongxi.FieldScheduledDate]); sr > 0 {
			out.Scheduled = time.Unix(int64(sr), 0).UTC().Format("2006-01-02")
		}
		if dd := toFloat(f[dongxi.FieldDeadline]); dd > 0 {
			out.Deadline = time.Unix(int64(dd), 0).UTC().Format("2006-01-02")
		}
		if sp := toFloat(f[dongxi.FieldStopDate]); sp > 0 {
			out.CompletedAt = time.Unix(int64(sp), 0).UTC().Format(time.RFC3339)
		}
		out.Notes = dongxi.NoteText(f[dongxi.FieldNote])
		if tg := toStringSlice(f[dongxi.FieldTagIDs]); len(tg) > 0 {
			out.Tags = tg
		}
		evening := toInt(f[dongxi.FieldStartBucket]) == 1
		out.Evening = &evening
	} else if item.entity == string(dongxi.EntityArea) {
		out.Type = "area"
		trashed := toBool(f[dongxi.FieldTrashed])
		out.Trashed = &trashed
	} else if item.entity == string(dongxi.EntityTag) {
		out.Type = "tag"
	} else if item.entity == string(dongxi.EntityChecklistItem) {
		out.Type = "checklist_item"
		switch dongxi.TaskStatus(toInt(f[dongxi.FieldStatus])) {
		case dongxi.TaskStatusOpen:
			out.Status = "open"
		case dongxi.TaskStatusCompleted:
			out.Status = "completed"
		}
		out.TaskUUID = firstString(f[dongxi.FieldTaskIDs])
	}

	return out
}

// ActionItemOutput represents a single item in a bulk action result.
type ActionItemOutput struct {
	UUID   string `json:"uuid"`
	Title  string `json:"title"`
	Action string `json:"action"`
}

// BulkActionOutput is the JSON result for complete, cancel, trash commands.
type BulkActionOutput struct {
	Items       []ActionItemOutput `json:"items"`
	ServerIndex int                `json:"server_index"`
}

// CreateOutput is the JSON result for the create command.
type CreateOutput struct {
	UUID           string `json:"uuid"`
	Type           string `json:"type"`
	Title          string `json:"title"`
	ChecklistCount int    `json:"checklist_count"`
	ServerIndex    int    `json:"server_index"`
}

// EditOutput is the JSON result for the edit and move commands.
type EditOutput struct {
	UUID        string   `json:"uuid"`
	Title       string   `json:"title"`
	Changes     []string `json:"changes"`
	ServerIndex int      `json:"server_index"`
}

// ReorderOutput is the JSON result for the reorder command.
type ReorderOutput struct {
	UUID        string `json:"uuid"`
	Title       string `json:"title"`
	Index       int    `json:"index"`
	ServerIndex int    `json:"server_index"`
}

// RepeatOutput is the JSON result for the repeat command.
type RepeatOutput struct {
	UUID        string `json:"uuid"`
	Title       string `json:"title"`
	Action      string `json:"action"`
	ServerIndex int    `json:"server_index"`
}

// TagActionOutput is the JSON result for tag/untag commands.
type TagActionOutput struct {
	TaskUUID    string `json:"task_uuid"`
	TagUUID     string `json:"tag_uuid"`
	Action      string `json:"action"`
	ServerIndex int    `json:"server_index"`
}

// ChecklistActionOutput is the JSON result for checklist add/complete/remove.
type ChecklistActionOutput struct {
	UUID        string `json:"uuid"`
	Title       string `json:"title"`
	TaskUUID    string `json:"task_uuid,omitempty"`
	Action      string `json:"action"`
	ServerIndex int    `json:"server_index"`
}

// InfoOutput is the JSON result for the info command.
type InfoOutput struct {
	Email         string `json:"email"`
	Status        string `json:"status"`
	MaildropEmail string `json:"maildrop_email"`
	HistoryKey    string `json:"history_key"`
	ServerIndex   int    `json:"server_index"`
	SchemaVersion int    `json:"schema_version"`
	IsEmpty       bool   `json:"is_empty"`
	ContentSize   int    `json:"content_size"`
}

// BatchOpOutput represents one item in a batch result.
type BatchOpOutput struct {
	UUID         string   `json:"uuid"`
	Descriptions []string `json:"descriptions"`
	Payload      any      `json:"payload,omitempty"`
}

// BatchOutput is the JSON result for the batch command.
type BatchOutput struct {
	DryRun      bool            `json:"dry_run,omitempty"`
	Committed   int             `json:"committed,omitempty"`
	Count       int             `json:"count,omitempty"`
	Operations  []BatchOpOutput `json:"operations"`
	ServerIndex int             `json:"server_index,omitempty"`
}
