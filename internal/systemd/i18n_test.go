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
		"Restart policy for the service.",
		"Restart=no|on-failure",
		"Restart=on-failure",
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("doc = %q, missing %q", doc, want)
		}
	}
	for _, unwanted := range []string{"**Description**", "**Syntax**", "**Example**", "**説明**", "**文法**", "**例**"} {
		if strings.Contains(doc, unwanted) {
			t.Fatalf("doc = %q, should not contain heading %q", doc, unwanted)
		}
	}
}

func TestSectionDocumentationIncludesSyntaxAndExample(t *testing.T) {
	doc := NewCatalog().SectionDocumentationFor("Service", LocaleJapanese)
	for _, want := range []string{
		"サービスプロセスの起動方法",
		"```ini\n[Service]\nType=simple\nExecStart=/usr/bin/example\n```",
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("doc = %q, missing %q", doc, want)
		}
	}
	if strings.Contains(doc, "[Service]\n[Service]") {
		t.Fatalf("doc = %q, contains a duplicate section header", doc)
	}
	for _, unwanted := range []string{"**説明**", "**文法**", "**例**", "**Description**", "**Syntax**", "**Example**"} {
		if strings.Contains(doc, unwanted) {
			t.Fatalf("doc = %q, should not contain heading %q", doc, unwanted)
		}
	}
}
