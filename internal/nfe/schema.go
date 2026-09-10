package nfe

import (
	"github.com/vendermais/fake-sefaz/internal/authorizer"
	"github.com/vendermais/fake-sefaz/internal/status"
)

func (s *Service) schemaRejection(request authorizer.Context) ([]byte, error) {
	switch request.Operation {
	case "enviNFe":
		return s.rejectBatch(request, status.RejectedBatchSchema)
	case "consStatServ":
		return s.rejectStatus(request, status.RejectedSchema)
	case "consReciNFe":
		return encode(RetConsReciNFe{
			Version: Version, Environment: request.Environment, VerAplic: VerAplic, UFCode: request.UFCode,
			Status: int(status.RejectedSchema), Reason: status.Message(status.RejectedSchema),
		})
	case "consSitNFe":
		return encode(RetConsSitNFe{
			Version: Version, Environment: request.Environment, VerAplic: VerAplic, UFCode: request.UFCode,
			Status: int(status.RejectedSchema), Reason: status.Message(status.RejectedSchema),
		})
	case "inutNFe":
		return encode(RetInutNFe{Version: Version, Info: InfInutRet{
			Environment: request.Environment, VerAplic: VerAplic, UFCode: request.UFCode,
			Status: int(status.RejectedSchema), Reason: status.Message(status.RejectedSchema),
		}})
	case "envEvento":
		return encode(RetEnvEvento{
			Version: Version, Environment: request.Environment, VerAplic: VerAplic, OrganCode: request.UFCode,
			Status: int(status.RejectedBatchSchema), Reason: status.Message(status.RejectedBatchSchema),
		})
	case "ConsCad":
		return encode(RetConsCad{Version: Version, Info: InfConsRet{
			VerAplic: VerAplic, QueriedAt: timestamp(s.engine.Now()),
			Status: int(status.RejectedSchema), Reason: status.Message(status.RejectedSchema),
		}})
	}
	return encode(RetDistDFeInt{
		Version: Version, Environment: request.Environment, VerAplic: VerAplic,
		RespondedAt: timestamp(s.engine.Now()),
		Status:      int(status.RejectedSchema), Reason: status.Message(status.RejectedSchema),
	})
}
