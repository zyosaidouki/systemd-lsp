package systemd

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestCatalogJSONRejectsAmbiguousAndTrailingInput(t *testing.T) {
	for _, input := range []string{
		`{"version":"v258","version":"v262","directives":[]}`,
		`{"directives":[{"section":"Service","name":"Type","name":"ExecStart"}]}`,
		"{\"directives\":[{\"name\":\"\xff\"}]}",
		`{"directives":[{"name":"\ud800"}]}`,
		`{"directives":[]} {"version":"v262"}`,
		`{"directives":[]} trailing garbage`,
	} {
		if _, err := DecodeCatalogFile(strings.NewReader(input)); err == nil {
			t.Fatalf("accepted invalid catalog %q", input)
		}
	}
	file, err := DecodeCatalogFile(strings.NewReader("{\"version\":\"v261\",\"directives\":[],\"futureMetadata\":{}} \n\t"))
	if err != nil || file.Version != "v261" {
		t.Fatalf("valid catalog with extension metadata: %#v, %v", file, err)
	}
}

func TestCatalogJSONPreservesGeneratedFormat(t *testing.T) {
	file := CatalogFile{Version: "v261", Directives: []CatalogDirective{{Section: "Service", Name: "Example", Doc: "日本 <value> &\u2028"}}}
	var buf bytes.Buffer
	if err := EncodeCatalogFile(&buf, file); err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"version\": \"v261\",\n  \"directives\": [\n    {\n      \"section\": \"Service\",\n      \"name\": \"Example\",\n      \"doc\": \"日本 \\u003cvalue\\u003e \\u0026\\u2028\"\n    }\n  ]\n}\n"
	if buf.String() != want {
		t.Fatalf("catalog formatting changed:\n%s\nwant:\n%s", buf.String(), want)
	}
	decoded, err := DecodeCatalogFile(&buf)
	if err != nil || decoded.Directives[0].Doc != file.Directives[0].Doc {
		t.Fatalf("round trip lost documentation: %#v, %v", decoded, err)
	}
	buf.Reset()
	if err := EncodeCatalogFile(&buf, CatalogFile{}); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "{\n  \"directives\": []\n}\n" {
		t.Fatalf("nil directives must encode as an empty array: %s", buf.String())
	}
}

type failedJSONIO struct{}

func (failedJSONIO) Read([]byte) (int, error)  { return 0, io.ErrUnexpectedEOF }
func (failedJSONIO) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestCatalogJSONPropagatesIOErrors(t *testing.T) {
	if _, err := DecodeCatalogFile(failedJSONIO{}); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("reader error = %v", err)
	}
	if err := EncodeCatalogFile(failedJSONIO{}, CatalogFile{}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("writer error = %v", err)
	}
}
