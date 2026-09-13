package lsp

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/zyosaidouki/systemd-lsp/internal/systemd"
)

func decodeWireObject(t *testing.T, payload []byte) map[string]jsontext.Value {
	t.Helper()
	var object map[string]jsontext.Value
	if err := json.Unmarshal(payload, &object); err != nil {
		t.Fatalf("invalid response %q: %v", payload, err)
	}
	return object
}

func TestJSONResponseRetainsRequiredEmptyValues(t *testing.T) {
	for _, tc := range []struct {
		name   string
		result any
		want   string
	}{
		{"null", nil, "null"},
		{"empty array", []CompletionItem{}, "[]"},
		{"nil slice", []DocumentSymbol(nil), "[]"},
		{"empty object", map[string]any{}, "{}"},
		{"false", false, "false"},
		{"zero", 0, "0"},
		{"empty string", "", `""`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			object := decodeWireObject(t, encodeResponse(jsontext.Value("0"), tc.result, nil))
			if string(object["id"]) != "0" || string(object["result"]) != tc.want {
				t.Fatalf("required values lost: %v", object)
			}
			if _, exists := object["error"]; exists {
				t.Fatal("success response contains error")
			}
		})
	}
}

func TestJSONResponsePreservesRequestID(t *testing.T) {
	for _, id := range []string{"0", "9007199254740993", `""`, `"日本😀"`} {
		server := NewServer(systemd.NewCatalog(), nil)
		responses := server.Handle([]byte(`{"jsonrpc":"2.0","id":` + id + `,"method":"shutdown"}`))
		if len(responses) != 1 {
			t.Fatalf("response count = %d", len(responses))
		}
		object := decodeWireObject(t, responses[0])
		if string(object["id"]) != id || string(object["result"]) != "null" {
			t.Fatalf("ID or null result lost: %s", responses[0])
		}
	}
}

func TestJSONNotificationsKeepParamsWithoutResponseFields(t *testing.T) {
	object := decodeWireObject(t, notification("test", map[string]any{}))
	if string(object["params"]) != "{}" {
		t.Fatal("empty params were omitted")
	}
	for _, name := range []string{"id", "result", "error"} {
		if _, exists := object[name]; exists {
			t.Fatalf("notification contains %s", name)
		}
	}
	object = decodeWireObject(t, notification("textDocument/publishDiagnostics", map[string]any{"uri": "file:///tmp/demo.service", "diagnostics": []Diagnostic(nil)}))
	params := decodeWireObject(t, object["params"])
	if string(params["diagnostics"]) != "[]" {
		t.Fatal("empty diagnostics must be an array")
	}
}

func TestJSONParseErrorsDoNotMutateDocuments(t *testing.T) {
	uri := "file:///tmp/demo.service"
	for _, payload := range []string{
		`{"jsonrpc":"2.0","id":1,"id":2,"method":"shutdown"}`,
		`{"jsonrpc":"2.0","method":"textDocument/didChange","params":{"textDocument":{"uri":"file:///tmp/demo.service"},"contentChanges":[{"text":"first","text":"second"}]}}`,
		"{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/didChange\",\"params\":{\"textDocument\":{\"uri\":\"file:///tmp/demo.service\"},\"contentChanges\":[{\"text\":\"\xff\"}]}}",
		`{"jsonrpc":"2.0","method":"textDocument/didChange","params":{"textDocument":{"uri":"file:///tmp/demo.service"},"contentChanges":[{"text":"\ud800"}]}}`,
	} {
		server := NewServer(systemd.NewCatalog(), nil)
		server.docs[uri] = "original"
		responses := server.Handle([]byte(payload))
		if len(responses) != 1 {
			t.Fatalf("expected parse error for %q", payload)
		}
		object := decodeWireObject(t, responses[0])
		if string(object["id"]) != "null" {
			t.Fatalf("parse error must contain id:null: %s", responses[0])
		}
		if _, exists := object["result"]; exists {
			t.Fatal("error response contains result")
		}
		var rpcErr rpcError
		if err := json.Unmarshal(object["error"], &rpcErr); err != nil || rpcErr.Code != -32700 {
			t.Fatalf("expected parse error, got %s", responses[0])
		}
		if server.shutdown || server.docs[uri] != "original" {
			t.Fatal("invalid JSON changed server state")
		}
	}
}

func TestJSONMarshalErrorReturnsValidRPCError(t *testing.T) {
	object := decodeWireObject(t, encodeResponse(jsontext.Value(`"request"`), "\xff", nil))
	if string(object["id"]) != `"request"` {
		t.Fatal("marshal error lost request ID")
	}
	if _, exists := object["result"]; exists {
		t.Fatal("marshal error contains a partial result")
	}
	var rpcErr rpcError
	if err := json.Unmarshal(object["error"], &rpcErr); err != nil || rpcErr.Code != -32603 {
		t.Fatalf("missing internal error: %v", object)
	}
}

func TestJSONOptionalZeroFieldsAndRequiredPositions(t *testing.T) {
	payload, err := json.Marshal(CompletionItem{Label: "CPUSetPartition"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), "kind") || strings.Contains(string(payload), "insertTextFormat") || strings.Contains(string(payload), "textEdit") {
		t.Fatalf("optional zero fields were emitted: %s", payload)
	}
	payload, err = json.Marshal(Range{})
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != `{"start":{"line":0,"character":0},"end":{"line":0,"character":0}}` {
		t.Fatalf("zero positions were lost: %s", payload)
	}
}
