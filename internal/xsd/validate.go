package xsd

import (
	"encoding/xml"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const failureLimit = 25

var ErrRootUnknown = errors.New("no schema declares this root element")

type Failure struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

func (f Failure) String() string {
	return f.Path + ": " + f.Message
}

type instance struct {
	XMLName    xml.Name
	Attributes []xml.Attr `xml:",any,attr"`
	Children   []instance `xml:",any"`
	Text       string     `xml:",chardata"`
}

func (s *Set) Validate(root, version string, document []byte) ([]Failure, error) {
	chosen, found := s.pick(root, version)
	if !found {
		return nil, ErrRootUnknown
	}
	compiled := s.unitFor(chosen.file)
	var value instance
	if err := xml.Unmarshal(toUTF8(document), &value); err != nil {
		return []Failure{{Path: root, Message: "documento XML malformado"}}, nil
	}
	declaration, known := compiled.elements[value.XMLName]
	if !known {
		declaration, known = compiled.elements[xml.Name{Space: value.XMLName.Space, Local: root}]
	}
	if !known {
		return nil, ErrRootUnknown
	}
	checker := &validator{unit: compiled}
	checker.element(value.XMLName.Local, declaration, value)
	return checker.failures, nil
}

type validator struct {
	unit     *unit
	failures []Failure
}

func (v *validator) fail(path, format string, arguments ...any) {
	if len(v.failures) >= failureLimit {
		return
	}
	v.failures = append(v.failures, Failure{Path: path, Message: fmt.Sprintf(format, arguments...)})
}

func (v *validator) full() bool {
	return len(v.failures) >= failureLimit
}

func (v *validator) element(path string, declaration *elementDecl, value instance) {
	if v.full() {
		return
	}
	if declaration.reference != (xml.Name{}) {
		referenced, found := v.unit.elements[declaration.reference]
		if !found {
			return
		}
		declaration = referenced
	}
	definition := v.typeOf(declaration)
	if definition == nil {
		if declaration.typeRef.Space == schemaNamespace {
			v.builtin(path, declaration.typeRef.Local, strings.TrimSpace(value.Text))
		}
		return
	}
	v.attributes(path, definition, value)
	if definition.content != nil {
		v.content(path, definition, value)
		return
	}
	text := strings.TrimSpace(value.Text)
	if definition.simple != nil {
		v.simple(path, definition.simple, text)
		return
	}
	if definition.simpleBase != (xml.Name{}) {
		v.simpleByName(path, definition.simpleBase, text)
	}
}

func (v *validator) typeOf(declaration *elementDecl) *typeDecl {
	if declaration.inlineType != nil {
		return declaration.inlineType
	}
	if declaration.typeRef == (xml.Name{}) {
		return nil
	}
	if definition, found := v.unit.types[declaration.typeRef]; found {
		return definition
	}
	return nil
}

func (v *validator) attributes(path string, definition *typeDecl, value instance) {
	for _, declaration := range definition.attributes {
		found := false
		for _, attribute := range value.Attributes {
			if attribute.Name.Local != declaration.name || attribute.Name.Space == "xmlns" {
				continue
			}
			found = true
			if declaration.inline != nil {
				v.simple(path+"/@"+declaration.name, declaration.inline, attribute.Value)
				break
			}
			v.simpleByName(path+"/@"+declaration.name, declaration.typeRef, attribute.Value)
			break
		}
		if !found && declaration.required {
			v.fail(path, "atributo obrigatorio %q ausente", declaration.name)
		}
	}
}

func (v *validator) content(path string, definition *typeDecl, value instance) {
	report := &mismatch{index: -1}
	consumed, bindings, matched := repeat(definition.content, value.Children, 0, report)
	if !matched {
		v.fail(path, "conteudo nao satisfaz o modelo declarado%s", report.describe(value.Children))
		return
	}
	if consumed < len(value.Children) {
		v.fail(path+"/"+value.Children[consumed].XMLName.Local, "elemento inesperado nesta posicao")
		return
	}
	for _, bound := range bindings {
		child := value.Children[bound.index]
		v.element(path+"/"+child.XMLName.Local, bound.declaration, child)
	}
}

type mismatch struct {
	index int
	names []string
}

func (m *mismatch) note(index int, names []string) {
	if index <= m.index || len(names) == 0 {
		return
	}
	m.index = index
	m.names = names
}

func (m *mismatch) merge(other *mismatch) {
	m.note(other.index, other.names)
}

func (m *mismatch) describe(children []instance) string {
	if len(m.names) == 0 {
		return ""
	}
	found := "fim do elemento"
	if m.index >= 0 && m.index < len(children) {
		found = children[m.index].XMLName.Local
	}
	return ", esperava " + strings.Join(m.names, " ou ") + " e encontrou " + found
}

func firstNames(group *particle) []string {
	switch group.kind {
	case "element":
		return []string{group.element.name.Local}
	case "any":
		return []string{"qualquer elemento"}
	}
	names := make([]string, 0, len(group.items))
	for _, item := range group.items {
		names = append(names, firstNames(item)...)
		if item.min > 0 && group.kind == "sequence" {
			break
		}
	}
	return names
}

type binding struct {
	index       int
	declaration *elementDecl
}

func matchGroup(group *particle, children []instance, start int, report *mismatch) (int, []binding, bool) {
	switch group.kind {
	case "element":
		if start < len(children) && sameName(children[start].XMLName, group.element.name) {
			return start + 1, []binding{{index: start, declaration: group.element}}, true
		}
		return start, nil, false
	case "any":
		if start < len(children) {
			return start + 1, nil, true
		}
		return start, nil, false
	case "sequence":
		index := start
		var collected []binding
		for _, item := range group.items {
			next, bound, matched := repeat(item, children, index, report)
			if !matched {
				return start, nil, false
			}
			index = next
			collected = append(collected, bound...)
		}
		return index, collected, true
	case "choice":
		for _, item := range group.items {
			if next, bound, matched := repeat(item, children, start, nil); matched && next > start {
				return next, bound, true
			}
		}
		for _, item := range group.items {
			if _, _, matched := repeat(item, children, start, nil); matched {
				return start, nil, true
			}
		}
		return start, nil, false
	}
	return start, nil, false
}

func repeat(item *particle, children []instance, start int, report *mismatch) (int, []binding, bool) {
	index := start
	var collected []binding
	count := 0
	scratch := &mismatch{index: -1}
	for item.max == unbounded || count < item.max {
		next, bound, matched := matchGroup(item, children, index, scratch)
		if !matched {
			break
		}
		collected = append(collected, bound...)
		count++
		if next == index {
			break
		}
		index = next
	}
	if count < item.min {
		if report != nil {
			report.merge(scratch)
			report.note(index, firstNames(item))
		}
		return start, nil, false
	}
	return index, collected, true
}

func sameName(left, right xml.Name) bool {
	if left.Local != right.Local {
		return false
	}
	return right.Space == "" || left.Space == right.Space
}

func (v *validator) simpleByName(path string, name xml.Name, text string) {
	if name == (xml.Name{}) {
		return
	}
	if name.Space == schemaNamespace {
		v.builtin(path, name.Local, text)
		return
	}
	definition, found := v.unit.types[name]
	if !found || definition.simple == nil {
		return
	}
	v.simple(path, definition.simple, text)
}

func (v *validator) simple(path string, declaration *simpleDecl, text string) {
	chain := v.chain(declaration)
	value := text
	for _, level := range chain {
		if level.whiteSpace == "collapse" {
			value = strings.Join(strings.Fields(value), " ")
		}
	}
	for _, level := range chain {
		v.facets(path, level, value)
		if v.full() {
			return
		}
	}
	base := chain[len(chain)-1].base
	if base.Space == schemaNamespace {
		v.builtin(path, base.Local, value)
	}
}

func (v *validator) chain(declaration *simpleDecl) []*simpleDecl {
	chain := []*simpleDecl{declaration}
	current := declaration
	for depth := 0; depth < 16; depth++ {
		if current.base.Space == schemaNamespace || current.base == (xml.Name{}) {
			break
		}
		definition, found := v.unit.types[current.base]
		if !found || definition.simple == nil {
			break
		}
		chain = append(chain, definition.simple)
		current = definition.simple
	}
	return chain
}

func (v *validator) facets(path string, declaration *simpleDecl, value string) {
	length := len([]rune(value))
	switch {
	case declaration.length >= 0 && length != declaration.length:
		v.fail(path, "deve ter exatamente %d caracteres e tem %d", declaration.length, length)
	case declaration.minLength >= 0 && length < declaration.minLength:
		v.fail(path, "deve ter ao menos %d caracteres e tem %d", declaration.minLength, length)
	case declaration.maxLength >= 0 && length > declaration.maxLength:
		v.fail(path, "deve ter no maximo %d caracteres e tem %d", declaration.maxLength, length)
	}
	if declaration.enumerations != nil && !declaration.enumerations[value] {
		v.fail(path, "valor %q fora da lista permitida", value)
	}
	if !matchesAnyPattern(declaration.patterns, value) {
		v.fail(path, "valor %q nao casa com o formato %s", value, declaration.patterns[0].String())
	}
	v.bounds(path, declaration, value)
}

func (v *validator) bounds(path string, declaration *simpleDecl, value string) {
	if declaration.minInclusive == "" && declaration.maxInclusive == "" {
		return
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return
	}
	if declaration.minInclusive != "" {
		if limit, err := strconv.ParseFloat(declaration.minInclusive, 64); err == nil && parsed < limit {
			v.fail(path, "valor %s menor que o minimo %s", value, declaration.minInclusive)
		}
	}
	if declaration.maxInclusive != "" {
		if limit, err := strconv.ParseFloat(declaration.maxInclusive, 64); err == nil && parsed > limit {
			v.fail(path, "valor %s maior que o maximo %s", value, declaration.maxInclusive)
		}
	}
}

func (v *validator) builtin(path, name, value string) {
	switch name {
	case "decimal", "double", "float":
		if _, err := strconv.ParseFloat(value, 64); err != nil && value != "" {
			v.fail(path, "valor %q nao e um decimal", value)
		}
	case "integer", "int", "long", "short", "nonNegativeInteger", "positiveInteger", "unsignedInt", "unsignedLong":
		if _, err := strconv.ParseInt(value, 10, 64); err != nil && value != "" {
			v.fail(path, "valor %q nao e um inteiro", value)
		}
	}
}

func matchesAnyPattern(patterns []*regexp.Regexp, value string) bool {
	if len(patterns) == 0 {
		return true
	}
	for _, pattern := range patterns {
		if pattern.MatchString(value) {
			return true
		}
	}
	return false
}
