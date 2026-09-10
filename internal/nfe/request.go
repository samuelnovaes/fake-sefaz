package nfe

import "encoding/xml"

const (
	Namespace = "http://www.portalfiscal.inf.br/nfe"
	Version   = "4.00"
	VerAplic  = "fake-sefaz"
)

type ConsStatServ struct {
	Environment int    `xml:"tpAmb"`
	UFCode      string `xml:"cUF"`
	Service     string `xml:"xServ"`
}

type EnviNFe struct {
	BatchID   string `xml:"idLote"`
	Sync      int    `xml:"indSinc"`
	Documents []NFe  `xml:"NFe"`
}

type NFe struct {
	Info      InfNFe    `xml:"infNFe"`
	Signature Signature `xml:"Signature"`
	Inner     []byte    `xml:",innerxml"`
}

type InfNFe struct {
	Identifier string `xml:"Id,attr"`
	Version    string `xml:"versao,attr"`
	Ide        Ide    `xml:"ide"`
	Issuer     Party  `xml:"emit"`
	Recipient  Party  `xml:"dest"`
}

type Ide struct {
	UFCode       string `xml:"cUF"`
	RandomCode   string `xml:"cNF"`
	Model        string `xml:"mod"`
	Series       int    `xml:"serie"`
	Number       int64  `xml:"nNF"`
	IssuedAt     string `xml:"dhEmi"`
	Environment  int    `xml:"tpAmb"`
	IssuanceKind int    `xml:"tpEmis"`
	CheckDigit   int    `xml:"cDV"`
}

type Party struct {
	TaxID     string `xml:"CNPJ"`
	PersonID  string `xml:"CPF"`
	LegalName string `xml:"xNome"`
}

func (p Party) Document() string {
	if p.TaxID != "" {
		return p.TaxID
	}
	return p.PersonID
}

type Signature struct {
	SignedInfo struct {
		Reference struct {
			URI         string `xml:"URI,attr"`
			DigestValue string `xml:"DigestValue"`
		} `xml:"Reference"`
	} `xml:"SignedInfo"`
	SignatureValue string `xml:"SignatureValue"`
}

type ConsReciNFe struct {
	Environment int    `xml:"tpAmb"`
	Receipt     string `xml:"nRec"`
}

type ConsSitNFe struct {
	Environment int    `xml:"tpAmb"`
	Service     string `xml:"xServ"`
	Key         string `xml:"chNFe"`
}

type InutNFe struct {
	Info InfInut `xml:"infInut"`
}

type InfInut struct {
	Identifier  string `xml:"Id,attr"`
	Environment int    `xml:"tpAmb"`
	Service     string `xml:"xServ"`
	UFCode      string `xml:"cUF"`
	Year        int    `xml:"ano"`
	TaxID       string `xml:"CNPJ"`
	Model       string `xml:"mod"`
	Series      int    `xml:"serie"`
	First       int64  `xml:"nNFIni"`
	Last        int64  `xml:"nNFFin"`
	Reason      string `xml:"xJust"`
}

type EnvEvento struct {
	BatchID string   `xml:"idLote"`
	Events  []Evento `xml:"evento"`
}

type Evento struct {
	Info  InfEvento `xml:"infEvento"`
	Inner []byte    `xml:",innerxml"`
}

type InfEvento struct {
	Identifier  string        `xml:"Id,attr"`
	OrganCode   string        `xml:"cOrgao"`
	Environment int           `xml:"tpAmb"`
	TaxID       string        `xml:"CNPJ"`
	PersonID    string        `xml:"CPF"`
	Key         string        `xml:"chNFe"`
	OccurredAt  string        `xml:"dhEvento"`
	Type        string        `xml:"tpEvento"`
	Sequence    int           `xml:"nSeqEvento"`
	Detail      EventoDetalhe `xml:"detEvento"`
}

type EventoDetalhe struct {
	Description string `xml:"descEvento"`
	Reason      string `xml:"xJust"`
	Protocol    string `xml:"nProt"`
	Correction  string `xml:"xCorrecao"`
}

func (i InfEvento) Document() string {
	if i.TaxID != "" {
		return i.TaxID
	}
	return i.PersonID
}

type ConsCad struct {
	Info InfConsCad `xml:"infCons"`
}

type InfConsCad struct {
	Service string `xml:"xServ"`
	UF      string `xml:"UF"`
	TaxID   string `xml:"CNPJ"`
	StateID string `xml:"IE"`
	Person  string `xml:"CPF"`
}

type DistDFeInt struct {
	Environment int    `xml:"tpAmb"`
	UFAuthor    string `xml:"cUFAutor"`
	TaxID       string `xml:"CNPJ"`
	PersonID    string `xml:"CPF"`
	DistNSU     struct {
		LastNSU string `xml:"ultNSU"`
	} `xml:"distNSU"`
	ConsNSU struct {
		NSU string `xml:"NSU"`
	} `xml:"consNSU"`
	ConsChNFe struct {
		Key string `xml:"chNFe"`
	} `xml:"consChNFe"`
}

func (d DistDFeInt) Document() string {
	if d.TaxID != "" {
		return d.TaxID
	}
	return d.PersonID
}

func unmarshal(payload []byte, root string, target any) error {
	wrapped := "<" + root + " xmlns=\"" + Namespace + "\">" + string(payload) + "</" + root + ">"
	return xml.Unmarshal([]byte(wrapped), target)
}
