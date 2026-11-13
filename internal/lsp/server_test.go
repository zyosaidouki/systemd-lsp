package lsp

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zyosaidouki/systemd-lsp/internal/systemd"
	"github.com/zyosaidouki/systemd-lsp/internal/version"
)

func TestInitialize(t *testing.T) {
	server := NewServer(systemd.NewCatalog(), nil)
	payload := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	responses := server.Handle(payload)
	if len(responses) != 1 {
		t.Fatal("Handle returned no response")
	}
	var msg rpcMessage
	if err := json.Unmarshal(responses[0], &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Error != nil {
		t.Fatalf("initialize error = %#v", msg.Error)
	}
	var result struct {
		Capabilities map[string]any `json:"capabilities"`
	}
	if err := json.Unmarshal(msg.Result, &result); err != nil {
		t.Fatal(err)
	}
	if result.Capabilities["hoverProvider"] != true {
		t.Fatalf("hoverProvider = %#v, want true", result.Capabilities["hoverProvider"])
	}
}

func TestDidOpenPublishesDiagnostics(t *testing.T) {
	server := NewServer(systemd.NewCatalog(), nil)
	payload := []byte(`{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///tmp/demo.service","languageId":"systemd","version":1,"text":"Description=outside\n"}}}`)
	responses := server.Handle(payload)
	if len(responses) != 1 {
		t.Fatal("Handle returned no notification")
	}
	var msg rpcMessage
	if err := json.Unmarshal(responses[0], &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Method != "textDocument/publishDiagnostics" {
		t.Fatalf("method = %q, want publishDiagnostics", msg.Method)
	}
	var params struct {
		Diagnostics []Diagnostic `json:"diagnostics"`
	}
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		t.Fatal(err)
	}
	if len(params.Diagnostics) != 1 {
		t.Fatalf("diagnostic count = %d, want 1", len(params.Diagnostics))
	}
}

func TestCompletionReturnsServiceDirectives(t *testing.T) {
	server := NewServer(systemd.NewCatalog(), nil)
	initializeWithSnippetSupport(t, server)
	open := []byte(`{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///tmp/demo.service","languageId":"systemd","version":1,"text":"[Service]\n"}}}`)
	if len(server.Handle(open)) != 1 {
		t.Fatal("didOpen did not publish diagnostics")
	}
	payload := []byte(`{"jsonrpc":"2.0","id":2,"method":"textDocument/completion","params":{"textDocument":{"uri":"file:///tmp/demo.service"},"position":{"line":1,"character":0}}}`)
	responses := server.Handle(payload)
	if len(responses) != 1 {
		t.Fatal("Handle returned no response")
	}
	var msg rpcMessage
	if err := json.Unmarshal(responses[0], &msg); err != nil {
		t.Fatal(err)
	}
	var items []CompletionItem
	if err := json.Unmarshal(msg.Result, &items); err != nil {
		t.Fatal(err)
	}
	if !hasCompletion(items, "ExecStart") {
		t.Fatalf("completion labels = %#v, missing ExecStart", items)
	}
	if !hasCompletion(items, "Delegate") {
		t.Fatalf("completion labels = %#v, missing Delegate", items)
	}
	if !hasCompletion(items, "CPUAffinity") {
		t.Fatalf("completion labels = %#v, missing CPUAffinity", items)
	}
	if !hasCompletion(items, "RestartMode") {
		t.Fatalf("completion labels = %#v, missing generated RestartMode", items)
	}
	item, ok := completionByLabel(items, "ExecStart")
	if !ok {
		t.Fatal("missing ExecStart completion")
	}
	if item.InsertText != "ExecStart=$0" {
		t.Fatalf("ExecStart insertText = %q, want snippet cursor after =", item.InsertText)
	}
	if item.InsertTextFormat != InsertTextFormatSnippet {
		t.Fatalf("ExecStart insertTextFormat = %d, want snippet", item.InsertTextFormat)
	}
}

func TestCompletionReturnsSectionSnippets(t *testing.T) {
	server := NewServer(systemd.NewCatalog(), nil)
	initializeWithSnippetSupport(t, server)
	open := []byte(`{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///tmp/demo.service","languageId":"systemd","version":1,"text":"["}}}`)
	if len(server.Handle(open)) != 1 {
		t.Fatal("didOpen did not publish diagnostics")
	}
	payload := []byte(`{"jsonrpc":"2.0","id":2,"method":"textDocument/completion","params":{"textDocument":{"uri":"file:///tmp/demo.service"},"position":{"line":0,"character":1}}}`)
	responses := server.Handle(payload)
	if len(responses) != 1 {
		t.Fatal("Handle returned no response")
	}
	var msg rpcMessage
	if err := json.Unmarshal(responses[0], &msg); err != nil {
		t.Fatal(err)
	}
	var items []CompletionItem
	if err := json.Unmarshal(msg.Result, &items); err != nil {
		t.Fatal(err)
	}
	item, ok := completionByLabel(items, "[Service]")
	if !ok {
		t.Fatalf("completion labels = %#v, missing [Service]", items)
	}
	if item.InsertText != "[Service]$0" {
		t.Fatalf("[Service] insertText = %q, want cursor after section", item.InsertText)
	}
	if item.InsertTextFormat != InsertTextFormatSnippet {
		t.Fatalf("[Service] insertTextFormat = %d, want snippet", item.InsertTextFormat)
	}
}

func TestCompletionUsesPlainTextWithoutSnippetSupport(t *testing.T) {
	server := NewServer(systemd.NewCatalog(), nil)
	init := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"capabilities":{}}}`)
	if len(server.Handle(init)) != 1 {
		t.Fatal("initialize returned no response")
	}
	open := []byte(`{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///tmp/demo.service","languageId":"systemd","version":1,"text":"["}}}`)
	server.Handle(open)
	sectionPayload := []byte(`{"jsonrpc":"2.0","id":2,"method":"textDocument/completion","params":{"textDocument":{"uri":"file:///tmp/demo.service"},"position":{"line":0,"character":1}}}`)
	responses := server.Handle(sectionPayload)

	var msg rpcMessage
	if err := json.Unmarshal(responses[0], &msg); err != nil {
		t.Fatal(err)
	}
	var items []CompletionItem
	if err := json.Unmarshal(msg.Result, &items); err != nil {
		t.Fatal(err)
	}
	sectionItem, ok := completionByLabel(items, "[Service]")
	if !ok {
		t.Fatal("missing [Service] completion")
	}
	if sectionItem.InsertText != "[Service]" {
		t.Fatalf("[Service] insertText = %q, want plain text", sectionItem.InsertText)
	}
	if sectionItem.InsertTextFormat != InsertTextFormatPlainText {
		t.Fatalf("[Service] insertTextFormat = %d, want plain text", sectionItem.InsertTextFormat)
	}

	change := []byte(`{"jsonrpc":"2.0","method":"textDocument/didChange","params":{"textDocument":{"uri":"file:///tmp/demo.service","version":2},"contentChanges":[{"text":"[Service]\n"}]}}`)
	server.Handle(change)
	payload := []byte(`{"jsonrpc":"2.0","id":2,"method":"textDocument/completion","params":{"textDocument":{"uri":"file:///tmp/demo.service"},"position":{"line":1,"character":0}}}`)
	responses = server.Handle(payload)

	msg = rpcMessage{}
	if err := json.Unmarshal(responses[0], &msg); err != nil {
		t.Fatal(err)
	}
	items = nil
	if err := json.Unmarshal(msg.Result, &items); err != nil {
		t.Fatal(err)
	}
	item, ok := completionByLabel(items, "ExecStart")
	if !ok {
		t.Fatal("missing ExecStart completion")
	}
	if item.InsertText != "ExecStart=" {
		t.Fatalf("ExecStart insertText = %q, want plain text", item.InsertText)
	}
	if item.InsertTextFormat != InsertTextFormatPlainText {
		t.Fatalf("ExecStart insertTextFormat = %d, want plain text", item.InsertTextFormat)
	}
}

