package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryEmailRepository_SaveAndGet(t *testing.T) {
	repo := NewMemoryEmailRepository()
	ctx := context.Background()

	email := &EmailLog{
		ID:           "email-123",
		MessageID:    "brevo-msg-456",
		To:           []string{"user@example.com"},
		Subject:      "Test Subject",
		TemplateType: "welcome",
		Status:       "sent",
		SentAt:       time.Now(),
	}

	err := repo.SaveEmail(ctx, email)
	require.NoError(t, err)

	retrieved, err := repo.GetEmail(ctx, "email-123")
	require.NoError(t, err)

	assert.Equal(t, email.ID, retrieved.ID)
	assert.Equal(t, email.MessageID, retrieved.MessageID)
	assert.Equal(t, email.Subject, retrieved.Subject)
	assert.Equal(t, email.Status, retrieved.Status)
}

func TestMemoryEmailRepository_GetNotFound(t *testing.T) {
	repo := NewMemoryEmailRepository()
	ctx := context.Background()

	_, err := repo.GetEmail(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMemoryEmailRepository_ListEmails(t *testing.T) {
	repo := NewMemoryEmailRepository()
	ctx := context.Background()

	now := time.Now()

	// Save multiple emails
	emails := []*EmailLog{
		{ID: "1", To: []string{"a@example.com"}, Status: "sent", SentAt: now},
		{ID: "2", To: []string{"b@example.com"}, Status: "delivered", SentAt: now.Add(time.Hour)},
		{ID: "3", To: []string{"a@example.com"}, Status: "sent", SentAt: now.Add(2 * time.Hour)},
	}

	for _, e := range emails {
		err := repo.SaveEmail(ctx, e)
		require.NoError(t, err)
	}

	// Test listing all
	result, err := repo.ListEmails(ctx, EmailFilter{})
	require.NoError(t, err)
	assert.Len(t, result, 3)

	// Test filtering by status
	sentStatus := "sent"
	result, err = repo.ListEmails(ctx, EmailFilter{Status: &sentStatus})
	require.NoError(t, err)
	assert.Len(t, result, 2)

	// Test filtering by recipient
	recipient := "a@example.com"
	result, err = repo.ListEmails(ctx, EmailFilter{To: &recipient})
	require.NoError(t, err)
	assert.Len(t, result, 2)

	// Test pagination
	result, err = repo.ListEmails(ctx, EmailFilter{Limit: 2})
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestMemoryEmailRepository_UpdateStatus(t *testing.T) {
	repo := NewMemoryEmailRepository()
	ctx := context.Background()

	email := &EmailLog{
		ID:     "email-123",
		Status: "sent",
		SentAt: time.Now(),
	}

	err := repo.SaveEmail(ctx, email)
	require.NoError(t, err)

	err = repo.UpdateStatus(ctx, "email-123", "delivered")
	require.NoError(t, err)

	retrieved, err := repo.GetEmail(ctx, "email-123")
	require.NoError(t, err)
	assert.Equal(t, "delivered", retrieved.Status)
}

func TestMemoryEmailRepository_UpdateStatusNotFound(t *testing.T) {
	repo := NewMemoryEmailRepository()
	ctx := context.Background()

	err := repo.UpdateStatus(ctx, "nonexistent", "delivered")
	assert.Error(t, err)
}

func TestMemoryEmailRepository_Clear(t *testing.T) {
	repo := NewMemoryEmailRepository()
	ctx := context.Background()

	email := &EmailLog{ID: "1", SentAt: time.Now()}
	err := repo.SaveEmail(ctx, email)
	require.NoError(t, err)

	assert.Equal(t, 1, repo.Count())

	repo.Clear()
	assert.Equal(t, 0, repo.Count())
}

func TestMemoryEmailRepository_SaveValidation(t *testing.T) {
	repo := NewMemoryEmailRepository()
	ctx := context.Background()

	// Test nil email
	err := repo.SaveEmail(ctx, nil)
	assert.Error(t, err)

	// Test empty ID
	err = repo.SaveEmail(ctx, &EmailLog{})
	assert.Error(t, err)
}
