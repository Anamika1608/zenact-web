package browser

import (
	"context"
	"fmt"
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

// Click clicks an element by CSS selector.
func (b *Browser) Click(selector string) error {
	timeoutCtx, cancel := context.WithTimeout(b.ctx, 5*time.Second)
	defer cancel()
	return chromedp.Run(timeoutCtx,
		chromedp.WaitVisible(selector, chromedp.ByQuery),
		chromedp.Click(selector, chromedp.ByQuery, chromedp.NodeVisible),
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

// Screenshot captures the viewport as a JPEG image.
func (b *Browser) Screenshot() ([]byte, error) {
	var buf []byte
	if err := chromedp.Run(b.ctx, chromedp.FullScreenshot(&buf, 80)); err != nil {
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

// WaitVisible waits for a selector to be visible.
func (b *Browser) WaitVisible(selector string) error {
	timeoutCtx, cancel := context.WithTimeout(b.ctx, 10*time.Second)
	defer cancel()
	return chromedp.Run(timeoutCtx,
		chromedp.WaitVisible(selector, chromedp.ByQuery),
	)
}
