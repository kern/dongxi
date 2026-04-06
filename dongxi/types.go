package dongxi

// Account holds the Things Cloud account information.
type Account struct {
	HistoryKey       string `json:"history-key"`
	Status           string `json:"status"`
	Email            string `json:"email"`
	MaildropEmail    string `json:"maildrop-email"`
	SLAVersionAccepted string `json:"SLA-version-accepted"`
}

// HistoryInfo holds summary information about a history.
type HistoryInfo struct {
	LatestServerIndex      int  `json:"latest-server-index"`
	LatestSchemaVersion    int  `json:"latest-schema-version"`
	IsEmpty                bool `json:"is-empty"`
	LatestTotalContentSize int  `json:"latest-total-content-size"`
}

// HistoryItems holds the response from the items endpoint.
type HistoryItems struct {
	CurrentItemIndex int              `json:"current-item-index"`
	Schema           int              `json:"schema"`
	Items            []map[string]any `json:"items"`
}

// CommitResponse holds the response from the commit endpoint.
type CommitResponse struct {
	ServerHeadIndex int `json:"server-head-index"`
}

// ResetResponse holds the response from the reset endpoint.
type ResetResponse struct {
	NewHistoryKey string `json:"new-history-key"`
}

// ItemType represents the operation type in a commit.
type ItemType int

const (
	ItemTypeCreate ItemType = 0
	ItemTypeModify ItemType = 1
	ItemTypeDelete ItemType = 2
)

// EntityType is the Things entity type string.
type EntityType string

const (
	EntityTask          EntityType = "Task6"
	EntityChecklistItem EntityType = "ChecklistItem3"
	EntityArea          EntityType = "Area3"
	EntityTag           EntityType = "Tag4"
)

// SyncMeta is the xx field required on every committed item.
var SyncMeta = map[string]any{"sn": map[string]any{}, "_t": "oo"}

// EmptyNote is the nt field for a task with no note.
var EmptyNote = map[string]any{"_t": "tx", "ch": 0, "v": "", "t": 1}

// TaskStatus maps to the ss field.
type TaskStatus int

const (
	TaskStatusOpen      TaskStatus = 0
	TaskStatusCancelled TaskStatus = 2
	TaskStatusCompleted TaskStatus = 3
)

// TaskDestination maps to the st field.
type TaskDestination int

const (
	TaskDestinationInbox   TaskDestination = 0
	TaskDestinationAnytime TaskDestination = 1
	TaskDestinationSomeday TaskDestination = 2
)

// TaskType maps to the tp field.
type TaskType int

const (
	TaskTypeTask    TaskType = 0
	TaskTypeProject TaskType = 1
	TaskTypeHeading TaskType = 2
)

// CommitItem represents a single item in a commit payload.
type CommitItem struct {
	T ItemType          `json:"t"`
	E EntityType        `json:"e"`
	P map[string]any    `json:"p"`
}
