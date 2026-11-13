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
		"StartupCPUWeight=200",
		"CPUWeight=500",
		"CPUQuota=50%",
		"CPUQuotaPeriodSec=20ms",
		"MemoryAccounting=yes",
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
		"",
	}, "\n")
	diagnostics := fragmentDiagnostics(NewCatalog(), text, "service")
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want none", diagnostics)
	}
}

func TestDiagnosticsRejectsResourceControlInTimer(t *testing.T) {
	diagnostics := fragmentDiagnostics(NewCatalog(), "[Timer]\nCPUWeight=500\n", "timer")
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
	diagnostics := fragmentDiagnostics(NewCatalog(), "[Service]\nRestart=\\\n  maybe\n", "service")
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

func TestDiagnosticsAllowsRepeatingAdditiveSettings(t *testing.T) {
	for _, line := range []string{
		"SystemCallFilter=@system-service", "SystemCallLog=@privileged", "SystemCallArchitectures=native",
		"RestrictAddressFamilies=AF_INET", "RestrictNamespaces=user", "RestrictFileSystems=ext4",
		"RestrictNetworkInterfaces=lo", "OpenFile=/a:b", "StandardInputText=a", "AllowedCPUs=1",
		"NUMAMask=0", "Delegate=cpu", "IOReadBandwidthMax=/dev/sda 1M",
	} {
		if diagnostics := fragmentDiagnostics(NewCatalog(), "[Service]\n"+line+"\n"+line+"\n", "service"); len(diagnostics) != 0 {
			t.Errorf("%s repeated: %#v, want no diagnostics", line, diagnostics)
		}
	}
	if diagnostics := Diagnostics(NewCatalog(), "[Socket]\nListenStream=80\nXAttrEntryPoint=user.a=1\nXAttrEntryPoint=user.b=2\n", "socket"); len(diagnostics) != 0 {
		t.Errorf("XAttrEntryPoint repeated: %#v, want no diagnostics", diagnostics)
	}
	if diagnostics := Diagnostics(NewCatalog(), "[Unit]\nWantsMountsFor=/a\nWantsMountsFor=/b\n", "service"); len(diagnostics) != 0 {
		t.Errorf("WantsMountsFor repeated: %#v, want no diagnostics", diagnostics)
	}
	// Settings where the later assignment wins are still reported.
	for _, line := range []string{"MemoryMax=1G", "PrivateMounts=yes", "ExitType=main"} {
		diagnostics := fragmentDiagnostics(NewCatalog(), "[Service]\n"+line+"\n"+line+"\n", "service")
		if len(diagnostics) != 1 || !strings.Contains(diagnostics[0].Message, "already set") {
			t.Errorf("%s repeated: %#v, want one duplicate warning", line, diagnostics)
		}
	}
}

func TestDiagnosticsAcceptsExtendedBooleanValues(t *testing.T) {
	for _, line := range []string{
		"PrivateTmp=disconnected", "ProtectSystem=on", "ProtectSystem=strict", "ProtectHome=tmpfs",
		"ProtectControlGroups=private", "ProtectHostname=private",
		// config_parse_tristate takes every boolean spelling.
		"MemoryKSM=true", "MemoryKSM=on", "SetLoginEnvironment=0", "PrivateMounts=t",
		"RemainAfterExit=y", "PrivateNetwork=F",
	} {
		if diagnostics := fragmentDiagnostics(NewCatalog(), "[Service]\n"+line+"\n", "service"); len(diagnostics) != 0 {
			t.Errorf("%s: %#v, want no diagnostics", line, diagnostics)
		}
	}
	for _, line := range []string{"MemoryKSM=auto", "PrivateTmp=connected", "ProtectSystem=maybe", "RemainAfterExit=x"} {
		diagnostics := fragmentDiagnostics(NewCatalog(), "[Service]\n"+line+"\n", "service")
		if len(diagnostics) != 1 || !strings.Contains(diagnostics[0].Message, "unexpected value") {
			t.Errorf("%s: %#v, want one unexpected value", line, diagnostics)
		}
	}
}

// fragmentDiagnostics checks an excerpt of a unit, leaving out the errors
// about settings that only a complete unit must have.
func fragmentDiagnostics(c *Catalog, text, unitType string) []Diagnostic {
	diagnostics := []Diagnostic{}
	for _, d := range Diagnostics(c, text, unitType) {
		if !strings.HasSuffix(d.Message, "systemd refuses to load the unit") {
			diagnostics = append(diagnostics, d)
		}
	}
	return diagnostics
}

func messages(diagnostics []Diagnostic) []string {
	out := []string{}
	for _, d := range diagnostics {
		out = append(out, d.Message)
	}
	return out
}

func TestDiagnosticsReportsTrailingComments(t *testing.T) {
	text := "[Service]\nExecStart=/bin/true # an argument\nEnvironment=A=1 # b\nType=simple # main\nRemainAfterExit=yes ; x\nTimeoutStartSec=5 # c\n"
	diagnostics := Diagnostics(NewCatalog(), text, "service")
	if len(diagnostics) != 3 {
		t.Fatalf("diagnostics = %q, want three trailing comments", messages(diagnostics))
	}
	for i, d := range diagnostics {
		if d.Range.Start.Line != i+3 || !strings.Contains(d.Message, "trailing comments") {
			t.Fatalf("diagnostic %d = %#v", i, d)
		}
	}
}

func TestDiagnosticsComparesSymbolicValuesCaseSensitively(t *testing.T) {
	text := "[Service]\nExecStart=/bin/true\nType=Simple\nRestart=Always\nRemainAfterExit=YES\nPrivateTmp=True\n"
	diagnostics := Diagnostics(NewCatalog(), text, "service")
	if len(diagnostics) != 2 ||
		diagnostics[0].Message != `unexpected value "Simple" for Type; values are case-sensitive, use "simple"` ||
		diagnostics[1].Message != `unexpected value "Always" for Restart; values are case-sensitive, use "always"` {
		t.Fatalf("diagnostics = %q", messages(diagnostics))
	}
}

func TestDiagnosticsReportsObsoleteSettings(t *testing.T) {
	text := "[Service]\nExecStart=/bin/true\nMemoryLimit=1G\nMemoryLimit=2G\nCPUAccounting=bogus\n"
	want := []string{
		"MemoryLimit= is no longer supported and systemd ignores it; use MemoryMax= instead",
		"MemoryLimit= is no longer supported and systemd ignores it; use MemoryMax= instead",
		"CPUAccounting= is no longer supported and systemd ignores it",
	}
	if got := messages(Diagnostics(NewCatalog(), text, "service")); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("diagnostics = %q, want %q", got, want)
	}
}

