// Package i18n provides bilingual (zh-CN / en) message catalogs for
// user-facing content and slog log messages.
package i18n

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

const (
	// LocaleZhCN is the Chinese catalog key; Normalize maps any zh-* tag here
	// so content and slog catalogs load the 中文 bundle.
	LocaleZhCN = "zh-CN"
	// LocaleEn is the English catalog key; Normalize maps any en-* tag here.
	LocaleEn = "en"
	// DefaultLocale is used when the locale is missing or unrecognized so the
	// UI and slog catalogs always have a bundle.
	DefaultLocale = LocaleZhCN
)

// Bundle holds content and log message catalogs for one locale.
type Bundle struct {
	locale  string
	content map[string]string
	log     map[string]string
}

type ctxKey struct{}

var (
	globalMu     sync.RWMutex
	globalBundle *Bundle
)

func init() {
	b, err := NewBundle(DefaultLocale)
	if err != nil {
		panic("i18n: load default bundle: " + err.Error())
	}
	globalBundle = b
}

// NewBundle loads content and log catalogs for the given locale.
// Unknown locales fall back to DefaultLocale.
func NewBundle(locale string) (*Bundle, error) {
	loc := Normalize(locale)
	content, err := loadCatalog("content", loc)
	if err != nil {
		return nil, err
	}
	logCat, err := loadCatalog("log", loc)
	if err != nil {
		return nil, err
	}
	return &Bundle{locale: loc, content: content, log: logCat}, nil
}

// Normalize maps a locale tag to a supported value.
func Normalize(locale string) string {
	s := strings.TrimSpace(locale)
	if s == "" {
		return DefaultLocale
	}
	lower := strings.ToLower(strings.ReplaceAll(s, "_", "-"))
	switch {
	case lower == "en" || strings.HasPrefix(lower, "en-"):
		return LocaleEn
	case lower == "zh" || strings.HasPrefix(lower, "zh-"):
		return LocaleZhCN
	default:
		return DefaultLocale
	}
}

// ResolveLocale picks locale by priority: flag > env > settings > default.
func ResolveLocale(flagValue, settingsLocale string) string {
	if flagValue != "" {
		return Normalize(flagValue)
	}
	if env := os.Getenv("NEXA_LOCALE"); env != "" {
		return Normalize(env)
	}
	if settingsLocale != "" {
		return Normalize(settingsLocale)
	}
	return DefaultLocale
}

// ParseAcceptLanguage picks the best supported locale from an Accept-Language header.
func ParseAcceptLanguage(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.Split(header, ",")
	for _, part := range parts {
		tag := strings.TrimSpace(strings.Split(part, ";")[0])
		if tag == "" || tag == "*" {
			continue
		}
		return Normalize(tag)
	}
	return ""
}

// SetGlobal replaces the process-wide default bundle (used for slog and CLI).
func SetGlobal(b *Bundle) {
	if b == nil {
		return
	}
	globalMu.Lock()
	globalBundle = b
	globalMu.Unlock()
}

// Global returns the process-wide default bundle.
func Global() *Bundle {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalBundle
}

// Locale returns the bundle's locale tag.
func (b *Bundle) Locale() string {
	if b == nil {
		return DefaultLocale
	}
	return b.locale
}

// T translates a content catalog key with optional named args.
// Missing keys return the key itself.
func (b *Bundle) T(key string, args ...any) string {
	if b == nil {
		return formatMsg(key, nil, args...)
	}
	return formatMsg(key, b.content, args...)
}

// TLog translates a log catalog key with optional named args.
func (b *Bundle) TLog(key string, args ...any) string {
	if b == nil {
		return formatMsg(key, nil, args...)
	}
	return formatMsg(key, b.log, args...)
}

// T is a package-level helper using the global bundle.
func T(key string, args ...any) string {
	return Global().T(key, args...)
}

// TLog is a package-level helper using the global bundle.
func TLog(key string, args ...any) string {
	return Global().TLog(key, args...)
}

// WithLocale stores a request-scoped bundle in context.
func WithLocale(ctx context.Context, b *Bundle) context.Context {
	if b == nil {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, b)
}

// FromContext returns the request-scoped bundle, or Global() if absent.
func FromContext(ctx context.Context) *Bundle {
	if ctx == nil {
		return Global()
	}
	if b, ok := ctx.Value(ctxKey{}).(*Bundle); ok && b != nil {
		return b
	}
	return Global()
}

// TCtx translates a content key using the context bundle.
func TCtx(ctx context.Context, key string, args ...any) string {
	return FromContext(ctx).T(key, args...)
}

func loadCatalog(kind, locale string) (map[string]string, error) {
	path := fmt.Sprintf("locales/%s/%s.json", kind, locale)
	data, err := localeFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return m, nil
}

// formatMsg looks up key in catalog and interpolates {name} placeholders.
// args may be a single map[string]any, or alternating key/value pairs.
func formatMsg(key string, catalog map[string]string, args ...any) string {
	msg := key
	if catalog != nil {
		if v, ok := catalog[key]; ok {
			msg = v
		}
	}
	if len(args) == 0 {
		return msg
	}
	vars := argsToMap(args...)
	if len(vars) == 0 {
		return msg
	}
	for k, v := range vars {
		msg = strings.ReplaceAll(msg, "{"+k+"}", fmt.Sprint(v))
	}
	return msg
}

func argsToMap(args ...any) map[string]any {
	if len(args) == 1 {
		if m, ok := args[0].(map[string]any); ok {
			return m
		}
	}
	out := make(map[string]any, len(args)/2)
	for i := 0; i+1 < len(args); i += 2 {
		k, ok := args[i].(string)
		if !ok {
			continue
		}
		out[k] = args[i+1]
	}
	return out
}
