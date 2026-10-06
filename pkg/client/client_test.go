package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"
)

func TestClient_RetryOnServerError(t *testing.T) {
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt32(&attempts, 1)
		// First two attempts fail with 502, third succeeds with 200
		if current < 3 {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error": "bad gateway"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	c := NewClient(
		WithBaseURL(server.URL),
		WithRetry(3),
		WithTimeout(2*time.Second),
	)

	ctx := context.Background()
	body, err := c.Get(ctx, "/test", url.Values{})
	if err != nil {
		t.Fatalf("expected retry to succeed, got error: %v", err)
	}

	if string(body) != `{"success": true}` {
		t.Errorf("unexpected response: %s", string(body))
	}

	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("expected 3 attempts, got %d", atomic.LoadInt32(&attempts))
	}
}

func TestClient_AuthHeaderAndURLPrefix(t *testing.T) {
	var receivedAuth string
	var receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	c := NewClient(
		WithBaseURL(server.URL),
		WithAPIKey("test_token_xyz"),
	)

	ctx := context.Background()
	_, err := c.Get(ctx, "/users/current", url.Values{})
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	if receivedAuth != "Bearer test_token_xyz" {
		t.Errorf("expected Bearer token header, got %q", receivedAuth)
	}

	if receivedPath != "/api/v0/users/current" {
		t.Errorf("expected /api/v0/users/current, got %q", receivedPath)
	}
}
