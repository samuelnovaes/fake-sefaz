package nfe

import (
	"fmt"

	"github.com/vendermais/fake-sefaz/internal/authorizer"
	"github.com/vendermais/fake-sefaz/internal/dfe"
)

func (s *Service) void(request authorizer.Context, payload []byte) ([]byte, error) {
	var command InutNFe
	if err := unmarshal(payload, "inutNFe", &command); err != nil {
		return encode(RetInutNFe{Version: Version, Info: InfInutRet{
			Environment: request.Environment, VerAplic: VerAplic, UFCode: request.UFCode,
			Status: 215, Reason: "Rejeicao: Falha no schema XML",
		}})
	}
	info := command.Info
	result := s.engine.Void(authorizer.VoidSubmission{
		Environment: info.Environment,
		UFCode:      firstNonEmpty(info.UFCode, request.UFCode),
		Year:        info.Year,
		IssuerTaxID: info.TaxID,
		Model:       dfe.Model(info.Model),
		Series:      info.Series,
		First:       info.First,
		Last:        info.Last,
	}, request)

	answer := InfInutRet{
		Environment: authorizer.Environment(info.Environment, request.Environment),
		VerAplic:    VerAplic,
		Status:      int(result.Status),
		Reason:      result.Reason,
		UFCode:      firstNonEmpty(info.UFCode, request.UFCode),
		Year:        info.Year,
		TaxID:       info.TaxID,
		Model:       info.Model,
		Series:      info.Series,
		First:       info.First,
		Last:        info.Last,
		Protocol:    result.Protocol,
	}
	if !result.ReceivedAt.IsZero() {
		answer.ReceivedAt = timestamp(result.ReceivedAt)
		answer.Identifier = voidingIdentifierTag(answer)
	}
	return deliver(RetInutNFe{Version: Version, Info: answer}, result.AnswerLost)
}

func voidingIdentifierTag(result InfInutRet) string {
	return fmt.Sprintf("ID%s%02d%s%s%03d%09d%09d",
		result.UFCode, result.Year%100, result.TaxID, result.Model, result.Series, result.First, result.Last)
}
