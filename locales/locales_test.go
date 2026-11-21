package locales_test

import (
	"testing"

	"github.com/imamfzn/go-bootstrap/internal/domain"
	"github.com/imamfzn/go-bootstrap/internal/platform/i18n"
	"github.com/imamfzn/go-bootstrap/locales"
)

// A domain error with no translation falls back to English silently, so
// this test is what keeps the catalogs complete.
func TestEveryErrorCodeIsTranslatedInEveryLocale(t *testing.T) {
	catalog, err := i18n.Load(locales.FS, "en")
	if err != nil {
		t.Fatalf("load catalogs: %v", err)
	}

	loaded := catalog.Locales()
	if len(loaded) < 2 {
		t.Fatalf("loaded locales = %v, want more than one", loaded)
	}

	for _, locale := range loaded {
		for _, e := range domain.Errors() {
			if !catalog.Has(locale, e.Code) {
				t.Errorf("locale %q has no message for %s", locale, e.Code)
			}
		}
	}
}

func TestEnglishCatalogMatchesDefaultMessages(t *testing.T) {
	catalog, err := i18n.Load(locales.FS, "en")
	if err != nil {
		t.Fatalf("load catalogs: %v", err)
	}

	for _, e := range domain.Errors() {
		if got := catalog.Lookup("en", e.Code, ""); got != e.Message {
			t.Errorf("en message for %s = %q, but the domain default is %q", e.Code, got, e.Message)
		}
	}
}
