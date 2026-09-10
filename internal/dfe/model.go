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
	Model       Model
	Name        string
	RootTag     string
	InfoTag     string
	KeyPrefix   string
	Implemented bool
}

var modelSpecs = map[Model]ModelSpec{
	ModelNFe:   {ModelNFe, "NF-e", "NFe", "infNFe", "NFe", true},
	ModelNFCe:  {ModelNFCe, "NFC-e", "NFe", "infNFe", "NFe", true},
	ModelCTe:   {ModelCTe, "CT-e", "CTe", "infCte", "CTe", false},
	ModelCTeOS: {ModelCTeOS, "CT-e OS", "CTeOS", "infCte", "CTe", false},
	ModelMDFe:  {ModelMDFe, "MDF-e", "MDFe", "infMDFe", "MDFe", false},
	ModelCFe:   {ModelCFe, "CF-e SAT", "CFe", "infCFe", "CFe", false},
	ModelBPe:   {ModelBPe, "BP-e", "BPe", "infBPe", "BPe", false},
	ModelNF3e:  {ModelNF3e, "NF3e", "NF3e", "infNF3e", "NF3e", false},
}

func Lookup(model Model) (ModelSpec, bool) {
	spec, found := modelSpecs[model]
	return spec, found
}

func Models() []ModelSpec {
	ordered := []Model{ModelNFe, ModelNFCe, ModelCTe, ModelCTeOS, ModelMDFe, ModelCFe, ModelBPe, ModelNF3e}
	specs := make([]ModelSpec, 0, len(ordered))
	for _, model := range ordered {
		specs = append(specs, modelSpecs[model])
	}
	return specs
}

func (m Model) Valid() bool {
	_, found := modelSpecs[m]
	return found
}

func (m Model) Implemented() bool {
	spec, found := modelSpecs[m]
	return found && spec.Implemented
}
