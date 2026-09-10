package xsd

import (
	"encoding/xml"
	"regexp"
	"strconv"
)

type elementDecl struct {
	name       xml.Name
	reference  xml.Name
	typeRef    xml.Name
	inlineType *typeDecl
	min        int
	max        int
}

type attributeDecl struct {
	name     string
	typeRef  xml.Name
	inline   *simpleDecl
	required bool
}

type typeDecl struct {
	simple       *simpleDecl
	content      *particle
	attributes   []*attributeDecl
	anyAttribute bool
	simpleBase   xml.Name
}

type simpleDecl struct {
	base         xml.Name
	patterns     []*regexp.Regexp
	enumerations map[string]bool
	minLength    int
	maxLength    int
	length       int
	minInclusive string
	maxInclusive string
	whiteSpace   string
}

type particle struct {
	kind    string
	min     int
	max     int
	element *elementDecl
	items   []*particle
}

type unit struct {
	elements map[xml.Name]*elementDecl
	types    map[xml.Name]*typeDecl
}

func (s *Set) unitFor(file string) *unit {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if compiled, found := s.units[file]; found {
		return compiled
	}
	compiled := &unit{elements: map[xml.Name]*elementDecl{}, types: map[xml.Name]*typeDecl{}}
	visited := map[string]bool{}
	s.absorb(compiled, file, visited)
	s.units[file] = compiled
	return compiled
}

func (s *Set) absorb(target *unit, file string, visited map[string]bool) {
	if visited[file] {
		return
	}
	visited[file] = true
	parsed, found := s.documents[file]
	if !found {
		return
	}
	for _, include := range parsed.includes {
		if resolved := s.locate(file, include); resolved != "" {
			s.absorb(target, resolved, visited)
		}
	}
	scope := parsed.root.scope(map[string]string{"xs": schemaNamespace})
	namespace := parsed.targetNamespace
	for _, child := range parsed.root.elements() {
		switch {
		case child.is("element"):
			declaration := compileElement(child, scope, namespace)
			target.elements[declaration.name] = declaration
		case child.is("complexType"):
			target.types[xml.Name{Space: namespace, Local: child.attribute("name")}] = compileComplexType(child, scope, namespace)
		case child.is("simpleType"):
			target.types[xml.Name{Space: namespace, Local: child.attribute("name")}] = &typeDecl{simple: compileSimpleType(child, scope)}
		}
	}
}

func compileElement(source node, parent map[string]string, namespace string) *elementDecl {
	scope := source.scope(parent)
	declaration := &elementDecl{
		name:    xml.Name{Space: namespace, Local: source.attribute("name")},
		typeRef: resolve(source.attribute("type"), scope),
		min:     occurrence(source.attribute("minOccurs"), 1),
		max:     occurrence(source.attribute("maxOccurs"), 1),
	}
	if reference := source.attribute("ref"); reference != "" {
		declaration.reference = resolve(reference, scope)
		declaration.name = declaration.reference
	}
	for _, child := range source.elements() {
		switch {
		case child.is("complexType"):
			declaration.inlineType = compileComplexType(child, scope, namespace)
		case child.is("simpleType"):
			declaration.inlineType = &typeDecl{simple: compileSimpleType(child, scope)}
		}
	}
	return declaration
}

func compileComplexType(source node, parent map[string]string, namespace string) *typeDecl {
	scope := source.scope(parent)
	declaration := &typeDecl{}
	for _, child := range source.elements() {
		switch {
		case child.is("sequence"), child.is("choice"):
			declaration.content = compileParticle(child, scope, namespace)
		case child.is("attribute"):
			declaration.attributes = append(declaration.attributes, compileAttribute(child, scope))
		case child.is("anyAttribute"):
			declaration.anyAttribute = true
		case child.is("simpleContent"):
			for _, inner := range child.elements() {
				if !inner.is("extension") && !inner.is("restriction") {
					continue
				}
				innerScope := inner.scope(scope)
				declaration.simpleBase = resolve(inner.attribute("base"), innerScope)
				for _, attribute := range inner.elements() {
					if attribute.is("attribute") {
						declaration.attributes = append(declaration.attributes, compileAttribute(attribute, innerScope))
					}
					if attribute.is("anyAttribute") {
						declaration.anyAttribute = true
					}
				}
			}
		}
	}
	return declaration
}

func compileAttribute(source node, parent map[string]string) *attributeDecl {
	scope := source.scope(parent)
	declaration := &attributeDecl{
		name:     source.attribute("name"),
		typeRef:  resolve(source.attribute("type"), scope),
		required: source.attribute("use") == "required",
	}
	for _, child := range source.elements() {
		if child.is("simpleType") {
			declaration.inline = compileSimpleType(child, scope)
		}
	}
	return declaration
}

func compileParticle(source node, parent map[string]string, namespace string) *particle {
	scope := source.scope(parent)
	group := &particle{
		kind: source.XMLName.Local,
		min:  occurrence(source.attribute("minOccurs"), 1),
		max:  occurrence(source.attribute("maxOccurs"), 1),
	}
	for _, child := range source.elements() {
		switch {
		case child.is("element"):
			group.items = append(group.items, &particle{
				kind:    "element",
				min:     occurrence(child.attribute("minOccurs"), 1),
				max:     occurrence(child.attribute("maxOccurs"), 1),
				element: compileElement(child, scope, namespace),
			})
		case child.is("sequence"), child.is("choice"):
			group.items = append(group.items, compileParticle(child, scope, namespace))
		case child.is("any"):
			group.items = append(group.items, &particle{
				kind: "any",
				min:  occurrence(child.attribute("minOccurs"), 1),
				max:  occurrence(child.attribute("maxOccurs"), 1),
			})
		}
	}
	return group
}

func compileSimpleType(source node, parent map[string]string) *simpleDecl {
	scope := source.scope(parent)
	declaration := &simpleDecl{minLength: -1, maxLength: -1, length: -1}
	for _, child := range source.elements() {
		if !child.is("restriction") {
			continue
		}
		restrictionScope := child.scope(scope)
		declaration.base = resolve(child.attribute("base"), restrictionScope)
		for _, facet := range child.elements() {
			applyFacet(declaration, facet)
		}
	}
	return declaration
}

func applyFacet(declaration *simpleDecl, facet node) {
	value := facet.attribute("value")
	switch {
	case facet.is("pattern"):
		compiled, err := regexp.Compile("^(?:" + value + ")$")
		if err == nil {
			declaration.patterns = append(declaration.patterns, compiled)
		}
	case facet.is("enumeration"):
		if declaration.enumerations == nil {
			declaration.enumerations = map[string]bool{}
		}
		declaration.enumerations[value] = true
	case facet.is("minLength"):
		declaration.minLength = number(value, -1)
	case facet.is("maxLength"):
		declaration.maxLength = number(value, -1)
	case facet.is("length"):
		declaration.length = number(value, -1)
	case facet.is("minInclusive"):
		declaration.minInclusive = value
	case facet.is("maxInclusive"):
		declaration.maxInclusive = value
	case facet.is("whiteSpace"):
		declaration.whiteSpace = value
	}
}

func occurrence(value string, fallback int) int {
	switch value {
	case "":
		return fallback
	case "unbounded":
		return unbounded
	}
	return number(value, fallback)
}

func number(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
