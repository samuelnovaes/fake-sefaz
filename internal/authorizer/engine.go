package authorizer

import (
	"errors"
	"strconv"
	"time"

	"github.com/vendermais/fake-sefaz/internal/dfe"
	"github.com/vendermais/fake-sefaz/internal/scenario"
	"github.com/vendermais/fake-sefaz/internal/status"
	"github.com/vendermais/fake-sefaz/internal/store"
	"github.com/vendermais/fake-sefaz/internal/uf"
)

const (
	EnvironmentProduction   = 1
	EnvironmentHomologation = 2
)

var ErrAnswerLost = errors.New("answer lost by scenario")

type Options struct {
	CancellationWindow       time.Duration
	CancellationWindowNFCe   time.Duration
	SubstitutionWindow       time.Duration
	OfflineDeadline          time.Duration
	MaxDistributionDocuments int
}

func DefaultOptions() Options {
	return Options{
		CancellationWindow:       24 * time.Hour,
		CancellationWindowNFCe:   30 * time.Minute,
		SubstitutionWindow:       168 * time.Hour,
		OfflineDeadline:          24 * time.Hour,
		MaxDistributionDocuments: 50,
	}
}

type Engine struct {
	documents *store.Store
	scenarios *scenario.Engine
	options   Options
	clock     func() time.Time
}

func New(documents *store.Store, scenarios *scenario.Engine, options Options, clock func() time.Time) *Engine {
	if clock == nil {
		clock = time.Now
	}
	return &Engine{documents: documents, scenarios: scenarios, options: options, clock: clock}
}

func (e *Engine) Now() time.Time              { return e.clock() }
func (e *Engine) Documents() *store.Store     { return e.documents }
func (e *Engine) Scenarios() *scenario.Engine { return e.scenarios }
func (e *Engine) Options() Options            { return e.options }

type SchemaFailure struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

func (f SchemaFailure) String() string {
	return f.Path + ": " + f.Message
}

type Context struct {
	Operation      string
	UFCode         string
	Environment    int
	ForcedStatus   status.Code
	SchemaFailures []SchemaFailure
}

func (c Context) SchemaBroken() bool {
	return len(c.SchemaFailures) > 0
}

func (c Context) SchemaReason() string {
	if len(c.SchemaFailures) == 0 {
		return ""
	}
	return c.SchemaFailures[0].String()
}

type Item = store.Item

type Submission struct {
	Key                string
	UFCode             string
	Environment        int
	IssuerTaxID        string
	DigestValue        string
	RecipientName      string
	RecipientDocument  string
	RecipientForeignID string
	RecipientStateID   string
	Signed             bool
	Purpose            int
	IssuedAt           string
	Items              []Item
	DiscountTotal      string
	Total              string
	ICMSTotal          string
	XML                string
}

type Result struct {
	Status      status.Code
	Reason      string
	Protocol    string
	Key         string
	Model       dfe.Model
	UFCode      string
	Environment int
	DigestValue string
	ReceivedAt  time.Time
	AnswerLost  bool
}

type outcome struct {
	status status.Code
	lost   bool
}

func (e *Engine) Authorize(submission Submission, context Context) Result {
	now := e.Now()
	environment := Environment(submission.Environment, context.Environment)
	result := Result{
		Key:         submission.Key,
		UFCode:      firstNonEmpty(submission.UFCode, context.UFCode),
		Environment: environment,
		DigestValue: submission.DigestValue,
		ReceivedAt:  now,
	}

	parsed, err := dfe.ParseAccessKey(submission.Key)
	if err != nil {
		return refuse(result, status.RejectedCheckDigit)
	}
	result.Model = parsed.Model
	result.UFCode = firstNonEmpty(submission.UFCode, parsed.UFCode)

	if !uf.Known(result.UFCode) || result.UFCode != parsed.UFCode {
		return refuse(result, status.RejectedIssuerUF)
	}
	if context.Environment != 0 && submission.Environment != 0 && submission.Environment != context.Environment {
		return refuse(result, status.RejectedEnvironment)
	}
	if !parsed.Model.Implemented() {
		return refuse(result, status.RejectedUncatalogued)
	}
	if !submission.Signed {
		return refuse(result, status.RejectedSignature)
	}
	expected := dfe.HomologationName(parsed.Model)
	if environment == EnvironmentHomologation && expected != "" && submission.RecipientDocument != "" && submission.RecipientName != expected {
		return refuse(result, status.RejectedHomologationName)
	}
	if code, rejected := amountRejection(submission); rejected {
		return refuse(result, code)
	}

	decision := e.outcome(context, matchOf(context, submission.IssuerTaxID, parsed))
	result.AnswerLost = decision.lost
	if decision.status != 0 {
		return e.settle(result, decision.status, submission, parsed, now)
	}
	if code, rejected := e.duplicateRejection(environment, submission.IssuerTaxID, parsed); rejected {
		return refuse(result, code)
	}
	return e.settle(result, e.authorizationStatus(submission, parsed, now), submission, parsed, now)
}

