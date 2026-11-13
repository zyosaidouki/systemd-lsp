package lsp

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"log"
	"net/url"
	"path"
	"strings"

	"github.com/zyosaidouki/systemd-lsp/internal/systemd"
	"github.com/zyosaidouki/systemd-lsp/internal/version"
)

type Server struct {
	catalog        *systemd.Catalog
	logger         *log.Logger
	docs           map[string]string
	shutdown       bool
	exited         bool
	nextRequestID  int
	locale         string
	snippetSupport bool
}

func NewServer(catalog *systemd.Catalog, logger *log.Logger) *Server {
	return &Server{
		catalog: catalog,
		logger:  logger,
		docs:    map[string]string{},
		locale:  systemd.LocaleEnglish,
	}
}

type rpcMessage struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      jsontext.Value `json:"id,omitzero"`
	Method  string         `json:"method,omitempty"`
	Params  jsontext.Value `json:"params,omitzero"`
	Result  jsontext.Value `json:"result,omitzero"`
	Error   *rpcError      `json:"error,omitzero"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Responses always contain id (null when it could not be determined). A
// non-nil raw result must be retained even when it encodes null, [] or {}.
// v2's omitempty would omit those JSON values; omitzero only omits a nil Value.
type rpcResponse struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      jsontext.Value `json:"id"`
	Result  jsontext.Value `json:"result,omitzero"`
	Error   *rpcError      `json:"error,omitzero"`
}

func (s *Server) Handle(payload jsontext.Value) [][]byte {
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

func (s *Server) dispatch(method string, params jsontext.Value) (any, [][]byte, error) {
	switch method {
	case "initialize":
		s.configure(params)
		return s.initialize(), nil, nil
	case "initialized":
		return nil, nil, nil
	case "shutdown":
		s.shutdown = true
		return nil, nil, nil
	case "exit":
		s.exited = true
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
			Text         *string                `json:"text,omitzero"`
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

// ExitCode reports whether the client sent exit, and the status the process
// must then end with: 0 after a shutdown request, 1 otherwise.
func (s *Server) ExitCode() (code int, exited bool) {
	if !s.shutdown {
		code = 1
	}
	return code, s.exited
}

func (s *Server) configure(params jsontext.Value) {
	var p struct {
		Locale       string `json:"locale,omitempty"`
		CatalogPath  string `json:"catalogPath,omitempty"`
		Capabilities struct {
			TextDocument struct {
				Completion struct {
					CompletionItem struct {
						SnippetSupport bool `json:"snippetSupport,omitzero"`
					} `json:"completionItem,omitempty"`
				} `json:"completion,omitempty"`
			} `json:"textDocument,omitempty"`
		} `json:"capabilities,omitempty"`
		InitializationOptions struct {
			Locale      string `json:"locale,omitempty"`
			Language    string `json:"language,omitempty"`
			CatalogPath string `json:"catalogPath,omitempty"`
		} `json:"initializationOptions,omitempty"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return
	}
	s.snippetSupport = p.Capabilities.TextDocument.Completion.CompletionItem.SnippetSupport
	locale := p.InitializationOptions.Locale
	if locale == "" {
		locale = p.InitializationOptions.Language
	}
	if locale == "" {
		locale = p.Locale
	}
	s.locale = systemd.NormalizeLocale(locale)

	catalogPath := p.InitializationOptions.CatalogPath
	if catalogPath == "" {
		catalogPath = p.CatalogPath
	}
	if catalogPath == "" {
		return
	}
	file, err := systemd.LoadCatalogFile(catalogPath)
	if err != nil {
		if s.logger != nil {
			s.logger.Printf("failed to load generated catalog %q: %v", catalogPath, err)
		}
		return
	}
	s.catalog.MergeCatalogFile(file)
	if s.logger != nil {
		s.logger.Printf("loaded generated catalog %q with %d directives", catalogPath, len(file.Directives))
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
	return path.Ext(documentPath(uri)) == ".service" && strings.TrimSpace(text) == ""
}

