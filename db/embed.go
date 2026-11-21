// Package db embeds the SQL migrations into the binary, so the image needs
// no extra files and always carries the schema that matches its code.
package db

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed migrations/*.sql
var files embed.FS

// Migrations returns the migration files at the root of a filesystem.
func Migrations() (fs.FS, error) {
	sub, err := fs.Sub(files, "migrations")
	if err != nil {
		return nil, fmt.Errorf("open embedded migrations: %w", err)
	}
	return sub, nil
}
