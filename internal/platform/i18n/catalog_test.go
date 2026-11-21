package i18n_test

import (
	"slices"
	"testing"
	"testing/fstest"

	"github.com/imamfzn/go-bootstrap/internal/platform/i18n"
)

func newCatalog(t *testing.T) *i18n.Catalog {
	t.Helper()

	fsys := fstest.MapFS{
		"en.json": {Data: []byte(`{"GREETING": "Hello", "ONLY_EN": "English only"}`)},
		"id.json": {Data: []byte(`{"GREETING": "Halo"}`)},
	}
	c, err := i18n.Load(fsys, "en")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return c
}

func TestLoad_MissingFallback(t *testing.T) {
	fsys := fstest.MapFS{"id.json": {Data: []byte(`{}`)}}

	if _, err := i18n.Load(fsys, "en"); err == nil {
		t.Fatal("Load succeeded without the fallback catalog")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	fsys := fstest.MapFS{"en.json": {Data: []byte(`{not json`)}}

	if _, err := i18n.Load(fsys, "en"); err == nil {
		t.Fatal("Load succeeded with a malformed catalog")
	}
}

func TestCatalog_Locales(t *testing.T) {
	got := newCatalog(t).Locales()

	if want := []string{"en", "id"}; !slices.Equal(got, want) {
		t.Fatalf("Locales() = %v, want %v", got, want)
	}
}

func TestCatalog_Lookup(t *testing.T) {
	c := newCatalog(t)

	tests := []struct {
		name, locale, key, def, want string
	}{
		{"translated", "id", "GREETING", "default", "Halo"},
		{"falls back to fallback locale", "id", "ONLY_EN", "default", "English only"},
		{"unknown locale uses fallback", "fr", "GREETING", "default", "Hello"},
		{"unknown key uses default", "id", "MISSING", "default", "default"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := c.Lookup(tt.locale, tt.key, tt.def); got != tt.want {
				t.Fatalf("Lookup(%q, %q) = %q, want %q", tt.locale, tt.key, got, tt.want)
			}
		})
	}
}

func TestCatalog_Match(t *testing.T) {
	c := newCatalog(t)

	tests := []struct {
		header, want string
	}{
		{"", "en"},
		{"id", "id"},
		{"id-ID,id;q=0.9,en;q=0.8", "id"},
		{"fr-FR,fr;q=0.9", "en"},
		{"fr;q=0.9,id;q=0.5", "id"},
		{"en;q=0.2,id;q=0.8", "id"},
		{"id;q=0,en", "en"},
		{"ID", "id"},
		{"*", "en"},
	}
	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			if got := c.Match(tt.header); got != tt.want {
				t.Fatalf("Match(%q) = %q, want %q", tt.header, got, tt.want)
			}
		})
	}
}

func TestLocaleContext(t *testing.T) {
	ctx := i18n.WithLocale(t.Context(), "id")

	if got := i18n.LocaleFrom(ctx); got != "id" {
		t.Fatalf("LocaleFrom = %q, want id", got)
	}
	if got := i18n.LocaleFrom(t.Context()); got != "" {
		t.Fatalf("LocaleFrom on a bare context = %q, want empty", got)
	}
}
