package main

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestShouldForward(t *testing.T) {
	tt := []struct {
		name     string
		rawURL   string
		wantPort string
		wantOK   bool
	}{
		// Direct localhost URLs
		{
			name:     "direct localhost with port",
			rawURL:   "http://localhost:12345/callback",
			wantPort: "12345",
			wantOK:   true,
		},
		{
			name:     "direct 127.0.0.1 with port",
			rawURL:   "http://127.0.0.1:54321/auth",
			wantPort: "54321",
			wantOK:   true,
		},
		{
			name:     "direct IPv6 loopback with port",
			rawURL:   "http://[::1]:9999/cb",
			wantPort: "9999",
			wantOK:   true,
		},
		// Query param extraction
		{
			name:     "redirect_uri in query",
			rawURL:   "https://login.microsoftonline.com/tenant/oauth2?redirect_uri=http%3A%2F%2Flocalhost%3A38947&client_id=foo",
			wantPort: "38947",
			wantOK:   true,
		},
		{
			name:     "callback in query 127.0.0.1",
			rawURL:   "https://accounts.google.com/o/oauth2?callback=http%3A%2F%2F127.0.0.1%3A12345%2Fcb",
			wantPort: "12345",
			wantOK:   true,
		},
		{
			name:     "azure cli oauth",
			rawURL:   "https://login.microsoftonline.com/organizations/oauth2/v2.0/authorize?client_id=xxx&response_type=code&redirect_uri=http%3A%2F%2Flocalhost%3A38947&scope=https%3A%2F%2Fmanagement.core.windows.net%2F%2F.default+offline_access+openid+profile&state=xxx&code_challenge=xxx&code_challenge_method=S256&nonce=xxx&client_info=1&claims=%7B%22access_token%22%3A+%7B%22xms_cc%22%3A+%7B%22values%22%3A+%5B%22CP1%22%5D%7D%7D%7D&prompt=select_account",
			wantPort: "38947",
			wantOK:   true,
		},
		{
			name:     "gcloud cli oauth",
			rawURL:   "https://accounts.google.com/o/oauth2/auth?response_type=code&client_id=32555940559.apps.googleusercontent.com&redirect_uri=http%3A%2F%2Flocalhost%3A8085%2F&scope=openid+https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fuserinfo.email+https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fcloud-platform+https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fappengine.admin+https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fsqlservice.login+https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fcompute+https%3A%2F%2Fwww.googleapis.com%2Fauth%2Faccounts.reauth&state=xxx&access_type=offline&code_challenge=xxx&code_challenge_method=S256",
			wantPort: "8085",
			wantOK:   true,
		},
		{
			name:     "first of multiple localhost params wins",
			rawURL:   "https://example.com/?foo=http%3A%2F%2Flocalhost%3A9999&bar=http%3A%2F%2Flocalhost%3A8888",
			wantPort: "9999",
			wantOK:   true,
		},
		// Should not forward
		{
			name:     "non-localhost URL no localhost in params",
			rawURL:   "https://accounts.google.com/o/oauth2",
			wantPort: "",
			wantOK:   false,
		},
		{
			name:     "redirect_uri is not localhost",
			rawURL:   "https://example.com/?redirect_uri=https%3A%2F%2Fexample.com%2Fcallback",
			wantPort: "",
			wantOK:   false,
		},
		{
			name:     "localhost without port implies 80",
			rawURL:   "https://example.com/?redirect_uri=http%3A%2F%2Flocalhost%2Fpath",
			wantPort: "",
			wantOK:   false,
		},
		{
			name:     "direct localhost path no port",
			rawURL:   "http://localhost/path",
			wantPort: "",
			wantOK:   false,
		},
		{
			name:     "standard port 443 skip",
			rawURL:   "http://localhost:443/path",
			wantPort: "",
			wantOK:   false,
		},
		{
			name:     "standard port 80 skip",
			rawURL:   "http://localhost:80/path",
			wantPort: "",
			wantOK:   false,
		},
		{
			name:     "non-numeric port rejected",
			rawURL:   "http://localhost:abc/path",
			wantPort: "",
			wantOK:   false,
		},
		{
			name:     "port 0 rejected",
			rawURL:   "http://localhost:0/path",
			wantPort: "",
			wantOK:   false,
		},
		{
			name:     "port exceeding 65535 rejected",
			rawURL:   "http://localhost:99999/path",
			wantPort: "",
			wantOK:   false,
		},
		{
			name:     "negative port rejected",
			rawURL:   "http://localhost:-1/path",
			wantPort: "",
			wantOK:   false,
		},
		{
			name:     "malformed URL",
			rawURL:   "not-a-url",
			wantPort: "",
			wantOK:   false,
		},
		{
			name:     "empty string",
			rawURL:   "",
			wantPort: "",
			wantOK:   false,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			port, ok := shouldForward(tc.rawURL)
			if ok != tc.wantOK || port != tc.wantPort {
				t.Errorf("shouldForward(%q) = %q, %v; want %q, %v", tc.rawURL, port, ok, tc.wantPort, tc.wantOK)
			}
		})
	}
}

