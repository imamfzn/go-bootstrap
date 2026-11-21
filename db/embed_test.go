package db_test

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"github.com/imamfzn/go-bootstrap/db"
)

var migrationName = regexp.MustCompile(`^\d{14}_[a-z0-9_]+\.sql$`)

func TestMigrations_AreEmbeddedAndWellFormed(t *testing.T) {
	fsys, err := db.Migrations()
	if err != nil {
		t.Fatalf("Migrations: %v", err)
	}

	names, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		t.Fatalf("list migrations: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("no migrations are embedded")
	}

	for _, name := range names {
		if !migrationName.MatchString(name) {
			t.Errorf("%s: name must be <YYYYMMDDHHMMSS>_<snake_case>.sql", name)
		}

		data, err := fs.ReadFile(fsys, name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		for _, marker := range []string{"-- +goose Up", "-- +goose Down"} {
			if !strings.Contains(string(data), marker) {
				t.Errorf("%s: missing %q section", name, marker)
			}
		}
	}
}
