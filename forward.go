package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

const sshForwardTimeout = 3 * time.Second
const cleanupInterval = 10 * time.Second

// shouldForward parses rawURL and returns a non-standard loopback port if the URL
// (or any query parameter value parsed as a URL) refers to localhost with a
// non-standard port. Query parameters are scanned in the order they appear in
// the raw URL. Returns "", false if no such port is found or on parse errors.
func shouldForward(rawURL string) (port string, ok bool) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", false
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", false
	}
	if p := loopbackPort(u); p != "" {
		return p, true
	}
	// Iterate over the raw query string to preserve parameter order.
	for _, param := range strings.Split(u.RawQuery, "&") {
		_, v, _ := strings.Cut(param, "=")
		v, err := url.QueryUnescape(v)
		if err != nil {
			continue
		}
		sub, err := url.Parse(v)
		if err != nil {
			continue
		}
		if p := loopbackPort(sub); p != "" {
			return p, true
		}
	}
	return "", false
}

// loopbackPort returns the port from u if u's host is loopback and the port is
// present and not 80 or 443. Otherwise returns "".
func loopbackPort(u *url.URL) string {
	host := u.Hostname()
	if host == "" {
		return ""
	}
	host = strings.ToLower(host)
	if host != "localhost" && host != "127.0.0.1" && host != "::1" {
		return ""
	}
	port := u.Port()
	if port == "" || port == "80" || port == "443" {
		return ""
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return ""
	}
	return port
}

type forwardTracker struct {
	mu         sync.Mutex
	active     map[string]time.Time // port -> last forwarded time
	ttl        time.Duration
	errOut     io.Writer
	sshControlCmdFunc func(ctx context.Context, operation, port string) *exec.Cmd
}

func newForwardTracker(controlSocket string, ttl time.Duration, errOut io.Writer) *forwardTracker {
	return &forwardTracker{
		active: make(map[string]time.Time),
		ttl:    ttl,
		errOut: errOut,
		sshControlCmdFunc: func(ctx context.Context, op, port string) *exec.Cmd {
			return exec.CommandContext(ctx, "ssh", "-S", controlSocket, "-O", op, "-L", port+":localhost:"+port, "none")
		},
	}
}

func (ft *forwardTracker) forward(port string) {
	ft.mu.Lock()
	if _, exists := ft.active[port]; exists {
		ft.active[port] = time.Now()
		ft.mu.Unlock()
		if ft.errOut != nil {
			fmt.Fprintf(ft.errOut, "opener: port %s already forwarded, refreshing TTL\n", port)
		}
		return
	}
	// Mark the port as in-progress to prevent duplicate ssh execs.
	ft.active[port] = time.Now()
	ft.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), sshForwardTimeout)
	defer cancel()
	cmd := ft.sshControlCmdFunc(ctx, "forward", port)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err != nil {
		ft.mu.Lock()
		delete(ft.active, port)
		ft.mu.Unlock()
		if ft.errOut != nil {
			fmt.Fprintf(ft.errOut, "opener: ssh forward -L %s:localhost:%s: %v: %s\n", port, port, err, stderr.String())
		}
		return
	}

	if ft.errOut != nil {
		fmt.Fprintf(ft.errOut, "opener: forwarded -L %s:localhost:%s\n", port, port)
	}
}

func (ft *forwardTracker) cancelForward(port string) error {
	ctx, cancel := context.WithTimeout(context.Background(), sshForwardTimeout)
	defer cancel()
	cmd := ft.sshControlCmdFunc(ctx, "cancel", port)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, stderr.String())
	}
	return nil
}

func (ft *forwardTracker) cleanup() {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	now := time.Now()
	for port, ts := range ft.active {
		if now.Sub(ts) <= ft.ttl {
			continue
		}
		err := ft.cancelForward(port)
		if err == nil {
			delete(ft.active, port)
		}
		if ft.errOut != nil {
			if err != nil {
				fmt.Fprintf(ft.errOut, "opener: ssh cancel -L %s:localhost:%s: %v\n", port, port, err)
			} else {
				fmt.Fprintf(ft.errOut, "opener: cancelled forward -L %s:localhost:%s\n", port, port)
			}
		}
	}
}

func (ft *forwardTracker) run(ctx context.Context) {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ft.cleanup()
		}
	}
}
