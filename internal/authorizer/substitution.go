package authorizer

import (
	"time"

	"github.com/vendermais/fake-sefaz/internal/dfe"
	"github.com/vendermais/fake-sefaz/internal/status"
	"github.com/vendermais/fake-sefaz/internal/store"
	"github.com/vendermais/fake-sefaz/internal/uf"
)

const substituteIssuanceWindow = 2 * time.Hour

const firstKeyYear = 2006

type keyRejection struct {
	code   status.Code
	detail string
}

func (e *Engine) cancelBySubstitution(result EventResult, submission EventSubmission, parsed dfe.AccessKey, now time.Time) EventResult {
	if parsed.IssuanceKind != dfe.IssuanceNormal {
		return refuseEvent(result, status.RejectedSubstitutionIssuance)
	}
	substitute, rejection, rejected := substituteKeyRejection(parsed, submission.SubstituteKey, now)
	if rejected {
		refused := refuseEvent(result, rejection.code)
		refused.Reason += " (" + rejection.detail + ")"
		return refused
	}
	if e.documents.HasEvent(parsed.Raw, submission.Type, submission.Sequence) {
		return refuseEvent(result, status.RejectedDuplicateEvent)
	}
	document, found := e.documents.Document(parsed.Raw)
	if !found {
		return refuseEvent(result, status.RejectedKeyInexistent)
	}
	if code, rejected := e.cancelledDocumentRejection(document, submission, now); rejected {
		return refuseEvent(result, code)
	}
	if code, rejected := e.substituteRejection(document, substitute); rejected {
		return refuseEvent(result, code)
	}
	return e.recordEvent(result, submission, parsed, status.EventLinked, true, now)
}

func (e *Engine) cancelledDocumentRejection(document store.Document, submission EventSubmission, now time.Time) (status.Code, bool) {
	if now.Sub(document.ReceivedAt) > e.options.SubstitutionWindow {
		return status.RejectedCancellationDeadline, true
	}
	if document.Cancelled || status.Denied(document.Status) {
		return status.RejectedEventNeedsAuthorized, true
	}
	if submission.Protocol != document.Protocol {
		return status.RejectedProtocolMismatch, true
	}
	return 0, false
}

func substituteKeyRejection(cancelled dfe.AccessKey, key string, now time.Time) (dfe.AccessKey, keyRejection, bool) {
	substitute, err := dfe.ParseAccessKey(key)
	if err != nil {
		return substitute, keyRejection{status.RejectedSubstituteKeyInvalid, "Digito"}, true
	}
	if detail, invalid := invalidKeyField(substitute, now); invalid {
		return substitute, keyRejection{status.RejectedSubstituteKeyInvalid, detail}, true
	}
	if detail, incorrect := incorrectKeyField(cancelled, substitute); incorrect {
		return substitute, keyRejection{status.RejectedSubstituteKeyIncorrect, detail}, true
	}
	return substitute, keyRejection{}, false
}

func invalidKeyField(key dfe.AccessKey, now time.Time) (string, bool) {
	switch {
	case !uf.Known(key.UFCode):
		return "Codigo UF", true
	case key.Year < firstKeyYear || key.Year > now.Year():
		return "Ano", true
	case key.Month < 1 || key.Month > 12:
		return "Mes", true
	case !validIssuerDocument(key):
		return "CNPJ/CPF", true
	case key.Model != dfe.ModelNFe && key.Model != dfe.ModelNFCe:
		return "Modelo", true
	case key.Number == 0:
		return "Numero", true
	}
	return "", false
}

func incorrectKeyField(cancelled, substitute dfe.AccessKey) (string, bool) {
	switch {
	case substitute.Raw == cancelled.Raw:
		return "mesma Chave de Acesso", true
	case substitute.UFCode != cancelled.UFCode:
		return "Codigo da UF", true
	case substitute.IssuerTaxID != cancelled.IssuerTaxID:
		return "CNPJ/CPF", true
	case yearMonth(substitute) > yearMonth(cancelled) || yearMonth(substitute) < yearMonth(cancelled)-1:
		return "Ano-Mes", true
	case substitute.Model != cancelled.Model:
		return "Modelo", true
	}
	return "", false
}

func yearMonth(key dfe.AccessKey) int {
	return key.Year*12 + key.Month - 1
}

func (e *Engine) substituteRejection(cancelled store.Document, key dfe.AccessKey) (status.Code, bool) {
	substitute, found := e.documents.Document(key.Raw)
	switch {
	case !found:
		return status.RejectedSubstituteMissing, true
	case substitute.Cancelled || status.Denied(substitute.Status):
		return status.RejectedSubstituteUnavailable, true
	case issuedTooLate(cancelled, substitute):
		return status.RejectedSubstituteIssuedLate, true
	case !sameDecimal(substitute.Total, cancelled.Total):
		return status.RejectedSubstituteTotal, true
	case !sameDecimal(substitute.ICMSTotal, cancelled.ICMSTotal):
		return status.RejectedSubstituteICMSTotal, true
	case substitute.Recipient != cancelled.Recipient:
		return status.RejectedSubstituteRecipient, true
	case len(substitute.Items) != len(cancelled.Items):
		return status.RejectedSubstituteItemCount, true
	case !sameItems(substitute.Items, cancelled.Items):
		return status.RejectedSubstituteItem, true
	case key.IssuanceKind == dfe.IssuanceNormal:
		return status.RejectedSubstituteNotContingency, true
	}
	return 0, false
}

func issuedTooLate(cancelled, substitute store.Document) bool {
	if cancelled.IssuedAt.IsZero() || substitute.IssuedAt.IsZero() {
		return false
	}
	return substitute.IssuedAt.Sub(cancelled.IssuedAt) > substituteIssuanceWindow
}

func sameItems(first, second []Item) bool {
	for index := range first {
		if !sameItem(first[index], second[index]) {
			return false
		}
	}
	return true
}

func sameItem(first, second Item) bool {
	return first.Code == second.Code &&
		first.EAN == second.EAN &&
		first.Description == second.Description &&
		first.NCM == second.NCM &&
		first.CFOP == second.CFOP &&
		first.Unit == second.Unit &&
		first.TotalIndicator == second.TotalIndicator &&
		sameDecimal(first.Quantity, second.Quantity) &&
		sameDecimal(first.UnitValue, second.UnitValue) &&
		sameDecimal(first.Value, second.Value)
}
