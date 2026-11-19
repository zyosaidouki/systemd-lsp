package lsp

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zyosaidouki/systemd-lsp/internal/config"
	"github.com/zyosaidouki/systemd-lsp/internal/systemd"
)

func initialize(t *testing.T, server *Server, params string) {
	t.Helper()
	if len(server.Handle([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":`+params+`}`))) != 1 {
		t.Fatal("initialize returned no response")
	}
}

// didOpen returns the method and parameters of the one message the server
// sends in reply.
func didOpen(t *testing.T, server *Server, uri, text string, params any) string {
	t.Helper()
	payload, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{
		"textDocument": TextDocumentItem{URI: uri, Text: text},
	}})
	if err != nil {
		t.Fatal(err)
	}
	responses := server.Handle(payload)
	if len(responses) != 1 {
		t.Fatalf("didOpen returned %d messages, want 1", len(responses))
	}
	var msg rpcMessage
	if err := json.Unmarshal(responses[0], &msg); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(msg.Params, params); err != nil {
		t.Fatal(err)
	}
	return msg.Method
}

func TestSettingsLocaleIsFallback(t *testing.T) {
	for _, tt := range []struct {
		name, params, want string
	}{
		{"no client option", `{}`, systemd.LocaleJapanese},
		{"client option wins", `{"initializationOptions":{"locale":"en"}}`, systemd.LocaleEnglish},
		{"over editor UI locale", `{"locale":"en-US"}`, systemd.LocaleJapanese},
	} {
		server := NewServer(systemd.NewCatalog(), nil)
		server.UseSettings(config.Settings{Locale: "ja"})
		initialize(t, server, tt.params)
		if server.locale != tt.want {
			t.Errorf("%s: locale = %q, want %q", tt.name, server.locale, tt.want)
		}
	}

	server := NewServer(systemd.NewCatalog(), nil)
	initialize(t, server, `{"locale":"ja"}`)
	if server.locale != systemd.LocaleJapanese {
		t.Errorf("without settings: locale = %q, want the client's", server.locale)
	}
}

func TestSettingsCatalogPathIsFallback(t *testing.T) {
	dir := t.TempDir()
	write := func(name, directive string) string {
		path := filepath.Join(dir, name)
		content := `{"directives":[{"section":"Service","name":"` + directive + `","valueKind":"string"}]}`
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	fromSettings, fromClient := write("settings.json", "FromSettings"), write("client.json", "FromClient")

	server := NewServer(systemd.NewCatalog(), nil)
	server.UseSettings(config.Settings{CatalogPath: fromSettings})
	initialize(t, server, `{}`)
	if _, ok := server.catalog.Directive("Service", "FromSettings"); !ok {
		t.Error("catalog from settings was not loaded")
	}

	server = NewServer(systemd.NewCatalog(), nil)
	server.UseSettings(config.Settings{CatalogPath: fromSettings})
	initialize(t, server, `{"initializationOptions":{"catalogPath":`+strconvQuote(fromClient)+`}}`)
	if _, ok := server.catalog.Directive("Service", "FromClient"); !ok {
		t.Error("catalog from the client was not loaded")
	}
	if _, ok := server.catalog.Directive("Service", "FromSettings"); ok {
		t.Error("catalog from settings was loaded although the client named one")
	}
}

func TestIsUserUnit(t *testing.T) {
	server := NewServer(systemd.NewCatalog(), nil)
	for uri, want := range map[string]bool{
		"file:///xdg/config/systemd/user/demo.service":       false,
		"file:///home/u/.config/systemd/user/a.service":      false,
		"file:///xdg/config/systemd/user/a.service.d/x.conf": false,
	} {
		if got := server.isUserUnit(uri); got != want {
			t.Errorf("without settings: isUserUnit(%q) = %v, want %v", uri, got, want)
		}
	}
	server.UseSettings(config.Settings{UserUnitDir: "/xdg/my config/systemd/user"})
	for uri, want := range map[string]bool{
		"file:///xdg/my%20config/systemd/user/demo.service":                     true,
		"file:///xdg/my%20config/systemd/user/demo.service.d/override.conf":     true,
		"file:///xdg/my%20config/systemd/user/.%23demo.service0123456789abcdef": true,
		"file:///xdg/my%20config/systemd/user":                                  false,
		"file:///xdg/my%20config/systemd/user.control/demo.service":             false,
		"file:///xdg/my%20config/systemd/system/demo.service":                   false,
		"file:///home/u/.config/systemd/user/demo.service":                      false,
		"file:///etc/systemd/system/demo.service":                               false,
		"untitled:/xdg/my%20config/systemd/user/demo.service":                   false,
	} {
		if got := server.isUserUnit(uri); got != want {
			t.Errorf("isUserUnit(%q) = %v, want %v", uri, got, want)
		}
	}
}

func TestUserUnitGetsUserTemplate(t *testing.T) {
	server := NewServer(systemd.NewCatalog(), nil)
	server.UseSettings(config.Settings{UserUnitDir: "/xdg/config/systemd/user"})
	for uri, wantedBy := range map[string]string{
		"file:///xdg/config/systemd/user/demo.service": "WantedBy=default.target\n",
		"file:///etc/systemd/system/demo.service":      "WantedBy=multi-user.target\n",
	} {
		var params struct {
			Edit struct {
				Changes map[string][]struct {
					NewText string `json:"newText"`
				} `json:"changes"`
			} `json:"edit"`
		}
		if method := didOpen(t, server, uri, "", &params); method != "workspace/applyEdit" {
			t.Fatalf("%s: method = %q, want workspace/applyEdit", uri, method)
		}
		edits := params.Edit.Changes[uri]
		if len(edits) != 1 || !strings.HasSuffix(edits[0].NewText, wantedBy) {
			t.Errorf("%s: edits = %#v, want a template ending in %q", uri, edits, wantedBy)
		}
	}
}

func TestUserUnitWarnsAboutSystemTargets(t *testing.T) {
	server := NewServer(systemd.NewCatalog(), nil)
	server.UseSettings(config.Settings{UserUnitDir: "/xdg/config/systemd/user"})
	text := "[Unit]\nDescription=demo\n\n[Service]\nExecStart=/bin/true\n\n[Install]\nWantedBy=multi-user.target\n"
	for uri, want := range map[string]int{
		"file:///xdg/config/systemd/user/demo.service":                 1,
		"file:///xdg/config/systemd/user/demo.service.d/override.conf": 1,
		"file:///etc/systemd/system/demo.service":                      0,
	} {
		var params struct {
			Diagnostics []Diagnostic `json:"diagnostics"`
		}
		if method := didOpen(t, server, uri, text, &params); method != "textDocument/publishDiagnostics" {
			t.Fatalf("%s: method = %q, want publishDiagnostics", uri, method)
		}
		if len(params.Diagnostics) != want {
			t.Errorf("%s: diagnostics = %#v, want %d", uri, params.Diagnostics, want)
			continue
		}
		if want == 1 && (params.Diagnostics[0].Range.Start.Line != 7 || !strings.Contains(params.Diagnostics[0].Message, "use default.target instead")) {
			t.Errorf("%s: diagnostic = %#v, want a default.target warning on line 7", uri, params.Diagnostics[0])
		}
	}
}
