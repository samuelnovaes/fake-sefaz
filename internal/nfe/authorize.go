package nfe

import (
	"time"

	"github.com/vendermais/fake-sefaz/internal/dfe"
	"github.com/vendermais/fake-sefaz/internal/scenario"
	"github.com/vendermais/fake-sefaz/internal/status"
	"github.com/vendermais/fake-sefaz/internal/store"
	"github.com/vendermais/fake-sefaz/internal/uf"
)

func (s *Service) serviceStatus(request Request) ([]byte, error) {
	var query ConsStatServ
	if err := unmarshal(request.Payload, "consStatServ", &query); err != nil {
		return s.rejectStatus(request, status.RejectedSchema)
	}
	code := s.scenarios.ServiceStatus()
	if request.ForcedStatus != 0 {
		code = request.ForcedStatus
	}
	response := RetConsStatServ{
		Version:     Version,
		Environment: environmentOf(query.Environment, request.Environment),
		VerAplic:    VerAplic,
		Status:      int(code),
		Reason:      status.Message(code),
		UFCode:      firstNonEmpty(query.UFCode, request.UFCode),
		ReceivedAt:  timestamp(s.now()),
		AverageTime: s.scenarios.AverageTime(),
	}
	if code != status.ServiceRunning {
		response.ReturnAt = timestamp(s.now().Add(5 * time.Minute))
		response.Note = "fake-sefaz scenario"
	}
	return encode(response)
}

func (s *Service) rejectStatus(request Request, code status.Code) ([]byte, error) {
	return encode(RetConsStatServ{
		Version:     Version,
		Environment: request.Environment,
		VerAplic:    VerAplic,
		Status:      int(code),
		Reason:      status.Message(code),
		UFCode:      request.UFCode,
		ReceivedAt:  timestamp(s.now()),
	})
}

