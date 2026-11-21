// Package ident generates identifiers.
package ident

import "uuid"

// New returns a new UUID v7 as a string. Version 7 identifiers sort by
// creation time, which keeps primary-key indexes compact.
func New() string { return uuid.NewV7().String() }
