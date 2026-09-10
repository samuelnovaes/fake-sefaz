package dfews

import (
	"errors"
	"strconv"
	"time"

	"github.com/vendermais/fake-sefaz/internal/authorizer"
	"github.com/vendermais/fake-sefaz/internal/dfe"
	"github.com/vendermais/fake-sefaz/internal/soap"
	"github.com/vendermais/fake-sefaz/internal/status"
	"github.com/vendermais/fake-sefaz/internal/store"
)

const VerAplic = "fake-sefaz"

var ErrUnknownOperation = errors.New("unknown operation")

type Service struct {
	engine *authorizer.Engine
}

func NewService(engine *authorizer.Engine) *Service {
	return &Service{engine: engine}
}

func (s *Service) Operations() map[string]soap.Endpoint { return Operations() }

func (s *Service) WebServices() []soap.Service { return WebServices() }

func (s *Service) Handle(request authorizer.Context, message soap.Message) ([]byte, error) {
	current, known := operations[request.Operation]
	if !known {
		return nil, ErrUnknownOperation
	}
	if request.SchemaBroken() {
		return []byte(s.answer(current, request.Environment, request.UFCode, status.RejectedSchema, current.model, nil).String()), nil
	}
	switch current.kind {
	case kindReceive:
		return s.receive(current, request, message)
	case kindStatus:
		return s.status(current, request, message)
	case kindQuery:
		return s.query(current, request, message)
	case kindEvent:
		return s.event(current, request, message)
	case kindOpen:
		return s.open(current, request, message)
	}
	return nil, ErrUnknownOperation
}

func (s *Service) receive(current operation, request authorizer.Context, message soap.Message) ([]byte, error) {
	document, err := dfe.ParseDocument(message.Element)
	if err != nil {
		return []byte(s.answer(current, request.Environment, request.UFCode, status.RejectedSchema, current.model, nil).String()), nil
	}
	result := s.engine.Authorize(authorizer.Submission{
		Key:               document.Key,
		UFCode:            document.UFCode,
		Environment:       document.Environment,
		IssuerTaxID:       document.IssuerTaxID,
		DigestValue:       document.DigestValue,
		RecipientName:     document.RecipientName,
		RecipientDocument: document.RecipientDocument,
		Signed:            document.Signed,
		XML:               document.XML,
	}, request)

	response := element(current.spec.ReceiveResponse).
		attribute("versao", current.version()).
		attribute("xmlns", current.namespace()).
		field("tpAmb", strconv.Itoa(result.Environment)).
		field("verAplic", VerAplic).
		field("cStat", strconv.Itoa(int(result.Status))).
		field("xMotivo", result.Reason).
		field("cUF", result.UFCode)
	if result.Protocol != "" {
		response.child(protocolElement(result.Model, current.version(), result.Key, result.Protocol,
			result.DigestValue, result.ReceivedAt, result.Environment, result.Status, result.Reason))
	}
	return []byte(response.String()), nil
}

func (s *Service) status(current operation, request authorizer.Context, message soap.Message) ([]byte, error) {
	fields := dfe.Fields(message.Payload)
	code := s.engine.ServiceStatus(request)
	response := element(current.spec.StatusResponse).
		attribute("versao", current.version()).
		attribute("xmlns", current.namespace()).
		field("tpAmb", strconv.Itoa(authorizer.Environment(number(fields["tpAmb"]), request.Environment))).
		field("verAplic", VerAplic).
		field("cStat", strconv.Itoa(int(code))).
		field("xMotivo", status.MessageFor(current.model, code)).
		field("cUF", firstNonEmpty(fields["cUF"], request.UFCode)).
		field("dhRecbto", timestamp(s.engine.Now())).
		number("tMed", s.engine.Scenarios().AverageTime())
	if code != status.ServiceRunning {
		response.field("dhRetorno", timestamp(s.engine.Now().Add(5*time.Minute))).field("xObs", "fake-sefaz scenario")
	}
	return []byte(response.String()), nil
}