func TestCompletionUsesJapaneseDocumentation(t *testing.T) {
	server := NewServer(systemd.NewCatalog(), nil)
	init := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"initializationOptions":{"locale":"ja"}}}`)
	if len(server.Handle(init)) != 1 {
		t.Fatal("initialize returned no response")
	}
	open := []byte(`{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///tmp/demo.service","languageId":"systemd","version":1,"text":"[Service]\n"}}}`)
	if len(server.Handle(open)) != 1 {
		t.Fatal("didOpen did not publish diagnostics")
	}
	payload := []byte(`{"jsonrpc":"2.0","id":2,"method":"textDocument/completion","params":{"textDocument":{"uri":"file:///tmp/demo.service"},"position":{"line":1,"character":0}}}`)
	responses := server.Handle(payload)
	if len(responses) != 1 {
		t.Fatal("Handle returned no response")
	}
	var msg rpcMessage
	if err := json.Unmarshal(responses[0], &msg); err != nil {
		t.Fatal(err)
	}
	var items []CompletionItem
	if err := json.Unmarshal(msg.Result, &items); err != nil {
		t.Fatal(err)
	}
	item, ok := completionByLabel(items, "ExecStart")
	if !ok {
		t.Fatal("missing ExecStart completion")
	}
	if item.Detail != "[Service] ディレクティブ" {
		t.Fatalf("detail = %q, want Japanese detail", item.Detail)
	}
	for _, want := range []string{
		"サービス起動時に実行するコマンドを指定します。",
		"ExecStart=<command>",
		"ExecStart=/usr/bin/example --foreground",
	} {
		if !strings.Contains(item.Documentation, want) {
			t.Fatalf("documentation = %q, missing %q", item.Documentation, want)
		}
	}
	for _, unwanted := range []string{"**説明**", "**文法**", "**例**"} {
		if strings.Contains(item.Documentation, unwanted) {
			t.Fatalf("documentation = %q, should not contain heading %q", item.Documentation, unwanted)
		}
	}
}

