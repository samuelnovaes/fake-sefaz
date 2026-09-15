package nfe

import (
	"github.com/vendermais/fake-sefaz/internal/authorizer"
	"github.com/vendermais/fake-sefaz/internal/status"
)

func (s *Service) receiveEvents(request authorizer.Context, payload []byte) ([]byte, error) {
	var batch EnvEvento
	if err := unmarshal(payload, "envEvento", &batch); err != nil {
		return encode(RetEnvEvento{
			Version: Version, Environment: request.Environment, VerAplic: VerAplic, OrganCode: request.UFCode,
			Status: int(status.RejectedBatchSchema), Reason: status.Message(status.RejectedBatchSchema),
		})
	}
	response := RetEnvEvento{
		Version:     Version,
		BatchID:     batch.BatchID,
		Environment: request.Environment,
		VerAplic:    VerAplic,
		OrganCode:   request.UFCode,
		Status:      int(status.EventBatchProcessed),
		Reason:      status.Message(status.EventBatchProcessed),
	}
	if len(batch.Events) == 0 {
		response.Status = int(status.RejectedBatchSchema)
		response.Reason = status.Message(status.RejectedBatchSchema)
		return encode(response)
	}
	lost := false
	for _, event := range batch.Events {
		result := s.engine.RegisterEvent(eventSubmissionOf(event), request)
		if status.RefusesRequest(result.Status) {
			response.Status = int(result.Status)
			response.Reason = result.Reason
			response.Events = nil
			return encode(response)
		}
		lost = lost || result.AnswerLost
		rendered := eventResponseOf(event, result)
		if response.Environment == 0 {
			response.Environment = result.Environment
		}
		if response.OrganCode == "" {
			response.OrganCode = result.OrganCode
		}
		response.Events = append(response.Events, rendered)
	}
	return deliver(response, lost)
}

func eventSubmissionOf(event Evento) authorizer.EventSubmission {
	info := event.Info
	return authorizer.EventSubmission{
		Key:           info.Key,
		Type:          info.Type,
		Sequence:      info.Sequence,
		Environment:   info.Environment,
		OrganCode:     info.OrganCode,
		IssuerTaxID:   info.Document(),
		Protocol:      info.Detail.Protocol,
		SubstituteKey: info.Detail.Substitute,
		XML:           `<evento xmlns="` + Namespace + `" versao="` + Version + `">` + string(event.Inner) + `</evento>`,
	}
}

func eventResponseOf(event Evento, result authorizer.EventResult) RetEvento {
	info := InfEventoRet{
		Identifier:  "ID" + event.Info.Type + event.Info.Key + pad(event.Info.Sequence),
		Environment: result.Environment,
		VerAplic:    VerAplic,
		OrganCode:   result.OrganCode,
		Status:      int(result.Status),
		Reason:      result.Reason,
		Key:         event.Info.Key,
		Type:        event.Info.Type,
		Description: result.Description,
		Sequence:    event.Info.Sequence,
		Protocol:    result.Protocol,
	}
	if !result.RegisteredAt.IsZero() {
		info.RegisteredAt = timestamp(result.RegisteredAt)
	}
	return RetEvento{Version: Version, Info: info}
}
