package systemd

import "testing"

func TestValidateCatalogFileStats(t *testing.T) {
	file := CatalogFile{
		Directives: []CatalogDirective{
			{Section: "Service", Name: "ExecStart", Parser: "config_parse_exec", ValueKind: "command", Doc: "Starts the service.", ManPage: "systemd.service(5)"},
			{Section: "Unit", Name: "Description", Parser: "config_parse_unit_string_printf", ValueKind: "string"},
		},
	}
	stats, errs := ValidateCatalogFile(file, CatalogValidationOptions{MinDirectives: 2})
	if len(errs) != 0 {
		t.Fatalf("errs = %#v, want none", errs)
	}
	if stats.Directives != 2 || stats.Sections != 2 || stats.Documented != 1 || stats.WithManPage != 1 {
		t.Fatalf("stats = %#v", stats)
	}
}

func TestValidateCatalogFileFindsProblems(t *testing.T) {
	file := CatalogFile{
		Directives: []CatalogDirective{
			{Section: "Service", Name: "ExecStart"},
			{Section: "Service", Name: "ExecStart"},
			{Section: "", Name: "Broken"},
		},
	}
	_, errs := ValidateCatalogFile(file, CatalogValidationOptions{MinDirectives: 4, RequireDocs: true})
	if len(errs) < 4 {
		t.Fatalf("errs = %#v, want duplicate, empty key, missing docs, and min count errors", errs)
	}
}
