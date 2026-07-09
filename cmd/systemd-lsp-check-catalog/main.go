package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/zyosaidouki/systemd-lsp/internal/systemd"
)

func main() {
	minDirectives := flag.Int("min-directives", 1, "minimum number of directives required")
	requireDocs := flag.Bool("require-docs", false, "fail if any directive has no documentation")
	flag.Parse()
	if flag.NArg() != 1 {
		log.Fatalf("usage: systemd-lsp-check-catalog [-min-directives N] [-require-docs] catalog.json")
	}

	file, err := systemd.LoadCatalogFile(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	stats, errs := systemd.ValidateCatalogFile(file, systemd.CatalogValidationOptions{
		MinDirectives: *minDirectives,
		RequireDocs:   *requireDocs,
	})
	fmt.Printf("version: %s\n", file.Version)
	fmt.Printf("directives: %d\n", stats.Directives)
	fmt.Printf("sections: %d\n", stats.Sections)
	fmt.Printf("with parser: %d\n", stats.WithParser)
	fmt.Printf("with value kind: %d\n", stats.WithValueKind)
	fmt.Printf("documented: %d\n", stats.Documented)
	fmt.Printf("with man page: %d\n", stats.WithManPage)
	fmt.Printf("duplicates: %d\n", stats.DuplicateEntries)

	if len(errs) == 0 {
		return
	}
	for _, err := range errs {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}
	os.Exit(1)
}
