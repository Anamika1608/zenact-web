package agent

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/anamika/zenact-web/server/browser"
	"github.com/anamika/zenact-web/server/config"
	"github.com/anamika/zenact-web/server/llm"
	"github.com/anamika/zenact-web/server/models"
	"github.com/google/uuid"
)

const maxConsecutiveLLMErrors = 3

type Agent struct {
	cfg       *config.Config
	llmClient *llm.Client

	tasks map[string]*models.Task
	mu    sync.RWMutex

	subscribers map[string][]chan models.WSEvent
	subMu       sync.RWMutex
}

func New(cfg *config.Config, llmClient *llm.Client) *Agent {
	return &Agent{
		cfg:         cfg,
		llmClient:   llmClient,
		tasks:       make(map[string]*models.Task),
		subscribers: make(map[string][]chan models.WSEvent),
	}
}

// StartTask creates a new task and launches the agent loop.
func (a *Agent) StartTask(prompt string) string {
	taskID := uuid.New().String()
	task := &models.Task{
		ID:        taskID,
		Prompt:    prompt,
		Status:    models.TaskStatusPending,
		Steps:     []models.Step{},
		CreatedAt: time.Now(),
	}

	a.mu.Lock()
	a.tasks[taskID] = task
	a.mu.Unlock()

	go a.runLoop(taskID)
	return taskID
}

// GetTask returns a copy of the task.
func (a *Agent) GetTask(taskID string) (*models.Task, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	task, ok := a.tasks[taskID]
	if !ok {
		return nil, false
	}
	copied := *task
	copied.Steps = make([]models.Step, len(task.Steps))
	copy(copied.Steps, task.Steps)
	return &copied, true
}

// Subscribe registers a channel to receive events for a task.
func (a *Agent) Subscribe(taskID string) chan models.WSEvent {
	ch := make(chan models.WSEvent, 50)
	a.subMu.Lock()
	a.subscribers[taskID] = append(a.subscribers[taskID], ch)
	a.subMu.Unlock()
	return ch
}

// Unsubscribe removes a channel from the subscriber list.
func (a *Agent) Unsubscribe(taskID string, ch chan models.WSEvent) {
	a.subMu.Lock()
	defer a.subMu.Unlock()
	subs := a.subscribers[taskID]
	for i, sub := range subs {
		if sub == ch {
			a.subscribers[taskID] = append(subs[:i], subs[i+1:]...)
			close(ch)
			return
		}
	}
}

// broadcast sends an event to all subscribers of a task.
func (a *Agent) broadcast(taskID string, event models.WSEvent) {
	a.subMu.RLock()
	defer a.subMu.RUnlock()
	for _, ch := range a.subscribers[taskID] {
		select {
		case ch <- event:
		default:
			log.Printf("WARNING: dropping event for task %s (slow consumer)", taskID)
		}
	}
}

