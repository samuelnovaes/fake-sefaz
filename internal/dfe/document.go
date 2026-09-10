package dfe

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"strconv"
	"strings"
)

var ErrNoAccessKey = errors.New("no signed element carrying an access key was found")

type Document struct {
	Key               string
	Model             Model
	RootTag           string
	InfoTag           string
	UFCode            string
	Environment       int
	Series            int
	Number            int64
	IssuerTaxID       string
	RecipientName     string
	RecipientDocument string
	DigestValue       string
	Signed            bool
	XML               string
}

var numberTags = map[string]bool{"nNF": true, "nCT": true, "nMDF": true, "nBP": true, "nNF3": true}

func ParseDocument(raw []byte) (Document, error) {
	document := Document{XML: string(raw)}
	decoder := xml.NewDecoder(bytes.NewReader(raw))
	stack := make([]string, 0, 8)
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return document, err
		}
		start, isStart := token.(xml.StartElement)
		if !isStart {
			if _, isEnd := token.(xml.EndElement); isEnd && len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			continue
		}
		if document.RootTag == "" {
			document.RootTag = start.Name.Local
		}
		if strings.HasPrefix(start.Name.Local, "inf") && document.Key == "" {
			if key := accessKeyIn(start); key != "" {
				document.Key = key
				document.InfoTag = start.Name.Local
			}
		}
		parent := ""
		if len(stack) > 0 {
			parent = stack[len(stack)-1]
		}
		if !leafOfInterest(start.Name.Local, parent) {
			stack = append(stack, start.Name.Local)
			continue
		}
		var value string
		if err := decoder.DecodeElement(&value, &start); err != nil {
			return document, err
		}
		assign(&document, start.Name.Local, parent, strings.TrimSpace(value))
	}
	if document.Key == "" {
		return document, ErrNoAccessKey
	}
	if parsed, err := ParseAccessKey(document.Key); err == nil {
		document.Model = parsed.Model
	}
	return document, nil
}

func accessKeyIn(start xml.StartElement) string {
	for _, attribute := range start.Attr {
		if attribute.Name.Local != "Id" || len(attribute.Value) < KeyLength {
			continue
		}
		return attribute.Value[len(attribute.Value)-KeyLength:]
	}
	return ""
}

func leafOfInterest(name, parent string) bool {
	switch name {
	case "cUF", "tpAmb", "mod", "serie", "DigestValue", "SignatureValue":
		return true
	case "CNPJ", "CPF", "xNome":
		return parent == "emit" || parent == "dest" || parent == "comp"
	}
	return numberTags[name]
}

func assign(document *Document, name, parent, value string) {
	if value == "" {
		return
	}
	switch name {
	case "cUF":
		if document.UFCode == "" {
			document.UFCode = value
		}
	case "tpAmb":
		if document.Environment == 0 {
			document.Environment, _ = strconv.Atoi(value)
		}
	case "serie":
		if document.Series == 0 {
			document.Series, _ = strconv.Atoi(value)
		}
	case "DigestValue":
		if document.DigestValue == "" {
			document.DigestValue = value
		}
	case "SignatureValue":
		document.Signed = true
	case "CNPJ", "CPF":
		if parent == "emit" && document.IssuerTaxID == "" {
			document.IssuerTaxID = value
			return
		}
		if document.RecipientDocument == "" {
			document.RecipientDocument = value
		}
	case "xNome":
		if parent != "emit" && document.RecipientName == "" {
			document.RecipientName = value
		}
	default:
		if numberTags[name] && document.Number == 0 {
			document.Number, _ = strconv.ParseInt(value, 10, 64)
		}
	}
}

func Fields(raw []byte) map[string]string {
	fields := map[string]string{}
	decoder := xml.NewDecoder(bytes.NewReader(raw))
	var current string
	var text strings.Builder
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fields
		}
		switch element := token.(type) {
		case xml.StartElement:
			current = element.Name.Local
			text.Reset()
		case xml.CharData:
			text.Write(element)
		case xml.EndElement:
			if element.Name.Local == current {
				if value := strings.TrimSpace(text.String()); value != "" {
					if _, seen := fields[current]; !seen {
						fields[current] = value
					}
				}
			}
			current = ""
			text.Reset()
		}
	}
	return fields
}

func FieldKey(fields map[string]string) string {
	for name, value := range fields {
		if strings.HasPrefix(name, "ch") && len(value) == KeyLength {
			return value
		}
	}
	return ""
}
