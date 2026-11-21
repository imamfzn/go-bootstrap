package repository_test

import (
	"context"
	"fmt"
	"reflect"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/imamfzn/go-bootstrap/internal/platform/postgres"
)

// fakeDB stands in for the database handle. It records the statement a
// repository ran and answers with whatever the test prepared, so repository
// tests need no PostgreSQL.
type fakeDB struct {
	// Prepared answers.
	execTag  pgconn.CommandTag
	execErr  error
	rows     *fakeRows
	queryErr error

	// What the repository ran.
	sql  string
	args []any
}

func (db *fakeDB) Conn(context.Context) postgres.Querier { return db }

func (db *fakeDB) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	db.sql, db.args = sql, args
	return db.execTag, db.execErr
}

func (db *fakeDB) Query(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
	db.sql, db.args = sql, args
	if db.queryErr != nil {
		return nil, db.queryErr
	}
	return db.rows, nil
}

func (db *fakeDB) QueryRow(context.Context, string, ...any) pgx.Row {
	panic("QueryRow is not used by the repositories")
}

// namedArgs returns the named arguments of the recorded statement.
func (db *fakeDB) namedArgs() pgx.NamedArgs {
	if len(db.args) != 1 {
		return nil
	}
	named, _ := db.args[0].(pgx.NamedArgs)
	return named
}

// fakeRows is an in-memory pgx.Rows: column names and the rows under them.
type fakeRows struct {
	pgx.Rows

	columns []string
	data    [][]any
	err     error

	next int
}

func newFakeRows(columns []string, data ...[]any) *fakeRows {
	return &fakeRows{columns: columns, data: data}
}

func (r *fakeRows) Close()     {}
func (r *fakeRows) Err() error { return r.err }

func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription {
	fields := make([]pgconn.FieldDescription, len(r.columns))
	for i, name := range r.columns {
		fields[i] = pgconn.FieldDescription{Name: name}
	}
	return fields
}

func (r *fakeRows) Next() bool {
	if r.err != nil || r.next >= len(r.data) {
		return false
	}
	r.next++
	return true
}

func (r *fakeRows) Scan(dest ...any) error {
	row := r.data[r.next-1]
	if len(dest) != len(row) {
		return fmt.Errorf("scan: %d destinations for %d columns", len(dest), len(row))
	}
	for i, value := range row {
		target := reflect.ValueOf(dest[i]).Elem()
		v := reflect.ValueOf(value)
		if !v.Type().AssignableTo(target.Type()) {
			return fmt.Errorf("scan column %s: cannot store %T in %s", r.columns[i], value, target.Type())
		}
		target.Set(v)
	}
	return nil
}
