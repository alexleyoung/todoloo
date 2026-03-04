package models

import (
	"time"
)

type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusDone       JobStatus = "done"
	JobStatusFailed     JobStatus = "failed"
)

type RawJob struct {
	ID            int64      `json:"id"`
	RawText       string     `json:"raw_text"`
	SubmittedAt   time.Time  `json:"submitted_at"`
	Status        JobStatus  `json:"status"`
	Attempts      int        `json:"attempts"`
	LastAttempted *time.Time `json:"last_attempted,omitempty"`
	ErrorMsg      *string    `json:"error_msg,omitempty"`
}

type TodoStatus string

const (
	TodoStatusOpen      TodoStatus = "open"
	TodoStatusDone      TodoStatus = "done"
	TodoStatusCancelled TodoStatus = "cancelled"
)

type Todo struct {
	ID          int64      `json:"id"`
	RawID       int64      `json:"raw_id,omitempty"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Category    string     `json:"category,omitempty"`
	DueDate     string     `json:"due_date,omitempty"`
	DueTime     string     `json:"due_time,omitempty"`
	Urgency     int        `json:"urgency"`
	Tags        string     `json:"tags,omitempty"`
	Location    string     `json:"location,omitempty"`
	Recurrence  string     `json:"recurrence,omitempty"`
	Status      TodoStatus `json:"status"`
	SourceText  string     `json:"source_text,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type ParsedTodo struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	DueDate     string   `json:"due_date"`
	DueTime     string   `json:"due_time"`
	Urgency     int      `json:"urgency"`
	Location    string   `json:"location"`
	Tags        []string `json:"tags"`
	Notes       string   `json:"notes"`
	Recurrence  string   `json:"recurrence"`
}

type TodoFilters struct {
	Category   string
	Status     string
	DueBefore  string
	DueAfter   string
	UrgencyMin int
	Tags       string
	Query      string
	Sort       string
	Order      string
	Limit      int
	Offset     int
}
