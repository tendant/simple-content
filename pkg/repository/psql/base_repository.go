// Deprecated: This package is deprecated as of 2025-10-01 and will be removed in 3 months.
// Please migrate to github.com/tendant/simple-content/pkg/simplecontent/repo/postgres which provides:
//   - Unified Repository interface
//   - Better error handling
//   - Status management operations
//   - Soft delete support
//   - Dedicated 'content' schema support
// See MIGRATION_FROM_LEGACY.md for migration guide.
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
