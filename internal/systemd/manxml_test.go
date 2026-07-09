package systemd

import (
	"strings"
	"testing"
)

func TestParseManXMLDocsExtractsDirectiveDocs(t *testing.T) {
	source := `<?xml version="1.0"?>
<refentry>
  <refsect1>
    <variablelist>
      <varlistentry>
        <term><varname>ExecStart=</varname></term>
        <listitem>
          <para>Commands that are executed when this service is started.</para>
          <programlisting>ExecStart=/usr/bin/example --foreground</programlisting>
        </listitem>
      </varlistentry>
      <varlistentry>
        <term><varname>MemoryMax=<replaceable>bytes</replaceable></varname></term>
        <term><varname>StartupMemoryMax=<replaceable>bytes</replaceable></varname></term>
        <listitem>
          <para>Specify the absolute limit on memory usage of the executed processes.</para>
        </listitem>
      </varlistentry>
    </variablelist>
  </refsect1>
</refentry>`
	docs, err := ParseManXMLDocs(strings.NewReader(source), "systemd.service.xml")
	if err != nil {
		t.Fatal(err)
	}
	execStart := docs["ExecStart"][0]
	if execStart.Doc != "Commands that are executed when this service is started." {
		t.Fatalf("ExecStart doc = %q", execStart.Doc)
	}
	if execStart.Syntax != "ExecStart=" {
		t.Fatalf("ExecStart syntax = %q", execStart.Syntax)
	}
	if execStart.Example != "ExecStart=/usr/bin/example --foreground" {
		t.Fatalf("ExecStart example = %q", execStart.Example)
	}
	if execStart.ManPage != "systemd.service(5)" {
		t.Fatalf("ExecStart manPage = %q", execStart.ManPage)
	}
	if docs["MemoryMax"][0].Syntax != "MemoryMax=bytes" {
		t.Fatalf("MemoryMax syntax = %q", docs["MemoryMax"][0].Syntax)
	}
	if docs["StartupMemoryMax"][0].Doc == "" {
		t.Fatal("StartupMemoryMax doc should be populated from shared varlistentry")
	}
}

func TestEnrichCatalogWithManDocs(t *testing.T) {
	file := CatalogFile{
		Directives: []CatalogDirective{
			{Section: "Service", Name: "ExecStart", Parser: "config_parse_exec", ValueKind: "command"},
		},
	}
	file = EnrichCatalogWithManDocs(file, ManDirectiveDocs{
		"ExecStart": {
			{
				Doc:     "Commands that are executed when this service is started.",
				Syntax:  "ExecStart=",
				Example: "ExecStart=/usr/bin/example --foreground",
				ManPage: "systemd.service(5)",
			},
		},
	})
	got := file.Directives[0]
	if got.Doc == "" || got.Syntax == "" || got.Example == "" || got.ManPage == "" {
		t.Fatalf("directive was not enriched: %#v", got)
	}
}

func TestEnrichCatalogWithManDocsPrefersSectionPage(t *testing.T) {
	file := CatalogFile{
		Directives: []CatalogDirective{
			{Section: "Mount", Name: "Where"},
			{Section: "Automount", Name: "Where"},
		},
	}
	file = EnrichCatalogWithManDocs(file, ManDirectiveDocs{
		"Where": {
			{Doc: "Automount point path.", Syntax: "Where=", ManPage: "systemd.automount(5)"},
			{Doc: "Mount point path.", Syntax: "Where=", ManPage: "systemd.mount(5)"},
		},
	})
	if file.Directives[0].Doc != "Mount point path." {
		t.Fatalf("Mount Where doc = %q", file.Directives[0].Doc)
	}
	if file.Directives[1].Doc != "Automount point path." {
		t.Fatalf("Automount Where doc = %q", file.Directives[1].Doc)
	}
}
