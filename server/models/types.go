package models

import "time"

// --- Task Status ---

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
)

// --- Task ---

type Task struct {
	ID          string     `json:"id"`
	Prompt      string     `json:"prompt"`
	Status      TaskStatus `json:"status"`
	Steps       []Step     `json:"steps"`
	Error       string     `json:"error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// --- Step (one iteration of the agent loop) ---

type Step struct {
	Iteration  int       `json:"iteration"`
	Screenshot string    `json:"screenshot,omitempty"` // base64 PNG
	URL        string    `json:"url"`
	Title      string    `json:"title"`
	Thought    string    `json:"thought"`
	Action     Action    `json:"action"`
	Timestamp  time.Time `json:"timestamp"`
}

// --- Action ---

type ActionType string

const (
	ActionNavigate ActionType = "navigate"
	ActionClick    ActionType = "click"
	ActionTypeText ActionType = "type"
	ActionScroll   ActionType = "scroll"
	ActionWait     ActionType = "wait"
	ActionDone     ActionType = "done"
	ActionHold     ActionType = "hold"
	ActionDrag     ActionType = "drag"
)

type Action struct {
	Type     ActionType `json:"action"`
	Selector string     `json:"selector,omitempty"`
	Value    string     `json:"value,omitempty"`
	Done     bool       `json:"done"`
	Success  bool       `json:"success"`
}

// --- LLM Response (parsed from vision model) ---

type LLMResponse struct {
	Thought  string `json:"thought"`
	Action   string `json:"action"`
	Selector string `json:"selector"`
	Value    string `json:"value"`
	Done     bool   `json:"done"`
	Success  bool   `json:"success"`
}

// --- API Request/Response ---

type CreateTaskRequest struct {
	Prompt string `json:"prompt"`
}

type CreateTaskResponse struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"`
}

// --- WebSocket Events ---

type WSEventType string

const (
	WSEventScreenshot   WSEventType = "screenshot"
	WSEventStepComplete WSEventType = "step_complete"
	WSEventTaskComplete WSEventType = "task_complete"
	WSEventTaskFailed   WSEventType = "task_failed"
)

type WSEvent struct {
	Type       WSEventType `json:"type"`
	TaskID     string      `json:"task_id"`
	Step       *Step       `json:"step,omitempty"`
	Screenshot string      `json:"screenshot,omitempty"`
	Error      string      `json:"error,omitempty"`
	Message    string      `json:"message,omitempty"`
}
