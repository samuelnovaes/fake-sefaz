package authorizer

import (
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

type Options struct {
	CancellationWindow       time.Duration
	CancellationWindowNFCe   time.Duration
	MaxDistributionDocuments int
}

func DefaultOptions() Options {
	return Options{
		CancellationWindow:       24 * time.Hour,
		CancellationWindowNFCe:   30 * time.Minute,
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

type Context struct {
	Operation    string
	UFCode       string
	Environment  int
	ForcedStatus status.Code
}

type Submission struct {
	Key               string
	UFCode            string
	Environment       int
	IssuerTaxID       string
	DigestValue       string
	RecipientName     string
	RecipientDocument string
	Signed            bool
	XML               string
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

	if forced, matched := e.forced(context, scenario.Match{
		Operation:   context.Operation,
		IssuerTaxID: submission.IssuerTaxID,
		Key:         submission.Key,
		Model:       string(parsed.Model),
	}); matched {
		return e.settle(result, forced, submission, parsed, now)
	}

	if _, found := e.documents.Document(parsed.Raw); found {
		return refuse(result, status.RejectedDuplicate)
	}
	if _, found := e.documents.DocumentByNumber(environment, submission.IssuerTaxID, parsed.Model, parsed.Series, parsed.Number); found {
		return refuse(result, status.RejectedDuplicateOtherKey)
	}
	if e.documents.NumberVoided(environment, submission.IssuerTaxID, parsed.Model, parsed.Series, parsed.Number) {
		return refuse(result, status.RejectedDuplicate)
	}
	return e.settle(result, status.Authorized, submission, parsed, now)
}

func (e *Engine) settle(result Result, code status.Code, submission Submission, parsed dfe.AccessKey, now time.Time) Result {
	if code != status.Authorized && !status.Denied(code) {
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
		XML:         submission.XML,
	})
	return result
}

func (e *Engine) forced(context Context, match scenario.Match) (status.Code, bool) {
	if code, matched := e.scenarios.Resolve(match); matched {
		return code, true
	}
	if context.ForcedStatus != 0 {
		return context.ForcedStatus, true
	}
	return 0, false
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
