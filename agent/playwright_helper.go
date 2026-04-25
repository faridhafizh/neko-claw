package main

import (
	"fmt"
	"strings"

	"github.com/playwright-community/playwright-go"
)

func searchWebAndRead(url string) (string, error) {
	pw, err := playwright.Run()
	if err != nil {
		return "", fmt.Errorf("could not start playwright: %w", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		return "", fmt.Errorf("could not launch browser: %w", err)
	}
	defer browser.Close()

	page, err := browser.NewPage()
	if err != nil {
		return "", fmt.Errorf("could not create page: %w", err)
	}

	if _, err = page.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		return "", fmt.Errorf("could not goto: %w", err)
	}

	content, err := page.Locator("body").InnerText()
	if err != nil {
		return "", fmt.Errorf("could not get text: %w", err)
	}

	if len(content) > 10000 {
		content = content[:10000] + "\n... (truncated)"
	}

	return strings.TrimSpace(content), nil
}