func TestInitializeLoadsGeneratedCatalogPath(t *testing.T) {
	catalogPath := filepath.Join(t.TempDir(), "catalog.json")
	if err := os.WriteFile(catalogPath, []byte(`{
  "version": "test",
  "directives": [
    {
      "section": "Service",
      "name": "RestartMode",
      "parser": "config_parse_service_restart_mode",
      "valueKind": "string",
      "values": ["normal", "direct", "debug"]
    }
  ]
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	server := NewServer(systemd.NewCatalog(), nil)
	init := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"initializationOptions":{"catalogPath":` + strconvQuote(catalogPath) + `}}}`)
	if len(server.Handle(init)) != 1 {
		t.Fatal("initialize returned no response")
	}
	open := []byte(`{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///tmp/demo.service","languageId":"systemd","version":1,"text":"[Service]\n"}}}`)
	if len(server.Handle(open)) != 1 {
		t.Fatal("didOpen did not publish diagnostics")
	}
	payload := []byte(`{"jsonrpc":"2.0","id":2,"method":"textDocument/completion","params":{"textDocument":{"uri":"file:///tmp/demo.service"},"position":{"line":1,"character":0}}}`)
	responses := server.Handle(payload)
	if len(responses) != 1 {
		t.Fatal("Handle returned no response")
	}
	var msg rpcMessage
	if err := json.Unmarshal(responses[0], &msg); err != nil {
		t.Fatal(err)
	}
	var items []CompletionItem
	if err := json.Unmarshal(msg.Result, &items); err != nil {
		t.Fatal(err)
	}
	item, ok := completionByLabel(items, "RestartMode")
	if !ok {
		t.Fatalf("completion labels = %#v, missing RestartMode", items)
	}
	if !strings.Contains(item.Documentation, "RestartMode=normal|direct|debug") {
		t.Fatalf("documentation = %q, missing generated value syntax", item.Documentation)
	}
}

func TestHoverUsesJapaneseDocumentation(t *testing.T) {
	server := NewServer(systemd.NewCatalog(), nil)
	init := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"initializationOptions":{"locale":"ja"}}}`)
	if len(server.Handle(init)) != 1 {
		t.Fatal("initialize returned no response")
	}
	open := []byte(`{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///tmp/demo.service","languageId":"systemd","version":1,"text":"[Service]\nExecStart=/bin/true\n"}}}`)
	if len(server.Handle(open)) != 1 {
		t.Fatal("didOpen did not publish diagnostics")
	}
	payload := []byte(`{"jsonrpc":"2.0","id":2,"method":"textDocument/hover","params":{"textDocument":{"uri":"file:///tmp/demo.service"},"position":{"line":1,"character":3}}}`)
	responses := server.Handle(payload)
	if len(responses) != 1 {
		t.Fatal("Handle returned no response")
	}
	var msg rpcMessage
	if err := json.Unmarshal(responses[0], &msg); err != nil {
		t.Fatal(err)
	}
	var hover Hover
	if err := json.Unmarshal(msg.Result, &hover); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"サービス起動時に実行するコマンドを指定します。",
		"ExecStart=<command>",
		"ExecStart=/usr/bin/example --foreground",
	} {
		if !strings.Contains(hover.Contents.Value, want) {
			t.Fatalf("hover = %q, missing %q", hover.Contents.Value, want)
		}
	}
}

