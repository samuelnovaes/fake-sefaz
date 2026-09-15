package authorizer

import (
	"github.com/vendermais/fake-sefaz/internal/dfe"
	"github.com/vendermais/fake-sefaz/internal/status"
	"github.com/vendermais/fake-sefaz/internal/store"
)

type ConsultResult struct {
	Status   status.Code
	Reason   string
	UFCode   string
	Document store.Document
	Found    bool
}

func (e *Engine) Consult(key string, environment int) ConsultResult {
	parsed, err := dfe.ParseAccessKey(key)
	if err != nil {
		return consultAnswer(ConsultResult{}, status.RejectedCheckDigit)
	}
	result := ConsultResult{UFCode: parsed.UFCode}
	if document, found := e.documents.Document(parsed.Raw); found {
		return situation(result, document)
	}
	numbering := store.Numbering{Environment: environment, IssuerTaxID: parsed.IssuerTaxID, Model: parsed.Model, Series: parsed.Series}
	stored, found := e.documents.DocumentByNumber(numbering, parsed.Number)
	if !found {
		return consultAnswer(result, status.RejectedNotFound)
	}
	return keyDivergence(result, parsed, stored)
}

func situation(result ConsultResult, document store.Document) ConsultResult {
	result.Document = document
	result.Found = true
	if document.Cancelled {
		return consultAnswer(result, status.CancellationAuthorized)
	}
	return consultAnswer(result, document.Status)
}

func keyDivergence(result ConsultResult, queried dfe.AccessKey, stored store.Document) ConsultResult {
	existing, _ := dfe.ParseAccessKey(stored.Key)
	if existing.RandomCode != queried.RandomCode {
		answered := consultAnswer(result, status.RejectedRandomCodeMismatch)
		answered.Reason += " [chNFe:" + stored.Key + "]"
		return answered
	}
	if existing.Month != queried.Month {
		return consultAnswer(result, status.RejectedMonthMismatch)
	}
	return consultAnswer(result, status.RejectedKeyMismatch)
}

func consultAnswer(result ConsultResult, code status.Code) ConsultResult {
	result.Status = code
	result.Reason = status.Message(code)
	return result
}
