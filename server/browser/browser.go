package browser

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

type Browser struct {
	allocCancel context.CancelFunc
	ctx         context.Context
	ctxCancel   context.CancelFunc
}

type ElementInfo struct {
	Tag         string `json:"tag"`
	ID          string `json:"id"`
	Classes     string `json:"classes"`
	Text        string `json:"text"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	Placeholder string `json:"placeholder"`
	Role        string `json:"role"`
	Visible     bool   `json:"visible"`
	Selector    string `json:"selector"`
	AXNode      string `json:"ax_node,omitempty"`
}

type AccessibilityNode struct {
	Role       string              `json:"role"`
	Name       string              `json:"name"`
	Value      string              `json:"value"`
	Properties map[string]string   `json:"properties"`
	Children   []AccessibilityNode `json:"children,omitempty"`
}

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

func (b *Browser) Close() {
	b.ctxCancel()
	b.allocCancel()
}

func (b *Browser) Navigate(url string) error {
	return chromedp.Run(b.ctx, chromedp.Navigate(url))
}

func (b *Browser) Screenshot() ([]byte, error) {
	var buf []byte
	if err := chromedp.Run(b.ctx, chromedp.CaptureScreenshot(&buf)); err != nil {
		return nil, fmt.Errorf("screenshot failed: %w", err)
	}
	return buf, nil
}

func (b *Browser) GetURL() (string, error) {
	var url string
	if err := chromedp.Run(b.ctx, chromedp.Location(&url)); err != nil {
		return "", err
	}
	return url, nil
}

func (b *Browser) GetTitle() (string, error) {
	var title string
	if err := chromedp.Run(b.ctx, chromedp.Title(&title)); err != nil {
		return "", err
	}
	return title, nil
}

func (b *Browser) GetFullDOM() (string, error) {
	var domContent string

	script := `
		(async () => {
			try {
				let html = '<!DOCTYPE html><html>';
				
				function serializeNode(node, depth = 0) {
					if (depth > 10) return '';
					let result = '';
					
					if (node.nodeType === Node.ELEMENT_NODE) {
						let attrs = '';
						if (node.id) attrs += ' id="' + node.id.replace(/"/g, '&quot;') + '"';
						if (node.className && typeof node.className === 'string') {
							const classes = node.className.split(' ').filter(Boolean).join(' ');
							if (classes) attrs += ' class="' + classes.replace(/"/g, '&quot;') + '"';
						}
						
						let role = node.getAttribute('role') || '';
						let text = (node.textContent || '').trim().substring(0, 100);
						let placeholder = node.getAttribute('placeholder') || '';
						let typeAttr = node.getAttribute('type') || '';
						let name = node.getAttribute('name') || '';
						
						result += '<div class="__browser_elem" data-role="' + role + '" data-text="' + text.replace(/"/g, '&quot;') + 
						           '" data-placeholder="' + placeholder.replace(/"/g, '&quot;') + 
						           '" data-type="' + typeAttr + 
						           '" data-name="' + name.replace(/"/g, '&quot;') + '">';
						result += '<' + node.tagName.toLowerCase() + attrs + '>';
						
						for (let child of node.childNodes) {
							result += serializeNode(child, depth + 1);
						}
						
						result += '</' + node.tagName.toLowerCase() + '>';
						result += '</div>';
					} else if (node.nodeType === Node.TEXT_NODE) {
						let text = node.textContent.trim();
						if (text) result += '<span class="__browser_text">' + text.replace(/</g, '&lt;').replace(/>/g, '&gt;') + '</span>';
					}
					
					return result;
				}
				
				const doc = document.documentElement;
				html += serializeNode(doc);
				html += '</html>';
				
				return html;
			} catch (e) {
				return 'ERROR: ' + e.toString();
			}
		})();
	`

	err := chromedp.Run(b.ctx, chromedp.Evaluate(script, &domContent))
	if err != nil {
		return "", err
	}

	if strings.Contains(domContent, "ERROR:") {
		return "", fmt.Errorf("DOM extraction failed: %s", domContent)
	}

	return domContent, nil
}

func (b *Browser) GetAccessibilityTree() (string, error) {
	var axTree string

	script := `
		(async () => {
			try {
				function getAXNode(node) {
					if (!node || node.nodeType !== Node.ELEMENT_NODE) return null;
					
					let axNode = {
						role: node.getAttribute('role') || getImplicitRole(node),
						name: getAccessibleName(node),
						properties: {}
					};
					
					const rect = node.getBoundingClientRect();
					if (rect.width > 0 && rect.height > 0) {
						axNode.properties['visible'] = 'true';
						axNode.properties['bounds'] = Math.round(rect.left) + ',' + Math.round(rect.top) + ',' + 
						                           Math.round(rect.width) + ',' + Math.round(rect.height);
					}
					
					if (node.tagName === 'INPUT' || node.tagName === 'SELECT' || node.tagName === 'TEXTAREA') {
						axNode.value = node.value || '';
					}
					
					if (node.tagName === 'BUTTON' || node.getAttribute('role') === 'button') {
						axNode.properties['interactive'] = 'true';
					}
					
					return axNode;
				}
				
				function getImplicitRole(node) {
					const tag = node.tagName.toUpperCase();
					const type = node.getAttribute('type') || '';
					
					switch (tag) {
						case 'BUTTON': return 'button';
						case 'A': return node.href ? 'link' : 'button';
						case 'INPUT':
							if (type === 'button' || type === 'submit' || type === 'reset') return 'button';
							if (type === 'checkbox') return 'checkbox';
							if (type === 'radio') return 'radio';
							return 'textbox';
						case 'SELECT': return 'combobox';
						case 'TEXTAREA': return 'textbox';
						case 'H1': return 'heading';
						case 'H2': return 'heading';
						case 'H3': return 'heading';
						case 'H4': return 'heading';
						case 'A': return 'link';
						default:
							if (node.getAttribute('role')) return node.getAttribute('role');
							return 'generic';
					}
				}
				
				function getAccessibleName(node) {
					const ariaLabel = node.getAttribute('aria-label') || '';
					if (ariaLabel) return ariaLabel;
					
					const ariaLabelledBy = node.getAttribute('aria-labelledby') || '';
					if (ariaLabelledBy) {
						const ids = ariaLabelledBy.split(' ').map(id => document.getElementById(id)).filter(Boolean);
						return ids.map(el => el.textContent).join(' ').trim();
					}
					
					const title = node.getAttribute('title') || '';
					if (title) return title;
					
					let text = '';
					for (let child of node.childNodes) {
						if (child.nodeType === Node.TEXT_NODE) {
							text += child.textContent;
						}
					}
					return text.trim().substring(0, 200);
				}
				
				const interactive = ['button', 'link', 'textbox', 'checkbox', 'radio', 'combobox', 'heading', 'menuitem'];
				const elements = Array.from(document.querySelectorAll('*'));
				
				const axNodes = [];
				for (const el of elements) {
					if (el.offsetParent === null) continue;
					const rect = el.getBoundingClientRect();
					if (rect.width < 5 || rect.height < 5) continue;
					
					const role = getImplicitRole(el);
					if (!interactive.includes(role) && !el.onclick && !el.getAttribute('role')) continue;
					
					const node = getAXNode(el);
					if (node && (node.role !== 'generic' || el.id || el.className)) {
						let selector = el.tagName.toLowerCase();
						if (el.id) selector += '#' + el.id;
						else if (el.className && typeof el.className === 'string') {
							const cls = el.className.split(' ').filter(Boolean)[0];
							if (cls) selector += '.' + cls;
						}
						node.selector = selector;
						axNodes.push(node);
					}
				}
				
				return JSON.stringify(axNodes.slice(0, 30));
			} catch (e) {
				return 'ERROR: ' + e.toString();
			}
		})();
	`

	err := chromedp.Run(b.ctx, chromedp.Evaluate(script, &axTree))
	if err != nil {
		return "", err
	}

	if strings.Contains(axTree, "ERROR:") {
		return "[]", nil
	}

	return axTree, nil
}

func (b *Browser) FindElementByText(text string) (string, error) {
	var selector string

	script := fmt.Sprintf(`
		(function() {
			const text = "%s";
			const elements = Array.from(document.querySelectorAll('button, a, input, [role="button"], [role="link"]'));
			for (const el of elements) {
				const elText = (el.textContent || '').trim().toLowerCase();
				if (elText.includes(text.toLowerCase())) {
					if (el.id) return '#' + el.id;
					if (el.className && typeof el.className === 'string') {
						const cls = el.className.split(' ').filter(Boolean)[0];
						if (cls) return el.tagName.toLowerCase() + '.' + cls;
					}
					return el.tagName.toLowerCase();
				}
			}
			return 'NOT_FOUND';
		})();
	`, strings.ReplaceAll(text, `"`, `\"`))

	err := chromedp.Run(b.ctx, chromedp.Evaluate(script, &selector))
	if err != nil {
		return "", err
	}

	if selector == "NOT_FOUND" {
		return "", fmt.Errorf("element with text '%s' not found", text)
	}

	return selector, nil
}

func (b *Browser) FindElementsByText(text string) ([]string, error) {
	var selectors []string

	script := fmt.Sprintf(`
		(function() {
			const text = "%s";
			const elements = Array.from(document.querySelectorAll('button, a, input, [role="button"], [role="link"]'));
			const results = [];
			for (const el of elements) {
				const elText = (el.textContent || '').trim().toLowerCase();
				if (elText.includes(text.toLowerCase())) {
					let selector = el.tagName.toLowerCase();
					if (el.id) selector = '#' + el.id;
					else if (el.className && typeof el.className === 'string') {
						const cls = el.className.split(' ').filter(Boolean)[0];
						if (cls) selector += '.' + cls;
					}
					results.push(selector);
				}
			}
			return JSON.stringify(results);
		})();
	`, strings.ReplaceAll(text, `"`, `\"`))

	var jsonResult string
	err := chromedp.Run(b.ctx, chromedp.Evaluate(script, &jsonResult))
	if err != nil {
		return nil, err
	}

	if jsonResult == "[]" || jsonResult == "" {
		return []string{}, nil
	}

	return selectors, nil
}

func (b *Browser) Click(selector string) error {
	timeoutCtx, cancel := context.WithTimeout(b.ctx, 10*time.Second)
	defer cancel()

	return chromedp.Run(timeoutCtx,
		chromedp.WaitVisible(selector, chromedp.ByQuery),
		chromedp.Click(selector, chromedp.ByQuery, chromedp.NodeVisible),
	)
}

func (b *Browser) ClickByID(id string) error {
	selector := "#" + id
	return b.Click(selector)
}

func (b *Browser) Type(selector, text string) error {
	timeoutCtx, cancel := context.WithTimeout(b.ctx, 10*time.Second)
	defer cancel()

	return chromedp.Run(timeoutCtx,
		chromedp.WaitVisible(selector, chromedp.ByQuery),
		chromedp.Clear(selector, chromedp.ByQuery),
		chromedp.SendKeys(selector, text, chromedp.ByQuery),
	)
}

func (b *Browser) TypeByName(name, text string) error {
	selector := fmt.Sprintf(`input[name="%s"], textarea[name="%s"]`, name, name)
	return b.Type(selector, text)
}

func (b *Browser) Scroll(direction string) error {
	pixels := 500
	if direction == "up" {
		pixels = -500
	}
	script := fmt.Sprintf("window.scrollBy(0, %d)", pixels)
	return chromedp.Run(b.ctx, chromedp.Evaluate(script, nil))
}

func (b *Browser) ScrollToElement(selector string) error {
	script := fmt.Sprintf(`
		(function() {
			const el = document.querySelector('%s');
			if (el) {
				el.scrollIntoView({ behavior: 'smooth', block: 'center' });
				return 'OK';
			}
			return 'NOT_FOUND';
		})();
	`, strings.ReplaceAll(selector, `'`, `\\'`))

	var result string
	err := chromedp.Run(b.ctx, chromedp.Evaluate(script, &result))
	if err != nil {
		return err
	}

	if result == "NOT_FOUND" {
		return fmt.Errorf("element not found for scrolling: %s", selector)
	}

	return nil
}

func (b *Browser) WaitForLoad() error {
	time.Sleep(2 * time.Second)
	return nil
}

func (b *Browser) PressKey(key string) error {
	script := fmt.Sprintf("document.activeElement.dispatchEvent(new KeyboardEvent('keydown', { key: '%s', code: '%s', bubbles: true }));", key, key)
	return chromedp.Run(b.ctx, chromedp.Evaluate(script, nil))
}

func (b *Browser) ClickAt(selector string, x, y float64) error {
	script := fmt.Sprintf(`
		(function() {
			const el = document.querySelector('%s');
			if (!el) return 'NOT_FOUND';
			const rect = el.getBoundingClientRect();
			const clientX = rect.left + %f;
			const clientY = rect.top + %f;
			const event = new MouseEvent('click', { bubbles: true, cancelable: true, clientX: clientX, clientY: clientY });
			el.dispatchEvent(event);
			return 'OK';
		})();
	`, strings.ReplaceAll(selector, `'`, `\\'`), x, y)

	var result string
	err := chromedp.Run(b.ctx, chromedp.Evaluate(script, &result))
	if err != nil {
		return err
	}

	if result == "NOT_FOUND" {
		return fmt.Errorf("element not found: %s", selector)
	}

	return nil
}

func (b *Browser) ExecuteScript(script string) (string, error) {
	var result string
	err := chromedp.Run(b.ctx, chromedp.Evaluate(script, &result))
	if err != nil {
		return "", err
	}
	return result, nil
}

func (b *Browser) GetBase64Screenshot() (string, error) {
	buf, err := b.Screenshot()
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf), nil
}

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
	`, strings.ReplaceAll(selector, `'`, `\\'`), ms)
	_, err := b.ExecuteScript(script)
	return err
}

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
		`, strings.ReplaceAll(sourceSelector, `'`, `\\'`), offsetY)
	} else if strings.Contains(target, ",") {
		parts := strings.Split(target, ",")
		if len(parts) == 2 {
			x, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
			y, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
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
			`, strings.ReplaceAll(sourceSelector, `'`, `\\'`), x, y, x, y)
		}
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
		`, strings.ReplaceAll(sourceSelector, `'`, `\\'`), strings.ReplaceAll(target, `'`, `\\'`))
	}

	_, err := b.ExecuteScript(script)
	return err
}
