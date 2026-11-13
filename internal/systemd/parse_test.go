package systemd

import "testing"

func TestParseSystemdUnit(t *testing.T) {
	doc := Parse("[Unit]\nDescription=demo\n\n[Service]\nExecStart=/bin/true\n")
	if len(doc.Entries) != 6 {
		t.Fatalf("entry count = %d, want 6", len(doc.Entries))
	}
	if doc.Entries[0].Kind != EntrySection || doc.Entries[0].Section != "Unit" {
		t.Fatalf("first entry = %#v, want Unit section", doc.Entries[0])
	}
	if doc.Entries[1].Kind != EntryDirective || doc.Entries[1].Key != "Description" || doc.Entries[1].Value != "demo" {
		t.Fatalf("second entry = %#v, want Description directive", doc.Entries[1])
	}
}

func TestParseInvalidEntries(t *testing.T) {
	doc := Parse("[Service] trailing\nnot-a-directive\n=missing-key\n")
	for i, entry := range doc.Entries[:3] {
		if entry.Kind != EntryInvalid {
			t.Fatalf("entry %d kind = %v, want invalid", i, entry.Kind)
		}
	}
}

func TestParseJoinsContinuationLines(t *testing.T) {
	doc := Parse("[Service]\nExecStart=/usr/bin/foo \\\n    --option=bar \\\n# skipped\n    --verbose\nType=simple\n")
	kinds := []EntryKind{EntrySection, EntryDirective, EntryContinuation, EntryComment, EntryContinuation, EntryDirective, EntryBlank}
	if len(doc.Entries) != len(kinds) {
		t.Fatalf("entry count = %d, want %d", len(doc.Entries), len(kinds))
	}
	for i, want := range kinds {
		if doc.Entries[i].Kind != want || doc.Entries[i].Line != i {
			t.Fatalf("entry %d = %#v, want kind %v", i, doc.Entries[i], want)
		}
	}
	// As in systemd, each backslash becomes a space.
	command := doc.Entries[1]
	if command.Key != "ExecStart" || command.Value != "/usr/bin/foo  --option=bar  --verbose" {
		t.Fatalf("joined directive = %#v", command)
	}
	if command.Raw != "ExecStart=/usr/bin/foo \\" {
		t.Fatalf("raw = %q, want only the first line", command.Raw)
	}
}

func TestParseContinuationEdgeCases(t *testing.T) {
	// An escaped backslash does not continue the line.
	doc := Parse("ExecStart=/bin/echo \\\\\nType=simple")
	if doc.Entries[1].Kind != EntryDirective || doc.Entries[1].Key != "Type" {
		t.Fatalf("entry after escaped backslash = %#v", doc.Entries[1])
	}
	// CRLF line endings, and a continuation that runs into the end of the file.
	doc = Parse("Environment=A=1 \\\r\n  B=2 \\\r\n")
	if doc.Entries[0].Value != "A=1  B=2" || doc.Entries[1].Kind != EntryContinuation {
		t.Fatalf("CRLF continuation = %#v", doc.Entries)
	}
	// The key range stays on the first line when = only appears later.
	doc = Parse("Exec\\\nStart=/bin/true")
	if entry := doc.Entries[0]; entry.Kind != EntryDirective || entry.KeyEnd > len(entry.Raw) {
		t.Fatalf("split key = %#v", entry)
	}
}

func TestUTF16Length(t *testing.T) {
	for text, want := range map[string]int{"": 0, "Type": 4, "日本語": 3, "a😀b": 4} {
		if got := UTF16Length(text); got != want {
			t.Errorf("UTF16Length(%q) = %d, want %d", text, got, want)
		}
	}
}
