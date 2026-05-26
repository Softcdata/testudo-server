package i18n

import "testing"

func TestNormalizeLocale(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
		ok   bool
	}{
		{name: "zh cn", raw: "zh-CN", want: LocaleZhCN, ok: true},
		{name: "zh underscore", raw: "zh_CN", want: LocaleZhCN, ok: true},
		{name: "zh prefix", raw: "zh-Hans-CN", want: LocaleZhCN, ok: true},
		{name: "en", raw: "en", want: LocaleEnUS, ok: true},
		{name: "en prefix", raw: "en-GB", want: LocaleEnUS, ok: true},
		{name: "invalid", raw: "fr-FR", want: "", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := NormalizeLocale(tt.raw)
			if ok != tt.ok {
				t.Fatalf("NormalizeLocale(%q) ok=%v, want %v", tt.raw, ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("NormalizeLocale(%q)=%q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestResolveLocale(t *testing.T) {
	tests := []struct {
		name     string
		language string
		accept   string
		want     string
	}{
		{
			name:     "x language wins",
			language: "en-US",
			accept:   "zh-CN",
			want:     LocaleEnUS,
		},
		{
			name:   "accept language quality",
			accept: "fr-FR, en-US;q=0.7, zh-CN;q=0.3",
			want:   LocaleEnUS,
		},
		{
			name:     "invalid x language falls back to accept language",
			language: "fr-FR",
			accept:   "en-US",
			want:     LocaleEnUS,
		},
		{
			name:     "invalid headers use default",
			language: "fr-FR",
			accept:   "de-DE",
			want:     DefaultLocale,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveLocale(tt.language, tt.accept)
			if got != tt.want {
				t.Fatalf("ResolveLocale(%q,%q)=%q, want %q", tt.language, tt.accept, got, tt.want)
			}
		})
	}
}

func TestResolveWebSocketLocale(t *testing.T) {
	got := ResolveWebSocketLocale("en-US", "zh-CN", "")
	if got != LocaleEnUS {
		t.Fatalf("ResolveWebSocketLocale()=%q, want %q", got, LocaleEnUS)
	}

	got = ResolveWebSocketLocale("", "", "en-US")
	if got != LocaleEnUS {
		t.Fatalf("ResolveWebSocketLocale()=%q, want %q", got, LocaleEnUS)
	}
}
