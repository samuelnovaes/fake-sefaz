package nfe

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"strconv"

	"github.com/vendermais/fake-sefaz/internal/status"
	"github.com/vendermais/fake-sefaz/internal/store"
)

func (s *Service) distribute(request Request) ([]byte, error) {
	var query DistDFeInt
	if err := unmarshal(request.Payload, "distDFeInt", &query); err != nil {
		return encode(RetDistDFeInt{
			Version: Version, Environment: request.Environment, VerAplic: VerAplic,
			Status: int(status.RejectedSchema), Reason: status.Message(status.RejectedSchema),
			RespondedAt: timestamp(s.now()),
		})
	}
	environment := environmentOf(query.Environment, request.Environment)
	response := RetDistDFeInt{
		Version:     Version,
		Environment: environment,
		VerAplic:    VerAplic,
		RespondedAt: timestamp(s.now()),
	}

	selected := s.selectForDistribution(query, environment)
	response.MaxNSU = fmt.Sprintf("%015d", s.documents.LastNSU())
	if len(selected) == 0 {
		response.Status = int(status.NoDocumentFound)
		response.Reason = status.Message(status.NoDocumentFound)
		response.LastNSU = fmt.Sprintf("%015d", lastNSU(query))
		return encode(response)
	}

	batch := &LoteDistDFeInt{}
	highest := int64(0)
	for _, document := range selected {
		content, err := compress(procNFe(document))
		if err != nil {
			return nil, err
		}
		batch.Documents = append(batch.Documents, DocZip{
			NSU:     fmt.Sprintf("%015d", document.NSU),
			Schema:  "procNFe_v4.00.xsd",
			Content: content,
		})
		if document.NSU > highest {
			highest = document.NSU
		}
	}
	response.Status = int(status.DocumentFound)
	response.Reason = status.Message(status.DocumentFound)
	response.LastNSU = fmt.Sprintf("%015d", highest)
	response.Batch = batch
	return encode(response)
}

func (s *Service) selectForDistribution(query DistDFeInt, environment int) []store.Document {
	if query.ConsChNFe.Key != "" {
		document, found := s.documents.Document(query.ConsChNFe.Key)
		if !found {
			return nil
		}
		return []store.Document{document}
	}
	documents := s.documents.Documents(store.DocumentFilter{
		IssuerTaxID: query.Document(),
		Environment: environment,
		AfterNSU:    lastNSU(query),
	})
	if query.ConsNSU.NSU != "" {
		target, _ := strconv.ParseInt(query.ConsNSU.NSU, 10, 64)
		for _, document := range documents {
			if document.NSU == target {
				return []store.Document{document}
			}
		}
		return nil
	}
	if len(documents) > s.options.MaxDistributionDocuments {
		documents = documents[:s.options.MaxDistributionDocuments]
	}
	return documents
}

func lastNSU(query DistDFeInt) int64 {
	value, _ := strconv.ParseInt(query.DistNSU.LastNSU, 10, 64)
	return value
}

func procNFe(document store.Document) string {
	protocol := protocolOf(document)
	rendered, err := encode(protocol)
	if err != nil {
		return document.XML
	}
	return `<nfeProc versao="` + Version + `" xmlns="` + Namespace + `">` + document.XML + string(rendered) + `</nfeProc>`
}

func compress(content string) (string, error) {
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	if _, err := writer.Write([]byte(content)); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buffer.Bytes()), nil
}