func TestDidOpenEmptyServiceRequestsTemplateInsertion(t *testing.T) {
	server := NewServer(systemd.NewCatalog(), nil)
	payload := []byte(`{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///tmp/demo.service","languageId":"systemd","version":1,"text":""}}}`)
	responses := server.Handle(payload)
	if len(responses) != 1 {
		t.Fatalf("response count = %d, want 1", len(responses))
	}
	var msg rpcMessage
	if err := json.Unmarshal(responses[0], &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Method != "workspace/applyEdit" {
		t.Fatalf("method = %q, want workspace/applyEdit", msg.Method)
	}
	var params struct {
		Label string `json:"label"`
		Edit  struct {
			Changes map[string][]struct {
				NewText string `json:"newText"`
			} `json:"changes"`
		} `json:"edit"`
	}
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		t.Fatal(err)
	}
	edits := params.Edit.Changes["file:///tmp/demo.service"]
	if len(edits) != 1 {
		t.Fatalf("edit count = %d, want 1", len(edits))
	}
	if edits[0].NewText != defaultServiceTemplate {
		t.Fatalf("newText = %q, want default service template", edits[0].NewText)
	}
}

func TestDidOpenEmptyServiceDropInDoesNotRequestTemplateInsertion(t *testing.T) {
	server := NewServer(systemd.NewCatalog(), nil)
	payload := []byte(`{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///etc/systemd/system/demo.service.d/override.conf","languageId":"systemd","version":1,"text":""}}}`)
	responses := server.Handle(payload)
	if len(responses) != 1 {
		t.Fatalf("response count = %d, want diagnostics notification", len(responses))
	}
	var msg rpcMessage
	if err := json.Unmarshal(responses[0], &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Method != "textDocument/publishDiagnostics" {
		t.Fatalf("method = %q, want publishDiagnostics", msg.Method)
	}
}

func TestDidOpenEmptyTimerDoesNotRequestTemplateInsertion(t *testing.T) {
	server := NewServer(systemd.NewCatalog(), nil)
	payload := []byte(`{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///tmp/demo.timer","languageId":"systemd","version":1,"text":""}}}`)
	responses := server.Handle(payload)
	if len(responses) != 1 {
		t.Fatalf("response count = %d, want diagnostics notification", len(responses))
	}
	var msg rpcMessage
	if err := json.Unmarshal(responses[0], &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Method == "workspace/applyEdit" {
		t.Fatal("empty .timer unexpectedly requested template insertion")
	}
}

func hasCompletion(items []CompletionItem, label string) bool {
	_, ok := completionByLabel(items, label)
	return ok
}

func completionByLabel(items []CompletionItem, label string) (CompletionItem, bool) {
	for _, item := range items {
		if item.Label == label {
			return item, true
		}
	}
	return CompletionItem{}, false
}

func strconvQuote(value string) string {
	b, _ := json.Marshal(value)
	return string(b)
}

func initializeWithSnippetSupport(t *testing.T, server *Server) {
	t.Helper()
	payload := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"capabilities":{"textDocument":{"completion":{"completionItem":{"snippetSupport":true}}}}}}`)
	if len(server.Handle(payload)) != 1 {
		t.Fatal("initialize returned no response")
	}
}

func TestInitializeReportsBuildVersion(t *testing.T) {
	server := NewServer(systemd.NewCatalog(), nil)
	responses := server.Handle([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`))
	var msg struct {
		Result struct {
			ServerInfo struct {
				Version string `json:"version"`
			} `json:"serverInfo"`
		} `json:"result"`
	}
	if err := json.Unmarshal(responses[0], &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Result.ServerInfo.Version != version.Value {
		t.Fatalf("serverInfo.version = %q, want %q", msg.Result.ServerInfo.Version, version.Value)
	}
}

