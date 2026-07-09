package lsp

import (
	"encoding/json"
	"testing"

	"github.com/zyosaidouki/systemd-lsp/internal/systemd"
)

func TestInitialize(t *testing.T) {
	server := NewServer(systemd.NewCatalog(), nil)
	payload := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	response, ok := server.Handle(payload)
	if !ok {
		t.Fatal("Handle returned no response")
	}
	var msg rpcMessage
	if err := json.Unmarshal(response, &msg); err != nil {
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
	response, ok := server.Handle(payload)
	if !ok {
		t.Fatal("Handle returned no notification")
	}
	var msg rpcMessage
	if err := json.Unmarshal(response, &msg); err != nil {
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
	if _, ok := server.Handle(open); !ok {
		t.Fatal("didOpen did not publish diagnostics")
	}
	payload := []byte(`{"jsonrpc":"2.0","id":2,"method":"textDocument/completion","params":{"textDocument":{"uri":"file:///tmp/demo.service"},"position":{"line":1,"character":0}}}`)
	response, ok := server.Handle(payload)
	if !ok {
		t.Fatal("Handle returned no response")
	}
	var msg rpcMessage
	if err := json.Unmarshal(response, &msg); err != nil {
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
}

func hasCompletion(items []CompletionItem, label string) bool {
	for _, item := range items {
		if item.Label == label {
			return true
		}
	}
	return false
}
