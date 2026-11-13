package systemd

import (
	"bytes"
	"strings"
	"testing"
)

func TestLatestFeatureCatalog(t *testing.T) {
	c := NewCatalog()
	for _, tc := range []struct{ section, name, value string }{
		{"Service", "PrivateUsers", "full"},
		{"Service", "CPUSetPartition", "isolated"},
		{"Service", "CPUPressureWatch", "skip"},
		{"Service", "IOPressureWatch", "auto"},
		{"Service", "FileDescriptorStorePreserve", "on-success"},
		{"Service", "RefreshOnReload", "credentials"},
		{"Service", "NUMAPolicy", "weighted-interleave"},
		{"Service", "SecureBits", "exec-deny-interactive-locked"},
		{"Slice", "ActivatingConcurrencyMax", "infinity"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, ok := c.Directive(tc.section, tc.name)
			if !ok || !contains(d.Values, tc.value) {
				t.Fatalf("missing %s=%s: %#v", tc.name, tc.value, d)
			}
			if d.Doc == "" || d.DocJapanese == "" || d.ManPage == "" {
				t.Fatalf("missing feature documentation: %#v", d)
			}
		})
	}
	if d, _ := c.Directive("Service", "SecureBits"); d.ValueKind == "duration" {
		t.Fatal("securebits are not a duration")
	}
}

func TestVersionedCatalogReplacesNewerSnapshot(t *testing.T) {
	c := NewCatalog()
	c.MergeCatalogFile(CatalogFile{Version: "v258.11", Directives: []CatalogDirective{
		{Section: "Service", Name: "PrivateUsers", Parser: "config_parse_private_users"},
		{Section: "Service", Name: "FileDescriptorStorePreserve", Parser: "config_parse_exec_preserve_mode"},
		{Section: "Service", Name: "NUMAPolicy", Parser: "config_parse_numa_policy"},
		{Section: "Service", Name: "SecureBits", Parser: "config_parse_exec_secure_bits"},
	}})
	for _, name := range []string{"CPUSetPartition", "RestartRandomizedDelaySec", "ExecStart"} {
		if _, ok := c.Directive("Service", name); ok {
			t.Fatalf("snapshot leaked %s from embedded catalog", name)
		}
	}
	if !contains(c.ValuesFor("Service", "PrivateUsers"), "full") {
		t.Fatal("v258 must support full")
	}
	for name, value := range map[string]string{"FileDescriptorStorePreserve": "on-success", "NUMAPolicy": "preferred-many", "SecureBits": "exec-restrict-file"} {
		if contains(c.ValuesFor("Service", name), value) {
			t.Fatalf("v258 leaked %s=%s", name, value)
		}
	}
	diagnostics := Diagnostics(c, "[Service]\nRestartRandomizedDelaySec=5s\n", "service")
	if len(diagnostics) != 1 || !strings.Contains(diagnostics[0].Message, "catalog v258.11") {
		t.Fatalf("missing version context: %#v", diagnostics)
	}
}

