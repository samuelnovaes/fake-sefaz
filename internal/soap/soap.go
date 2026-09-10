package soap

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	EnvelopeNamespace = "http://www.w3.org/2003/05/soap-envelope"
	ContentType       = "application/soap+xml; charset=utf-8"
)

var ErrOperationNotFound = errors.New("no known operation element in the request body")

type Message struct {
	Operation string
	Namespace string
	Version   string
	Payload   []byte
	Element   []byte
	Enveloped bool
}

type node struct {
	XMLName    xml.Name
	Attributes []xml.Attr `xml:",any,attr"`
	Inner      []byte     `xml:",innerxml"`
}

func Decode(body []byte, operations map[string]bool) (Message, error) {
	message, err := decode(body, operations)
	if !errors.Is(err, ErrOperationNotFound) {
		return message, err
	}
	inflated, found := inflate(body)
	if !found {
		return message, err
	}
	inflatedMessage, inflatedError := decode(inflated, operations)
	if inflatedError != nil {
		return message, err
	}
	inflatedMessage.Enveloped = bytes.Contains(body, []byte("Envelope"))
	return inflatedMessage, nil
}

func inflate(body []byte) ([]byte, bool) {
	decoder := xml.NewDecoder(bytes.NewReader(body))
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) || err != nil {
			return nil, false
		}
		start, isStart := token.(xml.StartElement)
		if !isStart || !strings.HasSuffix(start.Name.Local, "DadosMsg") {
			continue
		}
		var content string
		if err := decoder.DecodeElement(&content, &start); err != nil {
			return nil, false
		}
		packed, err := base64.StdEncoding.DecodeString(strings.TrimSpace(content))
		if err != nil {
			return nil, false
		}
		reader, err := gzip.NewReader(bytes.NewReader(packed))
		if err != nil {
			return nil, false
		}
		defer reader.Close()
		plain, err := io.ReadAll(reader)
		if err != nil {
			return nil, false
		}
		return plain, true
	}
}

func decode(body []byte, operations map[string]bool) (Message, error) {
	decoder := xml.NewDecoder(bytes.NewReader(body))
	enveloped := false
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return Message{}, err
		}
		start, isStart := token.(xml.StartElement)
		if !isStart {
			continue
		}
		if start.Name.Local == "Envelope" || start.Name.Local == "Body" {
			enveloped = true
			continue
		}
		if !operations[start.Name.Local] {
			continue
		}
		var element node
		if err := decoder.DecodeElement(&element, &start); err != nil {
			return Message{}, err
		}
		return Message{
			Operation: start.Name.Local,
			Namespace: start.Name.Space,
			Version:   attribute(element.Attributes, "versao"),
			Payload:   element.Inner,
			Element:   rebuild(start.Name, element),
			Enveloped: enveloped,
		}, nil
	}
	return Message{}, ErrOperationNotFound
}

func rebuild(name xml.Name, element node) []byte {
	var builder strings.Builder
	builder.WriteString("<" + name.Local)
	if name.Space != "" {
		builder.WriteString(fmt.Sprintf(` xmlns=%q`, name.Space))
	}
	for _, attribute := range element.Attributes {
		if attribute.Name.Local == "xmlns" {
			continue
		}
		builder.WriteString(fmt.Sprintf(` %s=%q`, attribute.Name.Local, attribute.Value))
	}
	builder.WriteString(">")
	builder.Write(element.Inner)
	builder.WriteString("</" + name.Local + ">")
	return []byte(builder.String())
}

func attribute(attributes []xml.Attr, name string) string {
	for _, candidate := range attributes {
		if candidate.Name.Local == name {
			return candidate.Value
		}
	}
	return ""
}

func Encode(wsdlNamespace, resultTag string, payload []byte, enveloped bool) []byte {
	if !enveloped {
		return append([]byte(xml.Header), payload...)
	}
	var builder bytes.Buffer
	builder.WriteString(xml.Header)
	builder.WriteString(`<soap:Envelope xmlns:soap="` + EnvelopeNamespace + `"><soap:Body>`)
	builder.WriteString(`<` + resultTag + ` xmlns="` + wsdlNamespace + `">`)
	builder.Write(payload)
	builder.WriteString(`</` + resultTag + `>`)
	builder.WriteString(`</soap:Body></soap:Envelope>`)
	return builder.Bytes()
}

func Fault(code, reason string) []byte {
	var builder bytes.Buffer
	builder.WriteString(xml.Header)
	builder.WriteString(`<soap:Envelope xmlns:soap="` + EnvelopeNamespace + `"><soap:Body><soap:Fault>`)
	builder.WriteString(`<soap:Code><soap:Value>soap:` + code + `</soap:Value></soap:Code>`)
	builder.WriteString(`<soap:Reason><soap:Text xml:lang="pt-BR">`)
	xml.EscapeText(&builder, []byte(reason))
	builder.WriteString(`</soap:Text></soap:Reason>`)
	builder.WriteString(`</soap:Fault></soap:Body></soap:Envelope>`)
	return builder.Bytes()
}

type Endpoint struct {
	WSDLNamespace string `json:"wsdlNamespace"`
	ResultTag     string `json:"resultTag"`
}

type Service struct {
	Operation string `json:"operation"`
	Name      string `json:"name"`
	Models    string `json:"models"`
}
