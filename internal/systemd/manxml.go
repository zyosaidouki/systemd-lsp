package systemd

import (
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type ManDirectiveDoc struct {
	Name    string
	Doc     string
	Syntax  string
	Example string
	ManPage string
}

type ManDirectiveDocs map[string][]ManDirectiveDoc

func EnrichCatalogWithManDir(file CatalogFile, dir string) (CatalogFile, error) {
	docs, err := LoadManXMLDocs(dir)
	if err != nil {
		return CatalogFile{}, err
	}
	return EnrichCatalogWithManDocs(file, docs), nil
}

func LoadManXMLDocs(dir string) (ManDirectiveDocs, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "systemd.*.xml"))
	if err != nil {
		return nil, err
	}
	docs := ManDirectiveDocs{}
	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		pageDocs, parseErr := ParseManXMLDocs(f, filepath.Base(path))
		closeErr := f.Close()
		if parseErr != nil {
			return nil, parseErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		for name, entries := range pageDocs {
			docs[name] = append(docs[name], entries...)
		}
	}
	return docs, nil
}

func EnrichCatalogWithManDocs(file CatalogFile, docs ManDirectiveDocs) CatalogFile {
	for i := range file.Directives {
		doc, ok := selectManDoc(file.Directives[i].Section, file.Directives[i].Name, docs)
		if !ok {
			continue
		}
		if file.Directives[i].Doc == "" {
			file.Directives[i].Doc = doc.Doc
		}
		if file.Directives[i].Syntax == "" {
			file.Directives[i].Syntax = doc.Syntax
		}
		if file.Directives[i].Example == "" {
			file.Directives[i].Example = doc.Example
		}
		if file.Directives[i].ManPage == "" {
			file.Directives[i].ManPage = doc.ManPage
		}
	}
	return file
}

func ParseManXMLDocs(r io.Reader, filename string) (ManDirectiveDocs, error) {
	page := manPageName(filename)
	decoder := xml.NewDecoder(r)
	docs := ManDirectiveDocs{}
	var entry *manEntry
	stack := []string{}

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch tok := token.(type) {
		case xml.StartElement:
			name := tok.Name.Local
			stack = append(stack, name)
			if name == "varlistentry" {
				entry = &manEntry{ManPage: page}
			}
			if entry == nil {
				continue
			}
			switch name {
			case "term":
				entry.inTerm = true
				entry.term.Reset()
			case "para", "simpara":
				if entry.inListItem && entry.Doc == "" {
					entry.inDocPara = true
					entry.para.Reset()
				}
			case "programlisting":
				if entry.Example == "" {
					entry.inProgramListing = true
					entry.programListing.Reset()
				}
			}
		case xml.EndElement:
			name := tok.Name.Local
			if entry != nil {
				switch name {
				case "term":
					entry.addTerm(entry.term.String())
					entry.inTerm = false
				case "listitem":
					entry.inListItem = false
				case "para", "simpara":
					if entry.inDocPara {
						entry.Doc = cleanXMLText(entry.para.String())
						entry.inDocPara = false
					}
				case "programlisting":
					if entry.inProgramListing {
						entry.Example = cleanProgramListing(entry.programListing.String())
						entry.inProgramListing = false
					}
				case "varlistentry":
					for _, doc := range entry.docs() {
						docs[doc.Name] = append(docs[doc.Name], doc)
					}
					entry = nil
				}
			}
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			if entry == nil {
				continue
			}
			text := string(tok)
			if entry.inTerm {
				entry.term.WriteString(text)
			}
			if entry.inDocPara {
				entry.para.WriteString(text)
			}
			if entry.inProgramListing {
				entry.programListing.WriteString(text)
			}
		}
		if entry != nil && len(stack) > 0 && stack[len(stack)-1] == "listitem" {
			entry.inListItem = true
		}
	}
	return docs, nil
}

func selectManDoc(section, name string, docs ManDirectiveDocs) (ManDirectiveDoc, bool) {
	candidates := docs[name]
	if len(candidates) == 0 {
		return ManDirectiveDoc{}, false
	}
	for _, preferred := range preferredManPages(section) {
		for _, candidate := range candidates {
			if candidate.ManPage == preferred {
				return candidate, true
			}
		}
	}
	return candidates[0], true
}

func preferredManPages(section string) []string {
	switch section {
	case "Unit", "Install":
		return []string{"systemd.unit(5)"}
	case "Service":
		return []string{"systemd.service(5)", "systemd.exec(5)", "systemd.kill(5)", "systemd.resource-control(5)"}
	case "Socket":
		return []string{"systemd.socket(5)", "systemd.exec(5)", "systemd.kill(5)", "systemd.resource-control(5)"}
	case "Timer":
		return []string{"systemd.timer(5)"}
	case "Path":
		return []string{"systemd.path(5)"}
	case "Mount":
		return []string{"systemd.mount(5)", "systemd.exec(5)", "systemd.kill(5)", "systemd.resource-control(5)"}
	case "Automount":
		return []string{"systemd.automount(5)"}
	case "Swap":
		return []string{"systemd.swap(5)", "systemd.exec(5)", "systemd.kill(5)", "systemd.resource-control(5)"}
	case "Slice":
		return []string{"systemd.slice(5)", "systemd.resource-control(5)"}
	case "Scope":
		return []string{"systemd.scope(5)", "systemd.kill(5)", "systemd.resource-control(5)"}
	default:
		return nil
	}
}

type manEntry struct {
	Names            []string
	Syntax           map[string]string
	Doc              string
	Example          string
	ManPage          string
	inTerm           bool
	inListItem       bool
	inDocPara        bool
	inProgramListing bool
	term             strings.Builder
	para             strings.Builder
	programListing   strings.Builder
}

var directiveInTermRE = regexp.MustCompile(`([A-Za-z][A-Za-z0-9]+)=`)

func (e *manEntry) addTerm(term string) {
	if e.Syntax == nil {
		e.Syntax = map[string]string{}
	}
	term = cleanXMLText(term)
	matches := directiveInTermRE.FindAllStringSubmatch(term, -1)
	for _, match := range matches {
		name := match[1]
		if !contains(e.Names, name) {
			e.Names = append(e.Names, name)
		}
		if e.Syntax[name] == "" {
			e.Syntax[name] = term
		}
	}
}

func (e *manEntry) docs() []ManDirectiveDoc {
	sort.Strings(e.Names)
	result := make([]ManDirectiveDoc, 0, len(e.Names))
	for _, name := range e.Names {
		result = append(result, ManDirectiveDoc{
			Name:    name,
			Doc:     e.Doc,
			Syntax:  e.Syntax[name],
			Example: e.exampleFor(name),
			ManPage: e.ManPage,
		})
	}
	return result
}

func (e *manEntry) exampleFor(name string) string {
	if e.Example == "" || !strings.Contains(e.Example, name+"=") {
		return ""
	}
	lines := strings.Split(e.Example, "\n")
	for _, line := range lines {
		if strings.Contains(line, name+"=") {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

func cleanXMLText(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func cleanProgramListing(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	return strings.Join(lines, "\n")
}

func manPageName(filename string) string {
	name := strings.TrimSuffix(filename, ".xml")
	if name == "" {
		return ""
	}
	return name + "(5)"
}
