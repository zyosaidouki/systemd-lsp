package systemd

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
)

type CatalogFile struct {
	Version    string             `json:"version,omitempty"`
	Source     string             `json:"source,omitempty"`
	Directives []CatalogDirective `json:"directives"`
}

type CatalogDirective struct {
	Section   string   `json:"section"`
	Name      string   `json:"name"`
	Parser    string   `json:"parser,omitempty"`
	ValueKind string   `json:"valueKind,omitempty"`
	Doc       string   `json:"doc,omitempty"`
	Syntax    string   `json:"syntax,omitempty"`
	Example   string   `json:"example,omitempty"`
	ManPage   string   `json:"manPage,omitempty"`
	Multiple  bool     `json:"multiple,omitempty"`
	Values    []string `json:"values,omitempty"`
}

func LoadCatalogFile(path string) (CatalogFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return CatalogFile{}, err
	}
	defer f.Close()
	return DecodeCatalogFile(f)
}

func DecodeCatalogFile(r io.Reader) (CatalogFile, error) {
	var file CatalogFile
	if err := json.NewDecoder(r).Decode(&file); err != nil {
		return CatalogFile{}, err
	}
	return file, nil
}

func EncodeCatalogFile(w io.Writer, file CatalogFile) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(file)
}

func ParseLoadFragmentGperf(r io.Reader) (CatalogFile, error) {
	lines, macros, err := readGperfTemplate(r)
	if err != nil {
		return CatalogFile{}, err
	}

	directives := map[string]CatalogDirective{}
	for _, line := range expandGperfLines(lines, macros) {
		directive, ok := parseGperfDirective(line)
		if !ok {
			continue
		}
		key := directive.Section + "." + directive.Name
		if existing, ok := directives[key]; ok {
			directives[key] = mergeCatalogDirective(existing, directive)
			continue
		}
		directives[key] = directive
	}

	file := CatalogFile{
		Source:     "systemd src/core/load-fragment-gperf.gperf.in",
		Directives: make([]CatalogDirective, 0, len(directives)),
	}
	for _, directive := range directives {
		file.Directives = append(file.Directives, directive)
	}
	sort.Slice(file.Directives, func(i, j int) bool {
		if file.Directives[i].Section == file.Directives[j].Section {
			return file.Directives[i].Name < file.Directives[j].Name
		}
		return file.Directives[i].Section < file.Directives[j].Section
	})
	return file, nil
}

func (d CatalogDirective) directive() Directive {
	return Directive{
		Name:      d.Name,
		Doc:       d.Doc,
		Parser:    d.Parser,
		ValueKind: d.ValueKind,
		Syntax:    d.Syntax,
		Example:   d.Example,
		ManPage:   d.ManPage,
		Multiple:  d.Multiple,
		Values:    d.Values,
	}
}

type gperfMacro struct {
	Param string
	Lines []string
}

var macroStartRE = regexp.MustCompile(`^\{%-?\s*macro\s+([A-Za-z0-9_]+)\(([^)]*)\)\s*-?%\}`)
var macroCallRE = regexp.MustCompile(`^\{\{\s*([A-Za-z0-9_]+)\('([^']+)'\)\s*\}\}$`)

func readGperfTemplate(r io.Reader) ([]string, map[string]gperfMacro, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	lines := []string{}
	macros := map[string]gperfMacro{}
	var currentName string
	var current gperfMacro

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if currentName != "" {
			if strings.Contains(trimmed, "endmacro") {
				macros[currentName] = current
				currentName = ""
				current = gperfMacro{}
				continue
			}
			current.Lines = append(current.Lines, line)
			continue
		}
		if matches := macroStartRE.FindStringSubmatch(trimmed); len(matches) == 3 {
			currentName = matches[1]
			current.Param = strings.TrimSpace(matches[2])
			continue
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}
	return lines, macros, nil
}

func expandGperfLines(lines []string, macros map[string]gperfMacro) []string {
	expanded := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if matches := macroCallRE.FindStringSubmatch(trimmed); len(matches) == 3 {
			macro, ok := macros[matches[1]]
			if !ok {
				continue
			}
			arg := matches[2]
			for _, macroLine := range macro.Lines {
				expanded = append(expanded, strings.ReplaceAll(macroLine, "{{"+macro.Param+"}}", arg))
			}
			continue
		}
		expanded = append(expanded, line)
	}
	return expanded
}

func parseGperfDirective(line string) (CatalogDirective, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "{") || strings.HasPrefix(line, "%") || strings.HasPrefix(line, "#") {
		return CatalogDirective{}, false
	}
	if strings.Contains(line, "{{") || strings.Contains(line, "}}") {
		return CatalogDirective{}, false
	}

	fields := strings.Split(line, ",")
	if len(fields) < 2 {
		return CatalogDirective{}, false
	}
	sectionAndKey := strings.TrimSpace(fields[0])
	section, name, ok := strings.Cut(sectionAndKey, ".")
	if !ok || !validGperfIdentifier(section) || !validGperfIdentifier(name) {
		return CatalogDirective{}, false
	}
	parser := strings.TrimSpace(fields[1])
	valueKind := inferValueKind(parser, name)
	return CatalogDirective{
		Section:   section,
		Name:      name,
		Parser:    parser,
		ValueKind: valueKind,
		Multiple:  inferMultiple(parser, name),
		Values:    inferValues(section, parser, name),
	}, true
}

func validGperfIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r == '_' || r == '-' || r == ':' {
			return false
		}
		if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

