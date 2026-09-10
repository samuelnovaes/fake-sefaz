package authorizer

import (
	"time"

	"github.com/vendermais/fake-sefaz/internal/dfe"
	"github.com/vendermais/fake-sefaz/internal/scenario"
	"github.com/vendermais/fake-sefaz/internal/status"
	"github.com/vendermais/fake-sefaz/internal/store"
)

type EventSubmission struct {
	Key         string
	Type        string
	Sequence    int
	Environment int
	OrganCode   string
	IssuerTaxID string
	Protocol    string
	XML         string
}

type EventResult struct {
	Status       status.Code
	Reason       string
	Protocol     string
	Description  string
	OrganCode    string
	Environment  int
	Model        dfe.Model
	RegisteredAt time.Time
}

func (e *Engine) RegisterEvent(submission EventSubmission, context Context) EventResult {
	now := e.Now()
	result := EventResult{
		Environment: Environment(submission.Environment, context.Environment),
		OrganCode:   firstNonEmpty(submission.OrganCode, context.UFCode),
	}

	parsed, err := dfe.ParseAccessKey(submission.Key)
	if err != nil {
		return refuseEvent(result, status.RejectedCheckDigit)
	}
	result.Model = parsed.Model
	result.OrganCode = firstNonEmpty(submission.OrganCode, parsed.UFCode)

	description, known := dfe.EventDescription(parsed.Model, submission.Type)
	if !known {
		return refuseEvent(result, status.RejectedSchema)
	}
	result.Description = description

	if forced, matched := e.forced(context, scenario.Match{
		Operation:   context.Operation,
		IssuerTaxID: submission.IssuerTaxID,
		Key:         submission.Key,
		Model:       string(parsed.Model),
	}); matched && forced != status.EventLinked && forced != status.CancellationAuthorized {
		return refuseEvent(result, forced)
	}

	if e.documents.HasEvent(parsed.Raw, submission.Type, submission.Sequence) {
		return refuseEvent(result, status.RejectedDuplicateEvent)
	}

	document, found := e.documents.Document(parsed.Raw)
	if !found {
		if dfe.EventLinksToDocument(submission.Type) {
			return refuseEvent(result, status.RejectedNotFound)
		}
		return e.recordEvent(result, submission, parsed, status.EventNotLinked, now)
	}

	if submission.Type == dfe.EventCancellation {
		if document.Cancelled {
			return refuseEvent(result, status.RejectedDuplicateEvent)
		}
		if submission.Protocol != "" && submission.Protocol != document.Protocol {
			return refuseEvent(result, status.RejectedKeyMismatch)
		}
		if now.Sub(document.ReceivedAt) > e.cancellationWindow(parsed.Model) {
			return refuseEvent(result, status.RejectedCancellationDeadline)
		}
		return e.recordEvent(result, submission, parsed, status.CancellationAuthorized, now)
	}
	return e.recordEvent(result, submission, parsed, status.EventLinked, now)
}

func (e *Engine) recordEvent(result EventResult, submission EventSubmission, parsed dfe.AccessKey, code status.Code, now time.Time) EventResult {
	protocol := e.documents.NextProtocol(parsed.UFCode, now)
	result.Status = code
	result.Reason = status.MessageFor(result.Model, code)
	result.Protocol = protocol
	result.RegisteredAt = now
	e.documents.AppendEvent(parsed.Raw, store.Event{
		Key:          parsed.Raw,
		Type:         submission.Type,
		Description:  result.Description,
		Sequence:     submission.Sequence,
		Protocol:     protocol,
		Status:       code,
		RegisteredAt: now,
		XML:          submission.XML,
	})
	return result
}

func refuseEvent(result EventResult, code status.Code) EventResult {
	result.Status = code
	result.Reason = status.MessageFor(result.Model, code)
	result.Protocol = ""
	result.RegisteredAt = time.Time{}
	return result
}

func (e *Engine) cancellationWindow(model dfe.Model) time.Duration {
	if model == dfe.ModelNFCe {
		return e.options.CancellationWindowNFCe
	}
	return e.options.CancellationWindow
}
