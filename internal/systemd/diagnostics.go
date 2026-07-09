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
	Severity int    `json:"severity,omitempty"`
	Source   string `json:"source,omitempty"`
	Message  string `json:"message"`
}

const (
	SeverityError   = 1
	SeverityWarning = 2
	SeverityHint    = 4
)

func Diagnostics(catalog *Catalog, text, unitType string) []Diagnostic {
	doc := Parse(text)
	diagnostics := make([]Diagnostic, 0)
	section := ""
	sectionKnown := false
	seenSingletons := map[string]int{}
	serviceChecks := newServiceChecks()

	for _, entry := range doc.Entries {
		switch entry.Kind {
		case EntryBlank, EntryComment:
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
				diagnostics = append(diagnostics, diagnostic(entry, SeverityWarning, fmt.Sprintf("unknown directive %s in [%s]", entry.Key, section)))
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
			if len(directive.Values) > 0 && shouldValidateValue(directive, entry.Value) && !containsFold(directive.Values, entry.Value) {
				diagnostics = append(diagnostics, diagnostic(entry, SeverityWarning, fmt.Sprintf("unexpected value %q for %s", entry.Value, directive.Name)))
			}
			if unitType == "service" && section == "Service" {
				serviceChecks.record(entry)
			}
		}
	}
	diagnostics = append(diagnostics, serviceChecks.diagnostics()...)
	return diagnostics
}

func isExtensionName(name string) bool {
	return strings.HasPrefix(name, "X-")
}

type serviceChecks struct {
	forkingType *Entry
	hasPIDFile  bool
}

func newServiceChecks() *serviceChecks {
	return &serviceChecks{}
}

func (c *serviceChecks) record(entry Entry) {
	switch entry.Key {
	case "Type":
		if strings.EqualFold(entry.Value, "forking") {
			c.forkingType = &entry
		}
	case "PIDFile":
		if strings.TrimSpace(entry.Value) != "" {
			c.hasPIDFile = true
		}
	}
}

func (c *serviceChecks) diagnostics() []Diagnostic {
	if c.forkingType == nil || c.hasPIDFile {
		return nil
	}
	return []Diagnostic{
		diagnostic(*c.forkingType, SeverityWarning, "Type=forking should specify PIDFile= so systemd can reliably track the main process"),
	}
}

func diagnostic(entry Entry, severity int, message string) Diagnostic {
	start, end := entry.KeyStart, entry.KeyEnd
	if start == end {
		start, end = 0, len(entry.Raw)
	}
	return Diagnostic{
		Range: Range{
			Start: Position{Line: entry.Line, Character: start},
			End:   Position{Line: entry.Line, Character: end},
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
