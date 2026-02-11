package agent

import (
	"fmt"
	"strconv"
	"time"

	"github.com/anamika/zenact-web/server/browser"
	"github.com/anamika/zenact-web/server/models"
)

// ExecuteAction dispatches an LLM response to the appropriate browser action.
func ExecuteAction(b *browser.Browser, resp *models.LLMResponse) error {
	switch models.ActionType(resp.Action) {
	case models.ActionNavigate:
		if resp.Value == "" {
			return fmt.Errorf("navigate action requires a URL in value")
		}
		if err := b.Navigate(resp.Value); err != nil {
			return fmt.Errorf("navigate to %s failed: %w", resp.Value, err)
		}
		// Wait for page to start loading
		time.Sleep(2 * time.Second)
		return nil

	case models.ActionClick:
		if resp.Selector == "" {
			return fmt.Errorf("click action requires a selector")
		}
		return b.Click(resp.Selector)

	case models.ActionTypeText:
		if resp.Selector == "" {
			return fmt.Errorf("type action requires a selector")
		}
		return b.Type(resp.Selector, resp.Value)

	case models.ActionScroll:
		direction := "down"
		if resp.Value != "" {
			direction = resp.Value
		}
		return b.Scroll(direction)

	case models.ActionWait:
		time.Sleep(2 * time.Second)
		return nil

	case models.ActionHold:
		if resp.Selector == "" {
			return fmt.Errorf("hold action requires a selector")
		}
		duration := 1000
		if resp.Value != "" {
			if d, err := strconv.Atoi(resp.Value); err == nil {
				duration = d
			}
		}
		return b.Hold(resp.Selector, time.Duration(duration)*time.Millisecond)

	case models.ActionDrag:
		if resp.Selector == "" {
			return fmt.Errorf("drag action requires a source selector")
		}
		if resp.Value == "" {
			return fmt.Errorf("drag action requires a target in value")
		}
		return b.Drag(resp.Selector, resp.Value)

	case models.ActionDone:
		return nil

	default:
		return fmt.Errorf("unknown action type: %s", resp.Action)
	}
}
