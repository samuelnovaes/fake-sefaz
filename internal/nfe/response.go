package nfe

import "encoding/xml"

type RetConsStatServ struct {
	XMLName     xml.Name `xml:"http://www.portalfiscal.inf.br/nfe retConsStatServ"`
	Version     string   `xml:"versao,attr"`
	Environment int      `xml:"tpAmb"`
	VerAplic    string   `xml:"verAplic"`
	Status      int      `xml:"cStat"`
	Reason      string   `xml:"xMotivo"`
	UFCode      string   `xml:"cUF"`
	ReceivedAt  string   `xml:"dhRecbto"`
	AverageTime int      `xml:"tMed,omitempty"`
	ReturnAt    string   `xml:"dhRetorno,omitempty"`
	Note        string   `xml:"xObs,omitempty"`
}

type InfProt struct {
	Identifier  string `xml:"Id,attr,omitempty"`
	Environment int    `xml:"tpAmb"`
	VerAplic    string `xml:"verAplic"`
	Key         string `xml:"chNFe,omitempty"`
	ReceivedAt  string `xml:"dhRecbto"`
	Protocol    string `xml:"nProt,omitempty"`
	DigestValue string `xml:"digVal,omitempty"`
	Status      int    `xml:"cStat"`
	Reason      string `xml:"xMotivo"`
}

type ProtNFe struct {
	XMLName xml.Name `xml:"protNFe"`
	Version string   `xml:"versao,attr"`
	Info    InfProt  `xml:"infProt"`
}

type InfRec struct {
	Receipt     string `xml:"nRec"`
	AverageTime int    `xml:"tMed"`
}

type RetEnviNFe struct {
	XMLName     xml.Name `xml:"http://www.portalfiscal.inf.br/nfe retEnviNFe"`
	Version     string   `xml:"versao,attr"`
	Environment int      `xml:"tpAmb"`
	VerAplic    string   `xml:"verAplic"`
	Status      int      `xml:"cStat"`
	Reason      string   `xml:"xMotivo"`
	UFCode      string   `xml:"cUF"`
	ReceivedAt  string   `xml:"dhRecbto"`
	Receipt     *InfRec  `xml:"infRec,omitempty"`
	Protocol    *ProtNFe `xml:"protNFe,omitempty"`
}

type RetConsReciNFe struct {
	XMLName     xml.Name  `xml:"http://www.portalfiscal.inf.br/nfe retConsReciNFe"`
	Version     string    `xml:"versao,attr"`
	Environment int       `xml:"tpAmb"`
	VerAplic    string    `xml:"verAplic"`
	Receipt     string    `xml:"nRec"`
	Status      int       `xml:"cStat"`
	Reason      string    `xml:"xMotivo"`
	UFCode      string    `xml:"cUF"`
	Protocols   []ProtNFe `xml:"protNFe"`
}

type ProcEventoNFe struct {
	XMLName xml.Name  `xml:"procEventoNFe"`
	Version string    `xml:"versao,attr"`
	Event   RetEvento `xml:"retEvento"`
}

type RetConsSitNFe struct {
	XMLName     xml.Name        `xml:"http://www.portalfiscal.inf.br/nfe retConsSitNFe"`
	Version     string          `xml:"versao,attr"`
	Environment int             `xml:"tpAmb"`
	VerAplic    string          `xml:"verAplic"`
	Status      int             `xml:"cStat"`
	Reason      string          `xml:"xMotivo"`
	UFCode      string          `xml:"cUF"`
	Key         string          `xml:"chNFe,omitempty"`
	Protocol    *ProtNFe        `xml:"protNFe,omitempty"`
	Events      []ProcEventoNFe `xml:"procEventoNFe,omitempty"`
}

type InfInutRet struct {
	Identifier  string `xml:"Id,attr,omitempty"`
	Environment int    `xml:"tpAmb"`
	VerAplic    string `xml:"verAplic"`
	Status      int    `xml:"cStat"`
	Reason      string `xml:"xMotivo"`
	UFCode      string `xml:"cUF"`
	Year        int    `xml:"ano,omitempty"`
	TaxID       string `xml:"CNPJ,omitempty"`
	Model       string `xml:"mod,omitempty"`
	Series      int    `xml:"serie,omitempty"`
	First       int64  `xml:"nNFIni,omitempty"`
	Last        int64  `xml:"nNFFin,omitempty"`
	ReceivedAt  string `xml:"dhRecbto,omitempty"`
	Protocol    string `xml:"nProt,omitempty"`
}

