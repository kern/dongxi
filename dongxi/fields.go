package dongxi

// Things Cloud API field key reference.
//
// All items in the history use short two-to-four character keys for fields.
// This file documents every known field and provides named constants.
//
// Entity types:
//   Task6            - Tasks, projects, and headings
//   ChecklistItem3   - Checklist items within tasks
//   Area3            - Areas of responsibility
//   Tag4             - Tags
//
// Operation types (the "t" field on each history entry):
//   0 = Create   - Creates a new item with all fields in "p"
//   1 = Modify   - Updates specific fields in "p" (merge semantics)
//   2 = Delete   - Removes the item entirely

// Commit structure keys (top-level keys on each history entry).
const (
	CommitKeyType    = "t" // int: operation type (0=create, 1=modify, 2=delete)
	CommitKeyEntity  = "e" // string: entity type (Task6, Area3, etc.)
	CommitKeyPayload = "p" // map: field payload
)

// Field keys for Task6 entities.
const (
	FieldTitle              = "tt"   // string: task title
	FieldStatus             = "ss"   // int: 0=open, 2=cancelled, 3=completed
	FieldDestination        = "st"   // int: 0=inbox, 1=anytime/today, 2=someday
	FieldType               = "tp"   // int: 0=task, 1=project, 2=heading
	FieldCreationDate       = "cd"   // float64: unix timestamp of creation
	FieldModificationDate   = "md"   // float64: unix timestamp of last modification (nil for new inbox tasks)
	FieldTrashed            = "tr"   // bool: whether the item is in the trash
	FieldIndex              = "ix"   // int: ordering index within its list (lower = higher in list)
	FieldTodayIndex         = "ti"   // int: ordering index within Today view
	FieldStopDate           = "sp"   // float64: unix timestamp when completed/cancelled (nil if open)
	FieldScheduledDate      = "sr"   // int64: unix timestamp of scheduled date (UTC midnight)
	FieldDeadline           = "dd"   // int64: unix timestamp of deadline date (UTC midnight)
	FieldNote               = "nt"   // map: note object {"_t":"tx", "ch":<len>, "v":"<text>", "t":1}
	FieldAreaIDs            = "ar"   // []string: area UUID(s) this item belongs to
	FieldProjectIDs         = "pr"   // []string: project UUID(s) this item belongs to
	FieldTagIDs             = "tg"   // []string: tag UUID(s) applied to this item
	FieldHeadingIDs         = "dl"   // []string: heading UUID(s) this item is under (within a project)
	FieldActionGroupIDs     = "agr"  // []string: action group UUIDs
	FieldRepeatRule         = "rr"   // map: repeat rule (see RepeatRule below), nil if not repeating
	FieldRepeatPaused       = "rp"   // any: repeat paused state
	FieldRepeatMethodDate   = "rmd"  // any: repeat method date
	FieldTodayIndexRef      = "tir"  // int64: today index reference timestamp (matches sr for today tasks)
	FieldDueDate            = "dds"  // any: due date state
	FieldLastAlarmInteract  = "lai"  // any: last alarm interaction
	FieldInstanceCreatedSrc = "icsd" // any: instance creation source date
	FieldAutoCompRepeatDate = "acrd" // any: auto-complete repeat date
	FieldSyncMeta           = "xx"   // map: sync metadata, always {"sn":{}, "_t":"oo"}
	FieldStartBucket        = "sb"   // int: 0=Today (morning), 1=This Evening
	FieldDueOrder           = "do"   // int: due/overdue ordering
	FieldChecklistCount     = "icc"  // int: total checklist item count
	FieldChecklistComplete  = "icp"  // bool: whether all checklist items are complete
	FieldLateTask           = "lt"   // bool: whether task is late/overdue
	FieldReminders          = "rt"   // []any: reminder objects
	FieldAutoTimeOffer      = "ato"  // any: auto time offer
)