func (s *Service) query(current operation, request authorizer.Context, message soap.Message) ([]byte, error) {
	fields := dfe.Fields(message.Payload)
	key := dfe.FieldKey(fields)
	keyTag := current.model.Spec().KeyTag

	parsed, err := dfe.ParseAccessKey(key)
	if err != nil {
		return []byte(s.answer(current, authorizer.Environment(number(fields["tpAmb"]), request.Environment),
			request.UFCode, status.RejectedCheckDigit, current.model, nil).String()), nil
	}
	document, found := s.engine.Documents().Document(parsed.Raw)
	if !found {
		response := s.answer(current, authorizer.Environment(number(fields["tpAmb"]), request.Environment),
			parsed.UFCode, status.RejectedNotFound, parsed.Model, nil)
		response.field(keyTag, key)
		return []byte(response.String()), nil
	}

	code := document.Status
	if document.Cancelled {
		code = status.CancellationAuthorized
	}
	response := element(current.spec.QueryResponse).
		attribute("versao", current.version()).
		attribute("xmlns", current.namespace()).
		field("tpAmb", strconv.Itoa(document.Environment)).
		field("verAplic", VerAplic).
		field("cStat", strconv.Itoa(int(code))).
		field("xMotivo", status.MessageFor(parsed.Model, code)).
		field("cUF", document.UFCode).
		field(keyTag, document.Key).
		child(protocolElement(parsed.Model, current.version(), document.Key, document.Protocol,
			document.DigestValue, document.ReceivedAt, document.Environment, document.Status, document.Reason))
	for _, event := range document.Events {
		response.child(element(parsed.Model.Spec().EventProcTag).
			attribute("versao", current.version()).
			child(eventElement(current, parsed.Model, document.Key, event)))
	}
	return []byte(response.String()), nil
}

func (s *Service) event(current operation, request authorizer.Context, message soap.Message) ([]byte, error) {
	fields := dfe.Fields(message.Payload)
	key := dfe.FieldKey(fields)
	result := s.engine.RegisterEvent(authorizer.EventSubmission{
		Key:         key,
		Type:        fields["tpEvento"],
		Sequence:    number(fields["nSeqEvento"]),
		Environment: number(fields["tpAmb"]),
		OrganCode:   fields["cOrgao"],
		IssuerTaxID: firstNonEmpty(fields["CNPJ"], fields["CPF"]),
		Protocol:    fields["nProt"],
		XML:         string(message.Element),
	}, request)

	model := result.Model
	if model == "" {
		model = current.model
	}
	info := element("infEvento").
		attribute("Id", "ID"+fields["tpEvento"]+key+pad(number(fields["nSeqEvento"]))).
		field("tpAmb", strconv.Itoa(result.Environment)).
		field("verAplic", VerAplic).
		field("cOrgao", result.OrganCode).
		field("cStat", strconv.Itoa(int(result.Status))).
		field("xMotivo", result.Reason).
		field(model.Spec().KeyTag, key).
		field("tpEvento", fields["tpEvento"]).
		field("xEvento", result.Description).
		field("nSeqEvento", fields["nSeqEvento"])
	if !result.RegisteredAt.IsZero() {
		info.field("dhRegEvento", timestamp(result.RegisteredAt)).field("nProt", result.Protocol)
	}
	response := element(current.spec.EventResponse).
		attribute("versao", current.version()).
		attribute("xmlns", current.namespace()).
		child(info)
	return []byte(response.String()), nil
}

