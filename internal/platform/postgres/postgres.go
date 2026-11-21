// Package postgres owns the connection pool, transactions and migrations.
// Repositories reach the database only through DB.
package postgres

import (
	"context"
	"fmt"
	"net"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Config describes how to reach the database.
type Config struct {
	// URL is a postgres:// connection string.
	URL string
	// InstanceConnectionName, when set, dials through the Cloud SQL
	// connector ("project:region:instance"). The host in URL is then ignored.
	InstanceConnectionName string
	// IAMAuth uses the service account as the database user, with no password.
	IAMAuth bool
	// MaxConns caps the pool. Keep max instances x MaxConns below the
	// database's connection limit.
	MaxConns int32
}

// Querier is what repositories run SQL on: the pool, or the transaction in
// progress.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// DB is the database handle shared by all repositories.
type DB struct {
	pool   *pgxpool.Pool
	dialer *cloudsqlconn.Dialer
}

// Open creates the pool and checks that the database answers.
func Open(ctx context.Context, cfg *Config) (*DB, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	if cfg.MaxConns > 0 {
		poolCfg.MaxConns = cfg.MaxConns
	}

	db := &DB{}
	if cfg.InstanceConnectionName != "" {
		var opts []cloudsqlconn.Option
		if cfg.IAMAuth {
			opts = append(opts, cloudsqlconn.WithIAMAuthN())
		}
		db.dialer, err = cloudsqlconn.NewDialer(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("create cloud sql dialer: %w", err)
		}
		poolCfg.ConnConfig.DialFunc = func(ctx context.Context, _, _ string) (net.Conn, error) {
			return db.dialer.Dial(ctx, cfg.InstanceConnectionName)
		}
	}

	db.pool, err = pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := db.pool.Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return db, nil
}

// Ping checks that the database answers.
func (db *DB) Ping(ctx context.Context) error {
	if err := db.pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	return nil
}

// Close releases every connection.
func (db *DB) Close() {
	if db.pool != nil {
		db.pool.Close()
	}
	if db.dialer != nil {
		_ = db.dialer.Close()
	}
}

type txKey struct{}

// InTx runs fn in a transaction. The transaction travels in the context, so
// every repository call made with that context joins it. Returning an error
// rolls back. A nested call joins the transaction already in progress.
func (db *DB) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}
	return pgx.BeginFunc(ctx, db.pool, func(tx pgx.Tx) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}

// Conn returns the transaction carried by ctx, or the pool when there is
// none.
func (db *DB) Conn(ctx context.Context) Querier {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return db.pool
}
