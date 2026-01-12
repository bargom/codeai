package brevo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: Config{
				APIKey: "test-api-key",
				DefaultSender: EmailAddress{
					Name:  "Test",
					Email: "test@example.com",
				},
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			cfg: Config{
				DefaultSender: EmailAddress{
					Name:  "Test",
					Email: "test@example.com",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestClient_SendTransactionalEmail(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/smtp/email", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "test-api-key", r.Header.Get("api-key"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req sendEmailRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		assert.Equal(t, "Test Subject", req.Subject)
		assert.Len(t, req.To, 1)
		assert.Equal(t, "user@example.com", req.To[0].Email)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(sendEmailResponse{
			MessageID: "test-message-id-123",
		})
	}))
	defer server.Close()

	client, err := NewClient(Config{
		APIKey: "test-api-key",
		DefaultSender: EmailAddress{
			Name:  "Test Sender",
			Email: "sender@example.com",
		},
		BaseURL: server.URL,
	})
	require.NoError(t, err)

	email := &TransactionalEmail{
		To: []EmailAddress{
			{Email: "user@example.com", Name: "Test User"},
		},
		Subject:     "Test Subject",
		HTMLContent: "<p>Test content</p>",
	}

	messageID, err := client.SendTransactionalEmail(context.Background(), email)
	assert.NoError(t, err)
	assert.Equal(t, "test-message-id-123", messageID)
}

func TestClient_SendTransactionalEmail_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "invalid email"}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	})
	require.NoError(t, err)

	email := &TransactionalEmail{
		To:      []EmailAddress{{Email: "invalid"}},
		Subject: "Test",
	}

	_, err = client.SendTransactionalEmail(context.Background(), email)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status 400")
}

func TestClient_SendTemplateEmail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req sendTemplateRequest
		json.NewDecoder(r.Body).Decode(&req)

		assert.Equal(t, int64(123), req.TemplateID)
		assert.Equal(t, "John", req.Params["userName"])

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(sendEmailResponse{
			MessageID: "template-message-id",
		})
	}))
	defer server.Close()

	client, err := NewClient(Config{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	})
	require.NoError(t, err)

	email := &TemplateEmail{
		To:         []EmailAddress{{Email: "user@example.com"}},
		TemplateID: 123,
		Params: map[string]interface{}{
			"userName": "John",
		},
	}

	messageID, err := client.SendTemplateEmail(context.Background(), email)
	assert.NoError(t, err)
	assert.Equal(t, "template-message-id", messageID)
}

func TestClient_GetEmailStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/smtp/emails/msg-123", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		w.WriteHeader(http.StatusOK)
		// Return raw JSON to avoid struct type mismatch
		w.Write([]byte(`{"messageId":"msg-123","event":"delivered","events":[]}`))

	}))
	defer server.Close()

	client, err := NewClient(Config{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	})
	require.NoError(t, err)

	status, err := client.GetEmailStatus(context.Background(), "msg-123")
	assert.NoError(t, err)
	assert.Equal(t, "msg-123", status.MessageID)
	assert.Equal(t, "delivered", status.Status)
}

func TestClient_GetEmailStatus_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := NewClient(Config{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	})
	require.NoError(t, err)

	_, err = client.GetEmailStatus(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestClient_Ping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/account", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"email": "test@example.com"}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	})
	require.NoError(t, err)

	err = client.Ping(context.Background())
	assert.NoError(t, err)
}
