package i18n

import "testing"

func TestTranslateWithArgs(t *testing.T) {
	got := T(LocaleEnUS, KeyValidationUnsupportedAction, map[string]any{"type": "pause"})
	want := "unsupported action type: pause"
	if got != want {
		t.Fatalf("T()=%q, want %q", got, want)
	}
}

func TestTranslateFallback(t *testing.T) {
	got := T("fr-FR", KeyValidationNameRequired, nil)
	want := "名称不能为空"
	if got != want {
		t.Fatalf("T()=%q, want %q", got, want)
	}

	got = T(LocaleEnUS, "missing.key", nil)
	if got != "missing.key" {
		t.Fatalf("missing key T()=%q, want key", got)
	}
}
