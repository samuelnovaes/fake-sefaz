package uf

import "sort"

type Authorizer string

const (
	AuthorizerOwn   Authorizer = "own"
	AuthorizerSVAN  Authorizer = "SVAN"
	AuthorizerSVRS  Authorizer = "SVRS"
	AuthorizerSVCAN Authorizer = "SVC-AN"
	AuthorizerSVCRS Authorizer = "SVC-RS"
)

type Unit struct {
	Acronym          string
	Code             string
	Name             string
	NFeAuthorizer    Authorizer
	NFCeAuthorizer   Authorizer
	ContingencyRoute Authorizer
}

var units = []Unit{
	{"RO", "11", "Rondonia", AuthorizerSVRS, AuthorizerSVRS, AuthorizerSVCAN},
	{"AC", "12", "Acre", AuthorizerSVRS, AuthorizerSVRS, AuthorizerSVCAN},
	{"AM", "13", "Amazonas", AuthorizerOwn, AuthorizerOwn, AuthorizerSVCRS},
	{"RR", "14", "Roraima", AuthorizerSVRS, AuthorizerSVRS, AuthorizerSVCAN},
	{"PA", "15", "Para", AuthorizerSVAN, AuthorizerSVRS, AuthorizerSVCAN},
	{"AP", "16", "Amapa", AuthorizerSVRS, AuthorizerSVRS, AuthorizerSVCAN},
	{"TO", "17", "Tocantins", AuthorizerSVRS, AuthorizerSVRS, AuthorizerSVCAN},
	{"MA", "21", "Maranhao", AuthorizerSVAN, AuthorizerSVRS, AuthorizerSVCRS},
	{"PI", "22", "Piaui", AuthorizerSVAN, AuthorizerSVRS, AuthorizerSVCAN},
	{"CE", "23", "Ceara", AuthorizerSVRS, AuthorizerOwn, AuthorizerSVCAN},
	{"RN", "24", "Rio Grande do Norte", AuthorizerSVRS, AuthorizerSVRS, AuthorizerSVCAN},
	{"PB", "25", "Paraiba", AuthorizerSVRS, AuthorizerSVRS, AuthorizerSVCAN},
	{"PE", "26", "Pernambuco", AuthorizerOwn, AuthorizerOwn, AuthorizerSVCRS},
	{"AL", "27", "Alagoas", AuthorizerSVRS, AuthorizerSVRS, AuthorizerSVCAN},
	{"SE", "28", "Sergipe", AuthorizerSVRS, AuthorizerSVRS, AuthorizerSVCAN},
	{"BA", "29", "Bahia", AuthorizerOwn, AuthorizerOwn, AuthorizerSVCRS},
	{"MG", "31", "Minas Gerais", AuthorizerOwn, AuthorizerOwn, AuthorizerSVCAN},
	{"ES", "32", "Espirito Santo", AuthorizerSVRS, AuthorizerSVRS, AuthorizerSVCAN},
	{"RJ", "33", "Rio de Janeiro", AuthorizerSVRS, AuthorizerSVRS, AuthorizerSVCAN},
	{"SP", "35", "Sao Paulo", AuthorizerOwn, AuthorizerOwn, AuthorizerSVCAN},
	{"PR", "41", "Parana", AuthorizerOwn, AuthorizerOwn, AuthorizerSVCRS},
	{"SC", "42", "Santa Catarina", AuthorizerSVRS, AuthorizerSVRS, AuthorizerSVCAN},
	{"RS", "43", "Rio Grande do Sul", AuthorizerOwn, AuthorizerOwn, AuthorizerSVCAN},
	{"MS", "50", "Mato Grosso do Sul", AuthorizerOwn, AuthorizerOwn, AuthorizerSVCRS},
	{"MT", "51", "Mato Grosso", AuthorizerOwn, AuthorizerOwn, AuthorizerSVCRS},
	{"GO", "52", "Goias", AuthorizerOwn, AuthorizerOwn, AuthorizerSVCRS},
	{"DF", "53", "Distrito Federal", AuthorizerSVRS, AuthorizerSVRS, AuthorizerSVCAN},
	{"AN", "91", "Ambiente Nacional", AuthorizerSVAN, AuthorizerSVAN, AuthorizerSVCAN},
}

var (
	byAcronym = map[string]Unit{}
	byCode    = map[string]Unit{}
)

func init() {
	for _, unit := range units {
		byAcronym[unit.Acronym] = unit
		byCode[unit.Code] = unit
	}
}

func All() []Unit {
	copied := make([]Unit, len(units))
	copy(copied, units)
	sort.Slice(copied, func(first, second int) bool { return copied[first].Acronym < copied[second].Acronym })
	return copied
}

func ByAcronym(acronym string) (Unit, bool) {
	unit, found := byAcronym[acronym]
	return unit, found
}

func ByCode(code string) (Unit, bool) {
	unit, found := byCode[code]
	return unit, found
}

func Known(code string) bool {
	_, found := byCode[code]
	return found
}
