package systemd

import (
	"strings"
	"testing"
)

func TestParseLoadFragmentGperfExpandsMacros(t *testing.T) {
	source := `{%- macro EXEC_CONTEXT_CONFIG_ITEMS(type) -%}
{{type}}.DynamicUser,                         config_parse_bool,                                  true,                               offsetof({{type}}, exec_context.dynamic_user)
{{type}}.Environment,                         config_parse_environ,                               0,                                  offsetof({{type}}, exec_context.environment)
{%- endmacro -%}
%%
Unit.Description,                             config_parse_unit_string_printf,                    0,                                  offsetof(Unit, description)
Service.Type,                                 config_parse_service_type,                          0,                                  offsetof(Service, type)
Mount.Type,                                   config_parse_unit_string_printf,                    0,                                  offsetof(Mount, parameters_fragment.fstype)
{{ EXEC_CONTEXT_CONFIG_ITEMS('Service') }}
Install.WantedBy,                             NULL,                                               0,                                  0
`
	file, err := ParseLoadFragmentGperf(strings.NewReader(source))
	if err != nil {
		t.Fatal(err)
	}
	directives := map[string]CatalogDirective{}
	for _, directive := range file.Directives {
		directives[directive.Section+"."+directive.Name] = directive
	}
	for _, key := range []string{"Unit.Description", "Service.Type", "Mount.Type", "Service.DynamicUser", "Service.Environment", "Install.WantedBy"} {
		if _, ok := directives[key]; !ok {
			t.Fatalf("missing parsed directive %s in %#v", key, directives)
		}
	}
	if got := directives["Service.DynamicUser"].Values; !contains(got, "true") || !contains(got, "false") {
		t.Fatalf("DynamicUser values = %#v, want bool values", got)
	}
	if !directives["Service.Environment"].Multiple {
		t.Fatal("Environment should be marked as multiple")
	}
	if directives["Service.Type"].ValueKind != "string" {
		t.Fatalf("Service.Type valueKind = %q, want string", directives["Service.Type"].ValueKind)
	}
	if !contains(directives["Service.Type"].Values, "forking") {
		t.Fatalf("Service.Type values = %#v, want service type values", directives["Service.Type"].Values)
	}
	if len(directives["Mount.Type"].Values) != 0 {
		t.Fatalf("Mount.Type values = %#v, want no service type values", directives["Mount.Type"].Values)
	}
}

func TestMergeCatalogFileAddsGeneratedDirective(t *testing.T) {
	catalog := NewCatalog()
	catalog.MergeCatalogFile(CatalogFile{
		Directives: []CatalogDirective{
			{
				Section:   "Service",
				Name:      "RestartMode",
				Parser:    "config_parse_service_restart_mode",
				ValueKind: "string",
				Values:    []string{"normal", "direct", "debug"},
			},
		},
	})

	directive, ok := catalog.Directive("Service", "RestartMode")
	if !ok {
		t.Fatal("missing generated RestartMode directive")
	}
	if directive.Parser != "config_parse_service_restart_mode" {
		t.Fatalf("parser = %q, want config_parse_service_restart_mode", directive.Parser)
	}
	if !contains(directive.Values, "debug") {
		t.Fatalf("values = %#v, want debug", directive.Values)
	}
}
