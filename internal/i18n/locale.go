package i18n

import (
	"sort"
	"strconv"
	"strings"
)

const (
	HeaderLanguage       = "X-Language"
	HeaderAcceptLanguage = "Accept-Language"
	HeaderContentLocale  = "Content-Language"
	HeaderVary           = "Vary"
	ContextLocaleKey     = "locale"

	DefaultLocale = "zh-CN"
	LocaleZhCN    = "zh-CN"
	LocaleEnUS    = "en-US"
)

var supportedLocales = map[string]string{
	"zh":         LocaleZhCN,
	"zh-cn":      LocaleZhCN,
	"zh-hans":    LocaleZhCN,
	"zh-hans-cn": LocaleZhCN,
	"en":         LocaleEnUS,
	"en-us":      LocaleEnUS,
}

func NormalizeLocale(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", false
	}
	value = strings.ToLower(strings.ReplaceAll(value, "_", "-"))
	if idx := strings.Index(value, ";"); idx >= 0 {
		value = strings.TrimSpace(value[:idx])
	}
	if locale, ok := supportedLocales[value]; ok {
		return locale, true
	}
	if strings.HasPrefix(value, "zh-") {
		return LocaleZhCN, true
	}
	if strings.HasPrefix(value, "en-") {
		return LocaleEnUS, true
	}
	return "", false
}

func ResolveLocale(languageHeader, acceptLanguageHeader string) string {
	if locale, ok := NormalizeLocale(languageHeader); ok {
		return locale
	}
	if locale, ok := matchAcceptLanguage(acceptLanguageHeader); ok {
		return locale
	}
	return DefaultLocale
}

func ResolveWebSocketLocale(langQuery, languageHeader, acceptLanguageHeader string) string {
	if locale, ok := NormalizeLocale(langQuery); ok {
		return locale
	}
	return ResolveLocale(languageHeader, acceptLanguageHeader)
}

func SupportedLocales() []string {
	return []string{LocaleZhCN, LocaleEnUS}
}

type languageCandidate struct {
	value string
	q     float64
	index int
}

func matchAcceptLanguage(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	parts := strings.Split(raw, ",")
	candidates := make([]languageCandidate, 0, len(parts))
	for i, part := range parts {
		value, q := parseAcceptLanguagePart(part)
		if value == "" || q <= 0 {
			continue
		}
		candidates = append(candidates, languageCandidate{value: value, q: q, index: i})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].q == candidates[j].q {
			return candidates[i].index < candidates[j].index
		}
		return candidates[i].q > candidates[j].q
	})
	for _, candidate := range candidates {
		if locale, ok := NormalizeLocale(candidate.value); ok {
			return locale, true
		}
	}
	return "", false
}

func parseAcceptLanguagePart(raw string) (string, float64) {
	part := strings.TrimSpace(raw)
	if part == "" {
		return "", 0
	}
	items := strings.Split(part, ";")
	value := strings.TrimSpace(items[0])
	q := 1.0
	for _, item := range items[1:] {
		item = strings.TrimSpace(item)
		if !strings.HasPrefix(item, "q=") {
			continue
		}
		parsed, err := strconv.ParseFloat(strings.TrimPrefix(item, "q="), 64)
		if err == nil {
			q = parsed
		}
	}
	return value, q
}
