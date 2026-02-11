package agent

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/anamika/zenact-web/server/browser"
	"github.com/anamika/zenact-web/server/config"
	"github.com/anamika/zenact-web/server/llm"
	"github.com/anamika/zenact-web/server/models"
	"github.com/google/uuid"
)

const maxConsecutiveLLMErrors = 5

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
	initialSummary := fmt.Sprintf("## Task Summary\n\n**Goal:** %s\n\n**Initial Context:**\n- Task just started\n- No pages visited yet\n- No actions taken\n\n**Progress:**\n- [ ] Started task\n", prompt)
	task := &models.Task{
		ID:        taskID,
		Prompt:    prompt,
		Status:    models.TaskStatusPending,
		Steps:     []models.Step{},
		Summary:   initialSummary,
		CreatedAt: time.Now(),
	}

	a.mu.Lock()
	a.tasks[taskID] = task
	a.mu.Unlock()

	go a.runLoop(taskID)
	return taskID
}

// updateSummary updates the task summary with new information after each step.
func (a *Agent) updateSummary(taskID string, iteration int, action models.Action, thought string, executionSuccess bool, executionError string, pageURL string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	task, ok := a.tasks[taskID]
	if !ok {
		return
	}

	var summary strings.Builder
	summary.WriteString(task.Summary)

	// Check if this step already exists in summary
	if strings.Contains(task.Summary, fmt.Sprintf("Step %d:", iteration)) {
		return
	}

	// Determine current phase based on iteration and content
	phase := "Progress"
	if strings.Contains(task.Summary, "Logged in") || strings.Contains(pageURL, "dashboard") || strings.Contains(pageURL, "account") {
		phase = "Logged in / Post-login"
	} else if strings.Contains(task.Summary, "Navigation") && !strings.Contains(task.Summary, "logged in") {
		phase = "Navigation"
	}

	// Build progress indicator
	progressIndicator := "[ ]"
	if executionSuccess {
		progressIndicator = "[x]"
	}

	// Write step summary
	summary.WriteString(fmt.Sprintf("\n### Step %d (%s)\n", iteration, phase))
	summary.WriteString(fmt.Sprintf("- **Action:** %s", action.Type))

	if action.Selector != "" {
		summary.WriteString(fmt.Sprintf(" on `%s`", action.Selector))
	}
	if action.Value != "" {
		if action.Type == models.ActionTypeText {
			summary.WriteString(fmt.Sprintf(" with value: \"%s\"", truncate(action.Value, 50)))
		} else {
			summary.WriteString(fmt.Sprintf(" with value: %s", truncate(action.Value, 50)))
		}
	}
	summary.WriteString(fmt.Sprintf("\n"))

	// Only include brief thought - not full context
	if thought != "" {
		summary.WriteString(fmt.Sprintf("- **Thought:** %s\n", truncate(thought, 150)))
	}

	if !executionSuccess && executionError != "" {
		summary.WriteString(fmt.Sprintf("- **Error:** %s\n", truncate(executionError, 200)))
		summary.WriteString(fmt.Sprintf("- **Status:** %s FAILED\n", progressIndicator))

		// Add what NOT to repeat - explicit guidance
		lessonsLearned := extractLessonsLearned(action, executionError)
		if lessonsLearned != "" {
			summary.WriteString(fmt.Sprintf("- **DO NOT REPEAT:** %s\n", lessonsLearned))
		}
	} else if executionSuccess {
		summary.WriteString(fmt.Sprintf("- **Status:** %s Completed\n", progressIndicator))

		// Extract and add key discoveries
		keyDiscoveries := extractKeyDiscoveries(action, executionSuccess, executionError, pageURL)
		if keyDiscoveries != "" {
			summary.WriteString(fmt.Sprintf("- **Key Discovery:** %s\n", keyDiscoveries))
		}
	}

	task.Summary = summary.String()
}

