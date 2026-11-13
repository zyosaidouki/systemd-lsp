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

func TestDiagnosticsAcceptsResourceControlDirectives(t *testing.T) {
	text := strings.Join([]string{
		"[Service]",
		"Slice=workload.slice",
		"Delegate=cpu io memory pids",
		"DisableControllers=cpu io",
		"AllowedCPUs=0-3",
		"AllowedMemoryNodes=0",
		"CPUAccounting=yes",
		"StartupCPUWeight=200",
		"CPUWeight=500",
		"CPUQuota=50%",
		"CPUQuotaPeriodSec=20ms",
		"MemoryAccounting=yes",
		"DefaultMemoryMin=128M",
		"DefaultMemoryLow=256M",
		"MemoryMin=64M",
		"MemoryLow=128M",
		"MemoryHigh=512M",
		"MemoryMax=1G",
		"MemorySwapMax=2G",
		"TasksAccounting=yes",
		"TasksMax=512",
		"IOAccounting=yes",
		"StartupIOWeight=200",
		"IOWeight=500",
		"IODeviceWeight=/dev/sda 1000",
		"IOReadBandwidthMax=/dev/sda 5M",
		"IOWriteBandwidthMax=/dev/sda 5M",
		"IOReadIOPSMax=/dev/sda 1K",
		"IOWriteIOPSMax=/dev/sda 1K",
		"IODeviceLatencyTargetSec=/dev/sda 25ms",
		"IPAccounting=yes",
		"IPAddressAllow=localhost",
		"IPAddressDeny=any",
		"IPIngressFilterPath=/sys/fs/bpf/ingress",
		"IPEgressFilterPath=/sys/fs/bpf/egress",
		"BPFProgram=egress:/sys/fs/bpf/egress-hook",
		"SocketBindAllow=ipv6:10000-65535",
		"SocketBindDeny=any",
		"DeviceAllow=/dev/null rw",
		"DevicePolicy=closed",
		"ManagedOOMSwap=kill",
		"ManagedOOMMemoryPressure=auto",
		"ManagedOOMMemoryPressureLimit=50%",
		"ManagedOOMPreference=avoid",
		"CPUShares=1024",
		"StartupCPUShares=2048",
		"MemoryLimit=1G",
		"BlockIOAccounting=yes",
		"StartupBlockIOWeight=500",
		"BlockIOWeight=500",
		"BlockIODeviceWeight=/dev/sda 500",
		"BlockIOReadBandwidth=/dev/sda 5M",
		"BlockIOWriteBandwidth=/dev/sda 5M",
		"",
	}, "\n")
	diagnostics := Diagnostics(NewCatalog(), text, "service")
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want none", diagnostics)
	}
}

func TestDiagnosticsRejectsResourceControlInTimer(t *testing.T) {
	diagnostics := Diagnostics(NewCatalog(), "[Timer]\nCPUWeight=500\n", "timer")
	if len(diagnostics) != 1 {
		t.Fatalf("diagnostic count = %d, want 1: %#v", len(diagnostics), diagnostics)
	}
	if !strings.Contains(diagnostics[0].Message, "unknown directive CPUWeight in [Timer]") {
		t.Fatalf("message = %q, want unknown directive warning", diagnostics[0].Message)
	}
}

func TestDiagnosticsIgnoresExtensionSectionsAndDirectives(t *testing.T) {
	text := strings.Join([]string{
		"[Unit]",
		"X-Owner=platform",
		"",
		"[X-Company]",
		"Anything=goes",
		"",
	}, "\n")
	diagnostics := Diagnostics(NewCatalog(), text, "service")
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want none", diagnostics)
	}
}

