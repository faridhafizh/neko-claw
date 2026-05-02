package main

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/playwright-community/playwright-go"
)

func isSafeURL(targetURL string) error {
	u, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid scheme: only http and https are allowed")
	}

	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("invalid URL: missing host")
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("could not resolve host: %w", err)
	}

	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() {
			return fmt.Errorf("URL resolves to a private or reserved IP address")
		}
	}

	return nil
}

func searchWebAndRead(targetURL string) (string, error) {
	if err := isSafeURL(targetURL); err != nil {
		return "", fmt.Errorf("unsafe URL: %w", err)
	}

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

	if _, err = page.Goto(targetURL, playwright.PageGotoOptions{
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
