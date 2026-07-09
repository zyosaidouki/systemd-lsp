package systemd

import "strings"

type EntryKind int

const (
	EntryUnknown EntryKind = iota
	EntrySection
	EntryDirective
	EntryComment
	EntryBlank
	EntryInvalid
)

type Entry struct {
	Kind     EntryKind
	Line     int
	Raw      string
	Section  string
	Key      string
	Value    string
	KeyStart int
	KeyEnd   int
	Message  string
}

type Document struct {
	Entries []Entry
}

func Parse(text string) Document {
	lines := strings.Split(text, "\n")
	entries := make([]Entry, 0, len(lines))
	for i, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		entry := Entry{Line: i, Raw: raw}
		switch {
		case trimmed == "":
			entry.Kind = EntryBlank
		case strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";"):
			entry.Kind = EntryComment
		case strings.HasPrefix(trimmed, "["):
			entry = parseSection(i, raw)
		default:
			entry = parseDirective(i, raw)
		}
		entries = append(entries, entry)
	}
	return Document{Entries: entries}
}

func parseSection(line int, raw string) Entry {
	start := strings.Index(raw, "[")
	end := strings.Index(raw[start:], "]")
	if start < 0 || end < 0 {
		return Entry{Kind: EntryInvalid, Line: line, Raw: raw, Message: "malformed section header"}
	}
	end += start
	name := strings.TrimSpace(raw[start+1 : end])
	if name == "" || strings.TrimSpace(raw[end+1:]) != "" {
		return Entry{Kind: EntryInvalid, Line: line, Raw: raw, Message: "malformed section header"}
	}
	return Entry{
		Kind:     EntrySection,
		Line:     line,
		Raw:      raw,
		Section:  name,
		KeyStart: start + 1,
		KeyEnd:   end,
	}
}

func parseDirective(line int, raw string) Entry {
	idx := strings.Index(raw, "=")
	if idx < 0 {
		return Entry{Kind: EntryInvalid, Line: line, Raw: raw, Message: "expected key=value directive"}
	}
	keyStart := firstNonSpace(raw)
	keyEnd := idx
	for keyEnd > keyStart && (raw[keyEnd-1] == ' ' || raw[keyEnd-1] == '\t') {
		keyEnd--
	}
	key := raw[keyStart:keyEnd]
	if key == "" {
		return Entry{Kind: EntryInvalid, Line: line, Raw: raw, Message: "missing directive name"}
	}
	return Entry{
		Kind:     EntryDirective,
		Line:     line,
		Raw:      raw,
		Key:      key,
		Value:    strings.TrimSpace(raw[idx+1:]),
		KeyStart: keyStart,
		KeyEnd:   keyEnd,
	}
}

func firstNonSpace(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] != ' ' && s[i] != '\t' {
			return i
		}
	}
	return len(s)
}
