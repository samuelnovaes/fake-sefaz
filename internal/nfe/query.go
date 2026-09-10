package nfe

import (
	"github.com/vendermais/fake-sefaz/internal/authorizer"
	"strconv"

	"github.com/vendermais/fake-sefaz/internal/dfe"
	"github.com/vendermais/fake-sefaz/internal/status"
	"github.com/vendermais/fake-sefaz/internal/store"
)

func (s *Service) batchResult(request authorizer.Context, payload []byte) ([]byte, error) {
	var query ConsReciNFe
	if err := unmarshal(payload, "consReciNFe", &query); err != nil {
		return encode(RetConsReciNFe{
			Version: Version, Environment: request.Environment, VerAplic: VerAplic,
			Status: int(status.RejectedSchema), Reason: status.Message(status.RejectedSchema), UFCode: request.UFCode,
		})
	}
	response := RetConsReciNFe{
		Version:     Version,
		Environment: authorizer.Environment(query.Environment, request.Environment),
		VerAplic:    VerAplic,
		Receipt:     query.Receipt,
		UFCode:      request.UFCode,
	}
	batch, found := s.engine.Documents().Batch(query.Receipt)
	if !found {
		response.Status = int(status.BatchNotFound)
		response.Reason = status.Message(status.BatchNotFound)
		return encode(response)
	}
	response.UFCode = batch.UFCode
	if s.engine.Now().Before(batch.ReleaseAt) {
		response.Status = int(status.BatchInProcess)
		response.Reason = status.Message(status.BatchInProcess)
		return encode(response)
	}
	response.Status = int(status.BatchProcessed)
	response.Reason = status.Message(status.BatchProcessed)
	for _, key := range batch.Keys {
		document, exists := s.engine.Documents().Document(key)
		if !exists {
			continue
		}
		response.Protocols = append(response.Protocols, protocolOf(document))
	}
	return encode(response)
}

func (s *Service) documentStatus(request authorizer.Context, payload []byte) ([]byte, error) {
	var query ConsSitNFe
	if err := unmarshal(payload, "consSitNFe", &query); err != nil {
		return encode(RetConsSitNFe{
			Version: Version, Environment: request.Environment, VerAplic: VerAplic,
			Status: int(status.RejectedSchema), Reason: status.Message(status.RejectedSchema), UFCode: request.UFCode,
		})
	}
	response := RetConsSitNFe{
		Version:     Version,
		Environment: authorizer.Environment(query.Environment, request.Environment),
		VerAplic:    VerAplic,
		UFCode:      request.UFCode,
		Key:         query.Key,
	}
	parsed, err := dfe.ParseAccessKey(query.Key)
	if err != nil {
		response.Status = int(status.RejectedCheckDigit)
		response.Reason = status.Message(status.RejectedCheckDigit)
		return encode(response)
	}
	response.UFCode = parsed.UFCode
	document, found := s.engine.Documents().Document(parsed.Raw)
	if !found {
		response.Status = int(status.RejectedNotFound)
		response.Reason = status.Message(status.RejectedNotFound)
		return encode(response)
	}
	protocol := protocolOf(document)
	response.Protocol = &protocol
	response.Status = int(document.Status)
	response.Reason = status.Message(document.Status)
	if document.Cancelled {
		response.Status = int(status.CancellationAuthorized)
		response.Reason = status.Message(status.CancellationAuthorized)
	}
	for _, event := range document.Events {
		response.Events = append(response.Events, ProcEventoNFe{
			Version: Version,
			Event: RetEvento{Version: Version, Info: InfEventoRet{
				Identifier:   "ID" + event.Type + event.Key + pad(event.Sequence),
				Environment:  document.Environment,
				VerAplic:     VerAplic,
				OrganCode:    document.UFCode,
				Status:       int(event.Status),
				Reason:       status.Message(event.Status),
				Key:          event.Key,
				Type:         event.Type,
				Description:  event.Description,
				Sequence:     event.Sequence,
				RegisteredAt: timestamp(event.RegisteredAt),
				Protocol:     event.Protocol,
			}},
		})
	}
	return encode(response)
}

func protocolOf(document store.Document) ProtNFe {
	return ProtNFe{Version: Version, Info: InfProt{
		Identifier:  "ID" + document.Protocol,
		Environment: document.Environment,
		VerAplic:    VerAplic,
		Key:         document.Key,
		ReceivedAt:  timestamp(document.ReceivedAt),
		Protocol:    document.Protocol,
		DigestValue: document.DigestValue,
		Status:      int(document.Status),
		Reason:      status.Message(document.Status),
	}}
}

func pad(sequence int) string {
	digits := "00" + itoa(sequence)
	return digits[len(digits)-2:]
}

func itoa(value int) string {
	return strconv.Itoa(value)
}
