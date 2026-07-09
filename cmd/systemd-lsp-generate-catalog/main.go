package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/zyosaidouki/systemd-lsp/internal/systemd"
)

func main() {
	version := flag.String("version", "", "systemd version or git tag represented by this catalog")
	manDir := flag.String("man-dir", "", "directory containing systemd man/*.xml files used to enrich documentation")
	flag.Parse()

	var in io.Reader = os.Stdin
	if flag.NArg() > 1 {
		log.Fatalf("usage: systemd-lsp-generate-catalog [-version vNNN] [load-fragment-gperf.gperf.in]")
	}
	if flag.NArg() == 1 {
		f, err := os.Open(flag.Arg(0))
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		in = f
	}

	catalog, err := systemd.ParseLoadFragmentGperf(in)
	if err != nil {
		log.Fatal(err)
	}
	catalog.Version = *version
	if *manDir != "" {
		catalog, err = systemd.EnrichCatalogWithManDir(catalog, *manDir)
		if err != nil {
			log.Fatal(err)
		}
	}
	if err := systemd.EncodeCatalogFile(os.Stdout, catalog); err != nil {
		log.Fatal(err)
	}
	if len(catalog.Directives) == 0 {
		fmt.Fprintln(os.Stderr, "warning: generated catalog contains no directives")
	}
}
