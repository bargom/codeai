package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bargom/codeai/internal/event/bus"
)

// ErrEventNotFound is returned when an event is not found.
var ErrEventNotFound = errors.New("event not found")

// PostgresEventRepository implements EventRepository using PostgreSQL.
type PostgresEventRepository struct {
	db *sql.DB
}

// NewPostgresEventRepository creates a new PostgresEventRepository.
func NewPostgresEventRepository(db *sql.DB) *PostgresEventRepository {
	return &PostgresEventRepository{db: db}
}

// SaveEvent persists an event to the database.
func (r *PostgresEventRepository) SaveEvent(ctx context.Context, event bus.Event) error {
	dataJSON, err := json.Marshal(event.Data)
	if err != nil {
		return fmt.Errorf("marshaling event data: %w", err)
	}

	metadataJSON, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("marshaling event metadata: %w", err)
	}

	query := `
		INSERT INTO events (id, type, source, timestamp, data, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err = r.db.ExecContext(ctx, query,
		event.ID,
		string(event.Type),
		event.Source,
		event.Timestamp,
		dataJSON,
		metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("inserting event: %w", err)
	}

	return nil
}

// GetEvent retrieves an event by its ID.
func (r *PostgresEventRepository) GetEvent(ctx context.Context, eventID string) (*bus.Event, error) {
	query := `
		SELECT id, type, source, timestamp, data, metadata
		FROM events
		WHERE id = $1
	`

	var event bus.Event
	var dataJSON, metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, eventID).Scan(
		&event.ID,
		&event.Type,
		&event.Source,
		&event.Timestamp,
		&dataJSON,
		&metadataJSON,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEventNotFound
		}
		return nil, fmt.Errorf("querying event: %w", err)
	}

	if err := json.Unmarshal(dataJSON, &event.Data); err != nil {
		return nil, fmt.Errorf("unmarshaling event data: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &event.Metadata); err != nil {
		return nil, fmt.Errorf("unmarshaling event metadata: %w", err)
	}

	return &event, nil
}

// ListEvents retrieves events matching the filter criteria.
func (r *PostgresEventRepository) ListEvents(ctx context.Context, filter EventFilter) ([]bus.Event, error) {
	query, args := r.buildFilterQuery(filter, false)
	return r.queryEvents(ctx, query, args)
}

// GetEventsByType retrieves events of a specific type.
func (r *PostgresEventRepository) GetEventsByType(ctx context.Context, eventType bus.EventType, limit int) ([]bus.Event, error) {
	filter := EventFilter{
		Types: []bus.EventType{eventType},
		Limit: limit,
	}
	return r.ListEvents(ctx, filter)
}

// GetEventsBySource retrieves events from a specific source.
func (r *PostgresEventRepository) GetEventsBySource(ctx context.Context, source string, limit int) ([]bus.Event, error) {
	filter := EventFilter{
		Sources: []string{source},
		Limit:   limit,
	}
	return r.ListEvents(ctx, filter)
}

// CountEvents returns the total count of events matching the filter.
func (r *PostgresEventRepository) CountEvents(ctx context.Context, filter EventFilter) (int64, error) {
	query, args := r.buildFilterQuery(filter, true)

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting events: %w", err)
	}

	return count, nil
}

// DeleteOldEvents removes events older than the specified time.
func (r *PostgresEventRepository) DeleteOldEvents(ctx context.Context, before time.Time) (int64, error) {
	query := `DELETE FROM events WHERE timestamp < $1`

	result, err := r.db.ExecContext(ctx, query, before)
	if err != nil {
		return 0, fmt.Errorf("deleting old events: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("getting rows affected: %w", err)
	}

	return count, nil
}

// buildFilterQuery constructs a SQL query from the filter.
func (r *PostgresEventRepository) buildFilterQuery(filter EventFilter, countOnly bool) (string, []interface{}) {
	var selectClause string
	if countOnly {
		selectClause = "SELECT COUNT(*)"
	} else {
		selectClause = "SELECT id, type, source, timestamp, data, metadata"
	}

	query := selectClause + " FROM events WHERE 1=1"
	args := make([]interface{}, 0)
	argNum := 1

	if len(filter.Types) > 0 {
		placeholders := make([]string, len(filter.Types))
		for i, t := range filter.Types {
			placeholders[i] = fmt.Sprintf("$%d", argNum)
			args = append(args, string(t))
			argNum++
		}
		query += fmt.Sprintf(" AND type IN (%s)", strings.Join(placeholders, ", "))
	}

	if len(filter.Sources) > 0 {
		placeholders := make([]string, len(filter.Sources))
		for i, s := range filter.Sources {
			placeholders[i] = fmt.Sprintf("$%d", argNum)
			args = append(args, s)
			argNum++
		}
		query += fmt.Sprintf(" AND source IN (%s)", strings.Join(placeholders, ", "))
	}

	if filter.StartTime != nil {
		query += fmt.Sprintf(" AND timestamp >= $%d", argNum)
		args = append(args, *filter.StartTime)
		argNum++
	}

	if filter.EndTime != nil {
		query += fmt.Sprintf(" AND timestamp <= $%d", argNum)
		args = append(args, *filter.EndTime)
		argNum++
	}

	if !countOnly {
		query += " ORDER BY timestamp DESC"

		if filter.Limit > 0 {
			query += fmt.Sprintf(" LIMIT $%d", argNum)
			args = append(args, filter.Limit)
			argNum++
		}

		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argNum)
			args = append(args, filter.Offset)
		}
	}

	return query, args
}

// queryEvents executes a query and returns the events.
func (r *PostgresEventRepository) queryEvents(ctx context.Context, query string, args []interface{}) ([]bus.Event, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying events: %w", err)
	}
	defer rows.Close()

	var events []bus.Event
	for rows.Next() {
		var event bus.Event
		var dataJSON, metadataJSON []byte

		if err := rows.Scan(
			&event.ID,
			&event.Type,
			&event.Source,
			&event.Timestamp,
			&dataJSON,
			&metadataJSON,
		); err != nil {
			return nil, fmt.Errorf("scanning event: %w", err)
		}

		if err := json.Unmarshal(dataJSON, &event.Data); err != nil {
			return nil, fmt.Errorf("unmarshaling event data: %w", err)
		}

		if err := json.Unmarshal(metadataJSON, &event.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshaling event metadata: %w", err)
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating events: %w", err)
	}

	return events, nil
}

// CreateEventsTable creates the events table if it doesn't exist.
func (r *PostgresEventRepository) CreateEventsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS events (
			id VARCHAR(36) PRIMARY KEY,
			type VARCHAR(100) NOT NULL,
			source VARCHAR(255) NOT NULL,
			timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
			data JSONB NOT NULL DEFAULT '{}',
			metadata JSONB NOT NULL DEFAULT '{}'
		);

		CREATE INDEX IF NOT EXISTS idx_events_type ON events(type);
		CREATE INDEX IF NOT EXISTS idx_events_source ON events(source);
		CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);
	`

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("creating events table: %w", err)
	}

	return nil
}