type RetInutNFe struct {
	XMLName xml.Name   `xml:"http://www.portalfiscal.inf.br/nfe retInutNFe"`
	Version string     `xml:"versao,attr"`
	Info    InfInutRet `xml:"infInut"`
}

type InfEventoRet struct {
	Identifier   string `xml:"Id,attr,omitempty"`
	Environment  int    `xml:"tpAmb"`
	VerAplic     string `xml:"verAplic"`
	OrganCode    string `xml:"cOrgao"`
	Status       int    `xml:"cStat"`
	Reason       string `xml:"xMotivo"`
	Key          string `xml:"chNFe,omitempty"`
	Type         string `xml:"tpEvento,omitempty"`
	Description  string `xml:"xEvento,omitempty"`
	Sequence     int    `xml:"nSeqEvento,omitempty"`
	TaxID        string `xml:"CNPJDest,omitempty"`
	RegisteredAt string `xml:"dhRegEvento,omitempty"`
	Protocol     string `xml:"nProt,omitempty"`
}

type RetEvento struct {
	XMLName xml.Name     `xml:"retEvento"`
	Version string       `xml:"versao,attr"`
	Info    InfEventoRet `xml:"infEvento"`
}

type RetEnvEvento struct {
	XMLName     xml.Name    `xml:"http://www.portalfiscal.inf.br/nfe retEnvEvento"`
	Version     string      `xml:"versao,attr"`
	BatchID     string      `xml:"idLote"`
	Environment int         `xml:"tpAmb"`
	VerAplic    string      `xml:"verAplic"`
	OrganCode   string      `xml:"cOrgao"`
	Status      int         `xml:"cStat"`
	Reason      string      `xml:"xMotivo"`
	Events      []RetEvento `xml:"retEvento,omitempty"`
}

type InfCad struct {
	StateID     string `xml:"IE"`
	TaxID       string `xml:"CNPJ,omitempty"`
	UF          string `xml:"UF"`
	StateStatus int    `xml:"cSit"`
	IndCredNFe  int    `xml:"indCredNFe"`
	IndCredCTe  int    `xml:"indCredCTe"`
	LegalName   string `xml:"xNome"`
	Regime      string `xml:"CNAE,omitempty"`
}

type InfConsRet struct {
	VerAplic  string   `xml:"verAplic"`
	Status    int      `xml:"cStat"`
	Reason    string   `xml:"xMotivo"`
	UF        string   `xml:"UF"`
	StateID   string   `xml:"IE,omitempty"`
	TaxID     string   `xml:"CNPJ,omitempty"`
	Person    string   `xml:"CPF,omitempty"`
	QueriedAt string   `xml:"dhCons"`
	UFCode    string   `xml:"cUF"`
	Records   []InfCad `xml:"infCad,omitempty"`
}

type RetConsCad struct {
	XMLName xml.Name   `xml:"http://www.portalfiscal.inf.br/nfe retConsCad"`
	Version string     `xml:"versao,attr"`
	Info    InfConsRet `xml:"infCons"`
}

type DocZip struct {
	NSU     string `xml:"NSU,attr"`
	Schema  string `xml:"schema,attr"`
	Content string `xml:",chardata"`
}

type LoteDistDFeInt struct {
	Documents []DocZip `xml:"docZip"`
}

type RetDistDFeInt struct {
	XMLName     xml.Name        `xml:"http://www.portalfiscal.inf.br/nfe retDistDFeInt"`
	Version     string          `xml:"versao,attr"`
	Environment int             `xml:"tpAmb"`
	VerAplic    string          `xml:"verAplic"`
	Status      int             `xml:"cStat"`
	Reason      string          `xml:"xMotivo"`
	RespondedAt string          `xml:"dhResp"`
	LastNSU     string          `xml:"ultNSU"`
	MaxNSU      string          `xml:"maxNSU"`
	Batch       *LoteDistDFeInt `xml:"loteDistDFeInt,omitempty"`
}
