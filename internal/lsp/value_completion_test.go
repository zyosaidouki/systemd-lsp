package lsp

import (
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/zyosaidouki/systemd-lsp/internal/systemd"
)

func completionRequest(t *testing.T, server *Server, section, markedLine string) []CompletionItem {
	t.Helper()
	caret := strings.Index(markedLine, "¦")
	if caret < 0 {
		t.Fatal("missing caret marker")
	}
	line := strings.Replace(markedLine, "¦", "", 1)
	uri := "file:///tmp/completion.service"
	open, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{"textDocument": TextDocumentItem{URI: uri, Text: "[" + section + "]\n" + line, Version: 1}}})
	server.Handle(open)
	req, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 20, "method": "textDocument/completion", "params": TextDocumentPositionParams{TextDocument: TextDocumentIdentifier{URI: uri}, Position: Position{Line: 1, Character: systemd.UTF16Length(markedLine[:caret])}}})
	responses := server.Handle(req)
	if len(responses) != 1 {
		t.Fatalf("responses = %d", len(responses))
	}
	var msg rpcMessage
	if err := json.Unmarshal(responses[0], &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Error != nil {
		t.Fatalf("completion error: %#v", msg.Error)
	}
	var items []CompletionItem
	if err := json.Unmarshal(msg.Result, &items); err != nil {
		t.Fatal(err)
	}
	return items
}

func TestValueCompletionPreservesSurroundingText(t *testing.T) {
	server := NewServer(systemd.NewCatalog(), nil)
	for _, tc := range []struct{ line, label, want string }{
		{"PrivateUsers=fu¦", "full", "PrivateUsers=full"},
		{"CPUSetPartition=iso¦lated", "isolated", "CPUSetPartition=isolated"},
		{"RefreshOnReload=extensions cre¦", "credentials", "RefreshOnReload=extensions credentials"},
		{"RefreshOnReload=~ext¦ensions credentials", "extensions", "RefreshOnReload=~extensions credentials"},
		{"RefreshOnReload=~extensions ¦", "credentials", "RefreshOnReload=~extensions credentials"},
		{"SecureBits=keep-caps no-s¦", "no-setuid-fixup", "SecureBits=keep-caps no-setuid-fixup"},
		{"SecureBits=keep-caps \"no-s¦\"", "no-setuid-fixup", "SecureBits=keep-caps \"no-setuid-fixup\""},
		{"  IOPressureWatch= au¦to", "auto", "  IOPressureWatch= auto"},
	} {
		t.Run(tc.line, func(t *testing.T) {
			items := completionRequest(t, server, "Service", tc.line)
			item, ok := completionByLabel(items, tc.label)
			if !ok || item.TextEdit == nil {
				t.Fatalf("missing completion %s: %#v", tc.label, items)
			}
			line := strings.Replace(tc.line, "¦", "", 1)
			edit := item.TextEdit
			// These fixtures are ASCII, so UTF-16 columns are also byte offsets.
			got := line[:edit.Range.Start.Character] + edit.NewText + line[edit.Range.End.Character:]
			if got != tc.want {
				t.Fatalf("edit result = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResourceListCompletionDoesNotMixBooleansOrDuplicateValues(t *testing.T) {
	server := NewServer(systemd.NewCatalog(), nil)
	for _, line := range []string{"RefreshOnReload=extensions ¦", "RefreshOnReload=~extensions ¦"} {
		items := completionRequest(t, server, "Service", line)
		if len(items) != 1 || items[0].Label != "credentials" {
			t.Fatalf("invalid next-resource candidates: %#v", items)
		}
	}
	items := completionRequest(t, server, "Service", "RefreshOnReload=~¦")
	if len(items) != 2 || !hasCompletion(items, "extensions") || !hasCompletion(items, "credentials") {
		t.Fatalf("invalid inverted candidates: %#v", items)
	}
}

func TestConditionSnippetsRespectClientCapabilities(t *testing.T) {
	for _, snippets := range []bool{false, true} {
		server := NewServer(systemd.NewCatalog(), nil)
		if snippets {
			initializeWithSnippetSupport(t, server)
		}
		items := completionRequest(t, server, "Unit", "¦")
		for _, name := range []string{"ConditionFraction", "AssertFraction", "ConditionMachineTag"} {
			item, ok := completionByLabel(items, name)
			if !ok {
				t.Fatalf("missing %s", name)
			}
			if snippets {
				if !strings.Contains(item.InsertText, "${1:") || item.InsertTextFormat != InsertTextFormatSnippet {
					t.Fatalf("missing editable placeholders: %#v", item)
				}
			} else if item.InsertText != name+"=" || item.InsertTextFormat != InsertTextFormatPlainText {
				t.Fatalf("snippet leaked to plain text client: %#v", item)
			}
		}
	}
}

func TestPinnedCatalogCompletionAndHover(t *testing.T) {
	c := systemd.NewCatalog()
	c.MergeCatalogFile(systemd.CatalogFile{Version: "v260", Directives: []systemd.CatalogDirective{{Section: "Service", Name: "FileDescriptorStorePreserve"}}})
	server := NewServer(c, nil)
	items := completionRequest(t, server, "Service", "FileDescriptorStorePreserve=¦")
	if hasCompletion(items, "on-success") || !hasCompletion(items, "restart") {
		t.Fatalf("wrong pinned values: %#v", items)
	}
	items = completionRequest(t, server, "Service", "¦")
	if hasCompletion(items, "CPUSetPartition") || hasCompletion(items, "RestartRandomizedDelaySec") {
		t.Fatal("newer directives leaked into pinned completion")
	}
	server = NewServer(systemd.NewCatalog(), nil)
	server.locale = "ja"
	completionRequest(t, server, "Service", "CPUSetPartition=¦")
	result := server.hover(TextDocumentPositionParams{TextDocument: TextDocumentIdentifier{URI: "file:///tmp/completion.service"}, Position: Position{Line: 1, Character: 3}}).(Hover)
	for _, want := range []string{"AllowedCPUs=", "v261", "CPU集合"} {
		if !strings.Contains(result.Contents.Value, want) {
			t.Fatalf("hover missing %q: %s", want, result.Contents.Value)
		}
	}
}

func TestValueCompletionUTF16Range(t *testing.T) {
	server := NewServer(systemd.NewCatalog(), nil)
	items := completionRequest(t, server, "Service", "SecureBits=\"日本😀\" no-s¦")
	item, ok := completionByLabel(items, "no-setuid-fixup")
	if !ok {
		t.Fatal("missing completion after non-ASCII token")
	}
	if got, want := item.TextEdit.Range.Start.Character, systemd.UTF16Length("SecureBits=\"日本😀\" "); got != want {
		t.Fatalf("start = %d, want %d", got, want)
	}
}
