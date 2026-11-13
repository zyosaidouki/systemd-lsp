package lsp

import (
	"strings"

	"github.com/zyosaidouki/systemd-lsp/internal/systemd"
)

func (s *Server) valueCompletions(section, key, value, line string, position Position) []CompletionItem {
	directive, ok := s.catalog.Directive(section, key)
	if !ok || len(directive.Values) == 0 {
		return []CompletionItem{}
	}
	before := prefixAt(line, position.Character)
	valueStart := len(before) - len(value)
	list := directive.ValueKind == "enum-list" || directive.ValueKind == "resource-list"
	tokenStart := 0
	if list {
		tokenStart = strings.LastIndexAny(value, " \t") + 1
	}
	for tokenStart < len(value) && (value[tokenStart] == ' ' || value[tokenStart] == '\t') {
		tokenStart++
	}
	prior := strings.TrimSpace(value[:tokenStart])
	inverted := directive.ValueKind == "resource-list" && strings.HasPrefix(strings.TrimSpace(value), "~")
	if inverted && tokenStart < len(value) && value[tokenStart] == '~' {
		tokenStart++
	}
	if tokenStart < len(value) && (value[tokenStart] == '\'' || value[tokenStart] == '"') {
		tokenStart++
	}
	prefix := value[tokenStart:]
	if strings.ContainsAny(prefix, "\\\"'") {
		return []CompletionItem{}
	}
	start := valueStart + tokenStart
	end := len(before)
	for end < len(line) && !strings.ContainsRune(" \t\r\n\"'", rune(line[end])) {
		end++
	}
	used := map[string]bool{}
	if list {
		for _, word := range strings.Fields(strings.TrimPrefix(prior, "~")) {
			used[strings.Trim(word, "\"'")] = true
		}
	}
	items := make([]CompletionItem, 0, len(directive.Values))
	for _, candidate := range directive.Values {
		if used[candidate] || !strings.HasPrefix(strings.ToLower(candidate), strings.ToLower(prefix)) {
			continue
		}
		if directive.ValueKind == "resource-list" && (inverted || prior != "") && candidate != "extensions" && candidate != "credentials" {
			continue
		}
		items = append(items, CompletionItem{
			Label: candidate, Kind: CompletionItemKindValue,
			Detail: s.localizedValueDetail(key), InsertText: candidate,
			Documentation: systemd.DirectiveDocumentationFor(section, directive, s.locale),
			TextEdit: &TextEdit{
				Range:   Range{Start: Position{Line: position.Line, Character: systemd.UTF16Length(line[:start])}, End: Position{Line: position.Line, Character: systemd.UTF16Length(line[:end])}},
				NewText: candidate,
			},
		})
	}
	return items
}
