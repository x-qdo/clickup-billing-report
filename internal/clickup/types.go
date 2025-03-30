package clickup

import (
	"encoding/json"
	"fmt"
)

// --- Common Structs ---

type User struct {
	ID             int    `json:"id"`
	Username       string `json:"username"`
	Email          string `json:"email"`
	Color          string `json:"color"`
	ProfilePicture string `json:"profilePicture"`
	Initials       string `json:"initials"`
}

type Status struct {
	Status     string `json:"status"`
	Color      string `json:"color"`
	Type       string `json:"type"`
	Orderindex int    `json:"orderindex"`
}

type Priority struct {
	ID       string `json:"id"`
	Priority string `json:"priority"`
	Color    string `json:"color"`
	Order    string `json:"orderindex"` // Note: ClickUp API might return string or int
}

type Tag struct {
	Name      string `json:"name"`
	TagFg     string `json:"tag_fg"`
	TagBg     string `json:"tag_bg"`
	Creator   int    `json:"creator"` // User ID
	IsEnabled bool   `json:"is_enabled"`
}

// --- API Response Structs ---

// UserResponse wraps the user object from /user endpoint.
type UserResponse struct {
	User User `json:"user"`
}

// Task represents a ClickUp task.
type Task struct {
	ID           string                 `json:"id"`
	CustomID     string                 `json:"custom_id"`
	Name         string                 `json:"name"`
	Status       Status                 `json:"status"`
	Orderindex   string                 `json:"orderindex"`
	DateCreated  string                 `json:"date_created"` // Milliseconds as string
	DateUpdated  string                 `json:"date_updated"` // Milliseconds as string
	DateClosed   string                 `json:"date_closed"`  // Milliseconds as string, can be null
	Archived     bool                   `json:"archived"`
	Creator      User                   `json:"creator"`
	Assignees    []User                 `json:"assignees"`
	Watchers     []User                 `json:"watchers"`
	Checklists   []interface{}          `json:"checklists"` // Define further if needed
	Tags         []Tag                  `json:"tags"`
	Parent       string                 `json:"parent"`        // Task ID, can be null
	Priority     *Priority              `json:"priority"`      // Pointer to handle null
	DueDate      string                 `json:"due_date"`      // Milliseconds as string, can be null
	StartDate    string                 `json:"start_date"`    // Milliseconds as string, can be null
	Points       interface{}            `json:"points"`        // Can be number or null
	TimeEstimate interface{}            `json:"time_estimate"` // Milliseconds, can be null
	CustomFields []TaskCustomFieldValue `json:"custom_fields"`
	List         struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Access bool   `json:"access"`
	} `json:"list"`
	Project struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Hidden bool   `json:"hidden"`
		Access bool   `json:"access"`
	} `json:"project"`
	Folder struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Hidden bool   `json:"hidden"`
		Access bool   `json:"access"`
	} `json:"folder"`
	Space struct {
		ID string `json:"id"`
	} `json:"space"`
	URL string `json:"url"`
}

// TaskCustomFieldValue represents a custom field value attached to a task.
// The 'Value' field can be of various types (string, number, array, etc.).
type TaskCustomFieldValue struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Type           string          `json:"type"`
	DateCreated    string          `json:"date_created"`
	HideFromGuests bool            `json:"hide_from_guests"`
	Value          json.RawMessage `json:"value"` // Use RawMessage to defer parsing
	Required       bool            `json:"required"`
}

// TasksResponse is the structure for the API response when fetching tasks.
type TasksResponse struct {
	Tasks    []Task `json:"tasks"`
	LastPage bool   `json:"last_page"`
}

// TimeEntry represents a ClickUp time entry.
type TimeEntry struct {
	ID          string `json:"id"`
	Task        Task   `json:"task"` // Can be partial task info
	Wid         string `json:"wid"`  // Workspace ID?
	User        User   `json:"user"`
	Billable    bool   `json:"billable"`
	Start       string `json:"start"`    // Milliseconds as string
	End         string `json:"end"`      // Milliseconds as string
	Duration    string `json:"duration"` // Milliseconds as string
	Description string `json:"description"`
	Tags        []Tag  `json:"tags"` // Or sometimes referred to as 'labels' in API
	Source      string `json:"source"`
	At          string `json:"at"` // Milliseconds as string (update time?)
	TaskURL     string `json:"task_url"`
}

// TimeEntriesResponse is the structure for the API response when fetching time entries.
type TimeEntriesResponse struct {
	Data []TimeEntry `json:"data"`
	// ClickUp might add pagination info here in the future
}

// UpdateTaskFieldRequest is used to update a custom field value.
type UpdateTaskFieldRequest struct {
	Value interface{} `json:"value"`
	// ValueOption string `json:"value_option,omitempty"` // For dropdowns, etc.
	// TimeDelta int `json:"time_delta,omitempty"` // For time tracking updates
}

// ErrorResponse represents a generic error response from the ClickUp API.
type ErrorResponse struct {
	ErrorCode string `json:"ECODE"`
	Error     string `json:"err"`
}

func (e *ErrorResponse) String() string {
	return fmt.Sprintf("ClickUp API Error: %s (%s)", e.Error, e.ErrorCode)
}
