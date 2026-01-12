package webhook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Send_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		Timeout:       5 * time.Second,
		MaxRetries:    1,
		MaxConcurrent: 10,
	})

	webhook := &Webhook{
		ID:      "test-webhook-1",
		URL:     server.URL,
		Payload: json.RawMessage(`{"test": true}`),
	}

	result, err := client.Send(context.Background(), webhook)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, http.StatusOK, result.StatusCode)
	assert.Equal(t, 1, result.Attempts)
}

func TestClient_Send_WithRetry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(Config{
		Timeout:       5 * time.Second,
		MaxRetries:    5,
		MaxConcurrent: 10,
	})

	webhook := &Webhook{
		ID:      "test-webhook-2",
		URL:     server.URL,
		Payload: json.RawMessage(`{}`),
		RetryPolicy: &RetryPolicy{
			MaxAttempts:    5,
			InitialBackoff: 10 * time.Millisecond,
			MaxBackoff:     100 * time.Millisecond,
			Multiplier:     1.5,
		},
	}

	result, err := client.Send(context.Background(), webhook)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, 3, result.Attempts)
	assert.Equal(t, 3, attempts)
}

func TestClient_Send_NoRetryOn4xx(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewClient(Config{
		Timeout:       5 * time.Second,
		MaxRetries:    5,
		MaxConcurrent: 10,
	})

	webhook := &Webhook{
		ID:      "test-webhook-3",
		URL:     server.URL,
		Payload: json.RawMessage(`{}`),
		RetryPolicy: &RetryPolicy{
			MaxAttempts:    3,
			InitialBackoff: 10 * time.Millisecond,
			MaxBackoff:     100 * time.Millisecond,
			Multiplier:     2.0,
		},
	}

	result, err := client.Send(context.Background(), webhook)

	require.Error(t, err)
	assert.False(t, result.Success)
	assert.Equal(t, 1, attempts) // No retry for 4xx
}

func TestClient_Send_WithSignature(t *testing.T) {
	var receivedSignature string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSignature = r.Header.Get("X-Webhook-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(DefaultConfig())

	webhook := &Webhook{
		ID:      "test-webhook-4",
		URL:     server.URL,
		Payload: json.RawMessage(`{"event": "test"}`),
		Secret:  "my-secret-key",
	}

	result, err := client.Send(context.Background(), webhook)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.NotEmpty(t, receivedSignature)
}

func TestClient_SendBatch(t *testing.T) {
	responseCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		responseCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(DefaultConfig())

	webhooks := []*Webhook{
		{ID: "wh-1", URL: server.URL, Payload: json.RawMessage(`{}`)},
		{ID: "wh-2", URL: server.URL, Payload: json.RawMessage(`{}`)},
		{ID: "wh-3", URL: server.URL, Payload: json.RawMessage(`{}`)},
	}

	results, err := client.SendBatch(context.Background(), webhooks)

	require.NoError(t, err)
	assert.Len(t, results, 3)
	assert.Equal(t, 3, responseCount)

	for _, result := range results {
		assert.True(t, result.Success)
	}
}

func TestRetryPolicy_CalculateBackoff(t *testing.T) {
	policy := &RetryPolicy{
		MaxAttempts:    5,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     30 * time.Second,
		Multiplier:     2.0,
	}

	// CalculateBackoff uses attempt as the number of previous attempts
	// attempt <= 0 returns InitialBackoff
	// attempt 1 means after first failure, returns 1s (backoff applied once from initial)
	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 1 * time.Second},  // Initial backoff for first retry
		{1, 1 * time.Second},  // After 1 failure, still initial (loop doesn't run)
		{2, 2 * time.Second},  // 1s * 2 = 2s
		{3, 4 * time.Second},  // 2s * 2 = 4s
		{4, 8 * time.Second},  // 4s * 2 = 8s
		{5, 16 * time.Second}, // 8s * 2 = 16s
	}

	for _, tt := range tests {
		result := policy.CalculateBackoff(tt.attempt)
		assert.Equal(t, tt.expected, result, "attempt %d", tt.attempt)
	}
}

func TestDefaultRetryPolicy(t *testing.T) {
	policy := DefaultRetryPolicy()

	assert.Equal(t, 3, policy.MaxAttempts)
	assert.Equal(t, 5*time.Second, policy.InitialBackoff)
	assert.Equal(t, 5*time.Minute, policy.MaxBackoff)
	assert.Equal(t, 2.0, policy.Multiplier)
}