func TestDiagnosticsValidatesTypedValues(t *testing.T) {
	valid := []string{
		"Nice=19", "Nice=-20", "OOMScoreAdjust=-1000", "FileDescriptorStoreMax=0x10",
		"TimeoutStartSec=5", "TimeoutStartSec=5min 30s", "TimeoutStartSec=1h30min", "TimeoutStartSec=infinity",
		"TimeoutStartSec=1.5h", "TimeoutStartSec=.5", "TimeoutStartSec=5 6", "TimeoutStartSec=10µs", "RestartSec=100ms",
		"LimitNOFILE=1024:524288", "LimitNOFILE=infinity", "LimitCORE=1G", "LimitMEMLOCK=64K:1M", "LimitCPU=1h", "LimitNICE=-5",
		"MemoryMax=1G", "MemoryMax=infinity", "MemoryMax=12.5%", "MemoryMax=1G 500M", "MemoryMax=10.M", "MemoryHigh=500‰",
		"UMask=0022", "UMask=7777", "RuntimeDirectoryMode=750",
	}
	for _, line := range valid {
		if diagnostics := fragmentDiagnostics(NewCatalog(), "[Service]\n"+line+"\n", "service"); len(diagnostics) != 0 {
			t.Errorf("%s: %q, want no diagnostics", line, messages(diagnostics))
		}
	}
	invalid := []string{
		"Nice=100", "Nice=abc", "OOMScoreAdjust=5000", "FileDescriptorStoreMax=-1",
		"TimeoutStartSec=abc", "TimeoutStartSec=5x", "TimeoutStartSec=-1", "TimeoutStartSec=5.", "RestartSec=1.2.3",
		"LimitNOFILE=abc", "LimitNOFILE=1K", "LimitNOFILE=1:2:3", "LimitCORE=1g",
		"MemoryMax=lots", "MemoryMax=1g", "MemoryMax=150%", "MemoryMax=500M 1G",
		"UMask=999", "UMask=77777", "RuntimeDirectoryMode=rwx",
	}
	for _, line := range invalid {
		diagnostics := fragmentDiagnostics(NewCatalog(), "[Service]\n"+line+"\n", "service")
		if len(diagnostics) != 1 || !strings.Contains(diagnostics[0].Message, " expects ") {
			t.Errorf("%s: %q, want one value diagnostic", line, messages(diagnostics))
		}
	}
	if diagnostics := fragmentDiagnostics(NewCatalog(), "[Socket]\nBindIPv6Only=false\nPipeSize=64K\n", "socket"); len(diagnostics) != 0 {
		t.Errorf("socket values: %q, want no diagnostics", messages(diagnostics))
	}
}

