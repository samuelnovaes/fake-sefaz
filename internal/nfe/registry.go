package nfe

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"github.com/vendermais/fake-sefaz/internal/status"
	"github.com/vendermais/fake-sefaz/internal/uf"
)

func (s *Service) registration(request Request) ([]byte, error) {
	var query ConsCad
	if err := unmarshal(request.Payload, "ConsCad", &query); err != nil {
		return encode(RetConsCad{Version: Version, Info: InfConsRet{
			VerAplic: VerAplic, Status: int(status.RejectedSchema), Reason: status.Message(status.RejectedSchema),
			QueriedAt: timestamp(s.now()),
		}})
	}
	info := query.Info
	unit, known := uf.ByAcronym(info.UF)
	result := InfConsRet{
		VerAplic:  VerAplic,
		UF:        info.UF,
		TaxID:     info.TaxID,
		StateID:   info.StateID,
		Person:    info.Person,
		QueriedAt: timestamp(s.now()),
		UFCode:    unit.Code,
	}
	if !known {
		result.Status = int(status.RejectedIssuerUF)
		result.Reason = status.Message(status.RejectedIssuerUF)
		return encode(RetConsCad{Version: Version, Info: result})
	}
	if request.ForcedStatus != 0 {
		result.Status = int(request.ForcedStatus)
		result.Reason = status.Message(request.ForcedStatus)
		return encode(RetConsCad{Version: Version, Info: result})
	}
	identity := firstNonEmpty(info.TaxID, info.Person, info.StateID)
	if identity == "" {
		result.Status = int(status.RejectedSchema)
		result.Reason = status.Message(status.RejectedSchema)
		return encode(RetConsCad{Version: Version, Info: result})
	}
	result.Status = int(status.RegistrationOneMatch)
	result.Reason = status.Message(status.RegistrationOneMatch)
	result.Records = []InfCad{{
		StateID:     firstNonEmpty(info.StateID, stateRegistration(unit.Code, identity)),
		TaxID:       info.TaxID,
		UF:          info.UF,
		StateStatus: 1,
		IndCredNFe:  1,
		IndCredCTe:  1,
		LegalName:   "CONTRIBUINTE FAKE SEFAZ " + identity,
	}}
	return encode(RetConsCad{Version: Version, Info: result})
}

func stateRegistration(ufCode, identity string) string {
	digest := sha256.Sum256([]byte(ufCode + identity))
	value := binary.BigEndian.Uint64(digest[:8]) % 1000000000
	return fmt.Sprintf("%09d", value)
}
