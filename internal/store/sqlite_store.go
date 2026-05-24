package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteExecutionStore persists execution state into a SQLite database.
type SQLiteExecutionStore struct {
	db *sql.DB
}

// NewSQLiteExecutionStore opens/initializes SQLite and applies defensive PRAGMAs.
func NewSQLiteExecutionStore(ctx context.Context, dbPath string) (*SQLiteExecutionStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	// Single connection avoids "database is locked" amplification under concurrent webhook load.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)

	store := &SQLiteExecutionStore{db: db}
	if err := store.init(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *SQLiteExecutionStore) init(ctx context.Context) error {
	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA synchronous=NORMAL;",
		"PRAGMA foreign_keys=ON;",
		"PRAGMA busy_timeout=5000;",
	}

	for _, stmt := range pragmas {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("apply pragma %q: %w", stmt, err)
		}
	}

	schema := `
CREATE TABLE IF NOT EXISTS executions (
	execution_id TEXT PRIMARY KEY,
	workflow_id TEXT NOT NULL,
	status TEXT NOT NULL,
	started_at TIMESTAMP NOT NULL,
	updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS execution_logs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	execution_id TEXT NOT NULL,
	step_id TEXT NOT NULL,
	status TEXT NOT NULL,
	message TEXT NOT NULL,
	created_at TIMESTAMP NOT NULL,
	FOREIGN KEY (execution_id) REFERENCES executions(execution_id)
);`

	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("initialize schema: %w", err)
	}

	return nil
}

// CreateExecution inserts a new execution row.
func (s *SQLiteExecutionStore) CreateExecution(ctx context.Context, executionID string, workflowID string) error {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO executions (execution_id, workflow_id, status, started_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)`,
		executionID,
		workflowID,
		"running",
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("insert execution %s: %w", executionID, err)
	}
	return nil
}

// UpdateStepStatus appends a step log and updates parent execution status timestamp.
func (s *SQLiteExecutionStore) UpdateStepStatus(
	ctx context.Context,
	executionID string,
	stepID string,
	status string,
	message string,
) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	now := time.Now().UTC()

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO execution_logs (execution_id, step_id, status, message, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		executionID,
		stepID,
		status,
		message,
		now,
	); err != nil {
		return fmt.Errorf("insert execution log (%s/%s): %w", executionID, stepID, err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`UPDATE executions
		 SET status = ?, updated_at = ?
		 WHERE execution_id = ?`,
		status,
		now,
		executionID,
	); err != nil {
		return fmt.Errorf("update execution status %s: %w", executionID, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit step status tx: %w", err)
	}
	return nil
}

// GetExecutionLogs returns logs ordered by insertion timestamp.
func (s *SQLiteExecutionStore) GetExecutionLogs(ctx context.Context, executionID string) ([]ExecutionLog, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT execution_id, step_id, status, message, created_at
		 FROM execution_logs
		 WHERE execution_id = ?
		 ORDER BY id ASC`,
		executionID,
	)
	if err != nil {
		return nil, fmt.Errorf("query execution logs %s: %w", executionID, err)
	}
	defer rows.Close()

	logs := make([]ExecutionLog, 0)
	for rows.Next() {
		var log ExecutionLog
		var createdAt time.Time
		if err := rows.Scan(&log.ExecutionID, &log.StepID, &log.Status, &log.Message, &createdAt); err != nil {
			return nil, fmt.Errorf("scan execution log row: %w", err)
		}
		log.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate execution logs: %w", err)
	}
	return logs, nil
}

// Close closes the underlying SQLite handle.
func (s *SQLiteExecutionStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}
