package dfews

import (
	"github.com/vendermais/fake-sefaz/internal/dfe"
	"github.com/vendermais/fake-sefaz/internal/soap"
)

const (
	kindReceive = "receive"
	kindStatus  = "status"
	kindQuery   = "query"
	kindEvent   = "event"
	kindOpen    = "open"
)

type spec struct {
	Model           dfe.Model
	ResultTag       string
	ReceiveResponse string
	StatusResponse  string
	QueryResponse   string
	EventResponse   string
	OpenResponse    string
}

type operation struct {
	spec    *spec
	kind    string
	service string
	model   dfe.Model
}

var (
	cte  = spec{dfe.ModelCTe, "cteResultMsg", "retCTe", "retConsStatServCte", "retConsSitCTe", "retEventoCTe", ""}
	mdfe = spec{dfe.ModelMDFe, "mdfeResultMsg", "retMDFe", "retConsStatServMDFe", "retConsSitMDFe", "retEventoMDFe", "retConsMDFeNaoEnc"}
	bpe  = spec{dfe.ModelBPe, "bpeResultMsg", "retBPe", "retConsStatServBPe", "retConsSitBPe", "retEventoBPe", ""}
	nf3e = spec{dfe.ModelNF3e, "nf3eResultMsg", "retNF3e", "retConsStatServNF3e", "retConsSitNF3e", "retEventoNF3e", ""}
)

var operations = map[string]operation{
	"CTe":             {&cte, kindReceive, "CTeRecepcaoSincV4", dfe.ModelCTe},
	"CTeOS":           {&cte, kindReceive, "CTeRecepcaoOSV4", dfe.ModelCTeOS},
	"consStatServCte": {&cte, kindStatus, "CTeStatusServicoV4", dfe.ModelCTe},
	"consSitCTe":      {&cte, kindQuery, "CTeConsultaV4", dfe.ModelCTe},
	"eventoCTe":       {&cte, kindEvent, "CTeRecepcaoEventoV4", dfe.ModelCTe},

	"MDFe":             {&mdfe, kindReceive, "MDFeRecepcaoSinc", dfe.ModelMDFe},
	"consStatServMDFe": {&mdfe, kindStatus, "MDFeStatusServico", dfe.ModelMDFe},
	"consSitMDFe":      {&mdfe, kindQuery, "MDFeConsulta", dfe.ModelMDFe},
	"eventoMDFe":       {&mdfe, kindEvent, "MDFeRecepcaoEvento", dfe.ModelMDFe},
	"consMDFeNaoEnc":   {&mdfe, kindOpen, "MDFeConsNaoEnc", dfe.ModelMDFe},

	"BPe":             {&bpe, kindReceive, "BPeRecepcao", dfe.ModelBPe},
	"consStatServBPe": {&bpe, kindStatus, "BPeStatusServico", dfe.ModelBPe},
	"consSitBPe":      {&bpe, kindQuery, "BPeConsulta", dfe.ModelBPe},
	"eventoBPe":       {&bpe, kindEvent, "BPeRecepcaoEvento", dfe.ModelBPe},

	"NF3e":             {&nf3e, kindReceive, "NF3eRecepcao", dfe.ModelNF3e},
	"consStatServNF3e": {&nf3e, kindStatus, "NF3eStatusServico", dfe.ModelNF3e},
	"consSitNF3e":      {&nf3e, kindQuery, "NF3eConsulta", dfe.ModelNF3e},
	"eventoNF3e":       {&nf3e, kindEvent, "NF3eRecepcaoEvento", dfe.ModelNF3e},
}

var serviceOrder = []string{
	"CTe", "CTeOS", "consSitCTe", "consStatServCte", "eventoCTe",
	"MDFe", "consSitMDFe", "consStatServMDFe", "eventoMDFe", "consMDFeNaoEnc",
	"BPe", "consSitBPe", "consStatServBPe", "eventoBPe",
	"NF3e", "consSitNF3e", "consStatServNF3e", "eventoNF3e",
}

func (o operation) namespace() string {
	return o.model.Spec().Namespace
}

func (o operation) version() string {
	return o.model.Spec().Version
}

func (o operation) wsdl() string {
	return o.namespace() + "/wsdl/" + o.service
}

func Operations() map[string]soap.Endpoint {
	endpoints := make(map[string]soap.Endpoint, len(operations))
	for name, current := range operations {
		endpoints[name] = soap.Endpoint{WSDLNamespace: current.wsdl(), ResultTag: current.spec.ResultTag}
	}
	return endpoints
}

func WebServices() []soap.Service {
	services := make([]soap.Service, 0, len(serviceOrder))
	for _, name := range serviceOrder {
		current := operations[name]
		services = append(services, soap.Service{
			Operation: name,
			Name:      current.service,
			Models:    string(current.model),
		})
	}
	return services
}
