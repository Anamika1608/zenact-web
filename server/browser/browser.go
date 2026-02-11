package browser

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/chromedp/chromedp"
)

type Browser struct {
	allocCancel context.CancelFunc
	ctx         context.Context
	ctxCancel   context.CancelFunc
}

// New creates a new browser instance.
func New(headless bool, width, height int) (*Browser, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.WindowSize(width, height),
		chromedp.Flag("headless", headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, ctxCancel := chromedp.NewContext(allocCtx)

	// Force browser startup
	if err := chromedp.Run(ctx); err != nil {
		allocCancel()
		return nil, fmt.Errorf("failed to start browser: %w", err)
	}

	return &Browser{
		allocCancel: allocCancel,
		ctx:         ctx,
		ctxCancel:   ctxCancel,
	}, nil
}

// Close shuts down the browser.
func (b *Browser) Close() {
	b.ctxCancel()
	b.allocCancel()
}

// Navigate goes to the given URL.
func (b *Browser) Navigate(url string) error {
	return chromedp.Run(b.ctx, chromedp.Navigate(url))
}

// Click clicks an element by CSS selector with comprehensive error checking.
func (b *Browser) Click(selector string) error {
	// 1. Check if element exists first
	exists, err := b.ElementExists(selector)
	if err != nil {
		return fmt.Errorf("failed to check if element exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("element not found: %s", selector)
	}

	// 2. Check if element is visible
	visible, err := b.ElementVisible(selector)
	if err != nil {
		return fmt.Errorf("failed to check element visibility: %w", err)
	}
	if !visible {
		return fmt.Errorf("element exists but is not visible: %s", selector)
	}

	// 3. Scroll element into view
	if err := b.ScrollIntoView(selector); err != nil {
		return fmt.Errorf("failed to scroll element into view: %w", err)
	}

	// 4. Wait for element to be ready and click
	timeoutCtx, cancel := context.WithTimeout(b.ctx, 5*time.Second)
	defer cancel()

	err = chromedp.Run(timeoutCtx,
		chromedp.WaitVisible(selector, chromedp.ByQuery),
		chromedp.Click(selector, chromedp.ByQuery, chromedp.NodeVisible),
	)

	if err != nil {
		return fmt.Errorf("click failed on %s: %w", selector, err)
	}

	return nil
}

// ElementExists checks if an element exists in the DOM
func (b *Browser) ElementExists(selector string) (bool, error) {
	var exists bool
	err := chromedp.Run(b.ctx,
		chromedp.Evaluate(fmt.Sprintf(`document.querySelector('%s') !== null`, selector), &exists),
	)
	return exists, err
}

// ElementVisible checks if an element is visible (has dimensions and not hidden)
func (b *Browser) ElementVisible(selector string) (bool, error) {
	var visible bool
	err := chromedp.Run(b.ctx,
		chromedp.Evaluate(fmt.Sprintf(`
            (() => {
                const el = document.querySelector('%s');
                if (!el) return false;
                const rect = el.getBoundingClientRect();
                const style = window.getComputedStyle(el);
                return rect.width > 0 && 
                       rect.height > 0 && 
                       style.visibility !== 'hidden' &&
                       style.display !== 'none' &&
                       style.opacity !== '0';
            })()
        `, selector), &visible),
	)
	return visible, err
}

// ScrollIntoView scrolls an element into the viewport
func (b *Browser) ScrollIntoView(selector string) error {
	return chromedp.Run(b.ctx,
		chromedp.Evaluate(fmt.Sprintf(`
            (() => {
                const el = document.querySelector('%s');
                if (el) {
                    el.scrollIntoView({ behavior: 'smooth', block: 'center' });
                }
            })()
        `, selector), nil),
	)
}

// Type types text into an element. Clears the field first.
func (b *Browser) Type(selector, text string) error {
	timeoutCtx, cancel := context.WithTimeout(b.ctx, 5*time.Second)
	defer cancel()
	return chromedp.Run(timeoutCtx,
		chromedp.WaitVisible(selector, chromedp.ByQuery),
		chromedp.Clear(selector, chromedp.ByQuery),
		chromedp.SendKeys(selector, text, chromedp.ByQuery),
	)
}

// Scroll scrolls the page up or down.
func (b *Browser) Scroll(direction string) error {
	pixels := 500
	if direction == "up" {
		pixels = -500
	}
	script := fmt.Sprintf("window.scrollBy(0, %d)", pixels)
	return chromedp.Run(b.ctx, chromedp.Evaluate(script, nil))
}

// Screenshot captures the current viewport as a PNG image.
func (b *Browser) Screenshot() ([]byte, error) {
	var buf []byte
	if err := chromedp.Run(b.ctx, chromedp.CaptureScreenshot(&buf)); err != nil {
		return nil, fmt.Errorf("screenshot failed: %w", err)
	}
	return buf, nil
}

// GetURL returns the current page URL.
func (b *Browser) GetURL() (string, error) {
	var url string
	if err := chromedp.Run(b.ctx, chromedp.Location(&url)); err != nil {
		return "", err
	}
	return url, nil
}

// GetTitle returns the current page title.
func (b *Browser) GetTitle() (string, error) {
	var title string
	if err := chromedp.Run(b.ctx, chromedp.Title(&title)); err != nil {
		return "", err
	}
	return title, nil
}

// GetVisibleElements returns a JSON string of visible interactive elements for the LLM.
func (b *Browser) GetVisibleElements() (string, error) {
	var result string
	script := `
		(() => {
			const elements = Array.from(document.querySelectorAll('a, button, input, select, textarea, [role="button"], [role="link"]'));
			const visible = elements
				.filter(el => {
					const rect = el.getBoundingClientRect();
					const style = window.getComputedStyle(el);
					return rect.width > 0 && rect.height > 0 && 
						   style.visibility !== 'hidden' && 
						   style.display !== 'none' &&
						   rect.top >= 0 && rect.top < window.innerHeight;
				})
				.slice(0, 20)
				.map(el => ({
					tag: el.tagName.toLowerCase(),
					id: el.id || '',
					class: (el.className || '').toString().split(' ').filter(Boolean).join(' '),
					text: (el.textContent || '').trim().substring(0, 50),
					type: el.getAttribute('type') || '',
					name: el.getAttribute('name') || '',
					placeholder: el.getAttribute('placeholder') || '',
					selector: generateSelector(el)
				}));
			return JSON.stringify(visible);
		})();

		function generateSelector(el) {
			if (el.id) return '#' + el.id;
			if (el.name) return el.tagName.toLowerCase() + '[name="' + el.getAttribute('name') + '"]';
			let path = [];
			while (el && el !== document.body) {
				let selector = el.tagName.toLowerCase();
				if (el.id) {
					selector = '#' + el.id;
					path.unshift(selector);
					break;
				}
				if (el.className && typeof el.className === 'string') {
					const classes = el.className.split(' ').filter(Boolean).join('.');
					if (classes) selector += '.' + classes;
				}
				path.unshift(selector);
				el = el.parentElement;
			}
			return path.join(' > ');
		}
	`
	err := chromedp.Run(b.ctx, chromedp.Evaluate(script, &result))
	return result, err
}

// WaitVisible waits for a selector to be visible.
func (b *Browser) WaitVisible(selector string) error {
	timeoutCtx, cancel := context.WithTimeout(b.ctx, 10*time.Second)
	defer cancel()
	return chromedp.Run(timeoutCtx,
		chromedp.WaitVisible(selector, chromedp.ByQuery),
	)
}

// Hold clicks and holds an element for the specified duration using JavaScript.
func (b *Browser) Hold(selector string, duration time.Duration) error {
	ms := strconv.Itoa(int(duration.Milliseconds()))
	script := fmt.Sprintf(`
		(function() {
			var el = document.querySelector('%s');
			if (!el) return 'Element not found';
			var event = new MouseEvent('mousedown', { bubbles: true, cancelable: true, clientX: 0, clientY: 0 });
			el.dispatchEvent(event);
			setTimeout(function() {
				var up = new MouseEvent('mouseup', { bubbles: true, cancelable: true, clientX: 0, clientY: 0 });
				el.dispatchEvent(up);
			}, %s);
			return 'OK';
		})();
	`, selector, ms)
	return chromedp.Run(b.ctx, chromedp.Evaluate(script, nil))
}

// Drag drags an element to a target location using JavaScript.
// target can be "up", "down", a CSS selector, or "x,y" coordinates.
func (b *Browser) Drag(sourceSelector string, target string) error {
	var script string

	if target == "down" || target == "up" {
		offsetY := 300
		if target == "up" {
			offsetY = -300
		}
		script = fmt.Sprintf(`
			(function() {
				var el = document.querySelector('%s');
				if (!el) return 'Element not found';
				var rect = el.getBoundingClientRect();
				var startX = rect.left + rect.width / 2;
				var startY = rect.top + rect.height / 2;
				var endX = startX;
				var endY = startY + %d;
				
				var down = new MouseEvent('mousedown', { bubbles: true, cancelable: true, clientX: startX, clientY: startY });
				el.dispatchEvent(down);
				
				var move = new MouseEvent('mousemove', { bubbles: true, cancelable: true, clientX: endX, clientY: endY });
				window.dispatchEvent(move);
				
				setTimeout(function() {
					var up = new MouseEvent('mouseup', { bubbles: true, cancelable: true, clientX: endX, clientY: endY });
					window.dispatchEvent(up);
				}, 100);
				return 'OK';
			})();
		`, sourceSelector, offsetY)
	} else if containsComma(target) {
		parts := splitAtComma(target)
		x, _ := strconv.Atoi(parts[0])
		y, _ := strconv.Atoi(parts[1])
		script = fmt.Sprintf(`
			(function() {
				var el = document.querySelector('%s');
				if (!el) return 'Element not found';
				var rect = el.getBoundingClientRect();
				var startX = rect.left + rect.width / 2;
				var startY = rect.top + rect.height / 2;
				
				var down = new MouseEvent('mousedown', { bubbles: true, cancelable: true, clientX: startX, clientY: startY });
				el.dispatchEvent(down);
				
				var move = new MouseEvent('mousemove', { bubbles: true, cancelable: true, clientX: startX + %d, clientY: startY + %d });
				window.dispatchEvent(move);
				
				setTimeout(function() {
					var up = new MouseEvent('mouseup', { bubbles: true, cancelable: true, clientX: startX + %d, clientY: startY + %d });
					window.dispatchEvent(up);
				}, 100);
				return 'OK';
			})();
		`, sourceSelector, x, y, x, y)
	} else {
		script = fmt.Sprintf(`
			(function() {
				var source = document.querySelector('%s');
				var target = document.querySelector('%s');
				if (!source || !target) return 'Element not found';
				
				var sourceRect = source.getBoundingClientRect();
				var targetRect = target.getBoundingClientRect();
				var startX = sourceRect.left + sourceRect.width / 2;
				var startY = sourceRect.top + sourceRect.height / 2;
				var endX = targetRect.left + targetRect.width / 2;
				var endY = targetRect.top + targetRect.height / 2;
				
				var down = new MouseEvent('mousedown', { bubbles: true, cancelable: true, clientX: startX, clientY: startY });
				source.dispatchEvent(down);
				
				var move = new MouseEvent('mousemove', { bubbles: true, cancelable: true, clientX: endX, clientY: endY });
				window.dispatchEvent(move);
				
				setTimeout(function() {
					var up = new MouseEvent('mouseup', { bubbles: true, cancelable: true, clientX: endX, clientY: endY });
					window.dispatchEvent(up);
				}, 100);
				return 'OK';
			})();
		`, sourceSelector, target)
	}

	return chromedp.Run(b.ctx, chromedp.Evaluate(script, nil))
}

func containsComma(s string) bool {
	for _, c := range s {
		if c == ',' {
			return true
		}
	}
	return false
}

func splitAtComma(s string) []string {
	var result []string
	current := ""
	for _, c := range s {
		if c == ',' {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	result = append(result, current)
	return result
}
