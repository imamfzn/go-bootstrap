package postgres_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/imamfzn/go-bootstrap/internal/platform/postgres"
)

func TestIsUniqueViolation(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"unique violation", &pgconn.PgError{Code: "23505"}, true},
		{"wrapped unique violation", fmt.Errorf("insert order: %w", &pgconn.PgError{Code: "23505"}), true},
		{"check violation", &pgconn.PgError{Code: "23514"}, false},
		{"other error", errors.New("connection reset"), false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := postgres.IsUniqueViolation(tt.err); got != tt.want {
				t.Fatalf("IsUniqueViolation(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestIsNoRows(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"no rows", pgx.ErrNoRows, true},
		{"wrapped no rows", fmt.Errorf("get order: %w", pgx.ErrNoRows), true},
		{"other error", errors.New("connection reset"), false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := postgres.IsNoRows(tt.err); got != tt.want {
				t.Fatalf("IsNoRows(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