func TestForwardTrackerCleanup(t *testing.T) {
	tt := []struct {
		name            string
		cancelCmd       string // "true" (exit 0) or "false" (exit 1)
		expiredRemoved  bool
	}{
		{"cancel succeeds", "true", true},
		{"cancel fails", "false", false},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ft := newForwardTracker("/unused", 100*time.Millisecond, io.Discard)
			ft.sshControlCmdFunc = func(ctx context.Context, op, port string) *exec.Cmd {
				return exec.Command(tc.cancelCmd)
			}
			ft.mu.Lock()
			ft.active["11111"] = time.Now().Add(-200 * time.Millisecond) // expired
			ft.active["22222"] = time.Now().Add(-50 * time.Millisecond)  // not expired
			ft.mu.Unlock()

			ft.cleanup()

			ft.mu.Lock()
			defer ft.mu.Unlock()
			_, expiredExists := ft.active["11111"]
			if tc.expiredRemoved && expiredExists {
				t.Error("expired entry 11111 should be removed after successful cancel")
			}
			if !tc.expiredRemoved && !expiredExists {
				t.Error("expired entry 11111 should remain when ssh cancel fails")
			}
			if _, ok := ft.active["22222"]; !ok {
				t.Error("non-expired entry 22222 should remain")
			}
		})
	}
}

func TestForwardTrackerRunExitsOnContextCancel(t *testing.T) {
	ft := newForwardTracker("/nonexistent/socket", time.Minute, io.Discard)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		ft.run(ctx)
		close(done)
	}()
	cancel()
	select {
	case <-done:
		// run() exited
	case <-time.After(2 * time.Second):
		t.Fatal("run() did not exit after context cancel")
	}
}

func TestForwardTrackerForwardLogsError(t *testing.T) {
	var buf bytes.Buffer
	ft := newForwardTracker("/nonexistent/control/socket", time.Minute, &buf)
	ft.forward("12345")
	if !strings.Contains(buf.String(), "opener: ssh forward -L") {
		t.Errorf("expected forward error log, got: %s", buf.String())
	}
}

func TestForwardTrackerForwardDedup(t *testing.T) {
	var calls atomic.Int32
	ft := newForwardTracker("/unused", time.Minute, io.Discard)
	ft.sshControlCmdFunc = func(ctx context.Context, op, port string) *exec.Cmd {
		calls.Add(1)
		return exec.Command("true")
	}

	ft.forward("12345")
	ft.forward("12345")

	if n := calls.Load(); n != 1 {
		t.Errorf("expected ssh to be called once, got %d", n)
	}
	ft.mu.Lock()
	defer ft.mu.Unlock()
	if _, ok := ft.active["12345"]; !ok {
		t.Error("port should remain in active map")
	}
}
