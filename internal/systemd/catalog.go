package systemd

import "sort"

type Directive struct {
	Name      string
	Doc       string
	Parser    string
	ValueKind string
	Syntax    string
	Example   string
	ManPage   string
	Multiple  bool
	UnitTypes []string
	Values    []string
}

type Section struct {
	Name      string
	Doc       string
	UnitTypes []string
}

type Catalog struct {
	sections   map[string]Section
	directives map[string]map[string]Directive
}

func NewCatalog() *Catalog {
	c := &Catalog{
		sections:   map[string]Section{},
		directives: map[string]map[string]Directive{},
	}
	for _, section := range builtinSections {
		c.sections[section.Name] = section
	}
	for section, directives := range builtinDirectives {
		c.directives[section] = map[string]Directive{}
		for _, directive := range directives {
			c.directives[section][directive.Name] = directive
		}
	}
	mergeEmbeddedCatalog(c)
	return c
}

func (c *Catalog) KnowsSection(section string) bool {
	_, ok := c.sections[section]
	return ok
}

func (c *Catalog) SectionAllowed(section, unitType string) bool {
	s, ok := c.sections[section]
	if !ok || unitType == "" || len(s.UnitTypes) == 0 {
		return ok
	}
	return contains(s.UnitTypes, unitType)
}

func (c *Catalog) SectionDoc(section string) string {
	if s, ok := c.sections[section]; ok {
		return s.Doc
	}
	return ""
}

func (c *Catalog) SectionsFor(unitType string) []string {
	var sections []string
	for name, section := range c.sections {
		if unitType == "" || len(section.UnitTypes) == 0 || contains(section.UnitTypes, unitType) {
			sections = append(sections, name)
		}
	}
	sort.Strings(sections)
	return sections
}

func (c *Catalog) Directive(section, name string) (Directive, bool) {
	directives, ok := c.directives[section]
	if !ok {
		return Directive{}, false
	}
	directive, ok := directives[name]
	return directive, ok
}

