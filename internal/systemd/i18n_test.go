package systemd

import "testing"

func TestNormalizeLocale(t *testing.T) {
	if got := NormalizeLocale("ja-JP"); got != LocaleJapanese {
		t.Fatalf("NormalizeLocale(ja-JP) = %q, want %q", got, LocaleJapanese)
	}
	if got := NormalizeLocale("en-US"); got != LocaleEnglish {
		t.Fatalf("NormalizeLocale(en-US) = %q, want %q", got, LocaleEnglish)
	}
}

func TestJapaneseDirectiveDocFallback(t *testing.T) {
	doc := DirectiveDocFor("Service", Directive{Name: "UnknownNewDirective", Doc: "English documentation."}, LocaleJapanese)
	want := "systemd の [Service] セクションで使用する UnknownNewDirective ディレクティブです。"
	if doc != want {
		t.Fatalf("doc = %q, want %q", doc, want)
	}
}

func TestEnglishDirectiveDocUsesCatalogText(t *testing.T) {
	doc := DirectiveDocFor("Service", Directive{Name: "ExecStart", Doc: "Command lines executed to start the service."}, LocaleEnglish)
	if doc != "Command lines executed to start the service." {
		t.Fatalf("doc = %q, want English catalog text", doc)
	}
}