func TestExitCodeFollowsShutdown(t *testing.T) {
	server := NewServer(systemd.NewCatalog(), nil)
	if _, exited := server.ExitCode(); exited {
		t.Fatal("exited before the exit notification")
	}
	server.Handle([]byte(`{"jsonrpc":"2.0","method":"exit"}`))
	if code, exited := server.ExitCode(); !exited || code != 1 {
		t.Fatalf("exit without shutdown = (%d, %v), want (1, true)", code, exited)
	}

	server = NewServer(systemd.NewCatalog(), nil)
	server.Handle([]byte(`{"jsonrpc":"2.0","id":1,"method":"shutdown"}`))
	if _, exited := server.ExitCode(); exited {
		t.Fatal("shutdown alone must not end the process")
	}
	server.Handle([]byte(`{"jsonrpc":"2.0","method":"exit"}`))
	if code, exited := server.ExitCode(); !exited || code != 0 {
		t.Fatalf("exit after shutdown = (%d, %v), want (0, true)", code, exited)
	}
}

func openDocument(t *testing.T, server *Server, uri, text string) {
	t.Helper()
	payload, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{
		"textDocument": TextDocumentItem{URI: uri, Text: text},
	}})
	if err != nil {
		t.Fatal(err)
	}
	server.Handle(payload)
}

func request(t *testing.T, server *Server, method string, params, result any) {
	t.Helper()
	payload, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	if err != nil {
		t.Fatal(err)
	}
	responses := server.Handle(payload)
	if len(responses) != 1 {
		t.Fatalf("%s returned %d responses", method, len(responses))
	}
	var msg rpcMessage
	if err := json.Unmarshal(responses[0], &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Error != nil {
		t.Fatalf("%s error = %#v", method, msg.Error)
	}
	if err := json.Unmarshal(msg.Result, result); err != nil {
		t.Fatal(err)
	}
}

func TestRangesCountUTF16Units(t *testing.T) {
	server := NewServer(systemd.NewCatalog(), nil)
	uri := "file:///tmp/demo.service"
	line := "Description=日本語😀のサービス"
	openDocument(t, server, uri, "[Unit]\n"+line+"\n")

	var symbols []DocumentSymbol
	request(t, server, "textDocument/documentSymbol", map[string]any{"textDocument": TextDocumentIdentifier{URI: uri}}, &symbols)
	if got, want := symbols[0].Children[0].Range.End.Character, len("Description=")+10; got != want {
		t.Fatalf("symbol end character = %d, want %d", got, want)
	}

	// A hover position after non-ASCII text must not be read as a byte offset.
	openDocument(t, server, uri, "[Unit]\nDescription=日本語 Documentation\n")
	var hover *Hover
	request(t, server, "textDocument/hover", TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{URI: uri}, Position: Position{Line: 1, Character: 18},
	}, &hover)
	if hover == nil || hover.Range.Start.Character != 16 || hover.Range.End.Character != 29 {
		t.Fatalf("hover = %#v, want Documentation at 16-29", hover)
	}
}

func TestContinuationLinesOfferNoCompletion(t *testing.T) {
	server := NewServer(systemd.NewCatalog(), nil)
	uri := "file:///tmp/demo.service"
	openDocument(t, server, uri, "[Service]\nExecStart=/usr/bin/foo \\\n    --type=\n")
	var items []CompletionItem
	request(t, server, "textDocument/completion", TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{URI: uri}, Position: Position{Line: 2, Character: 11},
	}, &items)
	if len(items) != 0 {
		t.Fatalf("completion on a continuation line = %#v, want none", items)
	}
}

