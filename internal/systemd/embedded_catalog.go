package systemd

import (
	"bytes"
	_ "embed"
)

//go:embed catalogdata/load-fragment-main.json
var embeddedLoadFragmentCatalog []byte

func mergeEmbeddedCatalog(c *Catalog) {
	file, err := DecodeCatalogFile(bytes.NewReader(embeddedLoadFragmentCatalog))
	if err != nil {
		return
	}
	c.MergeCatalogFile(file)
}