func TestFeatureValueVersions(t *testing.T) {
	for _, tc := range []struct {
		version, name, value string
		want                 bool
	}{
		{"v256", "PrivateUsers", "self", false},
		{"v257", "PrivateUsers", "self", true},
		{"v257", "PrivateUsers", "full", false},
		{"v258", "PrivateUsers", "full", true},
		{"v259", "PrivateUsers", "managed", false},
		{"v260", "PrivateUsers", "managed", true},
		{"v260", "FileDescriptorStorePreserve", "on-success", false},
		{"v261.3", "FileDescriptorStorePreserve", "on-success", true},
		{"v261", "NUMAPolicy", "weighted-interleave", false},
		{"v262-rc2", "NUMAPolicy", "weighted-interleave", true},
	} {
		t.Run(tc.version+"/"+tc.name+"/"+tc.value, func(t *testing.T) {
			file := EnrichCatalogFeatures(CatalogFile{Version: tc.version, Directives: []CatalogDirective{{Section: "Service", Name: tc.name}}})
			if got := contains(file.Directives[0].Values, tc.value); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestUnversionedCatalogOverlayKeepsOtherDirectives(t *testing.T) {
	c := NewCatalog()
	c.MergeCatalogFile(CatalogFile{Directives: []CatalogDirective{{Section: "Service", Name: "PrivateUsers", Values: []string{"yes", "no"}, Doc: "Custom documentation."}}})
	if _, ok := c.Directive("Service", "ExecStart"); !ok {
		t.Fatal("overlay removed other directives")
	}
	d, _ := c.Directive("Service", "PrivateUsers")
	if len(d.Values) != 2 || d.Doc != "Custom documentation." {
		t.Fatalf("explicit overlay not applied: %#v", d)
	}
}

func TestFeatureCatalogRoundTrip(t *testing.T) {
	file := EnrichCatalogFeatures(CatalogFile{Version: "v261", Directives: []CatalogDirective{{Section: "Unit", Name: "ConditionFraction"}}})
	var buf bytes.Buffer
	if err := EncodeCatalogFile(&buf, file); err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeCatalogFile(&buf)
	if err != nil {
		t.Fatal(err)
	}
	c := NewCatalog()
	c.MergeCatalogFile(decoded)
	d, ok := c.Directive("Unit", "ConditionFraction")
	if !ok || d.Since != 261 || !strings.Contains(d.Snippet, "${2:10}") {
		t.Fatalf("lost feature metadata: %#v", d)
	}
	for _, locale := range []string{"en", "ja"} {
		doc := DirectiveDocumentationFor("Unit", d, locale)
		if !strings.Contains(doc, "v261") || !strings.Contains(doc, "0%") || !strings.Contains(doc, "100%") {
			t.Fatalf("incomplete hover: %s", doc)
		}
	}
}

func TestNewFeatureDiagnostics(t *testing.T) {
	c := NewCatalog()
	for _, tc := range []struct{ name, text, unit, message string }{
		{"private users full", "[Service]\nPrivateUsers=full", "service", ""},
		{"boolean abbreviations", "[Service]\nPrivateUsers=Y\nCPUPressureWatch=n\nIOPressureWatch=t\nRefreshOnReload=f", "service", ""},
		{"private users typo", "[Service]\nPrivateUsers=ful", "service", "unexpected value"},
		{"resource controls", "[Service]\nCPUSetPartition=isolated\nAllowedCPUs=2-3\nCPUPressureWatch=auto\nIOPressureWatch=skip\nCPUPressureThresholdSec=200ms", "service", ""},
		{"partition requirement", "[Service]\nCPUSetPartition=root", "service", "requires AllowedCPUs="},
		{"partition typo", "[Service]\nCPUSetPartition=isolate", "service", "unexpected value"},
		{"pressure typo", "[Service]\nCPUPressureWatch=automatic", "service", "unexpected value"},
		{"fd store", "[Service]\nFileDescriptorStorePreserve=on-success", "service", ""},
		{"reload list", "[Service]\nRefreshOnReload=extensions credentials\nRefreshOnReload=~credentials\nRefreshOnReload=", "service", ""},
		{"reload mixed boolean", "[Service]\nRefreshOnReload=yes credentials", "service", "unexpected value"},
		{"reload inverted boolean", "[Service]\nRefreshOnReload=~yes", "service", "unexpected value"},
		{"fraction", "[Unit]\nConditionFraction=rollout 0.5%\nConditionFraction=|!other 100%\nAssertFraction=0%\nConditionFraction=", "service", ""},
		{"fraction range", "[Unit]\nConditionFraction=rollout 101%", "service", "between 0% and 100%"},
		{"fraction alternate units", "[Unit]\nConditionFraction=tag 5‰\nAssertFraction=50‱", "service", ""},
		{"fraction precision", "[Unit]\nConditionFraction=0.123%", "service", "two decimal places"},
		{"fraction missing percent", "[Unit]\nConditionFraction=10", "service", "expects an optional tag"},
		{"fraction extra word", "[Unit]\nConditionFraction=one two 10%", "service", "expects an optional tag"},
		{"machine tag", "[Unit]\nConditionMachineTag=|!production-*", "service", ""},
		{"secure bits", "[Service]\nSecureBits=keep-caps no-setuid-fixup\nSecureBits=exec-restrict-file\nSecureBits=", "service", ""},
		{"secure bits typo", "[Service]\nSecureBits=keep-caps imaginary", "service", "unexpected value"},
		{"concurrency zero", "[Slice]\nActivatingConcurrencyMax=0", "slice", ""},
		{"concurrency unlimited", "[Slice]\nActivatingConcurrencyMax=infinity", "slice", ""},
		{"concurrency hex", "[Slice]\nActivatingConcurrencyMax=0x10", "slice", ""},
		{"concurrency leading zero decimal", "[Slice]\nActivatingConcurrencyMax=08", "slice", "unsigned 32-bit"},
		{"concurrency negative", "[Slice]\nActivatingConcurrencyMax=-1", "slice", "unsigned 32-bit"},
		{"concurrency overflow", "[Slice]\nActivatingConcurrencyMax=4294967296", "slice", "unsigned 32-bit"},
		{"concurrency wrong section", "[Service]\nActivatingConcurrencyMax=4", "service", "unknown directive"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ds := Diagnostics(c, tc.text, tc.unit)
			if tc.message == "" {
				if len(ds) != 0 {
					t.Fatalf("unexpected diagnostics: %#v", ds)
				}
				return
			}
			if len(ds) != 1 || !strings.Contains(ds[0].Message, tc.message) {
				t.Fatalf("wanted %q, got %#v", tc.message, ds)
			}
			if tc.name == "partition requirement" && ds[0].Severity != SeverityHint {
				t.Fatal("incomplete document must only produce a hint")
			}
		})
	}
}

func TestOlderCatalogRejectsNewerBooleanExtensions(t *testing.T) {
	c := NewCatalog()
	c.MergeCatalogFile(CatalogFile{Version: "v256", Directives: []CatalogDirective{
		{Section: "Service", Name: "PrivateTmp", Parser: "config_parse_private_tmp"},
		{Section: "Service", Name: "ProtectSystem", Parser: "config_parse_protect_system"},
	}})
	if contains(c.ValuesFor("Service", "PrivateTmp"), "disconnected") {
		t.Fatal("v256 must not offer PrivateTmp=disconnected")
	}
	if !contains(c.ValuesFor("Service", "ProtectSystem"), "strict") {
		t.Fatal("v256 must offer ProtectSystem=strict")
	}
}

func TestCatalogsAreCorrectedOnLoad(t *testing.T) {
	// A catalog generated before additive parsers and tristates were handled.
	c := NewCatalog()
	c.MergeCatalogFile(CatalogFile{Version: "v262", Directives: []CatalogDirective{
		{Section: "Service", Name: "SystemCallFilter", Parser: "config_parse_syscall_filter"},
		{Section: "Service", Name: "SystemCallErrorNumber", Parser: "config_parse_syscall_errno"},
		{Section: "Service", Name: "MemoryKSM", Parser: "config_parse_tristate", Values: []string{"yes", "no", "auto"}},
	}})
	if d, _ := c.Directive("Service", "SystemCallFilter"); !d.Multiple {
		t.Fatal("SystemCallFilter must be repeatable")
	}
	if d, _ := c.Directive("Service", "SystemCallErrorNumber"); d.Multiple {
		t.Fatal("SystemCallErrorNumber is a single value")
	}
	if values := c.ValuesFor("Service", "MemoryKSM"); contains(values, "auto") || !contains(values, "true") {
		t.Fatalf("MemoryKSM values = %v, want booleans", values)
	}
}

func TestEmbeddedCatalogIsUpToDate(t *testing.T) {
	file, err := DecodeCatalogFile(bytes.NewReader(embeddedLoadFragmentCatalog))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := EncodeCatalogFile(&out, EnrichCatalogFeatures(file)); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Bytes(), embeddedLoadFragmentCatalog) {
		t.Fatal("catalogdata/load-fragment-main.json is stale; regenerate it after changing catalog inference or feature refinements")
	}
}
