package systemd

import (
	"strings"
	"testing"
)

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

func TestDirectiveDocumentationIncludesSyntaxAndExample(t *testing.T) {
	doc := DirectiveDocumentationFor("Service", Directive{Name: "Restart", Doc: "Restart policy for the service.", Values: []string{"no", "on-failure"}}, LocaleEnglish)
	for _, want := range []string{
		"**Description**",
		"Restart policy for the service.",
		"**Syntax**",
		"Restart=no|on-failure",
		"**Example**",
		"Restart=on-failure",
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("doc = %q, missing %q", doc, want)
		}
	}
}

func TestSectionDocumentationIncludesSyntaxAndExample(t *testing.T) {
	doc := NewCatalog().SectionDocumentationFor("Service", LocaleJapanese)
	for _, want := range []string{
		"**説明**",
		"サービスプロセスの起動方法",
		"**文法**",
		"[Service]",
		"**例**",
		"ExecStart=/usr/bin/example",
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("doc = %q, missing %q", doc, want)
		}
	}
}
