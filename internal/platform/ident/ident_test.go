package ident_test

import (
	"testing"
	"uuid"

	"github.com/imamfzn/go-bootstrap/internal/platform/ident"
)

func TestNew(t *testing.T) {
	a, b := ident.New(), ident.New()

	if a == b {
		t.Fatalf("New returned the same identifier twice: %s", a)
	}
	if _, err := uuid.Parse(a); err != nil {
		t.Fatalf("New returned %q, which is not a UUID: %v", a, err)
	}
	if a > b {
		t.Errorf("identifiers are not time-ordered: %s came before %s", a, b)
	}
}