func (s *Service) authorize(request Request) ([]byte, error) {
	var batch EnviNFe
	if err := unmarshal(request.Payload, "enviNFe", &batch); err != nil {
		return s.rejectBatch(request, status.RejectedBatchSchema)
	}
	if request.Version != "" && request.Version != Version {
		return s.rejectBatch(request, status.RejectedVersionUnsupported)
	}
	if len(batch.Documents) == 0 {
		return s.rejectBatch(request, status.RejectedBatchSchema)
	}
	if serviceCode := s.scenarios.ServiceStatus(); serviceCode != status.ServiceRunning {
		return s.rejectBatch(request, serviceCode)
	}

	now := s.now()
	protocols := make([]ProtNFe, 0, len(batch.Documents))
	ufCode := request.UFCode
	environment := request.Environment
	for _, document := range batch.Documents {
		protocol, resolvedUF, resolvedEnvironment := s.authorizeDocument(request, document, now)
		protocols = append(protocols, protocol)
		ufCode = firstNonEmpty(resolvedUF, ufCode)
		if environment == 0 {
			environment = resolvedEnvironment
		}
	}

	response := RetEnviNFe{
		Version:     Version,
		Environment: environment,
		VerAplic:    VerAplic,
		UFCode:      ufCode,
		ReceivedAt:  timestamp(now),
	}

	if batch.Sync == 1 && !s.scenarios.Asynchronous() {
		response.Status = int(status.BatchProcessed)
		response.Reason = status.Message(status.BatchProcessed)
		response.Protocol = &protocols[0]
		return encode(response)
	}

	receipt := s.documents.NextProtocol(ufCode, now)
	keys := make([]string, 0, len(protocols))
	for _, protocol := range protocols {
		keys = append(keys, protocol.Info.Key)
	}
	average := s.scenarios.AverageTime()
	s.documents.SaveBatch(store.Batch{
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

func (s *Service) rejectBatch(request Request, code status.Code) ([]byte, error) {
	return encode(RetEnviNFe{
		Version:     Version,
		Environment: request.Environment,
		VerAplic:    VerAplic,
		Status:      int(code),
		Reason:      status.Message(code),
		UFCode:      request.UFCode,
		ReceivedAt:  timestamp(s.now()),
	})
}

func (s *Service) authorizeDocument(request Request, document NFe, now time.Time) (ProtNFe, string, int) {
	info := document.Info
	environment := environmentOf(info.Ide.Environment, request.Environment)
	protocol := ProtNFe{Version: Version, Info: InfProt{
		Environment: environment,
		VerAplic:    VerAplic,
		ReceivedAt:  timestamp(now),
		DigestValue: document.Signature.SignedInfo.Reference.DigestValue,
	}}

	key := accessKeyOf(info.Identifier)
	protocol.Info.Key = key
	parsed, err := dfe.ParseAccessKey(key)
	if err != nil {
		return reject(protocol, status.RejectedCheckDigit), info.Ide.UFCode, environment
	}

	if !uf.Known(info.Ide.UFCode) || info.Ide.UFCode != parsed.UFCode {
		return reject(protocol, status.RejectedIssuerUF), info.Ide.UFCode, environment
	}
	if request.Environment != 0 && info.Ide.Environment != 0 && info.Ide.Environment != request.Environment {
		return reject(protocol, status.RejectedEnvironment), info.Ide.UFCode, environment
	}
	if !parsed.Model.Implemented() {
		return reject(protocol, status.RejectedUncatalogued), info.Ide.UFCode, environment
	}
	if document.Signature.SignatureValue == "" {
		return reject(protocol, status.RejectedSignature), info.Ide.UFCode, environment
	}
	if environment == EnvironmentHomologation && info.Recipient.Document() != "" && info.Recipient.LegalName != HomologationRecipientName {
		return reject(protocol, status.RejectedHomologationName), info.Ide.UFCode, environment
	}

	issuer := info.Issuer.Document()
	if forced, matched := s.scenarios.Resolve(scenario.Match{
		Operation: "enviNFe", IssuerTaxID: issuer, Key: key, Model: string(parsed.Model),
	}); matched {
		return s.settle(protocol, forced, document, parsed, issuer, now), info.Ide.UFCode, environment
	}
	if request.ForcedStatus != 0 {
		return s.settle(protocol, request.ForcedStatus, document, parsed, issuer, now), info.Ide.UFCode, environment
	}

	if existing, found := s.documents.Document(key); found {
		if existing.DigestValue == protocol.Info.DigestValue {
			protocol.Info.Protocol = existing.Protocol
			return reject(protocol, status.RejectedDuplicate), info.Ide.UFCode, environment
		}
		return reject(protocol, status.RejectedDuplicate), info.Ide.UFCode, environment
	}
	if _, found := s.documents.DocumentByNumber(environment, issuer, parsed.Model, int(parsed.Series), parsed.Number); found {
		return reject(protocol, status.RejectedDuplicateOtherKey), info.Ide.UFCode, environment
	}
	if s.documents.NumberVoided(environment, issuer, parsed.Model, int(parsed.Series), parsed.Number) {
		return reject(protocol, status.RejectedDuplicate), info.Ide.UFCode, environment
	}

	return s.settle(protocol, status.Authorized, document, parsed, issuer, now), info.Ide.UFCode, environment
}

func (s *Service) settle(protocol ProtNFe, code status.Code, document NFe, parsed dfe.AccessKey, issuer string, now time.Time) ProtNFe {
	if code != status.Authorized && !status.Denied(code) {
		return reject(protocol, code)
	}
	protocol.Info.Protocol = s.documents.NextProtocol(parsed.UFCode, now)
	protocol.Info.Identifier = "ID" + protocol.Info.Protocol
	protocol.Info.Status = int(code)
	protocol.Info.Reason = status.Message(code)
	s.documents.SaveDocument(store.Document{
		Key:         parsed.Raw,
		Model:       parsed.Model,
		Environment: protocol.Info.Environment,
		UFCode:      parsed.UFCode,
		IssuerTaxID: issuer,
		Series:      int(parsed.Series),
		Number:      parsed.Number,
		DigestValue: protocol.Info.DigestValue,
		Protocol:    protocol.Info.Protocol,
		Status:      code,
		Reason:      status.Message(code),
		ReceivedAt:  now,
		XML:         wrapDocument(document),
	})
	return protocol
}

func reject(protocol ProtNFe, code status.Code) ProtNFe {
	protocol.Info.Status = int(code)
	protocol.Info.Reason = status.Message(code)
	protocol.Info.Protocol = ""
	return protocol
}

func wrapDocument(document NFe) string {
	return `<NFe xmlns="` + Namespace + `">` + string(document.Inner) + `</NFe>`
}

func accessKeyOf(identifier string) string {
	if len(identifier) > 3 && (identifier[:3] == "NFe" || identifier[:3] == "CTe") {
		return identifier[3:]
	}
	return identifier
}

func environmentOf(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return EnvironmentHomologation
}

const (
	EnvironmentProduction   = 1
	EnvironmentHomologation = 2
)
