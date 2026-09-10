package server

import (
	"net/http"
	"strings"
	"testing"

	"github.com/vendermais/fake-sefaz/internal/dfe"
	"github.com/vendermais/fake-sefaz/internal/xsd"
)

func newValidatingHarness(t *testing.T) *harness {
	t.Helper()
	testHarness := newHarness(t)
	schemas, err := xsd.Load("testdata/schemas")
	if err != nil {
		t.Fatalf("load schemas: %v", err)
	}
	testHarness.handler.UseSchemas(schemas)
	return testHarness
}

func TestSchemaValidationLetsAValidDocumentThrough(t *testing.T) {
	testHarness := newValidatingHarness(t)

	body := testHarness.post(t, "/homologacao/SP/NFeAutorizacao4", buildBatch(defaultDocument(1)), nil)
	if !strings.Contains(body, "<cStat>100</cStat>") {
		t.Fatalf("a schema valid document must still be authorized: %s", body)
	}
}

func TestSchemaValidationRejectsTheBatch(t *testing.T) {
	testHarness := newValidatingHarness(t)
	cases := []struct {
		name     string
		from, to string
	}{
		{"CNPJ com treze digitos", "<CNPJ>" + issuerTaxID + "</CNPJ>", "<CNPJ>9999999900019</CNPJ>"},
		{"ambiente fora da lista", "<tpAmb>2</tpAmb>", "<tpAmb>7</tpAmb>"},
		{"modelo invalido", "<mod>65</mod>", "<mod>99</mod>"},
		{"elemento obrigatorio ausente", "<natOp>Venda</natOp>", ""},
		{"elemento desconhecido", "</ide>", "<inventado>1</inventado></ide>"},
	}
	for _, test := range cases {
		payload := strings.Replace(buildBatch(defaultDocument(2)), test.from, test.to, 1)
		if payload == buildBatch(defaultDocument(2)) {
			t.Fatalf("%s: a mutacao nao aplicou", test.name)
		}
		body := testHarness.post(t, "/homologacao/SP/NFeAutorizacao4", payload, nil)
		if got := statusOf(t, body); got != "225" {
			t.Errorf("%s: expected 225, got %s in %s", test.name, got, body)
		}
	}
}

func TestSchemaValidationRejectsOtherMessagesWith215(t *testing.T) {
	testHarness := newValidatingHarness(t)
	body := testHarness.post(t, "/homologacao/SP/NFeConsultaProtocolo4",
		`<consSitNFe versao="4.00" xmlns="http://www.portalfiscal.inf.br/nfe"><tpAmb>2</tpAmb><xServ>CONSULTAR</xServ><chNFe>123</chNFe></consSitNFe>`, nil)

	if got := statusOf(t, body); got != "215" {
		t.Fatalf("expected 215, got %s in %s", got, body)
	}
}

func TestSchemaValidationSkipsRootsWithoutASchema(t *testing.T) {
	testHarness := newValidatingHarness(t)
	body := testHarness.post(t, "/homologacao/SP/NFeStatusServico4",
		`<consStatServ versao="4.00" xmlns="http://www.portalfiscal.inf.br/nfe"><tpAmb>2</tpAmb><cUF>35</cUF><xServ>STATUS</xServ></consStatServ>`, nil)

	if got := statusOf(t, body); got != "107" {
		t.Fatalf("an operation without a schema must be served normally, got %s in %s", got, body)
	}
}

func TestSchemaValidationDoesNotStoreTheRejectedDocument(t *testing.T) {
	testHarness := newValidatingHarness(t)
	payload := strings.Replace(buildBatch(defaultDocument(3)), "<tpAmb>2</tpAmb>", "<tpAmb>7</tpAmb>", 1)

	testHarness.post(t, "/homologacao/SP/NFeAutorizacao4", payload, nil)
	if documents := testHarness.documents.Documents(storeFilter()); len(documents) != 0 {
		t.Fatalf("a document refused by the schema must not be stored, got %d", len(documents))
	}
}

func TestSchemaCatalogueIsReported(t *testing.T) {
	plain := newHarness(t)
	if body := plain.admin(t, http.MethodGet, "/admin/schemas", ""); !strings.Contains(body, `"loaded": false`) {
		t.Fatalf("without schemas the catalogue must say so: %s", body)
	}

	validating := newValidatingHarness(t)
	body := validating.admin(t, http.MethodGet, "/admin/schemas", "")
	if !strings.Contains(body, `"loaded": true`) || !strings.Contains(body, `"enviNFe"`) {
		t.Fatalf("expected the loaded roots to be listed: %s", body)
	}
}

func TestSchemaValidationKeepsOtherModelsWorking(t *testing.T) {
	testHarness := newValidatingHarness(t)
	_, payload := buildCTe(1, dfe.HomologationName(dfe.ModelCTe))

	body := testHarness.post(t, "/homologacao/SP/CTeRecepcaoSincV4", payload, nil)
	if got := statusOf(t, body); got != "100" {
		t.Fatalf("a model without a loaded schema must be served normally, got %s in %s", got, body)
	}
}