func TestDiagnosticsWarnsWhenForkingServiceOmitsPIDFile(t *testing.T) {
	text := strings.Join([]string{
		"[Service]",
		"Type=forking",
		"ExecStart=/usr/bin/exampled",
		"",
	}, "\n")
	diagnostics := Diagnostics(NewCatalog(), text, "service")
	if len(diagnostics) != 1 {
		t.Fatalf("diagnostic count = %d, want 1: %#v", len(diagnostics), diagnostics)
	}
	if !strings.Contains(diagnostics[0].Message, "Type=forking should specify PIDFile=") {
		t.Fatalf("message = %q, want PIDFile warning", diagnostics[0].Message)
	}
	if diagnostics[0].Range.Start.Line != 1 {
		t.Fatalf("diagnostic line = %d, want Type line", diagnostics[0].Range.Start.Line)
	}
}

func TestDiagnosticsAcceptsForkingServiceWithPIDFile(t *testing.T) {
	text := strings.Join([]string{
		"[Service]",
		"Type=forking",
		"PIDFile=/run/exampled.pid",
		"ExecStart=/usr/bin/exampled",
		"",
	}, "\n")
	diagnostics := Diagnostics(NewCatalog(), text, "service")
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want none", diagnostics)
	}
}

func TestDiagnosticsWarnsWhenForkingServiceHasEmptyPIDFile(t *testing.T) {
	text := strings.Join([]string{
		"[Service]",
		"Type=forking",
		"PIDFile=",
		"ExecStart=/usr/bin/exampled",
		"",
	}, "\n")
	diagnostics := Diagnostics(NewCatalog(), text, "service")
	if len(diagnostics) != 1 {
		t.Fatalf("diagnostic count = %d, want 1: %#v", len(diagnostics), diagnostics)
	}
	if !strings.Contains(diagnostics[0].Message, "Type=forking should specify PIDFile=") {
		t.Fatalf("message = %q, want PIDFile warning", diagnostics[0].Message)
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

func TestDiagnosticsAcceptsContinuationLines(t *testing.T) {
	text := "[Service]\nExecStart=/usr/bin/foo \\\n    --option=bar \\\n    --verbose\nEnvironment=A=1 \\\n  B=2\n"
	if diagnostics := Diagnostics(NewCatalog(), text, "service"); len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want none", diagnostics)
	}
	// The joined value is what gets validated.
	diagnostics := Diagnostics(NewCatalog(), "[Service]\nRestart=\\\n  maybe\n", "service")
	if len(diagnostics) != 1 || !strings.Contains(diagnostics[0].Message, `unexpected value "maybe"`) {
		t.Fatalf("diagnostics = %#v, want one unexpected value", diagnostics)
	}
}

func TestDiagnosticsUsesLastTypeForForkingCheck(t *testing.T) {
	diagnostics := Diagnostics(NewCatalog(), "[Service]\nType=forking\nType=simple\nExecStart=/bin/true\n", "service")
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Message, "PIDFile=") {
			t.Fatalf("overridden Type=forking still reported: %#v", diagnostic)
		}
	}
	diagnostics = Diagnostics(NewCatalog(), "[Service]\nPIDFile=/run/demo.pid\nType=forking\nPIDFile=\n", "service")
	if !hasMessageContaining(diagnosticMessages(diagnostics), "Type=forking should specify PIDFile=") {
		t.Fatalf("diagnostics = %#v, want warning after PIDFile= was reset", diagnostics)
	}
}

func TestDropInDiagnosticsOnlyHintsAtMissingPIDFile(t *testing.T) {
	diagnostics := DropInDiagnostics(NewCatalog(), "[Service]\nType=forking\n", "service")
	if len(diagnostics) != 1 || diagnostics[0].Severity != SeverityHint || !strings.Contains(diagnostics[0].Message, "main unit file") {
		t.Fatalf("diagnostics = %#v, want one hint", diagnostics)
	}
}

func TestDiagnosticRangesCountUTF16Units(t *testing.T) {
	diagnostics := Diagnostics(NewCatalog(), "[Unit]\n日本語😀だけの行\n", "service")
	if len(diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v, want one", diagnostics)
	}
	if got := diagnostics[0].Range.End.Character; got != 9 {
		t.Fatalf("end character = %d, want 9 UTF-16 units", got)
	}
}

func diagnosticMessages(diagnostics []Diagnostic) []string {
	messages := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		messages = append(messages, diagnostic.Message)
	}
	return messages
}
