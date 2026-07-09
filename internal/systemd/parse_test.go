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
