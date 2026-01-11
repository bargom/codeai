# Task: Event Emission and Subscription System

## Overview
Implement an event system for internal pub/sub messaging and external event publishing.

## Phase
Phase 3: Workflows and Jobs

## Priority
High - Core feature for event-driven architecture.

## Dependencies
- Phase 1 complete

## Description
Create an event module that supports internal event subscription, event emission, and integration with external message brokers.

## Detailed Requirements

### 1. Event Types (internal/modules/event/types.go)

```go
package event

import (
    "time"
)

type Event struct {
    ID          string
    Description string
    Payload     *PayloadDef
    PublishTo   []Publisher
    Trigger     *EventTrigger
}

type PayloadDef struct {
    Fields []PayloadField
}

type PayloadField struct {
    Name     string
    Type     string
    Required bool
}

type EventTrigger struct {
    Entity    string
    Condition string
}

type EventMessage struct {
    ID        string         `json:"id"`
    Type      string         `json:"type"`
    Payload   any            `json:"payload"`
    Metadata  EventMetadata  `json:"metadata"`
    Timestamp time.Time      `json:"timestamp"`
}

type EventMetadata struct {
    Source      string `json:"source"`
    TraceID     string `json:"trace_id,omitempty"`
    UserID      string `json:"user_id,omitempty"`
    RequestID   string `json:"request_id,omitempty"`
}

type EventHandler func(ctx *ExecutionContext, payload any) error
```

### 2. Event Module (internal/modules/event/module.go)

```go
package event

import (
    "context"
    "sync"
    "time"

    "github.com/google/uuid"
    "log/slog"
)

type EventModule interface {
    Module
    RegisterEvent(event *Event) error
    Subscribe(eventType string, handler EventHandler) error
    Unsubscribe(eventType string, handler EventHandler) error
    Emit(ctx context.Context, eventType string, payload any) error
    EmitAsync(ctx context.Context, eventType string, payload any) error
}

type eventModule struct {
    events      map[string]*Event
    subscribers map[string][]EventHandler
    publishers  map[string]Publisher
    queue       chan *eventJob
    mu          sync.RWMutex
    logger      *slog.Logger
    wg          sync.WaitGroup
    ctx         context.Context
    cancel      context.CancelFunc
}

type eventJob struct {
    event   *EventMessage
    def     *Event
}

func NewEventModule() EventModule {
    ctx, cancel := context.WithCancel(context.Background())

    m := &eventModule{
        events:      make(map[string]*Event),
        subscribers: make(map[string][]EventHandler),
        publishers:  make(map[string]Publisher),
        queue:       make(chan *eventJob, 1000),
        logger:      slog.Default().With("module", "event"),
        ctx:         ctx,
        cancel:      cancel,
    }

    return m
}

func (m *eventModule) Name() string { return "event" }

func (m *eventModule) Initialize(config *Config) error {
    return nil
}

func (m *eventModule) Start(ctx context.Context) error {
    // Start worker goroutines
    for i := 0; i < 5; i++ {
        m.wg.Add(1)
        go m.worker()
    }

    m.logger.Info("event module started")
    return nil
}

func (m *eventModule) Stop(ctx context.Context) error {
    m.cancel()
    close(m.queue)
    m.wg.Wait()
    return nil
}

func (m *eventModule) Health() HealthStatus {
    return HealthStatus{
        Status: "healthy",
        Details: map[string]any{
            "queue_size":   len(m.queue),
            "events":       len(m.events),
            "subscribers":  len(m.subscribers),
        },
    }
}

func (m *eventModule) RegisterEvent(event *Event) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    m.events[event.ID] = event
    return nil
}

func (m *eventModule) Subscribe(eventType string, handler EventHandler) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    m.subscribers[eventType] = append(m.subscribers[eventType], handler)
    m.logger.Debug("handler subscribed", "event", eventType)
    return nil
}

func (m *eventModule) Unsubscribe(eventType string, handler EventHandler) error {
    // Implementation for unsubscribe
    return nil
}

func (m *eventModule) Emit(ctx context.Context, eventType string, payload any) error {
    msg := m.createMessage(ctx, eventType, payload)

    m.mu.RLock()
    def := m.events[eventType]
    handlers := m.subscribers[eventType]
    m.mu.RUnlock()

    m.logger.Info("emitting event", "type", eventType, "id", msg.ID)

    // Execute local handlers synchronously
    execCtx := &ExecutionContext{ctx: ctx}
    for _, handler := range handlers {
        if err := handler(execCtx, payload); err != nil {
            m.logger.Error("handler error", "event", eventType, "error", err)
        }
    }

    // Publish to external systems
    if def != nil {
        for _, pub := range def.PublishTo {
            if err := pub.Publish(ctx, eventType, msg); err != nil {
                m.logger.Error("publish error", "event", eventType, "publisher", pub.Name(), "error", err)
            }
        }
    }

    return nil
}

func (m *eventModule) EmitAsync(ctx context.Context, eventType string, payload any) error {
    msg := m.createMessage(ctx, eventType, payload)

    m.mu.RLock()
    def := m.events[eventType]
    m.mu.RUnlock()

    select {
    case m.queue <- &eventJob{event: msg, def: def}:
        return nil
    default:
        return fmt.Errorf("event queue full")
    }
}

func (m *eventModule) createMessage(ctx context.Context, eventType string, payload any) *EventMessage {
    metadata := EventMetadata{
        Source: "codeai",
    }

    // Extract metadata from context
    if traceID, ok := ctx.Value("trace_id").(string); ok {
        metadata.TraceID = traceID
    }
    if userID, ok := ctx.Value("user_id").(string); ok {
        metadata.UserID = userID
    }
    if requestID, ok := ctx.Value("request_id").(string); ok {
        metadata.RequestID = requestID
    }

    return &EventMessage{
        ID:        uuid.New().String(),
        Type:      eventType,
        Payload:   payload,
        Metadata:  metadata,
        Timestamp: time.Now().UTC(),
    }
}

func (m *eventModule) worker() {
    defer m.wg.Done()

    for job := range m.queue {
        m.processJob(job)
    }
}

func (m *eventModule) processJob(job *eventJob) {
    ctx := context.Background()

    m.mu.RLock()
    handlers := m.subscribers[job.event.Type]
    m.mu.RUnlock()

    // Execute local handlers
    execCtx := &ExecutionContext{ctx: ctx}
    for _, handler := range handlers {
        if err := handler(execCtx, job.event.Payload); err != nil {
            m.logger.Error("async handler error",
                "event", job.event.Type,
                "error", err,
            )
        }
    }

    // Publish to external systems
    if job.def != nil {
        for _, pub := range job.def.PublishTo {
            if err := pub.Publish(ctx, job.event.Type, job.event); err != nil {
                m.logger.Error("async publish error",
                    "event", job.event.Type,
                    "publisher", pub.Name(),
                    "error", err,
                )
            }
        }
    }
}

func (m *eventModule) RegisterPublisher(name string, pub Publisher) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.publishers[name] = pub
}
```