func (s *Service) open(current operation, request authorizer.Context, message soap.Message) ([]byte, error) {
	fields := dfe.Fields(message.Payload)
	environment := authorizer.Environment(number(fields["tpAmb"]), request.Environment)
	documents := s.engine.Documents().Documents(store.DocumentFilter{
		IssuerTaxID: firstNonEmpty(fields["CNPJ"], fields["CPF"]),
		Environment: environment,
		Model:       dfe.ModelMDFe,
	})
	open := make([]store.Document, 0, len(documents))
	for _, document := range documents {
		if document.Cancelled || document.Status != status.Authorized || closed(document) {
			continue
		}
		open = append(open, document)
	}
	code := status.DocumentFound
	if len(open) == 0 {
		code = status.NoDocumentFound
	}
	response := element(current.spec.OpenResponse).
		attribute("versao", current.version()).
		attribute("xmlns", current.namespace()).
		field("tpAmb", strconv.Itoa(environment)).
		field("verAplic", VerAplic).
		field("cStat", strconv.Itoa(int(code))).
		field("xMotivo", status.Message(code)).
		field("cUF", firstNonEmpty(fields["cUF"], request.UFCode))
	for _, document := range open {
		response.child(element("infMDFe").
			field(dfe.ModelMDFe.Spec().KeyTag, document.Key).
			field("nProt", document.Protocol))
	}
	return []byte(response.String()), nil
}

func closed(document store.Document) bool {
	for _, event := range document.Events {
		if event.Type == dfe.EventClosing && event.Status == status.EventLinked {
			return true
		}
	}
	return false
}

func (s *Service) answer(current operation, environment int, ufCode string, code status.Code, model dfe.Model, body *node) *node {
	response := element(responseTag(current)).
		attribute("versao", current.version()).
		attribute("xmlns", current.namespace()).
		field("tpAmb", strconv.Itoa(authorizer.Environment(environment))).
		field("verAplic", VerAplic).
		field("cStat", strconv.Itoa(int(code))).
		field("xMotivo", status.MessageFor(model, code)).
		field("cUF", ufCode)
	return response.child(body)
}

func responseTag(current operation) string {
	switch current.kind {
	case kindReceive:
		return current.spec.ReceiveResponse
	case kindQuery:
		return current.spec.QueryResponse
	case kindOpen:
		return current.spec.OpenResponse
	}
	return current.spec.StatusResponse
}

func protocolElement(model dfe.Model, version, key, protocol, digest string, receivedAt time.Time, environment int, code status.Code, reason string) *node {
	specification := model.Spec()
	return element(specification.ProtocolTag).
		attribute("versao", version).
		child(element("infProt").
			attribute("Id", "ID"+protocol).
			field("tpAmb", strconv.Itoa(environment)).
			field("verAplic", VerAplic).
			field(specification.KeyTag, key).
			field("dhRecbto", timestamp(receivedAt)).
			field("nProt", protocol).
			field("digVal", digest).
			field("cStat", strconv.Itoa(int(code))).
			field("xMotivo", reason))
}

func eventElement(current operation, model dfe.Model, key string, event store.Event) *node {
	return element(current.spec.EventResponse).
		attribute("versao", current.version()).
		child(element("infEvento").
			attribute("Id", "ID"+event.Type+key+pad(event.Sequence)).
			field("tpAmb", strconv.Itoa(authorizer.EnvironmentHomologation)).
			field("verAplic", VerAplic).
			field("cStat", strconv.Itoa(int(event.Status))).
			field("xMotivo", status.MessageFor(model, event.Status)).
			field(model.Spec().KeyTag, key).
			field("tpEvento", event.Type).
			field("xEvento", event.Description).
			field("nSeqEvento", strconv.Itoa(event.Sequence)).
			field("dhRegEvento", timestamp(event.RegisteredAt)).
			field("nProt", event.Protocol))
}

func timestamp(moment time.Time) string {
	return moment.Format("2006-01-02T15:04:05-07:00")
}

func number(value string) int {
	parsed, _ := strconv.Atoi(value)
	return parsed
}

func pad(sequence int) string {
	digits := "00" + strconv.Itoa(sequence)
	return digits[len(digits)-2:]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
