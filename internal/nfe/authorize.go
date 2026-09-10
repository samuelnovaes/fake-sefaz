package nfe

import (
	"time"

	"github.com/vendermais/fake-sefaz/internal/authorizer"
	"github.com/vendermais/fake-sefaz/internal/status"
	"github.com/vendermais/fake-sefaz/internal/store"
)

func (s *Service) serviceStatus(request authorizer.Context, payload []byte) ([]byte, error) {
	var query ConsStatServ
	if err := unmarshal(payload, "consStatServ", &query); err != nil {
		return s.rejectStatus(request, status.RejectedSchema)
	}
	code := s.engine.ServiceStatus(request)
	response := RetConsStatServ{
		Version:     Version,
		Environment: authorizer.Environment(query.Environment, request.Environment),
		VerAplic:    VerAplic,
		Status:      int(code),
		Reason:      status.Message(code),
		UFCode:      firstNonEmpty(query.UFCode, request.UFCode),
		ReceivedAt:  timestamp(s.engine.Now()),
		AverageTime: s.engine.Scenarios().AverageTime(),
	}
	if code != status.ServiceRunning {
		response.ReturnAt = timestamp(s.engine.Now().Add(5 * time.Minute))
		response.Note = "fake-sefaz scenario"
	}
	return encode(response)
}

func (s *Service) rejectStatus(request authorizer.Context, code status.Code) ([]byte, error) {
	return encode(RetConsStatServ{
		Version:     Version,
		Environment: request.Environment,
		VerAplic:    VerAplic,
		Status:      int(code),
		Reason:      status.Message(code),
		UFCode:      request.UFCode,
		ReceivedAt:  timestamp(s.engine.Now()),
	})
}

func (s *Service) authorize(request authorizer.Context, version string, payload []byte) ([]byte, error) {
	var batch EnviNFe
	if err := unmarshal(payload, "enviNFe", &batch); err != nil {
		return s.rejectBatch(request, status.RejectedBatchSchema)
	}
	if version != "" && version != Version {
		return s.rejectBatch(request, status.RejectedVersionUnsupported)
	}
	if len(batch.Documents) == 0 {
		return s.rejectBatch(request, status.RejectedBatchSchema)
	}
	if serviceCode := s.engine.Scenarios().ServiceStatus(); serviceCode != status.ServiceRunning {
		return s.rejectBatch(request, serviceCode)
	}

	now := s.engine.Now()
	protocols := make([]ProtNFe, 0, len(batch.Documents))
	ufCode := request.UFCode
	environment := request.Environment
	for _, document := range batch.Documents {
		result := s.engine.Authorize(submissionOf(document), request)
		protocols = append(protocols, protocolFrom(result))
		ufCode = firstNonEmpty(result.UFCode, ufCode)
		if environment == 0 {
			environment = result.Environment
		}
	}

	response := RetEnviNFe{
		Version:     Version,
		Environment: environment,
		VerAplic:    VerAplic,
		UFCode:      ufCode,
		ReceivedAt:  timestamp(now),
	}
	if batch.Sync == 1 && !s.engine.Scenarios().Asynchronous() {
		response.Status = int(status.BatchProcessed)
		response.Reason = status.Message(status.BatchProcessed)
		response.Protocol = &protocols[0]
		return encode(response)
	}

	receipt := s.engine.Documents().NextProtocol(ufCode, now)
	keys := make([]string, 0, len(protocols))
	for _, protocol := range protocols {
		keys = append(keys, protocol.Info.Key)
	}
	average := s.engine.Scenarios().AverageTime()
	s.engine.Documents().SaveBatch(store.Batch{
		Receipt:     receipt,
		Environment: environment,
		UFCode:      ufCode,
		Keys:        keys,
		Status:      status.BatchProcessed,
		ReceivedAt:  now,
		ReleaseAt:   now.Add(time.Duration(average) * time.Second),
	})
	response.Status = int(status.BatchReceived)
	response.Reason = status.Message(status.BatchReceived)
	response.Receipt = &InfRec{Receipt: receipt, AverageTime: average}
	return encode(response)
}

func (s *Service) rejectBatch(request authorizer.Context, code status.Code) ([]byte, error) {
	return encode(RetEnviNFe{
		Version:     Version,
		Environment: request.Environment,
		VerAplic:    VerAplic,
		Status:      int(code),
		Reason:      status.Message(code),
		UFCode:      request.UFCode,
		ReceivedAt:  timestamp(s.engine.Now()),
	})
}

func submissionOf(document NFe) authorizer.Submission {
	info := document.Info
	return authorizer.Submission{
		Key:               accessKeyOf(info.Identifier),
		UFCode:            info.Ide.UFCode,
		Environment:       info.Ide.Environment,
		IssuerTaxID:       info.Issuer.Document(),
		DigestValue:       document.Signature.SignedInfo.Reference.DigestValue,
		RecipientName:     info.Recipient.LegalName,
		RecipientDocument: info.Recipient.Document(),
		Signed:            document.Signature.SignatureValue != "",
		XML:               `<NFe xmlns="` + Namespace + `">` + string(document.Inner) + `</NFe>`,
	}
}

func protocolFrom(result authorizer.Result) ProtNFe {
	protocol := ProtNFe{Version: Version, Info: InfProt{
		Environment: result.Environment,
		VerAplic:    VerAplic,
		Key:         result.Key,
		ReceivedAt:  timestamp(result.ReceivedAt),
		Protocol:    result.Protocol,
		DigestValue: result.DigestValue,
		Status:      int(result.Status),
		Reason:      result.Reason,
	}}
	if result.Protocol != "" {
		protocol.Info.Identifier = "ID" + result.Protocol
	}
	return protocol
}

func accessKeyOf(identifier string) string {
	if len(identifier) > 44 {
		return identifier[len(identifier)-44:]
	}
	return identifier
}

func timestamp(moment time.Time) string {
	return moment.Format("2006-01-02T15:04:05-07:00")
}
