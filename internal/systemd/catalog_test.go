package systemd

import "testing"

func TestResourceControlDirectivesCoverSystemd249ManPage(t *testing.T) {
	catalog := NewCatalog()
	names := []string{
		"AllowedCPUs",
		"AllowedMemoryNodes",
		"BlockIOAccounting",
		"BlockIODeviceWeight",
		"BlockIOReadBandwidth",
		"BlockIOWeight",
		"BlockIOWriteBandwidth",
		"BPFProgram",
		"CPUShares",
		"CPUAccounting",
		"CPUQuota",
		"CPUQuotaPeriodSec",
		"CPUWeight",
		"DefaultMemoryLow",
		"DefaultMemoryMin",
		"Delegate",
		"DeviceAllow",
		"DevicePolicy",
		"DisableControllers",
		"IPAddressAllow",
		"IPAddressDeny",
		"IPAccounting",
		"IPEgressFilterPath",
		"IPIngressFilterPath",
		"IODeviceLatencyTargetSec",
		"IODeviceWeight",
		"IOReadBandwidthMax",
		"IOReadIOPSMax",
		"IOWeight",
		"IOWriteBandwidthMax",
		"IOWriteIOPSMax",
		"IOAccounting",
		"ManagedOOMMemoryPressure",
		"ManagedOOMMemoryPressureLimit",
		"ManagedOOMPreference",
		"ManagedOOMSwap",
		"MemoryAccounting",
		"MemoryHigh",
		"MemoryLimit",
		"MemoryLow",
		"MemoryMax",
		"MemoryMin",
		"MemorySwapMax",
		"Slice",
		"SocketBindAllow",
		"SocketBindDeny",
		"StartupBlockIOWeight",
		"StartupCPUShares",
		"StartupCPUWeight",
		"StartupIOWeight",
		"TasksAccounting",
		"TasksMax",
	}
	for _, section := range []string{"Service", "Socket", "Mount", "Swap", "Slice", "Scope"} {
		for _, name := range names {
			if _, ok := catalog.Directive(section, name); !ok {
				t.Fatalf("[%s] missing resource-control directive %s", section, name)
			}
		}
	}
}

func TestResourceControlDirectivesAreNotAppliedToTimerOrPath(t *testing.T) {
	catalog := NewCatalog()
	for _, section := range []string{"Timer", "Path", "Automount"} {
		if _, ok := catalog.Directive(section, "CPUWeight"); ok {
			t.Fatalf("[%s] unexpectedly has resource-control directive CPUWeight", section)
		}
	}
}

func TestExecContextDirectivesIncludeCPUAffinity(t *testing.T) {
	catalog := NewCatalog()
	for _, section := range []string{"Service", "Socket", "Mount", "Swap"} {
		directive, ok := catalog.Directive(section, "CPUAffinity")
		if !ok {
			t.Fatalf("[%s] missing CPUAffinity directive", section)
		}
		if !directive.Multiple {
			t.Fatalf("[%s] CPUAffinity should allow repeated assignments", section)
		}
	}
}

func TestCatalogIncludesEmbeddedGeneratedDirectives(t *testing.T) {
	catalog := NewCatalog()
	directives := catalog.DirectivesFor("Service")
	if len(directives) < 200 {
		t.Fatalf("[Service] directive count = %d, want generated catalog coverage", len(directives))
	}
	for _, name := range []string{"RestartMode", "MemoryZSwapMax", "ManagedOOMMemoryPressureDurationSec"} {
		directive, ok := catalog.Directive("Service", name)
		if !ok {
			t.Fatalf("[Service] missing embedded generated directive %s", name)
		}
		if directive.Parser == "" {
			t.Fatalf("[Service] %s missing parser metadata", name)
		}
	}
}
