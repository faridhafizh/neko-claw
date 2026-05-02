package main

import (
	"strings"
	"testing"
)

func TestIsSafeURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		expectErr bool
		errPrefix string
	}{
		{
			name:      "Valid HTTPS URL",
			url:       "https://example.com",
			expectErr: false,
		},
		{
			name:      "Valid HTTP URL",
			url:       "http://example.com",
			expectErr: false,
		},
		{
			name:      "Invalid Scheme (file)",
			url:       "file:///etc/passwd",
			expectErr: true,
			errPrefix: "invalid scheme",
		},
		{
			name:      "Invalid Scheme (ftp)",
			url:       "ftp://example.com/file",
			expectErr: true,
			errPrefix: "invalid scheme",
		},
		{
			name:      "Localhost loopback",
			url:       "http://localhost",
			expectErr: true,
			errPrefix: "URL resolves to a private or reserved IP address",
		},
		{
			name:      "IPv4 Loopback",
			url:       "http://127.0.0.1",
			expectErr: true,
			errPrefix: "URL resolves to a private or reserved IP address",
		},
		{
			name:      "IPv4 Private (10.x.x.x)",
			url:       "http://10.0.0.1",
			expectErr: true,
			errPrefix: "URL resolves to a private or reserved IP address",
		},
		{
			name:      "IPv4 Private (192.168.x.x)",
			url:       "http://192.168.1.1",
			expectErr: true,
			errPrefix: "URL resolves to a private or reserved IP address",
		},
		{
			name:      "IPv6 Loopback",
			url:       "http://[::1]",
			expectErr: true,
			errPrefix: "URL resolves to a private or reserved IP address",
		},
		{
			name:      "Invalid URL format",
			url:       "://invalid",
			expectErr: true,
			errPrefix: "invalid URL",
		},
		{
			name:      "Missing Host",
			url:       "http://",
			expectErr: true,
			errPrefix: "invalid URL: missing host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := isSafeURL(tt.url)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error for URL %q, got none", tt.url)
				} else if tt.errPrefix != "" && !strings.HasPrefix(err.Error(), tt.errPrefix) {
					t.Errorf("expected error to start with %q, got %q", tt.errPrefix, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error for URL %q, got %v", tt.url, err)
				}
			}
		})
	}
}
