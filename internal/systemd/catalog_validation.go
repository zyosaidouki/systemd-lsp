package systemd

import (
	"fmt"
)

type CatalogStats struct {
	Directives       int
	Documented       int
	WithParser       int
	WithValueKind    int
	WithManPage      int
	Sections         int
	DuplicateEntries int
}

type CatalogValidationOptions struct {
	MinDirectives int
	RequireDocs   bool
}

func ValidateCatalogFile(file CatalogFile, opts CatalogValidationOptions) (CatalogStats, []error) {
	stats := CatalogStats{}
	sections := map[string]bool{}
	seen := map[string]bool{}
	var errs []error

	for _, directive := range file.Directives {
		stats.Directives++
		key := directive.Section + "." + directive.Name
		if directive.Section == "" || directive.Name == "" {
			errs = append(errs, fmt.Errorf("directive %d has empty section or name", stats.Directives))
			continue
		}
		if seen[key] {
			stats.DuplicateEntries++
			errs = append(errs, fmt.Errorf("duplicate directive %s", key))
		}
		seen[key] = true
		sections[directive.Section] = true
		if directive.Doc != "" {
			stats.Documented++
		} else if opts.RequireDocs {
			errs = append(errs, fmt.Errorf("%s is missing documentation", key))
		}
		if directive.Parser != "" {
			stats.WithParser++
		}
		if directive.ValueKind != "" {
			stats.WithValueKind++
		}
		if directive.ManPage != "" {
			stats.WithManPage++
		}
	}
	stats.Sections = len(sections)

	if opts.MinDirectives > 0 && stats.Directives < opts.MinDirectives {
		errs = append(errs, fmt.Errorf("catalog has %d directives, want at least %d", stats.Directives, opts.MinDirectives))
	}
	if stats.Directives == 0 {
		errs = append(errs, fmt.Errorf("catalog has no directives"))
	}
	return stats, errs
}