func (s *Server) applyEditRequest(uri, newText string) []byte {
	s.nextRequestID++
	id := jsontext.Value(fmt.Sprintf("%d", s.nextRequestID))
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
		raw = jsontext.Value("null")
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
			"version": version.Value,
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
				"triggerCharacters": []string{"[", "=", " ", "~"},
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
	if !s.catalog.SupportsUnitType(unitType) {
		// Editors also use the systemd filetype for formats such as .network.
		return [][]byte{notification("textDocument/publishDiagnostics", map[string]any{
			"uri":         uri,
			"diagnostics": []Diagnostic{},
		})}
	}
	diagnose := systemd.Diagnostics
	if isDropInURI(uri) {
		diagnose = systemd.DropInDiagnostics
	}
	diagnostics := diagnose(s.catalog, text, unitType)
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
	switch lineKind(doc, p.Position.Line) {
	case systemd.EntryComment, systemd.EntryContinuation:
		// Comments take no directives, and a continuation line is part of the
		// value of the directive above it.
		return []CompletionItem{}
	}
	if !s.catalog.SupportsUnitType(unitType) {
		return []CompletionItem{}
	}

	if strings.HasPrefix(trimmed, "[") {
		// Replace the header typed so far. [ is not a word character, so
		// without an explicit range clients would keep it and insert a second
		// one. The range ends at the cursor because some clients ignore
		// anything beyond it; a bracket the editor already paired is kept.
		headerRange := Range{
			Start: Position{Line: p.Position.Line, Character: systemd.UTF16Length(before[:len(before)-len(trimmed)])},
			End:   Position{Line: p.Position.Line, Character: systemd.UTF16Length(before)},
		}
		closed := strings.HasPrefix(line[len(before):], "]")
		items := make([]CompletionItem, 0, len(s.catalog.SectionsFor(unitType)))
		for _, section := range s.catalog.SectionsFor(unitType) {
			insertText, insertTextFormat := s.completionInsertText("[" + section + "]")
			kind := CompletionItemKindKeyword
			if s.snippetSupport {
				kind = CompletionItemKindSnippet
			}
			items = append(items, CompletionItem{
				Label:            "[" + section + "]",
				Kind:             kind,
				Detail:           s.localizedSectionDetail(),
				Documentation:    s.catalog.SectionDocumentationFor(section, s.locale),
				InsertText:       insertText,
				InsertTextFormat: insertTextFormat,
				TextEdit:         &TextEdit{Range: headerRange, NewText: insertText},
			})
			if closed {
				// Plain text: clients complete snippet items from the label.
				item := &items[len(items)-1]
				item.InsertText, item.InsertTextFormat = "["+section, InsertTextFormatPlainText
				item.TextEdit.NewText = item.InsertText
			}
		}
		return items
	}

	if key, value, hasEquals := strings.Cut(before, "="); hasEquals {
		section := sectionAt(s.catalog, doc, p.Position.Line)
		return s.valueCompletions(section, strings.TrimSpace(key), value, line, p.Position)
	}

	section := sectionAt(s.catalog, doc, p.Position.Line)
	if section == "" {
		return []CompletionItem{}
	}
	directives := s.catalog.DirectivesFor(section)
	items := make([]CompletionItem, 0, len(directives))
	for _, directive := range directives {
		if directive.Obsolete() {
			continue
		}
		insertText, insertTextFormat := s.completionInsertText(directive.Name + "=")
		if s.snippetSupport && directive.Snippet != "" {
			insertText = directive.Snippet
		}
		items = append(items, CompletionItem{
			Label:            directive.Name,
			Kind:             CompletionItemKindField,
			Detail:           s.localizedDirectiveDetail(section),
			Documentation:    systemd.DirectiveDocumentationFor(section, directive, s.locale),
			InsertText:       insertText,
			InsertTextFormat: insertTextFormat,
		})
	}
	return items
}

func (s *Server) completionInsertText(text string) (string, int) {
	if s.snippetSupport {
		return text + "$0", InsertTextFormatSnippet
	}
	return text, InsertTextFormatPlainText
}

func (s *Server) hover(p TextDocumentPositionParams) any {
	doc := s.docs[p.TextDocument.URI]
	line := lineAt(doc, p.Position.Line)
	wordRange, word := wordAt(line, len(prefixAt(line, p.Position.Character)))
	if word == "" || lineKind(doc, p.Position.Line) == systemd.EntryContinuation || !s.catalog.SupportsUnitType(unitTypeFromURI(p.TextDocument.URI)) {
		return nil
	}
	hoverRange := &Range{
		Start: Position{Line: p.Position.Line, Character: systemd.UTF16Length(line[:wordRange.Start])},
		End:   Position{Line: p.Position.Line, Character: systemd.UTF16Length(line[:wordRange.End])},
	}
	if strings.HasPrefix(strings.TrimSpace(line), "[") && strings.HasSuffix(strings.TrimSpace(line), "]") {
		section := strings.Trim(strings.TrimSpace(line), "[]")
		if doc := s.catalog.SectionDocumentationFor(section, s.locale); doc != "" {
			return Hover{
				Contents: MarkupContent{Kind: "markdown", Value: fmt.Sprintf("**[%s]**\n\n%s", section, doc)},
				Range:    hoverRange,
			}
		}
	}
	section := sectionAt(s.catalog, doc, p.Position.Line)
	if directive, ok := s.catalog.Directive(section, word); ok {
		return Hover{
			Contents: MarkupContent{Kind: "markdown", Value: fmt.Sprintf("**%s=**\n\n%s", directive.Name, systemd.DirectiveDocumentationFor(section, directive, s.locale))},
			Range:    hoverRange,
		}
	}
	return nil
}

