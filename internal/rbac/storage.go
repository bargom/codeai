package rbac

import (
	"context"
	"sync"
	"time"
)

// Storage defines the interface for policy storage backends.
type Storage interface {
	// GetRole retrieves a role by name.
	GetRole(ctx context.Context, name string) (*Role, error)

	// ListRoles returns all roles.
	ListRoles(ctx context.Context) ([]*Role, error)

	// SaveRole creates or updates a role.
	SaveRole(ctx context.Context, role *Role) error

	// DeleteRole removes a role.
	DeleteRole(ctx context.Context, name string) error
}

// MemoryStorage is an in-memory implementation of Storage.
type MemoryStorage struct {
	roles map[string]*Role
	mu    sync.RWMutex
}

// NewMemoryStorage creates a new in-memory storage.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		roles: make(map[string]*Role),
	}
}

// NewMemoryStorageWithDefaults creates a storage with default roles loaded.
func NewMemoryStorageWithDefaults() *MemoryStorage {
	s := NewMemoryStorage()
	for i := range DefaultRoles {
		role := DefaultRoles[i]
		s.roles[role.Name] = &role
	}
	return s
}

// GetRole retrieves a role by name.
func (s *MemoryStorage) GetRole(ctx context.Context, name string) (*Role, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	role, ok := s.roles[name]
	if !ok {
		return nil, ErrRoleNotFound
	}

	// Return a copy to prevent external modifications
	roleCopy := *role
	return &roleCopy, nil
}

// ListRoles returns all roles.
func (s *MemoryStorage) ListRoles(ctx context.Context) ([]*Role, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	roles := make([]*Role, 0, len(s.roles))
	for _, role := range s.roles {
		roleCopy := *role
		roles = append(roles, &roleCopy)
	}
	return roles, nil
}

// SaveRole creates or updates a role.
func (s *MemoryStorage) SaveRole(ctx context.Context, role *Role) error {
	if err := validateRoleName(role.Name); err != nil {
		return err
	}

	for _, perm := range role.Permissions {
		if err := validatePermission(perm); err != nil {
			return err
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Store a copy
	roleCopy := *role
	s.roles[role.Name] = &roleCopy
	return nil
}

// DeleteRole removes a role.
func (s *MemoryStorage) DeleteRole(ctx context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.roles[name]; !ok {
		return ErrRoleNotFound
	}

	delete(s.roles, name)
	return nil
}

// CachedStorage wraps another storage with TTL-based caching.
type CachedStorage struct {
	backend Storage
	cache   map[string]*cachedRole
	listTTL time.Duration
	ttl     time.Duration
	mu      sync.RWMutex
}

type cachedRole struct {
	role      *Role
	expiresAt time.Time
}

// CachedStorageOption configures the cached storage.
type CachedStorageOption func(*CachedStorage)

// WithTTL sets the cache TTL for individual role lookups.
func WithTTL(ttl time.Duration) CachedStorageOption {
	return func(s *CachedStorage) {
		s.ttl = ttl
	}
}

// WithListTTL sets the cache TTL for list operations.
func WithListTTL(ttl time.Duration) CachedStorageOption {
	return func(s *CachedStorage) {
		s.listTTL = ttl
	}
}

// NewCachedStorage creates a new cached storage wrapper.
func NewCachedStorage(backend Storage, opts ...CachedStorageOption) *CachedStorage {
	s := &CachedStorage{
		backend: backend,
		cache:   make(map[string]*cachedRole),
		ttl:     5 * time.Minute,
		listTTL: 1 * time.Minute,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// GetRole retrieves a role by name with caching.
func (s *CachedStorage) GetRole(ctx context.Context, name string) (*Role, error) {
	s.mu.RLock()
	if cached, ok := s.cache[name]; ok && time.Now().Before(cached.expiresAt) {
		s.mu.RUnlock()
		// Return a copy
		roleCopy := *cached.role
		return &roleCopy, nil
	}
	s.mu.RUnlock()

	// Fetch from backend
	role, err := s.backend.GetRole(ctx, name)
	if err != nil {
		return nil, err
	}

	// Cache the result
	s.mu.Lock()
	s.cache[name] = &cachedRole{
		role:      role,
		expiresAt: time.Now().Add(s.ttl),
	}
	s.mu.Unlock()

	// Return a copy
	roleCopy := *role
	return &roleCopy, nil
}

// ListRoles returns all roles.
func (s *CachedStorage) ListRoles(ctx context.Context) ([]*Role, error) {
	return s.backend.ListRoles(ctx)
}

// SaveRole creates or updates a role and invalidates cache.
func (s *CachedStorage) SaveRole(ctx context.Context, role *Role) error {
	err := s.backend.SaveRole(ctx, role)
	if err != nil {
		return err
	}

	// Invalidate cache for this role
	s.mu.Lock()
	delete(s.cache, role.Name)
	s.mu.Unlock()

	return nil
}

// DeleteRole removes a role and invalidates cache.
func (s *CachedStorage) DeleteRole(ctx context.Context, name string) error {
	err := s.backend.DeleteRole(ctx, name)
	if err != nil {
		return err
	}

	// Invalidate cache
	s.mu.Lock()
	delete(s.cache, name)
	s.mu.Unlock()

	return nil
}

// InvalidateCache clears the entire cache.
func (s *CachedStorage) InvalidateCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = make(map[string]*cachedRole)
}

// InvalidateRole removes a specific role from the cache.
func (s *CachedStorage) InvalidateRole(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.cache, name)
}
