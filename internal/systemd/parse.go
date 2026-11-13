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
	// EntryContinuation is a line that belongs to the directive above it,
	// because the preceding line ended in a backslash.
	EntryContinuation
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
	// Every line yields exactly one entry, so Entries[n] describes line n.
	for i := 0; i < len(lines); i++ {
		raw := lines[i]
		trimmed := strings.TrimSpace(raw)
		entry := Entry{Line: i, Raw: raw}
		switch {
		case trimmed == "":
			entry.Kind = EntryBlank
		case isComment(raw):
			entry.Kind = EntryComment
		case strings.HasPrefix(trimmed, "["):
			entry = parseSection(i, raw)
		default:
			logical, last := joinContinuations(lines, i)
			entry = parseDirective(i, logical)
			entry.Raw = raw
			// Offsets must stay on the first line even if = comes later.
			width := len(strings.TrimSuffix(raw, "\r"))
			entry.KeyStart = min(entry.KeyStart, width)
			entry.KeyEnd = min(entry.KeyEnd, width)
			entries = append(entries, entry)
			for i++; i <= last; i++ {
				kind := EntryContinuation
				if isComment(lines[i]) {
					kind = EntryComment
				}
				entries = append(entries, Entry{Kind: kind, Line: i, Raw: lines[i]})
			}
			i = last
			continue
		}
		entries = append(entries, entry)
	}
	return Document{Entries: entries}
}

func isComment(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	return strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";")
}

// joinContinuations follows systemd's config_parse(): a trailing unescaped
// backslash becomes a space and the next line is appended without its leading
// whitespace. Comment lines in between are skipped. It returns the joined line
// and the index of the last line it consumed.
func joinContinuations(lines []string, first int) (string, int) {
	logical := strings.TrimSuffix(lines[first], "\r")
	last := first
	for endsWithBackslash(logical) {
		next := last + 1
		for next < len(lines) && isComment(lines[next]) {
			next++
		}
		logical = logical[:len(logical)-1] + " "
		if next >= len(lines) {
			break
		}
		logical += strings.TrimLeft(strings.TrimSuffix(lines[next], "\r"), " \t")
		last = next
	}
	return logical, last
}

func endsWithBackslash(line string) bool {
	escaped := false
	for i := 0; i < len(line); i++ {
		escaped = !escaped && line[i] == '\\'
	}
	return escaped
}

// UTF16Length returns the length of text in UTF-16 code units, which is how
// LSP counts character offsets.
func UTF16Length(text string) int {
	n := 0
	for _, r := range text {
		n++
		if r > 0xffff {
			n++
		}
	}
	return n
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