func mergeCatalogDirective(existing, next CatalogDirective) CatalogDirective {
	if existing.Parser == "" {
		existing.Parser = next.Parser
	}
	if existing.ValueKind == "" {
		existing.ValueKind = next.ValueKind
	}
	if existing.Doc == "" {
		existing.Doc = next.Doc
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
	return existing
}

func inferMultiple(parser, name string) bool {
	if strings.HasPrefix(name, "Condition") || strings.HasPrefix(name, "Assert") {
		return true
	}
	if strings.HasPrefix(name, "Exec") {
		return true
	}
	switch name {
	case "Documentation", "Environment", "EnvironmentFile", "PassEnvironment", "UnsetEnvironment",
		"SupplementaryGroups", "LogExtraFields", "LogFilterPatterns", "CapabilityBoundingSet",
		"AmbientCapabilities", "SecureBits", "DeviceAllow", "IPAddressAllow", "IPAddressDeny",
		"SocketBindAllow", "SocketBindDeny", "DisableControllers", "ReadWritePaths", "ReadOnlyPaths",
		"InaccessiblePaths", "ExecPaths", "NoExecPaths", "BindPaths", "BindReadOnlyPaths",
		"TemporaryFileSystem", "StateDirectory", "CacheDirectory", "LogsDirectory", "RuntimeDirectory",
		"ConfigurationDirectory", "SetCredential", "SetCredentialEncrypted", "LoadCredential",
		"LoadCredentialEncrypted", "ImportCredential":
		return true
	}
	return strings.Contains(parser, "strv") ||
		strings.Contains(parser, "deps") ||
		strings.Contains(parser, "condition") ||
		strings.Contains(parser, "exec_directories") ||
		strings.Contains(parser, "credential") ||
		strings.Contains(parser, "bpf") ||
		strings.Contains(parser, "image")
}

func inferValues(section, parser, name string) []string {
	switch {
	case parser == "config_parse_bool":
		return boolValues
	case parser == "config_parse_tristate":
		return []string{"yes", "no", "auto"}
	}
	switch name {
	case "Type":
		if section == "Service" {
			return []string{"simple", "exec", "forking", "oneshot", "dbus", "notify", "notify-reload", "idle"}
		}
	case "ExitType":
		if section == "Service" {
			return []string{"main", "cgroup"}
		}
	case "Restart":
		if section == "Service" {
			return []string{"no", "on-success", "on-failure", "on-abnormal", "on-watchdog", "on-abort", "always"}
		}
	case "RestartMode":
		if section == "Service" {
			return []string{"normal", "direct", "debug"}
		}
	case "NotifyAccess":
		if section == "Service" {
			return []string{"none", "main", "exec", "all"}
		}
	case "OOMPolicy":
		if section == "Service" || section == "Scope" {
			return []string{"continue", "stop", "kill"}
		}
	case "KillMode":
		return []string{"control-group", "mixed", "process", "none"}
	case "CollectMode":
		return []string{"inactive", "inactive-or-failed"}
	case "DevicePolicy":
		return []string{"auto", "closed", "strict"}
	case "ManagedOOMSwap", "ManagedOOMMemoryPressure":
		return []string{"auto", "kill"}
	case "ManagedOOMPreference":
		return []string{"none", "avoid", "omit"}
	case "BindIPv6Only":
		return []string{"default", "both", "ipv6-only"}
	case "StandardInput":
		return []string{"null", "tty", "tty-force", "tty-fail", "data", "file", "socket", "fd"}
	case "StandardOutput", "StandardError":
		return []string{"inherit", "null", "tty", "journal", "kmsg", "journal+console", "kmsg+console", "file", "append", "truncate", "socket", "fd"}
	}
	return nil
}

func inferValueKind(parser, name string) string {
	switch {
	case parser == "NULL":
		return "install"
	case parser == "config_parse_bool":
		return "boolean"
	case parser == "config_parse_tristate":
		return "tristate"
	case strings.Contains(parser, "sec") || strings.Contains(parser, "nsec") || strings.Contains(name, "Timeout") || strings.HasSuffix(name, "Sec"):
		return "duration"
	case strings.Contains(parser, "size") || strings.Contains(parser, "memory_limit") || strings.Contains(parser, "quota"):
		return "size"
	case strings.Contains(parser, "exec") && strings.HasPrefix(name, "Exec"):
		return "command"
	case strings.Contains(parser, "path") || strings.Contains(parser, "mount_node") || strings.HasSuffix(name, "Path") || strings.HasSuffix(name, "Paths"):
		return "path"
	case strings.Contains(parser, "unit_deps"):
		return "unit-list"
	case strings.Contains(parser, "trigger_unit") || name == "Slice" || name == "Service" || name == "Unit":
		return "unit"
	case strings.Contains(parser, "environ"):
		return "environment"
	case strings.Contains(parser, "signal"):
		return "signal"
	case strings.Contains(parser, "mode"):
		return "mode"
	case strings.Contains(parser, "rlimit"):
		return "resource-limit"
	case strings.Contains(parser, "condition"):
		return "condition"
	case strings.Contains(parser, "socket") || strings.HasPrefix(name, "Listen"):
		return "socket"
	case strings.Contains(parser, "timer") || strings.Contains(name, "Calendar"):
		return "timer"
	case strings.Contains(parser, "warn_compat"):
		return "compatibility"
	case strings.Contains(parser, "string") || strings.Contains(parser, "printf"):
		return "string"
	case parser == "config_parse_unsigned" || parser == "config_parse_int" || parser == "config_parse_long":
		return "integer"
	default:
		return "string"
	}
}
