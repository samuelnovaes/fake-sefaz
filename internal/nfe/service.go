package nfe

import (
	"encoding/xml"
	"errors"

	"github.com/vendermais/fake-sefaz/internal/authorizer"
	"github.com/vendermais/fake-sefaz/internal/soap"
)

var ErrUnknownOperation = errors.New("unknown operation")

var endpoints = map[string]soap.Endpoint{
	"consStatServ": {WSDLNamespace: "http://www.portalfiscal.inf.br/nfe/wsdl/NFeStatusServico4", ResultTag: "nfeResultMsg"},
	"enviNFe":      {WSDLNamespace: "http://www.portalfiscal.inf.br/nfe/wsdl/NFeAutorizacao4", ResultTag: "nfeResultMsg"},
	"consReciNFe":  {WSDLNamespace: "http://www.portalfiscal.inf.br/nfe/wsdl/NFeRetAutorizacao4", ResultTag: "nfeResultMsg"},
	"consSitNFe":   {WSDLNamespace: "http://www.portalfiscal.inf.br/nfe/wsdl/NFeConsultaProtocolo4", ResultTag: "nfeResultMsg"},
	"inutNFe":      {WSDLNamespace: "http://www.portalfiscal.inf.br/nfe/wsdl/NFeInutilizacao4", ResultTag: "nfeResultMsg"},
	"envEvento":    {WSDLNamespace: "http://www.portalfiscal.inf.br/nfe/wsdl/NFeRecepcaoEvento4", ResultTag: "nfeResultMsg"},
	"ConsCad":      {WSDLNamespace: "http://www.portalfiscal.inf.br/nfe/wsdl/CadConsultaCadastro4", ResultTag: "nfeResultMsg"},
	"distDFeInt":   {WSDLNamespace: "http://www.portalfiscal.inf.br/nfe/wsdl/NFeDistribuicaoDFe", ResultTag: "nfeDistDFeInteresseResult"},
}

var webServices = []soap.Service{
	{Operation: "enviNFe", Name: "NFeAutorizacao4", Models: "55, 65"},
	{Operation: "consReciNFe", Name: "NFeRetAutorizacao4", Models: "55, 65"},
	{Operation: "consSitNFe", Name: "NFeConsultaProtocolo4", Models: "55, 65"},
	{Operation: "consStatServ", Name: "NFeStatusServico4", Models: "55, 65"},
	{Operation: "inutNFe", Name: "NFeInutilizacao4", Models: "55, 65"},
	{Operation: "envEvento", Name: "NFeRecepcaoEvento4", Models: "55, 65"},
	{Operation: "ConsCad", Name: "CadConsultaCadastro4", Models: "55, 65"},
	{Operation: "distDFeInt", Name: "NFeDistribuicaoDFe", Models: "55, 65"},
}

func WebServices() []soap.Service {
	copied := make([]soap.Service, len(webServices))
	copy(copied, webServices)
	return copied
}

func Operations() map[string]soap.Endpoint {
	copied := make(map[string]soap.Endpoint, len(endpoints))
	for name, value := range endpoints {
		copied[name] = value
	}
	return copied
}

type Service struct {
	engine *authorizer.Engine
}

func NewService(engine *authorizer.Engine) *Service {
	return &Service{engine: engine}
}

func (s *Service) Operations() map[string]soap.Endpoint { return Operations() }

func (s *Service) WebServices() []soap.Service { return WebServices() }

func (s *Service) Handle(request authorizer.Context, message soap.Message) ([]byte, error) {
	payload := message.Payload
	if request.SchemaBroken() {
		return s.schemaRejection(request)
	}
	switch request.Operation {
	case "consStatServ":
		return s.serviceStatus(request, payload)
	case "enviNFe":
		return s.authorize(request, message.Version, payload)
	case "consReciNFe":
		return s.batchResult(request, payload)
	case "consSitNFe":
		return s.documentStatus(request, payload)
	case "inutNFe":
		return s.void(request, payload)
	case "envEvento":
		return s.receiveEvents(request, payload)
	case "ConsCad":
		return s.registration(request, payload)
	case "distDFeInt":
		return s.distribute(request, payload)
	}
	return nil, ErrUnknownOperation
}

func encode(document any) ([]byte, error) {
	return xml.Marshal(document)
}

func deliver(document any, lost bool) ([]byte, error) {
	if lost {
		return nil, authorizer.ErrAnswerLost
	}
	return encode(document)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
