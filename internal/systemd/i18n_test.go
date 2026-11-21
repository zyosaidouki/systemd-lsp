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
	// A translation wins, then the untranslated description, then a generic sentence.
	for _, tc := range []struct {
		directive Directive
		want      string
	}{
		{Directive{Name: "UnknownNewDirective", Doc: "English documentation.", DocJapanese: "日本語の説明。"}, "日本語の説明。"},
		{Directive{Name: "UnknownNewDirective", Doc: "English documentation."}, "English documentation."},
		{Directive{Name: "UnknownNewDirective"}, "systemd の [Service] セクションで使用する UnknownNewDirective ディレクティブです。"},
	} {
		if doc := DirectiveDocFor("Service", tc.directive, LocaleJapanese); doc != tc.want {
			t.Errorf("doc = %q, want %q", doc, tc.want)
		}
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

func TestDocsDistinguishSettingsThatShareAName(t *testing.T) {
	c := NewCatalog()
	for _, tc := range []struct{ section, name, locale, want string }{
		{"Service", "Type", LocaleJapanese, "サービスの起動方式を指定します。"},
		{"Mount", "Type", LocaleJapanese, "マウントするファイルシステムの種類を指定します。"},
		{"Mount", "Type", LocaleEnglish, "File system type."},
		{"Swap", "Priority", LocaleJapanese, "swap の優先度を指定します。"},
		{"Socket", "Priority", LocaleJapanese, "このソケットから送信するトラフィックの優先度（SO_PRIORITY）を指定します。"},
		{"Socket", "Priority", LocaleEnglish, "Priority of the traffic sent from this socket (SO_PRIORITY)."},
		{"Service", "ExecStartPre", LocaleJapanese, "ExecStart= の前に実行するコマンドを指定します。"},
		{"Socket", "ExecStartPre", LocaleJapanese, "待ち受けソケットを作成してバインドする前に実行するコマンドを指定します。"},
		{"Socket", "ExecStopPost", LocaleEnglish, "Command lines executed after the listening sockets are closed and removed."},
		{"Service", "TimeoutStopSec", LocaleJapanese, "サービス停止に許可する最大時間を指定します。"},
		{"Scope", "TimeoutStopSec", LocaleJapanese, "scope の停止に許可する最大時間を指定します。"},
	} {
		directive, ok := c.Directive(tc.section, tc.name)
		if !ok {
			t.Fatalf("[%s] %s is not in the catalog", tc.section, tc.name)
		}
		if doc := DirectiveDocFor(tc.section, directive, tc.locale); doc != tc.want {
			t.Errorf("[%s] %s (%s) = %q, want %q", tc.section, tc.name, tc.locale, doc, tc.want)
		}
	}
	for _, tc := range []struct{ section, name, want string }{
		{"Service", "Type", "Type=simple"},
		{"Mount", "Type", "Type=ext4"},
		{"Swap", "Priority", "Priority=10"},
		{"Socket", "Priority", "Priority=6"},
		{"Mount", "Options", "Options=defaults,noatime"},
		{"Swap", "Options", "Options=discard"},
	} {
		directive, _ := c.Directive(tc.section, tc.name)
		if doc := DirectiveDocumentationFor(tc.section, directive, LocaleEnglish); !strings.Contains(doc, "\n"+tc.want+"\n") {
			t.Errorf("[%s] %s example missing %q in %q", tc.section, tc.name, tc.want, doc)
		}
	}
}
