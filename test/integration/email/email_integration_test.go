//go:build integration

// Package email provides integration tests for the email notification service.
package email

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	emailrepo "github.com/bargom/codeai/internal/notification/email/repository"
	"github.com/bargom/codeai/test/integration/testutil"
)

// EmailTestSuite holds resources for email integration tests.
type EmailTestSuite struct {
	Repository *emailrepo.MemoryEmailRepository
	Fixtures   *testutil.FixtureBuilder
	Ctx        context.Context
	Cancel     context.CancelFunc
}

// NewEmailTestSuite creates a new email test suite.
func NewEmailTestSuite(t *testing.T) *EmailTestSuite {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)

	return &EmailTestSuite{
		Repository: emailrepo.NewMemoryEmailRepository(),
		Fixtures:   testutil.NewFixtureBuilder(),
		Ctx:        ctx,
		Cancel:     cancel,
	}
}

// Teardown cleans up the test suite.
func (s *EmailTestSuite) Teardown() {
	s.Cancel()
}

// Reset clears the repository.
func (s *EmailTestSuite) Reset() {
	s.Repository.Clear()
}

func TestEmailRepositoryCRUD(t *testing.T) {
	suite := NewEmailTestSuite(t)
	defer suite.Teardown()

	t.Run("save and retrieve email log", func(t *testing.T) {
		suite.Reset()

		email := suite.Fixtures.CreateTestEmailLog(
			[]string{"test@example.com"},
			"Test Subject",
			"sent",
		)

		err := suite.Repository.SaveEmail(suite.Ctx, email)
		require.NoError(t, err)

		retrieved, err := suite.Repository.GetEmail(suite.Ctx, email.ID)
		require.NoError(t, err)
		assert.Equal(t, email.ID, retrieved.ID)
		assert.Equal(t, email.Subject, retrieved.Subject)
		assert.Equal(t, email.Status, retrieved.Status)
		assert.Equal(t, email.To, retrieved.To)
	})

	t.Run("update email status", func(t *testing.T) {
		suite.Reset()

		email := suite.Fixtures.CreateTestEmailLog(
			[]string{"test@example.com"},
			"Test Subject",
			"sent",
		)
		require.NoError(t, suite.Repository.SaveEmail(suite.Ctx, email))

		err := suite.Repository.UpdateStatus(suite.Ctx, email.ID, "delivered")
		require.NoError(t, err)

		retrieved, err := suite.Repository.GetEmail(suite.Ctx, email.ID)
		require.NoError(t, err)
		assert.Equal(t, "delivered", retrieved.Status)
	})

	t.Run("get non-existent email returns error", func(t *testing.T) {
		suite.Reset()

		_, err := suite.Repository.GetEmail(suite.Ctx, "non-existent-id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("save email with nil fails", func(t *testing.T) {
		suite.Reset()

		err := suite.Repository.SaveEmail(suite.Ctx, nil)
		assert.Error(t, err)
	})

	t.Run("save email without ID fails", func(t *testing.T) {
		suite.Reset()

		email := &emailrepo.EmailLog{
			To:      []string{"test@example.com"},
			Subject: "Test",
			Status:  "sent",
		}

		err := suite.Repository.SaveEmail(suite.Ctx, email)
		assert.Error(t, err)
	})
}

func TestEmailListingAndFiltering(t *testing.T) {
	suite := NewEmailTestSuite(t)
	defer suite.Teardown()

	t.Run("list emails by status", func(t *testing.T) {
		suite.Reset()

		// Create emails with different statuses
		sentEmail := suite.Fixtures.CreateTestEmailLog([]string{"a@test.com"}, "Sent", "sent")
		require.NoError(t, suite.Repository.SaveEmail(suite.Ctx, sentEmail))

		deliveredEmail := suite.Fixtures.CreateTestEmailLog([]string{"b@test.com"}, "Delivered", "delivered")
		require.NoError(t, suite.Repository.SaveEmail(suite.Ctx, deliveredEmail))

		failedEmail := suite.Fixtures.CreateTestEmailLog([]string{"c@test.com"}, "Failed", "failed")
		require.NoError(t, suite.Repository.SaveEmail(suite.Ctx, failedEmail))

		sentStatus := "sent"
		filter := emailrepo.EmailFilter{
			Status: &sentStatus,
			Limit:  10,
		}

		emails, err := suite.Repository.ListEmails(suite.Ctx, filter)
		require.NoError(t, err)
		assert.Len(t, emails, 1)
		assert.Equal(t, "sent", emails[0].Status)
	})

	t.Run("list emails by recipient", func(t *testing.T) {
		suite.Reset()

		email1 := suite.Fixtures.CreateTestEmailLog([]string{"alice@test.com"}, "Email 1", "sent")
		require.NoError(t, suite.Repository.SaveEmail(suite.Ctx, email1))

		email2 := suite.Fixtures.CreateTestEmailLog([]string{"bob@test.com"}, "Email 2", "sent")
		require.NoError(t, suite.Repository.SaveEmail(suite.Ctx, email2))

		email3 := suite.Fixtures.CreateTestEmailLog([]string{"alice@test.com", "charlie@test.com"}, "Email 3", "sent")
		require.NoError(t, suite.Repository.SaveEmail(suite.Ctx, email3))

		alice := "alice@test.com"
		filter := emailrepo.EmailFilter{
			To:    &alice,
			Limit: 10,
		}

		emails, err := suite.Repository.ListEmails(suite.Ctx, filter)
		require.NoError(t, err)
		assert.Len(t, emails, 2) // email1 and email3 contain alice
	})

	t.Run("list emails with pagination", func(t *testing.T) {
		suite.Reset()

		// Create 5 emails
		for i := 0; i < 5; i++ {
			email := suite.Fixtures.CreateTestEmailLog(
				[]string{"test@example.com"},
				"Email",
				"sent",
			)
			require.NoError(t, suite.Repository.SaveEmail(suite.Ctx, email))
		}

		// Get first page
		filter := emailrepo.EmailFilter{
			Limit:  2,
			Offset: 0,
		}
		emails, err := suite.Repository.ListEmails(suite.Ctx, filter)
		require.NoError(t, err)
		assert.Len(t, emails, 2)

		// Get second page
		filter.Offset = 2
		emails, err = suite.Repository.ListEmails(suite.Ctx, filter)
		require.NoError(t, err)
		assert.Len(t, emails, 2)

		// Get third page
		filter.Offset = 4
		emails, err = suite.Repository.ListEmails(suite.Ctx, filter)
		require.NoError(t, err)
		assert.Len(t, emails, 1)
	})

	t.Run("list emails by time range", func(t *testing.T) {
		suite.Reset()

		// Create old email
		oldEmail := suite.Fixtures.CreateTestEmailLog([]string{"old@test.com"}, "Old", "sent")
		oldEmail.SentAt = time.Now().Add(-2 * time.Hour)
		require.NoError(t, suite.Repository.SaveEmail(suite.Ctx, oldEmail))

		// Create recent email
		recentEmail := suite.Fixtures.CreateTestEmailLog([]string{"recent@test.com"}, "Recent", "sent")
		require.NoError(t, suite.Repository.SaveEmail(suite.Ctx, recentEmail))

		// Filter for recent emails only
		startTime := time.Now().Add(-1 * time.Hour)
		filter := emailrepo.EmailFilter{
			StartTime: &startTime,
			Limit:     10,
		}

		emails, err := suite.Repository.ListEmails(suite.Ctx, filter)
		require.NoError(t, err)
		assert.Len(t, emails, 1)
		assert.Equal(t, "Recent", emails[0].Subject)
	})
}

func TestEmailLogContent(t *testing.T) {
	suite := NewEmailTestSuite(t)
	defer suite.Teardown()

	t.Run("email with multiple recipients", func(t *testing.T) {
		suite.Reset()

		recipients := []string{"a@test.com", "b@test.com", "c@test.com"}
		email := suite.Fixtures.CreateTestEmailLog(recipients, "Multi-recipient", "sent")
		require.NoError(t, suite.Repository.SaveEmail(suite.Ctx, email))

		retrieved, err := suite.Repository.GetEmail(suite.Ctx, email.ID)
		require.NoError(t, err)
		assert.Equal(t, recipients, retrieved.To)
		assert.Len(t, retrieved.To, 3)
	})

	t.Run("email with metadata", func(t *testing.T) {
		suite.Reset()

		email := &emailrepo.EmailLog{
			ID:      "meta-test-id",
			To:      []string{"test@example.com"},
			Subject: "Test with metadata",
			Status:  "sent",
			SentAt:  time.Now(),
			Metadata: map[string]interface{}{
				"campaign":   "test-campaign",
				"priority":   "high",
				"templateId": 12345,
			},
		}
		require.NoError(t, suite.Repository.SaveEmail(suite.Ctx, email))

		retrieved, err := suite.Repository.GetEmail(suite.Ctx, email.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrieved.Metadata)
		assert.Equal(t, "test-campaign", retrieved.Metadata["campaign"])
		assert.Equal(t, "high", retrieved.Metadata["priority"])
	})

	t.Run("email status transitions", func(t *testing.T) {
		suite.Reset()

		email := suite.Fixtures.CreateTestEmailLog([]string{"test@example.com"}, "Status test", "sent")
		require.NoError(t, suite.Repository.SaveEmail(suite.Ctx, email))

		// Transition to delivered
		err := suite.Repository.UpdateStatus(suite.Ctx, email.ID, "delivered")
		require.NoError(t, err)

		retrieved, err := suite.Repository.GetEmail(suite.Ctx, email.ID)
		require.NoError(t, err)
		assert.Equal(t, "delivered", retrieved.Status)

		// Transition to opened
		err = suite.Repository.UpdateStatus(suite.Ctx, email.ID, "opened")
		require.NoError(t, err)

		retrieved, err = suite.Repository.GetEmail(suite.Ctx, email.ID)
		require.NoError(t, err)
		assert.Equal(t, "opened", retrieved.Status)
	})
}

func TestMockEmailClient(t *testing.T) {
	t.Run("send email successfully", func(t *testing.T) {
		client := testutil.NewMockEmailClient()

		messageID, err := client.Send(
			[]string{"test@example.com"},
			"Test Subject",
			"<h1>HTML</h1>",
			"Text content",
		)

		require.NoError(t, err)
		assert.Contains(t, messageID, "mock-message-id-")
		assert.Equal(t, 1, client.Count())

		emails := client.GetSentEmails()
		assert.Len(t, emails, 1)
		assert.Equal(t, []string{"test@example.com"}, emails[0].To)
		assert.Equal(t, "Test Subject", emails[0].Subject)
	})

	t.Run("simulate email failure", func(t *testing.T) {
		client := testutil.NewMockEmailClient()
		client.ShouldFail = true

		_, err := client.Send(
			[]string{"test@example.com"},
			"Test Subject",
			"",
			"",
		)

		assert.Error(t, err)
		assert.Equal(t, 0, client.Count()) // Failed email not counted
	})

	t.Run("fail after N emails", func(t *testing.T) {
		client := testutil.NewMockEmailClient()
		client.FailAfter = 2

		// First two should succeed
		_, err := client.Send([]string{"a@test.com"}, "Email 1", "", "")
		require.NoError(t, err)

		_, err = client.Send([]string{"b@test.com"}, "Email 2", "", "")
		require.NoError(t, err)

		// Third should fail
		_, err = client.Send([]string{"c@test.com"}, "Email 3", "", "")
		assert.Error(t, err)

		assert.Equal(t, 2, client.Count())
	})

	t.Run("reset clears state", func(t *testing.T) {
		client := testutil.NewMockEmailClient()
		client.ShouldFail = true

		client.Reset()

		assert.False(t, client.ShouldFail)
		assert.Equal(t, 0, client.Count())
	})
}

func TestEmailRepositoryCount(t *testing.T) {
	suite := NewEmailTestSuite(t)
	defer suite.Teardown()

	t.Run("count returns correct number", func(t *testing.T) {
		suite.Reset()

		assert.Equal(t, 0, suite.Repository.Count())

		for i := 0; i < 5; i++ {
			email := suite.Fixtures.CreateTestEmailLog(
				[]string{"test@example.com"},
				"Test",
				"sent",
			)
			require.NoError(t, suite.Repository.SaveEmail(suite.Ctx, email))
		}

		assert.Equal(t, 5, suite.Repository.Count())

		suite.Repository.Clear()
		assert.Equal(t, 0, suite.Repository.Count())
	})
}

func TestConcurrentEmailOperations(t *testing.T) {
	suite := NewEmailTestSuite(t)
	defer suite.Teardown()

	t.Run("concurrent saves are safe", func(t *testing.T) {
		suite.Reset()

		const numEmails = 100
		done := make(chan bool, numEmails)

		for i := 0; i < numEmails; i++ {
			go func(idx int) {
				email := suite.Fixtures.CreateTestEmailLog(
					[]string{"test@example.com"},
					"Concurrent test",
					"sent",
				)
				err := suite.Repository.SaveEmail(suite.Ctx, email)
				if err != nil {
					t.Errorf("failed to save email %d: %v", idx, err)
				}
				done <- true
			}(i)
		}

		for i := 0; i < numEmails; i++ {
			<-done
		}

		assert.Equal(t, numEmails, suite.Repository.Count())
	})

	t.Run("concurrent reads are safe", func(t *testing.T) {
		suite.Reset()

		// Create some emails
		emailIDs := make([]string, 10)
		for i := 0; i < 10; i++ {
			email := suite.Fixtures.CreateTestEmailLog(
				[]string{"test@example.com"},
				"Read test",
				"sent",
			)
			require.NoError(t, suite.Repository.SaveEmail(suite.Ctx, email))
			emailIDs[i] = email.ID
		}

		const numReaders = 50
		done := make(chan bool, numReaders)

		for i := 0; i < numReaders; i++ {
			go func(idx int) {
				emailID := emailIDs[idx%len(emailIDs)]
				_, err := suite.Repository.GetEmail(suite.Ctx, emailID)
				if err != nil {
					t.Errorf("failed to read email: %v", err)
				}
				done <- true
			}(i)
		}

		for i := 0; i < numReaders; i++ {
			<-done
		}
	})
}
