package database

import (
	"context"
	"database/sql"
)

// Executor can perform SQL queries.
type Executor interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// ContextExecutor can perform SQL queries with context.
type ContextExecutor interface {
	Executor

	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Transactor can commit and rollback, on top of being able to execute queries.
type Transactor interface {
	Commit() error
	Rollback() error

	Executor
}

type DatabaseCloser interface {
	Close() error
}
