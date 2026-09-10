package dfews

import (
	"encoding/xml"
	"strconv"
	"strings"
)

type node struct {
	name       string
	attributes [][2]string
	body       strings.Builder
}

func element(name string) *node {
	return &node{name: name}
}

func (n *node) attribute(name, value string) *node {
	if value == "" {
		return n
	}
	n.attributes = append(n.attributes, [2]string{name, value})
	return n
}

func (n *node) field(name, value string) *node {
	if value == "" {
		return n
	}
	n.body.WriteString("<" + name + ">")
	xml.EscapeText(&n.body, []byte(value))
	n.body.WriteString("</" + name + ">")
	return n
}

func (n *node) number(name string, value int) *node {
	if value == 0 {
		return n
	}
	return n.field(name, strconv.Itoa(value))
}

func (n *node) child(child *node) *node {
	if child == nil {
		return n
	}
	n.body.WriteString(child.String())
	return n
}

func (n *node) raw(content string) *node {
	n.body.WriteString(content)
	return n
}

func (n *node) String() string {
	var builder strings.Builder
	builder.WriteString("<" + n.name)
	for _, attribute := range n.attributes {
		builder.WriteString(" " + attribute[0] + "=\"" + attribute[1] + "\"")
	}
	builder.WriteString(">")
	builder.WriteString(n.body.String())
	builder.WriteString("</" + n.name + ">")
	return builder.String()
}
