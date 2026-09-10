package nfe

import (
	"encoding/xml"
	"errors"
	"time"

	"github.com/vendermais/fake-sefaz/internal/scenario"
	"github.com/vendermais/fake-sefaz/internal/status"
	"github.com/vendermais/fake-sefaz/internal/store"
)

const HomologationRecipientName = "NF-E EMITIDA EM AMBIENTE DE HOMOLOGACAO - SEM VALOR FISCAL"

var ErrUnknownOperation = errors.New("unknown operation")

type endpoint struct {
	WSDLNamespace string
	ResultTag     string
}

var endpoints = map[string]endpoint{
	"consStatServ": {"http://www.portalfiscal.inf.br/nfe/wsdl/NFeStatusServico4", "nfeResultMsg"},
	"enviNFe":      {"http://www.portalfiscal.inf.br/nfe/wsdl/NFeAutorizacao4", "nfeResultMsg"},
	"consReciNFe":  {"http://www.portalfiscal.inf.br/nfe/wsdl/NFeRetAutorizacao4", "nfeResultMsg"},
	"consSitNFe":   {"http://www.portalfiscal.inf.br/nfe/wsdl/NFeConsultaProtocolo4", "nfeResultMsg"},
	"inutNFe":      {"http://www.portalfiscal.inf.br/nfe/wsdl/NFeInutilizacao4", "nfeResultMsg"},
	"envEvento":    {"http://www.portalfiscal.inf.br/nfe/wsdl/NFeRecepcaoEvento4", "nfeResultMsg"},
	"ConsCad":      {"http://www.portalfiscal.inf.br/nfe/wsdl/CadConsultaCadastro4", "nfeResultMsg"},
	"distDFeInt":   {"http://www.portalfiscal.inf.br/nfe/wsdl/NFeDistribuicaoDFe", "nfeDistDFeInteresseResult"},
}

func Operations() map[string]bool {
	names := make(map[string]bool, len(endpoints))
	for name := range endpoints {
		names[name] = true
	}
	return names
}

func Endpoint(operation string) (endpoint, bool) {
	found, exists := endpoints[operation]
	return found, exists
}

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

type Service struct {
	documents *store.Store
	scenarios *scenario.Engine
	options   Options
	clock     func() time.Time
}

func NewService(documents *store.Store, scenarios *scenario.Engine, options Options, clock func() time.Time) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{documents: documents, scenarios: scenarios, options: options, clock: clock}
}

type Request struct {
	Operation    string
	Version      string
	Payload      []byte
	UFCode       string
	Environment  int
	ForcedStatus status.Code
}

func (s *Service) Handle(request Request) ([]byte, error) {
	switch request.Operation {
	case "consStatServ":
		return s.serviceStatus(request)
	case "enviNFe":
		return s.authorize(request)
	case "consReciNFe":
		return s.batchResult(request)
	case "consSitNFe":
		return s.documentStatus(request)
	case "inutNFe":
		return s.void(request)
	case "envEvento":
		return s.receiveEvents(request)
	case "ConsCad":
		return s.registration(request)
	case "distDFeInt":
		return s.distribute(request)
	}
	return nil, ErrUnknownOperation
}

func (s *Service) now() time.Time {
	return s.clock()
}

func timestamp(moment time.Time) string {
	return moment.Format("2006-01-02T15:04:05-07:00")
}

func encode(document any) ([]byte, error) {
	return xml.Marshal(document)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

type WebService struct {
	Operation string
	Name      string
}

var webServices = []WebService{
	{"enviNFe", "NFeAutorizacao4"},
	{"consReciNFe", "NFeRetAutorizacao4"},
	{"consSitNFe", "NFeConsultaProtocolo4"},
	{"consStatServ", "NFeStatusServico4"},
	{"inutNFe", "NFeInutilizacao4"},
	{"envEvento", "NFeRecepcaoEvento4"},
	{"ConsCad", "CadConsultaCadastro4"},
	{"distDFeInt", "NFeDistribuicaoDFe"},
}

func WebServices() []WebService {
	copied := make([]WebService, len(webServices))
	copy(copied, webServices)
	return copied
}
