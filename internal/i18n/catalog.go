package i18n

import (
	"embed"
	"fmt"
	"strings"
	"sync"

	"go.yaml.in/yaml/v3"
)

//go:embed locales/*.yaml
var localeFS embed.FS

var (
	catalogsOnce sync.Once
	catalogs     map[string]map[string]string
	catalogsErr  error
)

func T(locale string, key string, args map[string]any) string {
	if key == "" {
		return ""
	}
	value, ok := lookup(locale, key)
	if !ok {
		return key
	}
	return interpolate(value, args)
}

func LoadCatalogs() error {
	catalogsOnce.Do(func() {
		catalogs = map[string]map[string]string{}
		for _, locale := range SupportedLocales() {
			path := fmt.Sprintf("locales/%s.yaml", locale)
			raw, err := localeFS.ReadFile(path)
			if err != nil {
				catalogsErr = err
				return
			}
			values := map[string]string{}
			if err := yaml.Unmarshal(raw, &values); err != nil {
				catalogsErr = err
				return
			}
			catalogs[locale] = values
		}
	})
	return catalogsErr
}

func lookup(locale string, key string) (string, bool) {
	_ = LoadCatalogs()
	resolved, ok := NormalizeLocale(locale)
	if !ok {
		resolved = DefaultLocale
	}
	if messages, ok := catalogs[resolved]; ok {
		if value, ok := messages[key]; ok {
			return value, true
		}
	}
	if messages, ok := catalogs[DefaultLocale]; ok {
		value, ok := messages[key]
		return value, ok
	}
	return "", false
}

func interpolate(message string, args map[string]any) string {
	if len(args) == 0 {
		return message
	}
	replacements := make([]string, 0, len(args)*2)
	for key, value := range args {
		replacements = append(replacements, "{{"+key+"}}", fmt.Sprint(value))
	}
	return strings.NewReplacer(replacements...).Replace(message)
}
