package repository

import (
	"context"
	"fmt"
	"sync"
)

// MemoryEmailRepository is an in-memory implementation of EmailRepository.
type MemoryEmailRepository struct {
	mu     sync.RWMutex
	emails map[string]*EmailLog
}

// NewMemoryEmailRepository creates a new in-memory email repository.
func NewMemoryEmailRepository() *MemoryEmailRepository {
	return &MemoryEmailRepository{
		emails: make(map[string]*EmailLog),
	}
}

// SaveEmail persists an email log entry in memory.
func (r *MemoryEmailRepository) SaveEmail(ctx context.Context, email *EmailLog) error {
	if email == nil {
		return fmt.Errorf("email is nil")
	}
	if email.ID == "" {
		return fmt.Errorf("email ID is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Make a copy to prevent external modifications
	emailCopy := *email
	if email.To != nil {
		emailCopy.To = make([]string, len(email.To))
		copy(emailCopy.To, email.To)
	}
	if email.Metadata != nil {
		emailCopy.Metadata = make(map[string]interface{})
		for k, v := range email.Metadata {
			emailCopy.Metadata[k] = v
		}
	}

	r.emails[email.ID] = &emailCopy
	return nil
}

// GetEmail retrieves an email log by ID.
func (r *MemoryEmailRepository) GetEmail(ctx context.Context, emailID string) (*EmailLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	email, ok := r.emails[emailID]
	if !ok {
		return nil, fmt.Errorf("email not found: %s", emailID)
	}

	// Return a copy
	copy := *email
	return &copy, nil
}

// ListEmails retrieves email logs with optional filtering.
func (r *MemoryEmailRepository) ListEmails(ctx context.Context, filter EmailFilter) ([]EmailLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []EmailLog

	for _, email := range r.emails {
		// Apply filters
		if filter.Status != nil && email.Status != *filter.Status {
			continue
		}

		if filter.To != nil {
			found := false
			for _, to := range email.To {
				if to == *filter.To {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		if filter.StartTime != nil && email.SentAt.Before(*filter.StartTime) {
			continue
		}

		if filter.EndTime != nil && email.SentAt.After(*filter.EndTime) {
			continue
		}

		results = append(results, *email)
	}

	// Apply pagination
	if filter.Offset > 0 && filter.Offset < len(results) {
		results = results[filter.Offset:]
	} else if filter.Offset >= len(results) {
		return []EmailLog{}, nil
	}

	if filter.Limit > 0 && filter.Limit < len(results) {
		results = results[:filter.Limit]
	}

	return results, nil
}

// UpdateStatus updates the status of an email log.
func (r *MemoryEmailRepository) UpdateStatus(ctx context.Context, emailID string, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	email, ok := r.emails[emailID]
	if !ok {
		return fmt.Errorf("email not found: %s", emailID)
	}

	email.Status = status
	return nil
}

// Clear removes all emails from the repository (useful for testing).
func (r *MemoryEmailRepository) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.emails = make(map[string]*EmailLog)
}

// Count returns the number of emails in the repository.
func (r *MemoryEmailRepository) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.emails)
}