func TestDiagnosticsReportsUnitsSystemdRefusesToLoad(t *testing.T) {
	tests := []struct {
		unit, text string
		line       int
		message    string
	}{
		{"service", "[Unit]\nDescription=x\n[Service]\nType=simple\n", 2, "Service has no ExecStart=, ExecStop=, or SuccessAction="},
		{"service", "[Service]\nType=notify\nExecStop=/bin/a\n", 0, "Service has no ExecStart=, which is only allowed for Type=oneshot services"},
		{"service", "[Service]\nExecStop=/bin/a\n", 0, "Service has no ExecStart= and no SuccessAction=, and does not set RemainAfterExit=yes"},
		{"service", "[Service]\nExecStart=/bin/a\nExecStart=/bin/b\n", 2, "Service has more than one ExecStart= command"},
		{"service", "[Service]\nExecStart=/bin/a ; /bin/b\n", 1, "Service has more than one ExecStart= command"},
		{"service", "[Service]\nType=oneshot\nExecStart=/bin/a\nRestart=always\n", 3, "Restart=always and Restart=on-success are not allowed"},
		{"service", "[Service]\nType=oneshot\nExecStart=/bin/a\nExitType=cgroup\n", 3, "ExitType=cgroup is not allowed"},
		{"service", "[Service]\nType=dbus\nExecStart=/bin/a\n", 1, "Type=dbus requires BusName="},
		{"socket", "[Socket]\nListenStream=80\nListenDatagram=\nAccept=yes\n", 0, "Socket unit has no Listen setting"},
		{"timer", "[Timer]\nPersistent=yes\n", 0, "Timer unit has no trigger setting"},
		{"path", "[Path]\nUnit=a.service\n", 0, "Path unit has no path setting"},
		{"mount", "[Mount]\nWhere=/a\n", 0, "Mount unit has no What= setting"},
	}
	for _, test := range tests {
		diagnostics := Diagnostics(NewCatalog(), test.text, test.unit)
		if len(diagnostics) != 1 || diagnostics[0].Severity != SeverityError || diagnostics[0].Range.Start.Line != test.line ||
			!strings.HasPrefix(diagnostics[0].Message, test.message) || !strings.HasSuffix(diagnostics[0].Message, "; systemd refuses to load the unit") {
			t.Errorf("%q: %#v, want %q on line %d", test.text, diagnostics, test.message, test.line)
		}
	}
	loadable := []struct{ unit, text string }{
		{"service", "[Service]\nExecStart=/bin/a\nExecStart=\nExecStart=/bin/b\n"},
		{"service", "[Service]\nExecStart=/bin/sh -c 'a ; b' \\; c\n"},
		{"service", "[Service]\nType=oneshot\nExecStart=/bin/a\nExecStart=/bin/b\nRestart=on-failure\n"},
		{"service", "[Service]\nType=oneshot\nRemainAfterExit=yes\nExecStop=/bin/a\n"},
		{"service", "[Unit]\nSuccessAction=poweroff\n[Service]\nType=oneshot\n"},
		{"service", "[Service]\nBusName=org.example\nExecStart=/bin/a\n"},
		{"service", "[Unit]\nDescription=still being written\n"},
		{"socket", "[Socket]\nListenFIFO=/run/a\n"},
		{"timer", "[Timer]\nOnClockChange=yes\n"},
		{"timer", "[Timer]\nOnUnitActiveSec=1h\n"},
		{"path", "[Path]\nPathExists=/a\n"},
		{"mount", "[Mount]\nWhat=/dev/sda\nWhere=/a\n"},
		{"target", "[Unit]\nDescription=x\n"},
		{"", "[Service]\nType=simple\n"},
	}
	for _, test := range loadable {
		if diagnostics := Diagnostics(NewCatalog(), test.text, test.unit); len(diagnostics) != 0 {
			t.Errorf("%q: %q, want no diagnostics", test.text, messages(diagnostics))
		}
	}
	if diagnostics := DropInDiagnostics(NewCatalog(), "[Service]\nRestart=always\n", "service"); len(diagnostics) != 0 {
		t.Errorf("drop-in: %q, want no diagnostics", messages(diagnostics))
	}
}