func (c *Catalog) DirectivesFor(section string) []Directive {
	directives := c.directives[section]
	result := make([]Directive, 0, len(directives))
	for _, directive := range directives {
		result = append(result, directive)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

func (c *Catalog) ValuesFor(section, key string) []string {
	if directive, ok := c.Directive(section, key); ok {
		return directive.Values
	}
	return nil
}

func (c *Catalog) MergeCatalogFile(file CatalogFile) {
	for _, incoming := range file.Directives {
		if incoming.Section == "" || incoming.Name == "" {
			continue
		}
		if _, ok := c.sections[incoming.Section]; !ok {
			c.sections[incoming.Section] = Section{
				Name: incoming.Section,
				Doc:  "systemd section loaded from generated catalog.",
			}
		}
		if _, ok := c.directives[incoming.Section]; !ok {
			c.directives[incoming.Section] = map[string]Directive{}
		}
		next := incoming.directive()
		if existing, ok := c.directives[incoming.Section][incoming.Name]; ok {
			next = mergeDirective(existing, next)
		}
		c.directives[incoming.Section][incoming.Name] = next
	}
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

var builtinSections = []Section{
	{Name: "Unit", Doc: "Generic metadata and dependency ordering for any unit.", UnitTypes: []string{"service", "socket", "timer", "path", "mount", "automount", "swap", "target", "slice", "scope"}},
	{Name: "Install", Doc: "Installation information used by systemctl enable and disable.", UnitTypes: []string{"service", "socket", "timer", "path", "mount", "automount", "swap", "target"}},
	{Name: "Service", Doc: "Process execution and lifecycle settings for service units.", UnitTypes: []string{"service"}},
	{Name: "Socket", Doc: "Socket activation settings for socket units.", UnitTypes: []string{"socket"}},
	{Name: "Timer", Doc: "Calendar and monotonic timer settings for timer units.", UnitTypes: []string{"timer"}},
	{Name: "Path", Doc: "Path monitoring settings for path units.", UnitTypes: []string{"path"}},
	{Name: "Mount", Doc: "Mount point settings for mount units.", UnitTypes: []string{"mount"}},
	{Name: "Automount", Doc: "Automount point settings for automount units.", UnitTypes: []string{"automount"}},
	{Name: "Swap", Doc: "Swap device or file settings for swap units.", UnitTypes: []string{"swap"}},
	{Name: "Slice", Doc: "Resource control grouping settings for slice units.", UnitTypes: []string{"slice"}},
	{Name: "Scope", Doc: "Externally created process scope settings.", UnitTypes: []string{"scope"}},
}

var builtinDirectives = map[string][]Directive{
	"Unit": {
		{Name: "Description", Doc: "Human-readable unit description."},
		{Name: "Documentation", Doc: "Documentation URIs for this unit.", Multiple: true},
		{Name: "Requires", Doc: "Units that must be activated together with this unit.", Multiple: true},
		{Name: "Requisite", Doc: "Units that must already be active for this unit to start.", Multiple: true},
		{Name: "Wants", Doc: "Weaker requirement dependencies.", Multiple: true},
		{Name: "BindsTo", Doc: "Requirement dependency that stops this unit when the bound unit stops.", Multiple: true},
		{Name: "PartOf", Doc: "Propagates stop and restart operations from listed units.", Multiple: true},
		{Name: "Upholds", Doc: "Keeps listed units started while this unit is active.", Multiple: true},
		{Name: "Conflicts", Doc: "Negative requirement dependencies.", Multiple: true},
		{Name: "Before", Doc: "Ordering dependency; this unit starts before listed units.", Multiple: true},
		{Name: "After", Doc: "Ordering dependency; this unit starts after listed units.", Multiple: true},
		{Name: "OnFailure", Doc: "Units activated when this unit enters failed state.", Multiple: true},
		{Name: "OnSuccess", Doc: "Units activated when this unit becomes inactive successfully.", Multiple: true},
		{Name: "PropagatesReloadTo", Doc: "Reload propagation dependencies.", Multiple: true},
		{Name: "ReloadPropagatedFrom", Doc: "Reverse reload propagation dependencies.", Multiple: true},
		{Name: "JoinsNamespaceOf", Doc: "Share namespace setup with listed units.", Multiple: true},
		{Name: "RequiresMountsFor", Doc: "Adds mount dependencies for paths.", Multiple: true},
		{Name: "OnFailureJobMode", Doc: "Job mode used when activating OnFailure units.", Values: []string{"replace", "replace-irreversibly", "isolate", "flush", "ignore-dependencies", "ignore-requirements", "fail"}},
		{Name: "OnSuccessJobMode", Doc: "Job mode used when activating OnSuccess units.", Values: []string{"replace", "replace-irreversibly", "isolate", "flush", "ignore-dependencies", "ignore-requirements", "fail"}},
		{Name: "IgnoreOnIsolate", Doc: "Whether this unit is ignored during isolate operations.", Values: boolValues},
		{Name: "StopWhenUnneeded", Doc: "Whether this unit is stopped when no active unit needs it.", Values: boolValues},
		{Name: "RefuseManualStart", Doc: "Reject manual start requests.", Values: boolValues},
		{Name: "RefuseManualStop", Doc: "Reject manual stop requests.", Values: boolValues},
		{Name: "AllowIsolate", Doc: "Allow this unit to be used as isolate target.", Values: boolValues},
		{Name: "DefaultDependencies", Doc: "Whether implicit default dependencies are added.", Values: boolValues},
		{Name: "CollectMode", Doc: "Controls unloading of inactive unit state.", Values: []string{"inactive", "inactive-or-failed"}},
		{Name: "StartLimitIntervalSec", Doc: "Time interval for start rate limiting."},
		{Name: "StartLimitBurst", Doc: "Start attempts allowed in the start limit interval."},
		{Name: "ConditionPathExists", Doc: "Start condition requiring a path to exist.", Multiple: true},
		{Name: "ConditionPathExistsGlob", Doc: "Start condition requiring a path glob to match.", Multiple: true},
		{Name: "ConditionPathIsDirectory", Doc: "Start condition requiring a directory path.", Multiple: true},
		{Name: "ConditionPathIsSymbolicLink", Doc: "Start condition requiring a symbolic link.", Multiple: true},
		{Name: "ConditionPathIsMountPoint", Doc: "Start condition requiring a mount point.", Multiple: true},
		{Name: "ConditionPathIsReadWrite", Doc: "Start condition requiring a writable path.", Multiple: true},
		{Name: "ConditionDirectoryNotEmpty", Doc: "Start condition requiring a non-empty directory.", Multiple: true},
		{Name: "ConditionFileNotEmpty", Doc: "Start condition requiring a non-empty file.", Multiple: true},
		{Name: "ConditionFileIsExecutable", Doc: "Start condition requiring an executable file.", Multiple: true},
		{Name: "ConditionUser", Doc: "Start condition matching the user.", Multiple: true},
		{Name: "ConditionGroup", Doc: "Start condition matching the group.", Multiple: true},
		{Name: "ConditionHost", Doc: "Start condition matching the hostname.", Multiple: true},
		{Name: "ConditionKernelCommandLine", Doc: "Start condition matching the kernel command line.", Multiple: true},
		{Name: "ConditionKernelVersion", Doc: "Start condition matching the kernel version.", Multiple: true},
		{Name: "ConditionVirtualization", Doc: "Start condition matching virtualization state.", Multiple: true},
		{Name: "ConditionSecurity", Doc: "Start condition matching security technology.", Multiple: true},
		{Name: "ConditionCapability", Doc: "Start condition matching process capabilities.", Multiple: true},
		{Name: "ConditionACPower", Doc: "Start condition matching AC power state.", Values: boolValues, Multiple: true},
		{Name: "ConditionNeedsUpdate", Doc: "Start condition matching paths needing update.", Multiple: true},
		{Name: "AssertPathExists", Doc: "Start assertion requiring a path to exist.", Multiple: true},
		{Name: "AssertPathIsDirectory", Doc: "Start assertion requiring a directory path.", Multiple: true},
		{Name: "AssertFileNotEmpty", Doc: "Start assertion requiring a non-empty file.", Multiple: true},
	},
	"Install": {
		{Name: "Alias", Doc: "Additional names created when enabling this unit.", Multiple: true},
		{Name: "WantedBy", Doc: "Target units whose .wants directory receives a link.", Multiple: true},
		{Name: "RequiredBy", Doc: "Target units whose .requires directory receives a link.", Multiple: true},
		{Name: "UpheldBy", Doc: "Target units whose .upholds directory receives a link.", Multiple: true},
		{Name: "Also", Doc: "Additional units enabled or disabled together with this unit.", Multiple: true},
		{Name: "DefaultInstance", Doc: "Default instance name for template units."},
	},
	"Service":   combineDirectives(serviceDirectives, resourceControlDirectives),
	"Socket":    combineDirectives(socketDirectives, resourceControlDirectives),
	"Timer":     timerDirectives,
	"Path":      pathDirectives,
	"Mount":     combineDirectives(mountDirectives, resourceControlDirectives),
	"Automount": automountDirectives,
	"Swap":      combineDirectives(swapDirectives, resourceControlDirectives),
	"Slice":     resourceControlDirectives,
	"Scope":     combineDirectives(scopeDirectives, resourceControlDirectives),
}

var boolValues = []string{"true", "false", "yes", "no", "on", "off", "1", "0"}

var serviceDirectives = append([]Directive{
	{Name: "Type", Doc: "Service start-up type.", Values: []string{"simple", "exec", "forking", "oneshot", "dbus", "notify", "notify-reload", "idle"}},
	{Name: "ExitType", Doc: "Defines when systemd considers the service exited.", Values: []string{"main", "cgroup"}},
	{Name: "RemainAfterExit", Doc: "Consider service active after all processes exit.", Values: boolValues},
	{Name: "GuessMainPID", Doc: "Try to guess the main PID of a forking service.", Values: boolValues},
	{Name: "PIDFile", Doc: "PID file path for forking services."},
	{Name: "BusName", Doc: "D-Bus bus name for Type=dbus services."},
	{Name: "ExecStart", Doc: "Command lines executed to start the service.", Multiple: true},
	{Name: "ExecStartPre", Doc: "Command lines executed before ExecStart.", Multiple: true},
	{Name: "ExecStartPost", Doc: "Command lines executed after ExecStart.", Multiple: true},
	{Name: "ExecCondition", Doc: "Condition command lines evaluated before starting.", Multiple: true},
	{Name: "ExecReload", Doc: "Command lines executed to reload the service.", Multiple: true},
	{Name: "ExecStop", Doc: "Command lines executed to stop the service.", Multiple: true},
	{Name: "ExecStopPost", Doc: "Command lines executed after stopping the service.", Multiple: true},
	{Name: "RestartSec", Doc: "Delay before restarting the service."},
	{Name: "TimeoutStartSec", Doc: "Maximum time allowed for service start-up."},
	{Name: "TimeoutStopSec", Doc: "Maximum time allowed for service stop."},
	{Name: "TimeoutAbortSec", Doc: "Maximum time allowed for abort handling."},
	{Name: "TimeoutSec", Doc: "Shorthand for start and stop timeout."},
	{Name: "RuntimeMaxSec", Doc: "Maximum runtime before termination."},
	{Name: "RuntimeRandomizedExtraSec", Doc: "Random extra runtime added to RuntimeMaxSec."},
	{Name: "WatchdogSec", Doc: "Watchdog timeout for services using watchdog notification."},
	{Name: "Restart", Doc: "Restart policy for the service.", Values: []string{"no", "on-success", "on-failure", "on-abnormal", "on-watchdog", "on-abort", "always"}},
	{Name: "SuccessExitStatus", Doc: "Additional exit statuses considered successful.", Multiple: true},
	{Name: "RestartPreventExitStatus", Doc: "Exit statuses that prevent restart.", Multiple: true},
	{Name: "RestartForceExitStatus", Doc: "Exit statuses that force restart.", Multiple: true},
	{Name: "RootDirectoryStartOnly", Doc: "Apply root directory only to start commands.", Values: boolValues},
	{Name: "NonBlocking", Doc: "Set non-blocking mode for sockets passed to the service.", Values: boolValues},
	{Name: "NotifyAccess", Doc: "Controls access to the service manager notification socket.", Values: []string{"none", "main", "exec", "all"}},
	{Name: "Sockets", Doc: "Socket units to pass to the service.", Multiple: true},
	{Name: "FileDescriptorStoreMax", Doc: "Maximum file descriptors stored for the service."},
	{Name: "USBFunctionDescriptors", Doc: "Path to USB FunctionFS descriptors."},
	{Name: "USBFunctionStrings", Doc: "Path to USB FunctionFS strings."},
	{Name: "OOMPolicy", Doc: "Action when a process in the service is killed by the OOM killer.", Values: []string{"continue", "stop", "kill"}},
}, execContextDirectives()...)

func execContextDirectives() []Directive {
	return []Directive{
		{Name: "User", Doc: "Run processes as this user."},
		{Name: "Group", Doc: "Run processes as this group."},
		{Name: "SupplementaryGroups", Doc: "Supplementary groups for executed processes.", Multiple: true},
		{Name: "WorkingDirectory", Doc: "Working directory for executed processes."},
		{Name: "RootDirectory", Doc: "Root directory for executed processes."},
		{Name: "RootImage", Doc: "Root disk image for executed processes."},
		{Name: "Environment", Doc: "Environment variable assignments.", Multiple: true},
		{Name: "EnvironmentFile", Doc: "Files containing environment variable assignments.", Multiple: true},
		{Name: "PassEnvironment", Doc: "Environment variables inherited from the manager.", Multiple: true},
		{Name: "UnsetEnvironment", Doc: "Environment variables removed from the environment.", Multiple: true},
		{Name: "StandardInput", Doc: "Standard input source.", Values: []string{"null", "tty", "tty-force", "tty-fail", "data", "file", "socket", "fd"}},
		{Name: "StandardOutput", Doc: "Standard output target.", Values: []string{"inherit", "null", "tty", "journal", "kmsg", "journal+console", "kmsg+console", "file", "append", "truncate", "socket", "fd"}},
		{Name: "StandardError", Doc: "Standard error target.", Values: []string{"inherit", "null", "tty", "journal", "kmsg", "journal+console", "kmsg+console", "file", "append", "truncate", "socket", "fd"}},
		{Name: "TTYPath", Doc: "TTY device path."},
		{Name: "SyslogIdentifier", Doc: "Identifier used for syslog and journal messages."},
		{Name: "SyslogFacility", Doc: "Syslog facility name."},
		{Name: "SyslogLevel", Doc: "Default syslog level.", Values: []string{"emerg", "alert", "crit", "err", "warning", "notice", "info", "debug"}},
		{Name: "SyslogLevelPrefix", Doc: "Honor syslog priority prefixes in output.", Values: boolValues},
		{Name: "LogLevelMax", Doc: "Maximum log level for emitted messages.", Values: []string{"emerg", "alert", "crit", "err", "warning", "notice", "info", "debug"}},
		{Name: "LogExtraFields", Doc: "Additional journal fields.", Multiple: true},
		{Name: "CapabilityBoundingSet", Doc: "Restrict Linux capabilities.", Multiple: true},
		{Name: "AmbientCapabilities", Doc: "Capabilities added to the ambient set.", Multiple: true},
		{Name: "CPUAffinity", Doc: "CPU affinity mask for executed processes; accepts CPU indexes, ranges, or numa.", Multiple: true, ValueKind: "cpu-set"},
		{Name: "NoNewPrivileges", Doc: "Prevent gaining new privileges.", Values: boolValues},
		{Name: "SecureBits", Doc: "Securebits flags for executed processes.", Multiple: true},
		{Name: "PrivateTmp", Doc: "Use a private /tmp and /var/tmp namespace.", Values: boolValues},
		{Name: "PrivateDevices", Doc: "Use a private /dev namespace.", Values: boolValues},
		{Name: "PrivateNetwork", Doc: "Use a private network namespace.", Values: boolValues},
		{Name: "PrivateUsers", Doc: "Use a private user namespace.", Values: boolValues},
		{Name: "ProtectSystem", Doc: "Make selected parts of the file system read-only.", Values: []string{"true", "false", "yes", "no", "full", "strict"}},
		{Name: "ProtectHome", Doc: "Restrict access to home directories.", Values: []string{"true", "false", "yes", "no", "read-only", "tmpfs"}},
		{Name: "ReadWritePaths", Doc: "Paths that remain writable.", Multiple: true},
		{Name: "ReadOnlyPaths", Doc: "Paths made read-only.", Multiple: true},
		{Name: "InaccessiblePaths", Doc: "Paths made inaccessible.", Multiple: true},
		{Name: "ExecPaths", Doc: "Paths that may contain executable files.", Multiple: true},
		{Name: "NoExecPaths", Doc: "Paths that may not contain executable files.", Multiple: true},
		{Name: "StateDirectory", Doc: "State directories created below /var/lib.", Multiple: true},
		{Name: "CacheDirectory", Doc: "Cache directories created below /var/cache.", Multiple: true},
		{Name: "LogsDirectory", Doc: "Log directories created below /var/log.", Multiple: true},
		{Name: "RuntimeDirectory", Doc: "Runtime directories created below /run.", Multiple: true},
		{Name: "ConfigurationDirectory", Doc: "Configuration directories created below /etc.", Multiple: true},
		{Name: "UMask", Doc: "File mode creation mask."},
		{Name: "LimitCPU", Doc: "CPU time resource limit."},
		{Name: "LimitFSIZE", Doc: "File size resource limit."},
		{Name: "LimitDATA", Doc: "Data size resource limit."},
		{Name: "LimitSTACK", Doc: "Stack size resource limit."},
		{Name: "LimitCORE", Doc: "Core file size resource limit."},
		{Name: "LimitRSS", Doc: "Resident set size resource limit."},
		{Name: "LimitNOFILE", Doc: "Open files resource limit."},
		{Name: "LimitNPROC", Doc: "Process count resource limit."},
		{Name: "LimitMEMLOCK", Doc: "Locked memory resource limit."},
		{Name: "LimitLOCKS", Doc: "File lock resource limit."},
		{Name: "LimitSIGPENDING", Doc: "Pending signals resource limit."},
		{Name: "LimitMSGQUEUE", Doc: "POSIX message queue resource limit."},
		{Name: "LimitNICE", Doc: "Nice priority resource limit."},
		{Name: "LimitRTPRIO", Doc: "Realtime priority resource limit."},
		{Name: "LimitRTTIME", Doc: "Realtime runtime resource limit."},
	}
}

var socketDirectives = append([]Directive{
	{Name: "ListenStream", Doc: "Stream socket address or path to listen on.", Multiple: true},
	{Name: "ListenDatagram", Doc: "Datagram socket address or path to listen on.", Multiple: true},
	{Name: "ListenSequentialPacket", Doc: "Sequential packet socket address or path to listen on.", Multiple: true},
	{Name: "ListenFIFO", Doc: "FIFO path to listen on.", Multiple: true},
	{Name: "ListenSpecial", Doc: "Special file path to listen on.", Multiple: true},
	{Name: "ListenNetlink", Doc: "Netlink socket to listen on.", Multiple: true},
	{Name: "ListenMessageQueue", Doc: "POSIX message queue to listen on.", Multiple: true},
	{Name: "ListenUSBFunction", Doc: "USB FunctionFS endpoint path to listen on.", Multiple: true},
	{Name: "SocketProtocol", Doc: "Socket protocol name."},
	{Name: "BindIPv6Only", Doc: "IPv6-only bind behavior.", Values: []string{"default", "both", "ipv6-only"}},
	{Name: "Backlog", Doc: "Listen backlog."},
	{Name: "BindToDevice", Doc: "Network interface to bind to."},
	{Name: "SocketUser", Doc: "User owning the socket node."},
	{Name: "SocketGroup", Doc: "Group owning the socket node."},
	{Name: "SocketMode", Doc: "File mode for socket nodes."},
	{Name: "DirectoryMode", Doc: "File mode for parent directories created for socket nodes."},
	{Name: "Accept", Doc: "Whether each connection spawns a service instance.", Values: boolValues},
	{Name: "Writable", Doc: "Whether a special file socket is opened writable.", Values: boolValues},
	{Name: "FlushPending", Doc: "Flush pending socket data before entering listening state.", Values: boolValues},
	{Name: "MaxConnections", Doc: "Maximum simultaneous connections for Accept=yes sockets."},
	{Name: "MaxConnectionsPerSource", Doc: "Maximum simultaneous connections from one source."},
	{Name: "KeepAlive", Doc: "Enable TCP keep-alive.", Values: boolValues},
	{Name: "NoDelay", Doc: "Enable TCP_NODELAY.", Values: boolValues},
	{Name: "FreeBind", Doc: "Allow binding to non-local IP addresses.", Values: boolValues},
	{Name: "Transparent", Doc: "Enable transparent proxy socket option.", Values: boolValues},
	{Name: "Broadcast", Doc: "Enable datagram broadcast.", Values: boolValues},
	{Name: "PassCredentials", Doc: "Pass SCM_CREDENTIALS metadata.", Values: boolValues},
	{Name: "PassSecurity", Doc: "Pass SCM_SECURITY metadata.", Values: boolValues},
	{Name: "PassPacketInfo", Doc: "Pass packet information metadata.", Values: boolValues},
	{Name: "ReusePort", Doc: "Enable SO_REUSEPORT.", Values: boolValues},
	{Name: "MessageQueueMaxMessages", Doc: "Maximum messages for POSIX message queues."},
	{Name: "MessageQueueMessageSize", Doc: "Message size for POSIX message queues."},
	{Name: "Service", Doc: "Service unit activated by this socket."},
	{Name: "RemoveOnStop", Doc: "Remove socket nodes when stopped.", Values: boolValues},
	{Name: "Symlinks", Doc: "Symlink paths created for socket nodes.", Multiple: true},
	{Name: "FileDescriptorName", Doc: "Name assigned to passed file descriptors."},
	{Name: "TriggerLimitIntervalSec", Doc: "Time interval for trigger rate limiting."},
	{Name: "TriggerLimitBurst", Doc: "Trigger attempts allowed in the interval."},
}, execContextDirectives()...)

var timerDirectives = []Directive{
	{Name: "OnActiveSec", Doc: "Monotonic delay after timer activation.", Multiple: true},
	{Name: "OnBootSec", Doc: "Monotonic delay after boot.", Multiple: true},
	{Name: "OnStartupSec", Doc: "Monotonic delay after service manager start.", Multiple: true},
	{Name: "OnUnitActiveSec", Doc: "Monotonic delay after the triggered unit was last active.", Multiple: true},
	{Name: "OnUnitInactiveSec", Doc: "Monotonic delay after the triggered unit was last inactive.", Multiple: true},
	{Name: "OnCalendar", Doc: "Calendar event expression.", Multiple: true},
	{Name: "AccuracySec", Doc: "Timer accuracy window."},
	{Name: "RandomizedDelaySec", Doc: "Random delay added before firing."},
	{Name: "FixedRandomDelay", Doc: "Use a stable randomized delay.", Values: boolValues},
	{Name: "OnClockChange", Doc: "Trigger when the system clock changes.", Values: boolValues},
	{Name: "OnTimezoneChange", Doc: "Trigger when the timezone changes.", Values: boolValues},
	{Name: "Unit", Doc: "Unit activated by this timer."},
	{Name: "Persistent", Doc: "Catch up missed calendar events after downtime.", Values: boolValues},
	{Name: "WakeSystem", Doc: "Wake the system from suspend for this timer.", Values: boolValues},
	{Name: "RemainAfterElapse", Doc: "Keep the timer loaded after it elapsed.", Values: boolValues},
}

var pathDirectives = []Directive{
	{Name: "PathExists", Doc: "Trigger when the path exists.", Multiple: true},
	{Name: "PathExistsGlob", Doc: "Trigger when the path glob matches.", Multiple: true},
	{Name: "PathChanged", Doc: "Trigger when the path changes.", Multiple: true},
	{Name: "PathModified", Doc: "Trigger when the path is modified.", Multiple: true},
	{Name: "DirectoryNotEmpty", Doc: "Trigger when the directory is not empty.", Multiple: true},
	{Name: "Unit", Doc: "Unit activated by this path unit."},
	{Name: "MakeDirectory", Doc: "Create watched directories before watching.", Values: boolValues},
	{Name: "DirectoryMode", Doc: "File mode for directories created by MakeDirectory."},
	{Name: "TriggerLimitIntervalSec", Doc: "Time interval for trigger rate limiting."},
	{Name: "TriggerLimitBurst", Doc: "Trigger attempts allowed in the interval."},
}

var mountDirectives = append([]Directive{
	{Name: "What", Doc: "Source device, file, or remote resource to mount."},
	{Name: "Where", Doc: "Absolute mount point path."},
	{Name: "Type", Doc: "File system type."},
	{Name: "Options", Doc: "Mount options."},
	{Name: "SloppyOptions", Doc: "Tolerate unknown mount options.", Values: boolValues},
	{Name: "LazyUnmount", Doc: "Detach the file system immediately on unmount.", Values: boolValues},
	{Name: "ReadWriteOnly", Doc: "Fail if the mount cannot be mounted read-write.", Values: boolValues},
	{Name: "ForceUnmount", Doc: "Force unmount when stopping.", Values: boolValues},
	{Name: "DirectoryMode", Doc: "Mode for automatically created mount point directories."},
	{Name: "TimeoutSec", Doc: "Maximum time allowed for mount and unmount operations."},
}, execContextDirectives()...)

var automountDirectives = []Directive{
	{Name: "Where", Doc: "Absolute automount point path."},
	{Name: "DirectoryMode", Doc: "Mode for automatically created automount point directories."},
	{Name: "TimeoutIdleSec", Doc: "Idle time before automatic unmount."},
	{Name: "ExtraOptions", Doc: "Extra automount options."},
}

var swapDirectives = append([]Directive{
	{Name: "What", Doc: "Swap device or file path."},
	{Name: "Priority", Doc: "Swap priority."},
	{Name: "Options", Doc: "Swap options."},
	{Name: "TimeoutSec", Doc: "Maximum time allowed for swap activation and deactivation."},
}, execContextDirectives()...)

var resourceControlDirectives = []Directive{
	{Name: "AllowedCPUs", Doc: "Restrict processes to specific CPU indices or CPU ranges."},
	{Name: "AllowedMemoryNodes", Doc: "Restrict processes to specific memory NUMA node indices or ranges."},
	{Name: "BlockIOAccounting", Doc: "Deprecated legacy cgroup-v1 block I/O accounting setting; use IOAccounting= instead.", Values: boolValues},
	{Name: "BlockIODeviceWeight", Doc: "Deprecated per-device legacy block I/O weight; use IODeviceWeight= instead.", Multiple: true},
	{Name: "BlockIOReadBandwidth", Doc: "Deprecated per-device legacy read bandwidth limit; use IOReadBandwidthMax= instead.", Multiple: true},
	{Name: "BlockIOWeight", Doc: "Deprecated default legacy block I/O weight; use IOWeight= instead."},
	{Name: "BlockIOWriteBandwidth", Doc: "Deprecated per-device legacy write bandwidth limit; use IOWriteBandwidthMax= instead.", Multiple: true},
	{Name: "BPFProgram", Doc: "Attach a cgroup BPF program using type:program-path syntax.", Multiple: true},
	{Name: "CPUShares", Doc: "Deprecated legacy CPU share weight; use CPUWeight= instead."},
	{Name: "CPUAccounting", Doc: "Enable CPU accounting.", Values: boolValues},
	{Name: "CPUQuotaPeriodSec", Doc: "CPU quota measurement period."},
	{Name: "CPUWeight", Doc: "CPU scheduling weight."},
	{Name: "CPUQuota", Doc: "CPU time quota percentage."},
	{Name: "DefaultMemoryLow", Doc: "Default MemoryLow= allocation used by child cgroups."},
	{Name: "DefaultMemoryMin", Doc: "Default MemoryMin= allocation used by child cgroups."},
	{Name: "Delegate", Doc: "Delegate cgroup subtree management to the unit, optionally limited to selected controllers."},
	{Name: "DeviceAllow", Doc: "Device access rules.", Multiple: true},
	{Name: "DevicePolicy", Doc: "Device access policy.", Values: []string{"auto", "closed", "strict"}},
	{Name: "DisableControllers", Doc: "Prevent selected cgroup controllers from being enabled for child units.", Multiple: true},
	{Name: "IPAddressAllow", Doc: "Allow IP traffic to matching addresses or symbolic networks.", Multiple: true},
	{Name: "IPAddressDeny", Doc: "Deny IP traffic to matching addresses or symbolic networks.", Multiple: true},
	{Name: "IPAccounting", Doc: "Enable IP accounting.", Values: boolValues},
	{Name: "IPEgressFilterPath", Doc: "Attach a pinned BPF program for egress IP packet filtering.", Multiple: true},
	{Name: "IPIngressFilterPath", Doc: "Attach a pinned BPF program for ingress IP packet filtering.", Multiple: true},
	{Name: "IODeviceLatencyTargetSec", Doc: "Per-device target I/O latency.", Multiple: true},
	{Name: "IODeviceWeight", Doc: "Per-device I/O weight.", Multiple: true},
	{Name: "IOReadBandwidthMax", Doc: "Per-device read bandwidth limit.", Multiple: true},
	{Name: "IOReadIOPSMax", Doc: "Per-device read IOPS limit.", Multiple: true},
	{Name: "IOWeight", Doc: "Default I/O weight."},
	{Name: "IOWriteBandwidthMax", Doc: "Per-device write bandwidth limit.", Multiple: true},
	{Name: "IOWriteIOPSMax", Doc: "Per-device write IOPS limit.", Multiple: true},
	{Name: "IOAccounting", Doc: "Enable I/O accounting.", Values: boolValues},
	{Name: "ManagedOOMMemoryPressure", Doc: "Configure systemd-oomd memory pressure action for this unit's cgroup.", Values: []string{"auto", "kill"}},
	{Name: "ManagedOOMMemoryPressureLimit", Doc: "Override the memory pressure limit for systemd-oomd."},
	{Name: "ManagedOOMPreference", Doc: "Adjust systemd-oomd candidate preference for this cgroup.", Values: []string{"none", "avoid", "omit"}},
	{Name: "ManagedOOMSwap", Doc: "Configure systemd-oomd swap action for this unit's cgroup.", Values: []string{"auto", "kill"}},
	{Name: "MemoryAccounting", Doc: "Enable memory accounting.", Values: boolValues},
	{Name: "MemoryLimit", Doc: "Deprecated legacy memory limit; use MemoryMax= instead."},
	{Name: "MemoryMin", Doc: "Minimum protected memory."},
	{Name: "MemoryLow", Doc: "Best-effort protected memory."},
	{Name: "MemoryHigh", Doc: "Memory throttling threshold."},
	{Name: "MemoryMax", Doc: "Maximum memory usage."},
	{Name: "MemorySwapMax", Doc: "Maximum swap usage."},
	{Name: "Slice", Doc: "Slice unit to place this unit in."},
	{Name: "SocketBindAllow", Doc: "Allow binding sockets matching a cgroup socket bind rule.", Multiple: true},
	{Name: "SocketBindDeny", Doc: "Deny binding sockets matching a cgroup socket bind rule.", Multiple: true},
	{Name: "StartupBlockIOWeight", Doc: "Deprecated startup-phase legacy block I/O weight; use StartupIOWeight= instead."},
	{Name: "StartupCPUShares", Doc: "Deprecated startup-phase legacy CPU share weight; use StartupCPUWeight= instead."},
	{Name: "StartupCPUWeight", Doc: "Startup-phase CPU scheduling weight."},
	{Name: "StartupIOWeight", Doc: "Startup-phase I/O weight."},
	{Name: "TasksAccounting", Doc: "Enable task accounting.", Values: boolValues},
	{Name: "TasksMax", Doc: "Maximum number of tasks."},
}

var scopeDirectives = []Directive{
	{Name: "OOMPolicy", Doc: "Action when a process in the scope is killed by the OOM killer.", Values: []string{"continue", "stop", "kill"}},
	{Name: "RuntimeMaxSec", Doc: "Maximum runtime before termination."},
	{Name: "RuntimeRandomizedExtraSec", Doc: "Random extra runtime added to RuntimeMaxSec."},
	{Name: "TimeoutStopSec", Doc: "Maximum time allowed for stopping the scope."},
}

func combineDirectives(groups ...[]Directive) []Directive {
	result := make([]Directive, 0)
	seen := map[string]int{}
	for _, group := range groups {
		for _, directive := range group {
			if idx, ok := seen[directive.Name]; ok {
				result[idx] = mergeDirective(result[idx], directive)
				continue
			}
			seen[directive.Name] = len(result)
			result = append(result, directive)
		}
	}
	return result
}

func mergeDirective(existing, next Directive) Directive {
	if existing.Doc == "" {
		existing.Doc = next.Doc
	}
	if existing.Parser == "" {
		existing.Parser = next.Parser
	}
	if existing.ValueKind == "" {
		existing.ValueKind = next.ValueKind
	}
	if existing.Syntax == "" {
		existing.Syntax = next.Syntax
	}
	if existing.Example == "" {
		existing.Example = next.Example
	}
	if existing.ManPage == "" {
		existing.ManPage = next.ManPage
	}
	if next.Multiple {
		existing.Multiple = true
	}
	if len(existing.Values) == 0 {
		existing.Values = next.Values
	}
	if len(existing.UnitTypes) == 0 {
		existing.UnitTypes = next.UnitTypes
	}
	return existing
}
