// Deprecated: This module is deprecated and will be removed in a future version.
// Please use the new module instead.
package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DBTX is an interface that allows us to use either a database connection or a transaction
type DBTX interface {
	Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
	Query(context.Context, string, ...interface{}) (pgx.Rows, error)
	QueryRow(context.Context, string, ...interface{}) pgx.Row
}

// BaseRepository provides common functionality for all repositories
type BaseRepository struct {
	db DBTX
}

// NewBaseRepository creates a new base repository
func NewBaseRepository(db DBTX) BaseRepository {
	return BaseRepository{
		db: db,
	}
}