// runLoop is the core agent loop: observe → decide → execute → repeat.
func (a *Agent) runLoop(taskID string) {
	a.mu.Lock()
	task := a.tasks[taskID]
	task.Status = models.TaskStatusRunning
	a.mu.Unlock()

	// Create browser
	b, err := browser.New(a.cfg.BrowserHeadless, a.cfg.BrowserWidth, a.cfg.BrowserHeight)
	if err != nil {
		a.failTask(taskID, fmt.Sprintf("failed to start browser: %v", err))
		return
	}
	defer b.Close()

	ctx := context.Background()
	llmErrorStreak := 0

	for i := 0; i < a.cfg.MaxIterations; i++ {
		log.Printf("[Task %s] Iteration %d/%d", taskID, i+1, a.cfg.MaxIterations)

		// --- OBSERVE ---
		screenshotBytes, err := b.Screenshot()
		if err != nil {
			a.failTask(taskID, fmt.Sprintf("screenshot failed at iteration %d: %v", i+1, err))
			return
		}
		b64Screenshot := base64.StdEncoding.EncodeToString(screenshotBytes)

		pageURL, _ := b.GetURL()
		pageTitle, _ := b.GetTitle()

		// Send screenshot to WebSocket subscribers
		a.broadcast(taskID, models.WSEvent{
			Type:       models.WSEventScreenshot,
			TaskID:     taskID,
			Screenshot: b64Screenshot,
		})

		// --- DECIDE ---
		a.mu.RLock()
		history := make([]models.Step, len(task.Steps))
		copy(history, task.Steps)
		a.mu.RUnlock()

		llmResp, err := a.llmClient.Decide(ctx, SystemPrompt, screenshotBytes, pageURL, pageTitle, task.Prompt, history)
		if err != nil {
			llmErrorStreak++
			log.Printf("[Task %s] LLM error at iteration %d: %v", taskID, i+1, err)

			if statusCode, nonRetryable := nonRetryableLLMStatus(err); nonRetryable {
				a.failTask(taskID, fmt.Sprintf(
					"LLM request rejected with status %d. Likely invalid model input (for example oversized screenshot). %s",
					statusCode, clippedError(err),
				))
				return
			}

			if llmErrorStreak >= maxConsecutiveLLMErrors {
				a.failTask(taskID, fmt.Sprintf(
					"LLM failed %d times in a row. %s",
					llmErrorStreak, clippedError(err),
				))
				return
			}

			time.Sleep(3 * time.Second)
			continue
		}
		llmErrorStreak = 0

		log.Printf("[Task %s] LLM: thought=%q action=%s selector=%q value=%q done=%v success=%v",
			taskID, llmResp.Thought, llmResp.Action, llmResp.Selector, llmResp.Value, llmResp.Done, llmResp.Success)

		// Record step
		step := models.Step{
			Iteration:  i + 1,
			Screenshot: b64Screenshot,
			URL:        pageURL,
			Title:      pageTitle,
			Thought:    llmResp.Thought,
			Action: models.Action{
				Type:     models.ActionType(llmResp.Action),
				Selector: llmResp.Selector,
				Value:    llmResp.Value,
				Done:     llmResp.Done,
				Success:  llmResp.Success,
			},
			Timestamp: time.Now(),
		}

		a.mu.Lock()
		task.Steps = append(task.Steps, step)
		a.mu.Unlock()

		// Broadcast step
		a.broadcast(taskID, models.WSEvent{
			Type:   models.WSEventStepComplete,
			TaskID: taskID,
			Step:   &step,
		})

		// Check for completion
		if llmResp.Done {
			if llmResp.Success {
				a.completeTask(taskID)
			} else {
				a.failTask(taskID, llmResp.Thought)
			}
			return
		}

		// --- EXECUTE ---
		if err := ExecuteAction(b, llmResp); err != nil {
			log.Printf("[Task %s] Action error at iteration %d: %v", taskID, i+1, err)
			// Don't fail — let the LLM self-correct on next screenshot
		}

		// Brief pause for page to settle
		time.Sleep(1 * time.Second)
	}

	a.failTask(taskID, fmt.Sprintf("max iterations (%d) reached without completing task", a.cfg.MaxIterations))
}

func nonRetryableLLMStatus(err error) (int, bool) {
	var apiErr *llm.APIError
	if !errors.As(err, &apiErr) {
		return 0, false
	}

	statusCode := apiErr.StatusCode
	if statusCode >= 400 && statusCode < 500 && statusCode != http.StatusTooManyRequests {
		return statusCode, true
	}
	return statusCode, false
}

func clippedError(err error) string {
	const maxLen = 320
	msg := err.Error()
	if len(msg) <= maxLen {
		return msg
	}
	return msg[:maxLen] + "..."
}

func (a *Agent) completeTask(taskID string) {
	a.mu.Lock()
	task := a.tasks[taskID]
	task.Status = models.TaskStatusCompleted
	now := time.Now()
	task.CompletedAt = &now
	a.mu.Unlock()

	a.broadcast(taskID, models.WSEvent{
		Type:    models.WSEventTaskComplete,
		TaskID:  taskID,
		Message: "Task completed successfully",
	})
	log.Printf("[Task %s] COMPLETED", taskID)
}

func (a *Agent) failTask(taskID string, errMsg string) {
	a.mu.Lock()
	task := a.tasks[taskID]
	task.Status = models.TaskStatusFailed
	task.Error = errMsg
	now := time.Now()
	task.CompletedAt = &now
	a.mu.Unlock()

	a.broadcast(taskID, models.WSEvent{
		Type:   models.WSEventTaskFailed,
		TaskID: taskID,
		Error:  errMsg,
	})
	log.Printf("[Task %s] FAILED: %s", taskID, errMsg)
}