func (s *Server) localizedSectionDetail() string {
	if s.locale == systemd.LocaleJapanese {
		return "systemd セクション"
	}
	return "systemd section"
}

func (s *Server) localizedDirectiveDetail(section string) string {
	if s.locale == systemd.LocaleJapanese {
		return "[" + section + "] ディレクティブ"
	}
	return "[" + section + "] directive"
}

func (s *Server) localizedValueDetail(key string) string {
	if s.locale == systemd.LocaleJapanese {
		return key + " の値"
	}
	return key + " value"
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
					End:   Position{Line: entry.Line, Character: systemd.UTF16Length(entry.Raw)},
				},
				SelectionRange: Range{
					Start: Position{Line: entry.Line, Character: systemd.UTF16Length(entry.Raw[:entry.KeyStart])},
					End:   Position{Line: entry.Line, Character: systemd.UTF16Length(entry.Raw[:entry.KeyEnd])},
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
					End:   Position{Line: entry.Line, Character: systemd.UTF16Length(entry.Raw)},
				},
				SelectionRange: Range{
					Start: Position{Line: entry.Line, Character: systemd.UTF16Length(entry.Raw[:entry.KeyStart])},
					End:   Position{Line: entry.Line, Character: systemd.UTF16Length(entry.Raw[:entry.KeyEnd])},
				},
			})
		}
	}
	return symbols
}

func encodeResponse(id jsontext.Value, result any, rpcErr *rpcError) []byte {
	response := rpcResponse{JSONRPC: "2.0", ID: id, Error: rpcErr}
	if rpcErr == nil {
		if result == nil {
			response.Result = jsontext.Value("null")
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
		raw = jsontext.Value("null")
	}
	payload, _ := json.Marshal(rpcMessage{
		JSONRPC: "2.0",
		Method:  method,
		Params:  raw,
	})
	return payload
}

// documentPath returns the path a document is saved under. URIs are
// percent-encoded, and systemctl edit opens a temporary file that is named
// .#<name><random suffix> and sits next to the real one.
func documentPath(uri string) string {
	name, err := url.PathUnescape(uri)
	if err != nil {
		name = uri
	}
	dir, base := path.Split(name)
	if rest, ok := strings.CutPrefix(base, ".#"); ok {
		base = trimTemporarySuffix(rest)
	}
	return dir + base
}

// trimTemporarySuffix removes the random characters systemd appends to the
// extension: six from mkostemp(), or sixteen hexadecimal digits.
func trimTemporarySuffix(name string) string {
	dot := strings.LastIndex(name, ".")
	extension := name[dot+1:]
	for _, known := range append(systemd.UnitTypes(), "conf") {
		suffix, ok := strings.CutPrefix(extension, known)
		if !ok || len(suffix) != 6 && len(suffix) != 16 {
			continue
		}
		if strings.IndexFunc(suffix, func(r rune) bool {
			return !(r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z')
		}) < 0 {
			return name[:dot+1] + known
		}
	}
	return name
}

func isDropInURI(uri string) bool {
	name := documentPath(uri)
	return strings.HasSuffix(path.Base(name), ".conf") && strings.HasSuffix(path.Base(path.Dir(name)), ".d")
}

func unitTypeFromURI(uri string) string {
	name := documentPath(uri)
	base := path.Base(name)
	if isDropInURI(uri) {
		base = strings.TrimSuffix(path.Base(path.Dir(name)), ".d")
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
	units := 0
	for offset, r := range line {
		width := 1
		if r > 0xffff {
			width = 2
		}
		if units+width > character {
			return line[:offset]
		}
		units += width
	}
	return line
}

func lineKind(text string, lineNo int) systemd.EntryKind {
	entries := systemd.Parse(text).Entries
	if lineNo < 0 || lineNo >= len(entries) {
		return systemd.EntryUnknown
	}
	return entries[lineNo].Kind
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
