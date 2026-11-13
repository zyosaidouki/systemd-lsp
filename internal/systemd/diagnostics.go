package systemd

import (
	"fmt"
	"strings"
)

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Diagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity,omitzero"`
	Source   string `json:"source,omitempty"`
	Message  string `json:"message"`
}

const (
	SeverityError   = 1
	SeverityWarning = 2
	SeverityHint    = 4
)

// Diagnostics checks a complete unit file.
func Diagnostics(catalog *Catalog, text, unitType string) []Diagnostic {
	return diagnose(catalog, text, unitType, false)
}

// DropInDiagnostics checks a drop-in, whose settings only supplement the main
// unit file, so a missing companion setting is not necessarily a mistake.
func DropInDiagnostics(catalog *Catalog, text, unitType string) []Diagnostic {
	return diagnose(catalog, text, unitType, true)
}

func diagnose(catalog *Catalog, text, unitType string, dropIn bool) []Diagnostic {
	doc := Parse(text)
	diagnostics := make([]Diagnostic, 0)
	section := ""
	sectionKnown := false
	seenSingletons := map[string]int{}
	lastValues := map[string]map[string]Entry{}

	for _, entry := range doc.Entries {
		switch entry.Kind {
		case EntryBlank, EntryComment, EntryContinuation:
			continue
		case EntryInvalid:
			diagnostics = append(diagnostics, diagnostic(entry, SeverityError, entry.Message))
		case EntrySection:
			section = entry.Section
			sectionKnown = catalog.KnowsSection(section)
			if !sectionKnown {
				if isExtensionName(section) {
					continue
				}
				diagnostics = append(diagnostics, diagnostic(entry, SeverityWarning, fmt.Sprintf("unknown systemd section [%s]", section)))
				continue
			}
			if !catalog.SectionAllowed(section, unitType) {
				diagnostics = append(diagnostics, diagnostic(entry, SeverityWarning, fmt.Sprintf("[%s] is not normally used in .%s units", section, unitType)))
			}
		case EntryDirective:
			if section == "" {
				diagnostics = append(diagnostics, diagnostic(entry, SeverityError, "directive must be inside a section"))
				continue
			}
			if !sectionKnown {
				continue
			}
			if isExtensionName(entry.Key) {
				continue
			}
			directive, ok := catalog.Directive(section, entry.Key)
			if !ok {
				message := fmt.Sprintf("unknown directive %s in [%s]", entry.Key, section)
				if catalog.Version != "" {
					message += fmt.Sprintf(" (catalog %s)", catalog.Version)
				}
				diagnostics = append(diagnostics, diagnostic(entry, SeverityWarning, message))
				continue
			}
			if !directive.Multiple {
				key := section + "." + directive.Name
				if firstLine, ok := seenSingletons[key]; ok {
					diagnostics = append(diagnostics, diagnostic(entry, SeverityWarning, fmt.Sprintf("%s was already set on line %d; systemd will use the later assignment", directive.Name, firstLine+1)))
				} else {
					seenSingletons[key] = entry.Line
				}
			}
			if message := validateDirectiveValue(directive, entry.Value); message != "" {
				diagnostics = append(diagnostics, diagnostic(entry, SeverityWarning, message))
			}
			if lastValues[section] == nil {
				lastValues[section] = map[string]Entry{}
			}
			lastValues[section][entry.Key] = entry
		}
	}
	// Only the last assignment of each setting takes effect.
	if service := lastValues["Service"]; unitType == "service" && strings.EqualFold(service["Type"].Value, "forking") && service["PIDFile"].Value == "" {
		message := "Type=forking should specify PIDFile= so systemd can reliably track the main process"
		if dropIn {
			diagnostics = append(diagnostics, diagnostic(service["Type"], SeverityHint, message+"; it is not set in this drop-in (it may be supplied by the main unit file or another drop-in)"))
		} else {
			diagnostics = append(diagnostics, diagnostic(service["Type"], SeverityWarning, message))
		}
	}
	for _, name := range catalog.SectionsFor(unitType) {
		values := lastValues[name]
		partition, ok := values["CPUSetPartition"]
		if ok && contains([]string{"member", "root", "isolated"}, partition.Value) && values["AllowedCPUs"].Value == "" {
			diagnostics = append(diagnostics, diagnostic(partition, SeverityHint, "CPUSetPartition= requires AllowedCPUs=; it is not set in this document (it may be supplied by another unit file or drop-in)"))
		}
	}
	return diagnostics
}

func isExtensionName(name string) bool {
	return strings.HasPrefix(name, "X-")
}

func diagnostic(entry Entry, severity int, message string) Diagnostic {
	start, end := entry.KeyStart, entry.KeyEnd
	if start == end {
		start, end = 0, len(entry.Raw)
	}
	return Diagnostic{
		Range: Range{
			Start: Position{Line: entry.Line, Character: UTF16Length(entry.Raw[:start])},
			End:   Position{Line: entry.Line, Character: UTF16Length(entry.Raw[:end])},
		},
		Severity: severity,
		Source:   "systemd-lsp",
		Message:  message,
	}
}

func shouldValidateValue(directive Directive, value string) bool {
	if value == "" || strings.ContainsAny(value, " \t:/") {
		return false
	}
	for _, boolValue := range boolValues {
		if strings.EqualFold(value, boolValue) {
			return true
		}
	}
	switch directive.Name {
	case "Type", "ExitType", "Restart", "NotifyAccess", "OOMPolicy", "CollectMode", "OnFailureJobMode", "OnSuccessJobMode":
		return true
	default:
		return isBoolDirective(directive)
	}
}

func isBoolDirective(directive Directive) bool {
	if len(directive.Values) != len(boolValues) {
		return false
	}
	for _, value := range directive.Values {
		if !containsFold(boolValues, value) {
			return false
		}
	}
	return true
}

func containsFold(values []string, needle string) bool {
	for _, value := range values {
		if strings.EqualFold(value, needle) {
			return true
		}
	}
	return false
}
