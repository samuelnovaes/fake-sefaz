package nfe

import (
	"time"

	"github.com/vendermais/fake-sefaz/internal/dfe"
	"github.com/vendermais/fake-sefaz/internal/scenario"
	"github.com/vendermais/fake-sefaz/internal/status"
	"github.com/vendermais/fake-sefaz/internal/store"
)

func (s *Service) receiveEvents(request Request) ([]byte, error) {
	var batch EnvEvento
	if err := unmarshal(request.Payload, "envEvento", &batch); err != nil {
		return encode(RetEnvEvento{
			Version: Version, Environment: request.Environment, VerAplic: VerAplic, OrganCode: request.UFCode,
			Status: int(status.RejectedBatchSchema), Reason: status.Message(status.RejectedBatchSchema),
		})
	}
	now := s.now()
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
	for _, event := range batch.Events {
		result := s.receiveEvent(request, event, now)
		if response.Environment == 0 {
			response.Environment = result.Info.Environment
		}
		if response.OrganCode == "" {
			response.OrganCode = result.Info.OrganCode
		}
		response.Events = append(response.Events, result)
	}
	return encode(response)
}

func (s *Service) receiveEvent(request Request, event Evento, now time.Time) RetEvento {
	info := event.Info
	environment := environmentOf(info.Environment, request.Environment)
	description, known := dfe.EventDescription(info.Type)
	result := RetEvento{Version: Version, Info: InfEventoRet{
		Identifier:  "ID" + info.Type + info.Key + pad(info.Sequence),
		Environment: environment,
		VerAplic:    VerAplic,
		OrganCode:   firstNonEmpty(info.OrganCode, request.UFCode),
		Key:         info.Key,
		Type:        info.Type,
		Description: description,
		Sequence:    info.Sequence,
	}}
	if !known {
		return rejectEvent(result, status.RejectedSchema)
	}
	parsed, err := dfe.ParseAccessKey(info.Key)
	if err != nil {
		return rejectEvent(result, status.RejectedCheckDigit)
	}
	if result.Info.OrganCode == "" {
		result.Info.OrganCode = parsed.UFCode
	}

	if forced, matched := s.scenarios.Resolve(scenario.Match{
		Operation: "envEvento", IssuerTaxID: info.Document(), Key: info.Key, Model: string(parsed.Model),
	}); matched {
		if forced != status.EventLinked && forced != status.CancellationAuthorized {
			return rejectEvent(result, forced)
		}
	} else if request.ForcedStatus != 0 {
		return rejectEvent(result, request.ForcedStatus)
	}

	if s.documents.HasEvent(parsed.Raw, info.Type, info.Sequence) {
		return rejectEvent(result, status.RejectedDuplicateEvent)
	}

	document, found := s.documents.Document(parsed.Raw)
	if !found {
		if dfe.EventLinksToDocument(info.Type) {
			return rejectEvent(result, status.RejectedNotFound)
		}
		return s.registerEvent(result, event, parsed, status.EventNotLinked, now)
	}

	if info.Type == dfe.EventCancellation {
		if document.Cancelled {
			return rejectEvent(result, status.RejectedDuplicateEvent)
		}
		if info.Detail.Protocol != "" && info.Detail.Protocol != document.Protocol {
			return rejectEvent(result, status.RejectedKeyMismatch)
		}
		if now.Sub(document.ReceivedAt) > s.cancellationWindow(parsed.Model) {
			return rejectEvent(result, status.RejectedCancellationDeadline)
		}
		return s.registerEvent(result, event, parsed, status.CancellationAuthorized, now)
	}
	return s.registerEvent(result, event, parsed, status.EventLinked, now)
}

func (s *Service) registerEvent(result RetEvento, event Evento, parsed dfe.AccessKey, code status.Code, now time.Time) RetEvento {
	protocol := s.documents.NextProtocol(parsed.UFCode, now)
	result.Info.Status = int(code)
	result.Info.Reason = status.Message(code)
	result.Info.RegisteredAt = timestamp(now)
	result.Info.Protocol = protocol
	s.documents.AppendEvent(parsed.Raw, store.Event{
		Key:          parsed.Raw,
		Type:         event.Info.Type,
		Description:  result.Info.Description,
		Sequence:     event.Info.Sequence,
		Protocol:     protocol,
		Status:       code,
		RegisteredAt: now,
		XML:          `<evento xmlns="` + Namespace + `" versao="` + Version + `">` + string(event.Inner) + `</evento>`,
	})
	return result
}

func rejectEvent(result RetEvento, code status.Code) RetEvento {
	result.Info.Status = int(code)
	result.Info.Reason = status.Message(code)
	result.Info.Protocol = ""
	result.Info.RegisteredAt = ""
	return result
}

func (s *Service) cancellationWindow(model dfe.Model) time.Duration {
	if model == dfe.ModelNFCe {
		return s.options.CancellationWindowNFCe
	}
	return s.options.CancellationWindow
}
