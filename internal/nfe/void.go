package nfe

import (
	"fmt"

	"github.com/vendermais/fake-sefaz/internal/dfe"
	"github.com/vendermais/fake-sefaz/internal/scenario"
	"github.com/vendermais/fake-sefaz/internal/status"
	"github.com/vendermais/fake-sefaz/internal/store"
)

func (s *Service) void(request Request) ([]byte, error) {
	var command InutNFe
	if err := unmarshal(request.Payload, "inutNFe", &command); err != nil {
		return encode(RetInutNFe{Version: Version, Info: InfInutRet{
			Environment: request.Environment, VerAplic: VerAplic, UFCode: request.UFCode,
			Status: int(status.RejectedSchema), Reason: status.Message(status.RejectedSchema),
		}})
	}
	info := command.Info
	environment := environmentOf(info.Environment, request.Environment)
	now := s.now()
	result := InfInutRet{
		Environment: environment,
		VerAplic:    VerAplic,
		UFCode:      firstNonEmpty(info.UFCode, request.UFCode),
		Year:        info.Year,
		TaxID:       info.TaxID,
		Model:       info.Model,
		Series:      info.Series,
		First:       info.First,
		Last:        info.Last,
	}

	model := dfe.Model(info.Model)
	if !model.Implemented() {
		return encode(RetInutNFe{Version: Version, Info: rejectVoiding(result, status.RejectedUncatalogued)})
	}
	if info.First <= 0 || info.Last < info.First {
		return encode(RetInutNFe{Version: Version, Info: rejectVoiding(result, status.RejectedSchema)})
	}
	if forced, matched := s.scenarios.Resolve(scenario.Match{
		Operation: "inutNFe", IssuerTaxID: info.TaxID, Model: info.Model,
	}); matched && forced != status.VoidingAuthorized {
		return encode(RetInutNFe{Version: Version, Info: rejectVoiding(result, forced)})
	}
	if request.ForcedStatus != 0 && request.ForcedStatus != status.VoidingAuthorized {
		return encode(RetInutNFe{Version: Version, Info: rejectVoiding(result, request.ForcedStatus)})
	}

	identifier := store.VoidingIdentifier(environment, result.UFCode, info.Year, info.TaxID, model, info.Series, info.First, info.Last)
	if existing, found := s.documents.Voiding(identifier); found {
		result.Status = int(existing.Status)
		result.Reason = status.Message(existing.Status)
		result.Protocol = existing.Protocol
		result.ReceivedAt = timestamp(existing.ReceivedAt)
		result.Identifier = voidingIdentifierTag(result)
		return encode(RetInutNFe{Version: Version, Info: result})
	}
	for number := info.First; number <= info.Last; number++ {
		if _, found := s.documents.DocumentByNumber(environment, info.TaxID, model, info.Series, number); found {
			return encode(RetInutNFe{Version: Version, Info: rejectVoiding(result, status.RejectedDuplicate)})
		}
	}

	protocol := s.documents.NextProtocol(result.UFCode, now)
	s.documents.SaveVoiding(store.Voiding{
		Identifier:  identifier,
		Environment: environment,
		UFCode:      result.UFCode,
		Year:        info.Year,
		IssuerTaxID: info.TaxID,
		Model:       model,
		Series:      info.Series,
		First:       info.First,
		Last:        info.Last,
		Protocol:    protocol,
		Status:      status.VoidingAuthorized,
		ReceivedAt:  now,
	})
	result.Status = int(status.VoidingAuthorized)
	result.Reason = status.Message(status.VoidingAuthorized)
	result.Protocol = protocol
	result.ReceivedAt = timestamp(now)
	result.Identifier = voidingIdentifierTag(result)
	return encode(RetInutNFe{Version: Version, Info: result})
}

func rejectVoiding(result InfInutRet, code status.Code) InfInutRet {
	result.Status = int(code)
	result.Reason = status.Message(code)
	result.Protocol = ""
	result.ReceivedAt = ""
	return result
}

func voidingIdentifierTag(result InfInutRet) string {
	return fmt.Sprintf("ID%s%02d%s%s%03d%09d%09d",
		result.UFCode, result.Year%100, result.TaxID, result.Model, result.Series, result.First, result.Last)
}
