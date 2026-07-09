package systemd

import (
	"strings"
	"testing"
)

func TestDiagnosticsValidService(t *testing.T) {
	text := strings.Join([]string{
		"[Unit]",
		"Description=demo",
		"After=network-online.target",
		"",
		"[Service]",
		"Type=simple",
		"ExecStart=/bin/true",
		"Restart=on-failure",
		"",
		"[Install]",
		"WantedBy=multi-user.target",
		"",
	}, "\n")
	diagnostics := Diagnostics(NewCatalog(), text, "service")
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want none", diagnostics)
	}
}

func TestDiagnosticsFindsCommonProblems(t *testing.T) {
	text := strings.Join([]string{
		"Description=outside",
		"[Servce]",
		"ExecStrt=/bin/true",
		"[Service]",
		"Type=simple",
		"Type=never",
		"Restart=maybe",
		"",
	}, "\n")
	diagnostics := Diagnostics(NewCatalog(), text, "service")
	messages := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		messages = append(messages, diagnostic.Message)
	}
	wantSubstrings := []string{
		"directive must be inside a section",
		"unknown systemd section [Servce]",
		"already set",
		"unexpected value \"never\"",
		"unexpected value \"maybe\"",
	}
	for _, want := range wantSubstrings {
		if !hasMessageContaining(messages, want) {
			t.Fatalf("messages = %#v, missing %q", messages, want)
		}
	}
}

func TestDiagnosticsWarnsForWrongSectionType(t *testing.T) {
	diagnostics := Diagnostics(NewCatalog(), "[Timer]\nOnCalendar=daily\n", "service")
	if len(diagnostics) != 1 {
		t.Fatalf("diagnostic count = %d, want 1: %#v", len(diagnostics), diagnostics)
	}
	if !strings.Contains(diagnostics[0].Message, "not normally used") {
		t.Fatalf("message = %q, want section type warning", diagnostics[0].Message)
	}
}

func hasMessageContaining(messages []string, needle string) bool {
	for _, message := range messages {
		if strings.Contains(message, needle) {
			return true
		}
	}
	return false
}
