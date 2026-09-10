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
	if forced, matched := e.forced(context, scenario.Match{
		Operation:   context.Operation,
		IssuerTaxID: submission.IssuerTaxID,
		Model:       string(submission.Model),
	}); matched && forced != status.VoidingAuthorized {
		return refuseVoid(result, forced)
	}

	identifier := store.VoidingIdentifier(environment, submission.UFCode, submission.Year,
		submission.IssuerTaxID, submission.Model, submission.Series, submission.First, submission.Last)
	if existing, found := e.documents.Voiding(identifier); found {
		return VoidResult{
			Status:     existing.Status,
			Reason:     status.Message(existing.Status),
			Protocol:   existing.Protocol,
			ReceivedAt: existing.ReceivedAt,
		}
	}
	for number := submission.First; number <= submission.Last; number++ {
		if _, found := e.documents.DocumentByNumber(environment, submission.IssuerTaxID, submission.Model, submission.Series, number); found {
			return refuseVoid(result, status.RejectedDuplicate)
		}
	}

	protocol := e.documents.NextProtocol(submission.UFCode, now)
	e.documents.SaveVoiding(store.Voiding{
		Identifier:  identifier,
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
	}
}

func refuseVoid(result VoidResult, code status.Code) VoidResult {
	result.Status = code
	result.Reason = status.Message(code)
	result.Protocol = ""
	result.ReceivedAt = time.Time{}
	return result
}
