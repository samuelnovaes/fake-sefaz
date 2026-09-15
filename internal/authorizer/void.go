package authorizer

import (
	"time"

	"github.com/vendermais/fake-sefaz/internal/dfe"
	"github.com/vendermais/fake-sefaz/internal/scenario"
	"github.com/vendermais/fake-sefaz/internal/status"
	"github.com/vendermais/fake-sefaz/internal/store"
)

type VoidSubmission struct {
	Environment int
	UFCode      string
	Year        int
	IssuerTaxID string
	Model       dfe.Model
	Series      int
	First       int64
	Last        int64
}

type VoidResult struct {
	Status     status.Code
	Reason     string
	Protocol   string
	ReceivedAt time.Time
	AnswerLost bool
}

func (e *Engine) Void(submission VoidSubmission, context Context) VoidResult {
	now := e.Now()
	environment := Environment(submission.Environment, context.Environment)
	result := VoidResult{}

	if !submission.Model.Implemented() {
		return refuseVoid(result, status.RejectedUncatalogued)
	}
	if submission.First <= 0 || submission.Last < submission.First {
		return refuseVoid(result, status.RejectedSchema)
	}
	decision := e.outcome(context, scenario.Match{
		Operation:   context.Operation,
		IssuerTaxID: submission.IssuerTaxID,
		Model:       string(submission.Model),
	})
	result.AnswerLost = decision.lost
	if decision.status != 0 && decision.status != status.VoidingAuthorized {
		return refuseVoid(result, decision.status)
	}

	numbering := store.Numbering{Environment: environment, IssuerTaxID: submission.IssuerTaxID, Model: submission.Model, Series: submission.Series}
	if existing, found := e.documents.VoidingOfRange(numbering, submission.First, submission.Last); found {
		refused := refuseVoid(result, status.RejectedVoidingRepeated)
		refused.Protocol = existing.Protocol
		return refused
	}
	if e.documents.RangeVoided(numbering, submission.First, submission.Last) {
		return refuseVoid(result, status.RejectedRangeVoided)
	}
	if e.documents.RangeUsed(numbering, submission.First, submission.Last) {
		return refuseVoid(result, status.RejectedRangeUsed)
	}

	protocol := e.documents.NextProtocol(submission.UFCode, now)
	e.documents.SaveVoiding(store.Voiding{
		Identifier: store.VoidingIdentifier(environment, submission.UFCode, submission.Year,
			submission.IssuerTaxID, submission.Model, submission.Series, submission.First, submission.Last),
		Environment: environment,
		UFCode:      submission.UFCode,
		Year:        submission.Year,
		IssuerTaxID: submission.IssuerTaxID,
		Model:       submission.Model,
		Series:      submission.Series,
		First:       submission.First,
		Last:        submission.Last,
		Protocol:    protocol,
		Status:      status.VoidingAuthorized,
		ReceivedAt:  now,
	})
	return VoidResult{
		Status:     status.VoidingAuthorized,
		Reason:     status.Message(status.VoidingAuthorized),
		Protocol:   protocol,
		ReceivedAt: now,
		AnswerLost: result.AnswerLost,
	}
}

func refuseVoid(result VoidResult, code status.Code) VoidResult {
	result.Status = code
	result.Reason = status.Message(code)
	result.Protocol = ""
	result.ReceivedAt = time.Time{}
	return result
}