### 3. Publisher Interface

```go
// internal/modules/event/publisher.go
package event

import "context"

type Publisher interface {
    Name() string
    Publish(ctx context.Context, eventType string, msg *EventMessage) error
    Close() error
}
```

### 4. In-Memory Publisher (for testing)

```go
// internal/modules/event/memory_publisher.go
package event

type MemoryPublisher struct {
    messages []*EventMessage
    mu       sync.Mutex
}

func NewMemoryPublisher() *MemoryPublisher {
    return &MemoryPublisher{
        messages: make([]*EventMessage, 0),
    }
}

func (p *MemoryPublisher) Name() string { return "memory" }

func (p *MemoryPublisher) Publish(ctx context.Context, eventType string, msg *EventMessage) error {
    p.mu.Lock()
    defer p.mu.Unlock()
    p.messages = append(p.messages, msg)
    return nil
}

func (p *MemoryPublisher) Close() error { return nil }

func (p *MemoryPublisher) Messages() []*EventMessage {
    p.mu.Lock()
    defer p.mu.Unlock()
    return append([]*EventMessage{}, p.messages...)
}
```

## Acceptance Criteria
- [ ] Event registration with payload schema
- [ ] Synchronous event emission
- [ ] Asynchronous event emission with queue
- [ ] Handler subscription/unsubscription
- [ ] Event metadata propagation
- [ ] Publisher interface for external systems
- [ ] Graceful shutdown

## Testing Strategy
- Unit tests for event emission
- Unit tests for subscription
- Integration tests with handlers
- Concurrency tests

## Files to Create
- `internal/modules/event/types.go`
- `internal/modules/event/module.go`
- `internal/modules/event/publisher.go`
- `internal/modules/event/memory_publisher.go`
- `internal/modules/event/event_test.go`
