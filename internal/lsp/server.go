package lsp

import (
	"encoding/json"
	"fmt"
	"log"
	"path"
	"strings"

	"github.com/zyosaidouki/systemd-lsp/internal/systemd"
)

type Server struct {
	catalog       *systemd.Catalog
	logger        *log.Logger
	docs          map[string]string
	shutdown      bool
	nextRequestID int
}

func NewServer(catalog *systemd.Catalog, logger *log.Logger) *Server {
	return &Server{
		catalog: catalog,
		logger:  logger,
		docs:    map[string]string{},
	}
}

type rpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *Server) Handle(payload json.RawMessage) [][]byte {
	var msg rpcMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		return [][]byte{encodeResponse(nil, nil, &rpcError{Code: -32700, Message: err.Error()})}
	}
	if msg.Method == "" {
		return nil
	}

	result, notifications, err := s.dispatch(msg.Method, msg.Params)
	if len(msg.ID) == 0 {
		return notifications
	}
	if err != nil {
		return [][]byte{encodeResponse(msg.ID, nil, &rpcError{Code: -32603, Message: err.Error()})}
	}
	return [][]byte{encodeResponse(msg.ID, result, nil)}
}

func (s *Server) dispatch(method string, params json.RawMessage) (any, [][]byte, error) {
	switch method {
	case "initialize":
		return s.initialize(), nil, nil
	case "initialized":
		return nil, nil, nil
	case "shutdown":
		s.shutdown = true
		return nil, nil, nil
	case "exit":
		return nil, nil, nil
	case "textDocument/didOpen":
		var p struct {
			TextDocument TextDocumentItem `json:"textDocument"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, nil, err
		}
		s.docs[p.TextDocument.URI] = p.TextDocument.Text
		if shouldInsertServiceTemplate(p.TextDocument.URI, p.TextDocument.Text) {
			return nil, [][]byte{s.applyEditRequest(p.TextDocument.URI, defaultServiceTemplate)}, nil
		}
		return nil, s.publishDiagnostics(p.TextDocument.URI), nil
	case "textDocument/didChange":
		var p struct {
			TextDocument   VersionedTextDocumentIdentifier `json:"textDocument"`
			ContentChanges []struct {
				Text string `json:"text"`
			} `json:"contentChanges"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, nil, err
		}
		if len(p.ContentChanges) > 0 {
			s.docs[p.TextDocument.URI] = p.ContentChanges[len(p.ContentChanges)-1].Text
		}
		return nil, s.publishDiagnostics(p.TextDocument.URI), nil
	case "textDocument/didSave":
		var p struct {
			TextDocument TextDocumentIdentifier `json:"textDocument"`
			Text         *string                `json:"text,omitempty"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, nil, err
		}
		if p.Text != nil {
			s.docs[p.TextDocument.URI] = *p.Text
		}
		return nil, s.publishDiagnostics(p.TextDocument.URI), nil
	case "textDocument/didClose":
		var p struct {
			TextDocument TextDocumentIdentifier `json:"textDocument"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, nil, err
		}
		delete(s.docs, p.TextDocument.URI)
		return nil, [][]byte{notification("textDocument/publishDiagnostics", map[string]any{
			"uri":         p.TextDocument.URI,
			"diagnostics": []Diagnostic{},
		})}, nil
	case "textDocument/completion":
		var p TextDocumentPositionParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, nil, err
		}
		return s.completion(p), nil, nil
	case "textDocument/hover":
		var p TextDocumentPositionParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, nil, err
		}
		return s.hover(p), nil, nil
	case "textDocument/documentSymbol":
		var p struct {
			TextDocument TextDocumentIdentifier `json:"textDocument"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, nil, err
		}
		return s.documentSymbols(p.TextDocument.URI), nil, nil
	default:
		if s.logger != nil {
			s.logger.Printf("unhandled method: %s", method)
		}
		return nil, nil, nil
	}
}

const defaultServiceTemplate = `[Unit]
Description=

[Service]
Type=simple
ExecStart=
ExecStop=

[Install]
WantedBy=multi-user.target
`

func shouldInsertServiceTemplate(uri, text string) bool {
	return unitTypeFromURI(uri) == "service" && strings.TrimSpace(text) == ""
}

func (s *Server) applyEditRequest(uri, newText string) []byte {
	s.nextRequestID++
	id := json.RawMessage(fmt.Sprintf("%d", s.nextRequestID))
	params := map[string]any{
		"label": "Insert systemd service template",
		"edit": map[string]any{
			"changes": map[string]any{
				uri: []map[string]any{
					{
						"range": Range{
							Start: Position{Line: 0, Character: 0},
							End:   Position{Line: 0, Character: 0},
						},
						"newText": newText,
					},
				},
			},
		},
	}
	raw, err := json.Marshal(params)
	if err != nil {
		raw = json.RawMessage("null")
	}
	payload, _ := json.Marshal(rpcMessage{
		JSONRPC: "2.0",
		ID:      id,
		Method:  "workspace/applyEdit",
		Params:  raw,
	})
	return payload
}

func (s *Server) initialize() map[string]any {
	return map[string]any{
		"serverInfo": map[string]any{
			"name":    "systemd-lsp",
			"version": "0.1.0",
		},
		"capabilities": map[string]any{
			"textDocumentSync": map[string]any{
				"openClose": true,
				"change":    1,
				"save": map[string]any{
					"includeText": true,
				},
			},
			"completionProvider": map[string]any{
				"triggerCharacters": []string{"[", "="},
			},
			"hoverProvider":          true,
			"documentSymbolProvider": true,
		},
	}
}

func (s *Server) publishDiagnostics(uri string) [][]byte {
	text, ok := s.docs[uri]
	if !ok {
		return nil
	}
	unitType := unitTypeFromURI(uri)
	diagnostics := systemd.Diagnostics(s.catalog, text, unitType)
	return [][]byte{notification("textDocument/publishDiagnostics", map[string]any{
		"uri":         uri,
		"diagnostics": diagnostics,
	})}
}

func (s *Server) completion(p TextDocumentPositionParams) any {
	doc := s.docs[p.TextDocument.URI]
	unitType := unitTypeFromURI(p.TextDocument.URI)
	line := lineAt(doc, p.Position.Line)
	before := prefixAt(line, p.Position.Character)
	trimmed := strings.TrimLeft(before, " \t")

	if strings.HasPrefix(trimmed, "[") {
		items := make([]CompletionItem, 0, len(s.catalog.SectionsFor(unitType)))
		for _, section := range s.catalog.SectionsFor(unitType) {
			items = append(items, CompletionItem{
				Label:            "[" + section + "]",
				Kind:             CompletionItemKindSnippet,
				Detail:           "systemd section",
				Documentation:    s.catalog.SectionDoc(section),
				InsertText:       "[" + section + "]\n$0",
				InsertTextFormat: InsertTextFormatSnippet,
			})
		}
		return items
	}

	if key, value, hasEquals := strings.Cut(trimmed, "="); hasEquals {
		section := sectionAt(s.catalog, doc, p.Position.Line)
		return s.valueCompletions(section, strings.TrimSpace(key), strings.TrimSpace(value))
	}

	section := sectionAt(s.catalog, doc, p.Position.Line)
	if section == "" {
		return []CompletionItem{}
	}
	directives := s.catalog.DirectivesFor(section)
	items := make([]CompletionItem, 0, len(directives))
	for _, directive := range directives {
		items = append(items, CompletionItem{
			Label:            directive.Name,
			Kind:             CompletionItemKindField,
			Detail:           "[" + section + "] directive",
			Documentation:    directive.Doc,
			InsertText:       directive.Name + "=$0",
			InsertTextFormat: InsertTextFormatSnippet,
		})
	}
	return items
}

func (s *Server) valueCompletions(section, key, prefix string) []CompletionItem {
	values := s.catalog.ValuesFor(section, key)
	if len(values) == 0 {
		return []CompletionItem{}
	}
	items := make([]CompletionItem, 0, len(values))
	for _, value := range values {
		if prefix != "" && !strings.HasPrefix(strings.ToLower(value), strings.ToLower(prefix)) {
			continue
		}
		items = append(items, CompletionItem{
			Label:      value,
			Kind:       CompletionItemKindValue,
			Detail:     key + " value",
			InsertText: value,
		})
	}
	return items
}

func (s *Server) hover(p TextDocumentPositionParams) any {
	doc := s.docs[p.TextDocument.URI]
	line := lineAt(doc, p.Position.Line)
	wordRange, word := wordAt(line, p.Position.Character)
	if word == "" {
		return nil
	}
	if strings.HasPrefix(strings.TrimSpace(line), "[") && strings.HasSuffix(strings.TrimSpace(line), "]") {
		section := strings.Trim(strings.TrimSpace(line), "[]")
		if doc := s.catalog.SectionDoc(section); doc != "" {
			return Hover{
				Contents: MarkupContent{Kind: "markdown", Value: fmt.Sprintf("**[%s]**\n\n%s", section, doc)},
				Range:    &Range{Start: Position{Line: p.Position.Line, Character: wordRange.Start}, End: Position{Line: p.Position.Line, Character: wordRange.End}},
			}
		}
	}
	section := sectionAt(s.catalog, doc, p.Position.Line)
	if directive, ok := s.catalog.Directive(section, word); ok {
		return Hover{
			Contents: MarkupContent{Kind: "markdown", Value: fmt.Sprintf("**%s=**\n\n%s", directive.Name, directive.Doc)},
			Range:    &Range{Start: Position{Line: p.Position.Line, Character: wordRange.Start}, End: Position{Line: p.Position.Line, Character: wordRange.End}},
		}
	}
	return nil
}

func (s *Server) documentSymbols(uri string) []DocumentSymbol {
	doc := s.docs[uri]
	parsed := systemd.Parse(doc)
	symbols := make([]DocumentSymbol, 0)
	var current *DocumentSymbol
	for _, entry := range parsed.Entries {
		switch entry.Kind {
		case systemd.EntrySection:
			symbols = append(symbols, DocumentSymbol{
				Name: entry.Section,
				Kind: SymbolKindNamespace,
				Range: Range{
					Start: Position{Line: entry.Line, Character: 0},
					End:   Position{Line: entry.Line, Character: len(entry.Raw)},
				},
				SelectionRange: Range{
					Start: Position{Line: entry.Line, Character: entry.KeyStart},
					End:   Position{Line: entry.Line, Character: entry.KeyEnd},
				},
			})
			current = &symbols[len(symbols)-1]
		case systemd.EntryDirective:
			if current == nil {
				continue
			}
			current.Children = append(current.Children, DocumentSymbol{
				Name:   entry.Key,
				Detail: entry.Value,
				Kind:   SymbolKindProperty,
				Range: Range{
					Start: Position{Line: entry.Line, Character: 0},
					End:   Position{Line: entry.Line, Character: len(entry.Raw)},
				},
				SelectionRange: Range{
					Start: Position{Line: entry.Line, Character: entry.KeyStart},
					End:   Position{Line: entry.Line, Character: entry.KeyEnd},
				},
			})
		}
	}
	return symbols
}

func encodeResponse(id json.RawMessage, result any, rpcErr *rpcError) []byte {
	response := rpcMessage{JSONRPC: "2.0", ID: id, Error: rpcErr}
	if rpcErr == nil {
		if result == nil {
			response.Result = json.RawMessage("null")
		} else {
			raw, err := json.Marshal(result)
			if err != nil {
				response.Error = &rpcError{Code: -32603, Message: err.Error()}
			} else {
				response.Result = raw
			}
		}
	}
	payload, _ := json.Marshal(response)
	return payload
}

func notification(method string, params any) []byte {
	raw, err := json.Marshal(params)
	if err != nil {
		raw = json.RawMessage("null")
	}
	payload, _ := json.Marshal(rpcMessage{
		JSONRPC: "2.0",
		Method:  method,
		Params:  raw,
	})
	return payload
}

func unitTypeFromURI(uri string) string {
	base := path.Base(uri)
	if strings.HasSuffix(base, ".conf") {
		dir := path.Base(path.Dir(uri))
		if strings.HasSuffix(dir, ".d") {
			base = strings.TrimSuffix(dir, ".d")
		}
	}
	ext := path.Ext(base)
	if ext == "" {
		return ""
	}
	return strings.TrimPrefix(ext, ".")
}

func lineAt(text string, lineNo int) string {
	lines := strings.Split(text, "\n")
	if lineNo < 0 || lineNo >= len(lines) {
		return ""
	}
	return lines[lineNo]
}

func prefixAt(line string, character int) string {
	if character < 0 {
		return ""
	}
	runes := []rune(line)
	if character > len(runes) {
		character = len(runes)
	}
	return string(runes[:character])
}

func sectionAt(catalog *systemd.Catalog, text string, lineNo int) string {
	parsed := systemd.Parse(text)
	section := ""
	for _, entry := range parsed.Entries {
		if entry.Line >= lineNo {
			break
		}
		if entry.Kind == systemd.EntrySection && catalog.KnowsSection(entry.Section) {
			section = entry.Section
		}
	}
	return section
}

type simpleRange struct {
	Start int
	End   int
}

func wordAt(line string, character int) (simpleRange, string) {
	if character < 0 {
		character = 0
	}
	if character > len(line) {
		character = len(line)
	}
	start := character
	for start > 0 && isWordByte(line[start-1]) {
		start--
	}
	end := character
	for end < len(line) && isWordByte(line[end]) {
		end++
	}
	if start == end {
		return simpleRange{}, ""
	}
	return simpleRange{Start: start, End: end}, line[start:end]
}

func isWordByte(b byte) bool {
	return b == '_' || b == '-' || b == '.' || b == '@' || b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z' || b >= '0' && b <= '9'
}