// extractLessonsLearned provides explicit guidance on what NOT to repeat
func extractLessonsLearned(action models.Action, executionError string) string {
	if strings.Contains(executionError, "element not found") || strings.Contains(executionError, "does not have child") {
		if action.Type == models.ActionClick {
			return fmt.Sprintf("Selector `%s` doesn't work - element not found or incorrect", truncate(action.Selector, 50))
		}
		if action.Type == models.ActionTypeText {
			return "Textarea interaction failed - editor may use rich text or different DOM structure"
		}
	}

	if strings.Contains(executionError, "not visible") {
		return fmt.Sprintf("Element `%s` exists but is not visible - scroll or dismiss overlays first", truncate(action.Selector, 50))
	}

	if strings.Contains(executionError, "click failed") {
		return fmt.Sprintf("Click on `%s` failed - element may be covered or disabled", truncate(action.Selector, 50))
	}

	return ""
}

// extractKeyDiscoveries extracts important information from successful actions
func extractKeyDiscoveries(action models.Action, success bool, executionError string, pageURL string) string {
	if !success && strings.Contains(executionError, "element not found") {
		if action.Type == models.ActionClick {
			return fmt.Sprintf("Element selector '%s' was incorrect - need better selector or element may not exist", truncate(action.Selector, 50))
		}
	}

	if success {
		if action.Type == models.ActionTypeText {
			return fmt.Sprintf("Successfully typed into %s", truncate(action.Selector, 50))
		}
		if action.Type == models.ActionClick {
			return fmt.Sprintf("Successfully clicked %s", truncate(action.Selector, 50))
		}
		if action.Type == models.ActionNavigate {
			return fmt.Sprintf("Navigated to %s", pageURL)
		}
	}

	return ""
}

// truncate truncates a string to the given length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
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
		currentSummary := task.Summary
		blockedSelectors := task.BlockedSelectors
		a.mu.RUnlock()

		domContent, _ := b.GetFullDOM()
		axTree, _ := b.GetAccessibilityTree()

		llmResp, err := a.llmClient.Decide(ctx, SystemPrompt, screenshotBytes, pageURL, pageTitle, task.Prompt, history, domContent, axTree, currentSummary, blockedSelectors)
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

		// Check for completion BEFORE executing
		if llmResp.Done {
			// Create final step
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
				ExecutionSuccess: true,
				ExecutionError:   "",
				Timestamp:        time.Now(),
			}

			a.mu.Lock()
			task.Steps = append(task.Steps, step)
			a.mu.Unlock()

			// Update the task summary with final step
			a.updateSummary(taskID, i+1, step.Action, step.Thought, true, "", pageURL)

			a.broadcast(taskID, models.WSEvent{
				Type:   models.WSEventStepComplete,
				TaskID: taskID,
				Step:   &step,
			})

			if llmResp.Success {
				a.completeTask(taskID)
			} else {
				a.failTask(taskID, llmResp.Thought)
			}
			return
		}

		// --- EXECUTE ---
		execSuccess := true
		execError := ""

		if err := ExecuteAction(b, llmResp); err != nil {
			log.Printf("[Task %s] Action error at iteration %d: %v", taskID, i+1, err)
			execSuccess = false
			execError = err.Error()
		}

		// Wait for page to settle (longer for clicks/navigates)
		if llmResp.Action == "click" || llmResp.Action == "navigate" {
			time.Sleep(3 * time.Second)
		} else {
			time.Sleep(1 * time.Second)
		}

		// NOW create the step with execution results
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
			ExecutionSuccess: execSuccess,
			ExecutionError:   execError,
			Timestamp:        time.Now(),
		}

		// Store the step with execution results
		a.mu.Lock()
		task.Steps = append(task.Steps, step)
		a.mu.Unlock()

		// Update the task summary with this step's information
		a.updateSummary(taskID, i+1, step.Action, step.Thought, execSuccess, execError, pageURL)

		// Broadcast step with execution results
		a.broadcast(taskID, models.WSEvent{
			Type:   models.WSEventStepComplete,
			TaskID: taskID,
			Step:   &step,
		})
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
