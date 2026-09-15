package authorizer

import (
	"time"

	"github.com/vendermais/fake-sefaz/internal/dfe"
	"github.com/vendermais/fake-sefaz/internal/status"
	"github.com/vendermais/fake-sefaz/internal/store"
)

type EventSubmission struct {
	Key           string
	Type          string
	Sequence      int
	Environment   int
	OrganCode     string
	IssuerTaxID   string
	Protocol      string
	SubstituteKey string
	XML           string
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
	AnswerLost   bool
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

	decision := e.outcome(context, matchOf(context, submission.IssuerTaxID, parsed))
	result.AnswerLost = decision.lost
	if decision.status != 0 && decision.status != status.EventLinked && decision.status != status.CancellationAuthorized {
		return refuseEvent(result, decision.status)
	}

	if parsed.Model == dfe.ModelNFCe && submission.Type == dfe.EventCancellationBySwap {
		return e.cancelBySubstitution(result, submission, parsed, now)
	}

	if e.documents.HasEvent(parsed.Raw, submission.Type, submission.Sequence) {
		return refuseEvent(result, status.RejectedDuplicateEvent)
	}

	document, found := e.documents.Document(parsed.Raw)
	if !found {
		if dfe.EventLinksToDocument(submission.Type) {
			return refuseEvent(result, status.RejectedNotFound)
		}
		return e.recordEvent(result, submission, parsed, status.EventNotLinked, false, now)
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
		return e.recordEvent(result, submission, parsed, status.CancellationAuthorized, true, now)
	}
	return e.recordEvent(result, submission, parsed, status.EventLinked, false, now)
}

func (e *Engine) recordEvent(result EventResult, submission EventSubmission, parsed dfe.AccessKey, code status.Code, cancelling bool, now time.Time) EventResult {
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
		Cancelling:   cancelling,
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