func TestDropInDoesNotWarnAboutMissingPIDFile(t *testing.T) {
	server := NewServer(systemd.NewCatalog(), nil)
	payload := []byte(`{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///etc/systemd/system/demo.service.d/override.conf","text":"[Service]\nType=forking\n"}}}`)
	var msg struct {
		Params struct {
			Diagnostics []Diagnostic `json:"diagnostics"`
		} `json:"params"`
	}
	if err := json.Unmarshal(server.Handle(payload)[0], &msg); err != nil {
		t.Fatal(err)
	}
	if len(msg.Params.Diagnostics) != 1 || msg.Params.Diagnostics[0].Severity != SeverityHint {
		t.Fatalf("diagnostics = %#v, want one hint", msg.Params.Diagnostics)
	}
}

func TestSectionCompletionReplacesTypedHeader(t *testing.T) {
	server := NewServer(systemd.NewCatalog(), nil)
	uri := "file:///tmp/demo.service"
	for _, tc := range []struct {
		line      string
		character int
		newText   string
	}{
		{"  [Ser", 6, "[Service]"},
		// A bracket the editor paired stays in place.
		{"[Ser]", 4, "[Service"},
	} {
		openDocument(t, server, uri, "[Unit]\n"+tc.line+"\n")
		var items []CompletionItem
		request(t, server, "textDocument/completion", TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{URI: uri}, Position: Position{Line: 1, Character: tc.character},
		}, &items)
		var service *CompletionItem
		for i := range items {
			if items[i].Label == "[Service]" {
				service = &items[i]
			}
		}
		if service == nil || service.TextEdit == nil {
			t.Fatalf("%q: no [Service] item with a text edit: %#v", tc.line, items)
		}
		edit := service.TextEdit
		start := strings.Index(tc.line, "[")
		if service.InsertTextFormat != InsertTextFormatPlainText {
			t.Fatalf("%q: insertTextFormat = %d, want plain text without snippet support", tc.line, service.InsertTextFormat)
		}
		if edit.NewText != tc.newText || edit.Range.Start.Character != start || edit.Range.End.Character != tc.character {
			t.Fatalf("%q: edit = %#v, want %q over %d-%d", tc.line, edit, tc.newText, start, tc.character)
		}
	}
}

func TestCommentLinesOfferNoCompletion(t *testing.T) {
	server := NewServer(systemd.NewCatalog(), nil)
	uri := "file:///tmp/demo.service"
	openDocument(t, server, uri, "[Service]\n# see the \n")
	var items []CompletionItem
	request(t, server, "textDocument/completion", TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{URI: uri}, Position: Position{Line: 1, Character: 10},
	}, &items)
	if len(items) != 0 {
		t.Fatalf("completion in a comment returned %d items, want none", len(items))
	}
}

func TestOtherSystemdFormatsAreLeftAlone(t *testing.T) {
	server := NewServer(systemd.NewCatalog(), nil)
	for _, uri := range []string{"file:///etc/systemd/network/10-eth.network", "file:///etc/systemd/network/br0.netdev", "file:///etc/systemd/journald.conf"} {
		payload, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{
			"textDocument": TextDocumentItem{URI: uri, Text: "[Match]\nName=eth0\n\n"},
		}})
		if err != nil {
			t.Fatal(err)
		}
		var msg struct {
			Params struct {
				Diagnostics []Diagnostic `json:"diagnostics"`
			} `json:"params"`
		}
		if err := json.Unmarshal(server.Handle(payload)[0], &msg); err != nil {
			t.Fatal(err)
		}
		if len(msg.Params.Diagnostics) != 0 {
			t.Errorf("%s: diagnostics = %#v, want none", uri, msg.Params.Diagnostics)
		}
		var items []CompletionItem
		request(t, server, "textDocument/completion", TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{URI: uri}, Position: Position{Line: 2, Character: 0},
		}, &items)
		if len(items) != 0 {
			t.Errorf("%s: completion returned %d items, want none", uri, len(items))
		}
	}
}
