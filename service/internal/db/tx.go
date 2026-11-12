package db

import (
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
)

// Deferred after Beginx: once Commit has run, Rollback reports ErrTxDone,
// which is expected and not worth logging.
func (db *DB) rollbackUnlessCommitted(tx *sqlx.Tx) {
	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		db.Logger().Warnf("Transaction rollback failed: %v", err)
	}
}
