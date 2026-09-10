package xsd

import (
	"encoding/xml"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"unicode/utf8"
)

const (
	schemaNamespace = "http://www.w3.org/2001/XMLSchema"
	unbounded       = -1
)

type node struct {
	XMLName    xml.Name
	Attributes []xml.Attr `xml:",any,attr"`
	Children   []node     `xml:",any"`
}

func (n node) attribute(name string) string {
	for _, attribute := range n.Attributes {
		if attribute.Name.Local == name && attribute.Name.Space == "" {
			return attribute.Value
		}
	}
	return ""
}

func (n node) is(name string) bool {
	return n.XMLName.Space == schemaNamespace && n.XMLName.Local == name
}

func (n node) elements() []node {
	kept := make([]node, 0, len(n.Children))
	for _, child := range n.Children {
		if child.XMLName.Space == schemaNamespace && !child.is("annotation") {
			kept = append(kept, child)
		}
	}
	return kept
}

func (n node) scope(parent map[string]string) map[string]string {
	declarations := map[string]string{}
	for _, attribute := range n.Attributes {
		switch {
		case attribute.Name.Space == "xmlns":
			declarations[attribute.Name.Local] = attribute.Value
		case attribute.Name.Space == "" && attribute.Name.Local == "xmlns":
			declarations[""] = attribute.Value
		}
	}
	if len(declarations) == 0 {
		return parent
	}
	scoped := copyMap(parent)
	for prefix, uri := range declarations {
		scoped[prefix] = uri
	}
	return scoped
}

func copyMap(source map[string]string) map[string]string {
	copied := make(map[string]string, len(source)+1)
	for key, value := range source {
		copied[key] = value
	}
	return copied
}

func resolve(value string, scope map[string]string) xml.Name {
	if value == "" {
		return xml.Name{}
	}
	prefix, local := "", value
	if index := strings.Index(value, ":"); index >= 0 {
		prefix, local = value[:index], value[index+1:]
	}
	return xml.Name{Space: scope[prefix], Local: local}
}

type document struct {
	file            string
	root            node
	targetNamespace string
	includes        []string
	roots           []string
}

var versionPattern = regexp.MustCompile(`_v([0-9]+\.[0-9]+)\.xsd$`)

func versionOf(file string) string {
	match := versionPattern.FindStringSubmatch(strings.ToLower(file))
	if match == nil {
		return ""
	}
	return match[1]
}

type entry struct {
	version string
	file    string
}

type Set struct {
	documents map[string]*document
	byName    map[string]string
	roots     map[string][]entry
	units     map[string]*unit
	mutex     sync.Mutex
}

func (s *Set) locate(from, location string) string {
	candidate := filepath.Join(filepath.Dir(from), filepath.Base(location))
	if _, found := s.documents[candidate]; found {
		return candidate
	}
	return s.byName[filepath.Base(location)]
}

func Load(directory string) (*Set, error) {
	set := &Set{
		documents: map[string]*document{},
		byName:    map[string]string{},
		roots:     map[string][]entry{},
		units:     map[string]*unit{},
	}
	err := filepath.WalkDir(directory, func(path string, item fs.DirEntry, err error) error {
		if err != nil || item.IsDir() || !strings.EqualFold(filepath.Ext(path), ".xsd") {
			return err
		}
		parsed, parseError := parseDocument(path)
		if parseError != nil {
			return parseError
		}
		set.documents[path] = parsed
		set.byName[filepath.Base(path)] = path
		for _, root := range parsed.roots {
			set.roots[root] = append(set.roots[root], entry{version: versionOf(path), file: path})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return set, nil
}

func parseDocument(path string) (*document, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var root node
	if err := xml.Unmarshal(toUTF8(content), &root); err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	parsed := &document{file: path, root: root, targetNamespace: root.attribute("targetNamespace")}
	for _, child := range root.elements() {
		switch {
		case child.is("include"), child.is("import"):
			if location := child.attribute("schemaLocation"); location != "" {
				parsed.includes = append(parsed.includes, location)
			}
		case child.is("element"):
			if name := child.attribute("name"); name != "" {
				parsed.roots = append(parsed.roots, name)
			}
		}
	}
	return parsed, nil
}

func (s *Set) Roots() []string {
	names := make([]string, 0, len(s.roots))
	for name := range s.roots {
		names = append(names, name)
	}
	return names
}

func (s *Set) Documents() int {
	return len(s.documents)
}

func (s *Set) Knows(root string) bool {
	_, found := s.roots[root]
	return found
}

func (s *Set) pick(root, version string) (entry, bool) {
	candidates, found := s.roots[root]
	if !found || len(candidates) == 0 {
		return entry{}, false
	}
	best := candidates[0]
	for _, candidate := range candidates {
		if candidate.version == version {
			return candidate, true
		}
		if candidate.version > best.version {
			best = candidate
		}
	}
	return best, true
}

func toUTF8(content []byte) []byte {
	if utf8.Valid(content) {
		return content
	}
	converted := make([]rune, 0, len(content))
	for _, character := range content {
		converted = append(converted, rune(character))
	}
	return []byte(string(converted))
}
