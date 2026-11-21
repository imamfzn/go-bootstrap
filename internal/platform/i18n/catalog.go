// Package i18n loads message catalogs and picks a locale for a request.
// A catalog file is "<locale>.json" holding a flat object of key to text.
package i18n

import (
	"cmp"
	"context"
	"encoding/json/v2"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"slices"
	"strconv"
	"strings"
)

// Catalog holds the messages of every loaded locale.
type Catalog struct {
	fallback string
	messages map[string]map[string]string
}

// Load reads every *.json file at the root of fsys. fallback is the locale
// used when a request asks for one that is not loaded; it must exist.
func Load(fsys fs.FS, fallback string) (*Catalog, error) {
	files, err := fs.Glob(fsys, "*.json")
	if err != nil {
		return nil, fmt.Errorf("list catalogs: %w", err)
	}

	c := &Catalog{fallback: fallback, messages: make(map[string]map[string]string, len(files))}
	for _, name := range files {
		data, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, fmt.Errorf("read catalog %s: %w", name, err)
		}
		var msgs map[string]string
		if err := json.Unmarshal(data, &msgs); err != nil {
			return nil, fmt.Errorf("parse catalog %s: %w", name, err)
		}
		c.messages[strings.ToLower(strings.TrimSuffix(name, path.Ext(name)))] = msgs
	}

	if _, ok := c.messages[fallback]; !ok {
		return nil, fmt.Errorf("fallback catalog %q not found", fallback)
	}
	return c, nil
}

// Locales returns the loaded locales in alphabetical order.
func (c *Catalog) Locales() []string {
	return slices.Sorted(maps.Keys(c.messages))
}

// Has reports whether the locale has a message for the key.
func (c *Catalog) Has(locale, key string) bool {
	_, ok := c.messages[locale][key]
	return ok
}

// Lookup returns the message for key in locale. It falls back to the
// fallback locale, then to def.
func (c *Catalog) Lookup(locale, key, def string) string {
	return cmp.Or(c.messages[locale][key], c.messages[c.fallback][key], def)
}

// Match picks the best loaded locale for an Accept-Language header value.
func (c *Catalog) Match(acceptLanguage string) string {
	type choice struct {
		tag string
		q   float64
	}

	var choices []choice
	for part := range strings.SplitSeq(acceptLanguage, ",") {
		tag, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" {
			continue
		}
		q := 1.0
		if v, ok := strings.CutPrefix(strings.TrimSpace(params), "q="); ok {
			parsed, err := strconv.ParseFloat(v, 64)
			if err != nil {
				continue
			}
			q = parsed
		}
		if q > 0 {
			choices = append(choices, choice{tag, q})
		}
	}
	slices.SortStableFunc(choices, func(a, b choice) int { return cmp.Compare(b.q, a.q) })

	for _, ch := range choices {
		if _, ok := c.messages[ch.tag]; ok {
			return ch.tag
		}
		if base, _, found := strings.Cut(ch.tag, "-"); found {
			if _, ok := c.messages[base]; ok {
				return base
			}
		}
	}
	return c.fallback
}

type localeKey struct{}

// WithLocale returns a context carrying the locale.
func WithLocale(ctx context.Context, locale string) context.Context {
	return context.WithValue(ctx, localeKey{}, locale)
}

// LocaleFrom returns the locale in the context, or an empty string.
func LocaleFrom(ctx context.Context) string {
	locale, _ := ctx.Value(localeKey{}).(string)
	return locale
}