func (e *Engine) duplicateRejection(environment int, issuerTaxID string, parsed dfe.AccessKey) (status.Code, bool) {
	if document, found := e.documents.Document(parsed.Raw); found {
		return storedDocumentRejection(document), true
	}
	numbering := store.Numbering{Environment: environment, IssuerTaxID: issuerTaxID, Model: parsed.Model, Series: parsed.Series}
	if _, found := e.documents.DocumentByNumber(numbering, parsed.Number); found {
		return status.RejectedDuplicateOtherKey, true
	}
	if e.documents.RangeVoided(numbering, parsed.Number, parsed.Number) {
		return status.RejectedVoided, true
	}
	return 0, false
}

func storedDocumentRejection(document store.Document) status.Code {
	if document.Cancelled {
		return status.RejectedCancelled
	}
	if status.Denied(document.Status) {
		return status.RejectedDenied
	}
	return status.RejectedDuplicate
}

func (e *Engine) authorizationStatus(submission Submission, parsed dfe.AccessKey, now time.Time) status.Code {
	if parsed.Model != dfe.ModelNFCe || !contingencyIssuance(parsed.IssuanceKind) {
		return status.Authorized
	}
	issuedAt := parseMoment(submission.IssuedAt)
	if issuedAt.IsZero() || now.Sub(issuedAt) <= e.options.OfflineDeadline {
		return status.Authorized
	}
	return status.AuthorizedLate
}

func contingencyIssuance(kind int) bool {
	return kind == dfe.IssuanceEPEC || kind == dfe.IssuanceOffline
}

func parseMoment(text string) time.Time {
	moment, err := time.Parse(time.RFC3339, text)
	if err != nil {
		return time.Time{}
	}
	return moment
}

func (e *Engine) settle(result Result, code status.Code, submission Submission, parsed dfe.AccessKey, now time.Time) Result {
	if !status.InUse(code) && !status.Denied(code) {
		return refuse(result, code)
	}
	result.Protocol = e.documents.NextProtocol(parsed.UFCode, now)
	result.Status = code
	result.Reason = status.MessageFor(result.Model, code)
	e.documents.SaveDocument(store.Document{
		Key:         parsed.Raw,
		Model:       parsed.Model,
		Environment: result.Environment,
		UFCode:      parsed.UFCode,
		IssuerTaxID: submission.IssuerTaxID,
		Series:      parsed.Series,
		Number:      parsed.Number,
		DigestValue: submission.DigestValue,
		Protocol:    result.Protocol,
		Status:      code,
		Reason:      status.MessageFor(result.Model, code),
		ReceivedAt:  now,
		IssuedAt:    parseMoment(submission.IssuedAt),
		Total:       submission.Total,
		ICMSTotal:   submission.ICMSTotal,
		Recipient: store.Recipient{
			Document:  submission.RecipientDocument,
			ForeignID: submission.RecipientForeignID,
			StateID:   submission.RecipientStateID,
		},
		Items: submission.Items,
		XML:   submission.XML,
	})
	return result
}

func matchOf(context Context, issuerTaxID string, parsed dfe.AccessKey) scenario.Match {
	return scenario.Match{
		Operation:   context.Operation,
		IssuerTaxID: issuerTaxID,
		Key:         parsed.Raw,
		Model:       string(parsed.Model),
		Issuance:    strconv.Itoa(parsed.IssuanceKind),
	}
}

func (e *Engine) outcome(context Context, match scenario.Match) outcome {
	rule, matched := e.scenarios.Resolve(match)
	if matched && rule.Status != 0 {
		return outcome{status: rule.Status, lost: rule.LoseAnswer}
	}
	return outcome{status: context.ForcedStatus, lost: matched && rule.LoseAnswer}
}

func refuse(result Result, code status.Code) Result {
	result.Status = code
	result.Reason = status.MessageFor(result.Model, code)
	result.Protocol = ""
	return result
}

func Environment(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return EnvironmentHomologation
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func (e *Engine) ServiceStatus(context Context) status.Code {
	if context.ForcedStatus != 0 {
		return context.ForcedStatus
	}
	return e.scenarios.ServiceStatus()
}
