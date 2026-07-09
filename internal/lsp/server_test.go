package lsp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zyosaidouki/systemd-lsp/internal/systemd"
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
		"**説明**",
		"サービス起動時に実行するコマンドを指定します。",
		"**文法**",
		"ExecStart=<command>",
		"**例**",
		"ExecStart=/usr/bin/example --foreground",
	} {
		if !strings.Contains(item.Documentation, want) {
			t.Fatalf("documentation = %q, missing %q", item.Documentation, want)
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
