package dfe

type Model string

const (
	ModelNFe   Model = "55"
	ModelNFCe  Model = "65"
	ModelCTe   Model = "57"
	ModelCTeOS Model = "67"
	ModelMDFe  Model = "58"
	ModelCFe   Model = "59"
	ModelBPe   Model = "63"
	ModelNF3e  Model = "66"
)

type ModelSpec struct {
	Model            Model    `json:"model"`
	Name             string   `json:"name"`
	Namespace        string   `json:"namespace"`
	Version          string   `json:"version"`
	RootTags         []string `json:"rootTags"`
	InfoTag          string   `json:"infoTag"`
	KeyTag           string   `json:"keyTag"`
	ProtocolTag      string   `json:"protocolTag"`
	ProcTag          string   `json:"procTag"`
	EventTag         string   `json:"eventTag"`
	EventProcTag     string   `json:"eventProcTag"`
	HomologationName string   `json:"homologationName,omitempty"`
	Transport        string   `json:"transport"`
}

const (
	homologationNFe = "NF-E EMITIDA EM AMBIENTE DE HOMOLOGACAO - SEM VALOR FISCAL"
	homologationCTe = "CT-E EMITIDO EM AMBIENTE DE HOMOLOGACAO - SEM VALOR FISCAL"

	TransportSOAP = "soap"
	TransportSAT  = "sat"
)

var modelSpecs = map[Model]ModelSpec{
	ModelNFe: {ModelNFe, "NF-e", "http://www.portalfiscal.inf.br/nfe", "4.00",
		[]string{"NFe"}, "infNFe", "chNFe", "protNFe", "nfeProc", "evento", "procEventoNFe", homologationNFe, TransportSOAP},
	ModelNFCe: {ModelNFCe, "NFC-e", "http://www.portalfiscal.inf.br/nfe", "4.00",
		[]string{"NFe"}, "infNFe", "chNFe", "protNFe", "nfeProc", "evento", "procEventoNFe", homologationNFe, TransportSOAP},
	ModelCTe: {ModelCTe, "CT-e", "http://www.portalfiscal.inf.br/cte", "4.00",
		[]string{"CTe"}, "infCte", "chCTe", "protCTe", "cteProc", "eventoCTe", "procEventoCTe", homologationCTe, TransportSOAP},
	ModelCTeOS: {ModelCTeOS, "CT-e OS", "http://www.portalfiscal.inf.br/cte", "4.00",
		[]string{"CTeOS"}, "infCte", "chCTe", "protCTe", "cteProc", "eventoCTe", "procEventoCTe", homologationCTe, TransportSOAP},
	ModelMDFe: {ModelMDFe, "MDF-e", "http://www.portalfiscal.inf.br/mdfe", "3.00",
		[]string{"MDFe"}, "infMDFe", "chMDFe", "protMDFe", "mdfeProc", "eventoMDFe", "procEventoMDFe", "", TransportSOAP},
	ModelBPe: {ModelBPe, "BP-e", "http://www.portalfiscal.inf.br/bpe", "1.00",
		[]string{"BPe"}, "infBPe", "chBPe", "protBPe", "bpeProc", "eventoBPe", "procEventoBPe", "", TransportSOAP},
	ModelNF3e: {ModelNF3e, "NF3e", "http://www.portalfiscal.inf.br/nf3e", "1.00",
		[]string{"NF3e"}, "infNF3e", "chNF3e", "protNF3e", "nf3eProc", "eventoNF3e", "procEventoNF3e", "", TransportSOAP},
	ModelCFe: {ModelCFe, "CF-e SAT", "", "0.08",
		[]string{"CFe"}, "infCFe", "chCFe", "", "", "", "", "", TransportSAT},
}

var modelOrder = []Model{ModelNFe, ModelNFCe, ModelCTe, ModelCTeOS, ModelMDFe, ModelBPe, ModelNF3e, ModelCFe}

func Lookup(model Model) (ModelSpec, bool) {
	spec, found := modelSpecs[model]
	return spec, found
}

func Models() []ModelSpec {
	specs := make([]ModelSpec, 0, len(modelOrder))
	for _, model := range modelOrder {
		specs = append(specs, modelSpecs[model])
	}
	return specs
}

func HomologationName(model Model) string {
	return modelSpecs[model].HomologationName
}

func (m Model) Valid() bool {
	_, found := modelSpecs[m]
	return found
}

func (m Model) Implemented() bool {
	_, found := modelSpecs[m]
	return found
}

func (m Model) Spec() ModelSpec {
	return modelSpecs[m]
}