// Field keys for ChecklistItem3 entities.
const (
	// FieldTitle is shared (tt)
	// FieldStatus is shared (ss): 0=open, 3=completed
	// FieldStopDate is shared (sp)
	// FieldCreationDate is shared (cd)
	// FieldModificationDate is shared (md)
	// FieldIndex is shared (ix)
	// FieldLateTask is shared (lt)
	// FieldSyncMeta is shared (xx)
	FieldTaskIDs = "ts" // []string: parent task UUID(s) this checklist item belongs to
)

// Field keys for Area3 entities.
// Uses: FieldTitle (tt), FieldTrashed (tr), FieldIndex (ix), FieldSyncMeta (xx)

// Field keys for Tag4 entities.
// Uses: FieldTitle (tt), FieldIndex (ix), FieldSyncMeta (xx)
// Also has: "sh" (shortcut key)
const (
	FieldShortcut = "sh" // string: keyboard shortcut for the tag
)

// RepeatRule field keys (nested inside FieldRepeatRule).
//
// Example: repeat every 2 weeks on Mondays
//
//	{
//	  "rrv": 4,            // repeat rule version
//	  "tp":  1,            // 0=after completion, 1=fixed schedule
//	  "fu":  256,          // frequency unit: 8=monthly, 16=daily, 256=weekly
//	  "fa":  2,            // frequency amount (every N units)
//	  "of":  [{"wd": 1}], // offset: wd=weekday (0=Sun..6=Sat), dy=day of month
//	  "ia":  1609459200,   // instance anchor (start date, unix timestamp)
//	  "sr":  1609459200,   // scheduled reference (unix timestamp)
//	  "ed":  64092211200,  // end date (far future = never ends)
//	  "rc":  0,            // repeat count (0 = unlimited)
//	  "ts":  0,            // time shift in days (negative = remind before)
//	}
const (
	RepeatVersion       = "rrv" // int: always 4
	RepeatType          = "tp"  // int: 0=after completion, 1=fixed schedule
	RepeatFreqUnit      = "fu"  // int: 8=monthly, 16=daily, 256=weekly
	RepeatFreqAmount    = "fa"  // int: every N units
	RepeatOffset        = "of"  // []map: offset array (dy=day of month, wd=weekday)
	RepeatAnchor        = "ia"  // int64: instance anchor timestamp
	RepeatScheduledRef  = "sr"  // int64: scheduled reference timestamp
	RepeatEndDate       = "ed"  // int64: end date (64092211200 = never)
	RepeatCount         = "rc"  // int: max repetitions (0 = unlimited)
	RepeatTimeShift     = "ts"  // int: days to shift reminder (negative = before)
)

// Repeat offset keys (nested inside RepeatOffset array elements).
const (
	OffsetDay     = "dy" // int: day of month (1-31) or 0 for daily
	OffsetWeekday = "wd" // int: day of week (0=Sunday .. 6=Saturday)
)

// Frequency unit values for RepeatFreqUnit.
const (
	FreqUnitMonthly = 8
	FreqUnitDaily   = 16
	FreqUnitWeekly  = 256
)

// Repeat type values for RepeatType.
const (
	RepeatAfterCompletion = 0
	RepeatFixedSchedule   = 1
)

// Repeat end date sentinel.
const (
	RepeatEndNever int64 = 64092211200 // Far-future sentinel meaning "repeat forever"
)

// Note object keys (nested inside FieldNote).
const (
	NoteKeyType    = "_t" // always "tx"
	NoteKeyLength  = "ch" // int: character count
	NoteKeyValue   = "v"  // string: note text content
	NoteKeyVersion = "t"  // always 1
)

// NewNote constructs a Things note object with the given text.
func NewNote(text string) map[string]any {
	return map[string]any{NoteKeyType: "tx", NoteKeyLength: len(text), NoteKeyValue: text, NoteKeyVersion: 1}
}

// NoteText extracts the text content from a note field value, or "".
func NoteText(v any) string {
	m, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	s, _ := m[NoteKeyValue].(string)
	return s
}

// Default index values for new items.
const (
	DefaultIndex      = -373
	DefaultTodayIndex = -485
)
