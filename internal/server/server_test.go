package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vendermais/fake-sefaz/internal/dfe"
	"github.com/vendermais/fake-sefaz/internal/nfe"
	"github.com/vendermais/fake-sefaz/internal/scenario"
	"github.com/vendermais/fake-sefaz/internal/store"
)

const issuerTaxID = "99999999000191"

type harness struct {
	server    *httptest.Server
	documents *store.Store
	scenarios *scenario.Engine
	clock     time.Time
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	testHarness := &harness{
		documents: store.New(),
		scenarios: scenario.New(),
		clock:     time.Date(2026, time.September, 10, 9, 0, 0, 0, time.UTC),
	}
	service := nfe.NewService(testHarness.documents, testHarness.scenarios, nfe.DefaultOptions(), func() time.Time {
		return testHarness.clock
	})
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	testHarness.server = httptest.NewServer(New(testHarness.documents, testHarness.scenarios, service, logger))
	t.Cleanup(testHarness.server.Close)
	return testHarness
}

func (h *harness) post(t *testing.T, path, payload string, headers map[string]string) string {
	t.Helper()
	envelope := `<?xml version="1.0" encoding="utf-8"?>` +
		`<soap12:Envelope xmlns:soap12="http://www.w3.org/2003/05/soap-envelope"><soap12:Body>` +
		`<nfeDadosMsg xmlns="http://www.portalfiscal.inf.br/nfe/wsdl/NFeAutorizacao4">` + payload +
		`</nfeDadosMsg></soap12:Body></soap12:Envelope>`
	request, err := http.NewRequest(http.MethodPost, h.server.URL+path, bytes.NewBufferString(envelope))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	request.Header.Set("Content-Type", "application/soap+xml; charset=utf-8")
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response, err := h.server.Client().Do(request)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected http status %d: %s", response.StatusCode, body)
	}
	return string(body)
}

func (h *harness) admin(t *testing.T, method, path, payload string) string {
	t.Helper()
	var body io.Reader
	if payload != "" {
		body = strings.NewReader(payload)
	}
	request, err := http.NewRequest(method, h.server.URL+path, body)
	if err != nil {
		t.Fatalf("build admin request: %v", err)
	}
	response, err := h.server.Client().Do(request)
	if err != nil {
		t.Fatalf("admin request: %v", err)
	}
	defer response.Body.Close()
	content, _ := io.ReadAll(response.Body)
	if response.StatusCode >= http.StatusBadRequest {
		t.Fatalf("admin %s %s failed with %d: %s", method, path, response.StatusCode, content)
	}
	return string(content)
}

func accessKey(number int64, model dfe.Model) string {
	issued := time.Date(2026, time.September, 10, 9, 0, 0, 0, time.UTC)
	return dfe.BuildAccessKey("35", issued, issuerTaxID, model, 1, number, 1, "12345678")
}

type documentOptions struct {
	Number        int64
	Model         dfe.Model
	Environment   int
	RecipientName string
	Sync          int
	Signed        bool
	Key           string
}

func defaultDocument(number int64) documentOptions {
	return documentOptions{
		Number:        number,
		Model:         dfe.ModelNFCe,
		Environment:   2,
		RecipientName: nfe.HomologationRecipientName,
		Sync:          1,
		Signed:        true,
	}
}

func buildBatch(options documentOptions) string {
	key := options.Key
	if key == "" {
		key = accessKey(options.Number, options.Model)
	}
	signature := ""
	if options.Signed {
		signature = `<Signature xmlns="http://www.w3.org/2000/09/xmldsig#"><SignedInfo><Reference URI="#NFe` + key + `">` +
			`<DigestValue>t0Yl0Cq5UlY=</DigestValue></Reference></SignedInfo><SignatureValue>c2lnbmF0dXJl</SignatureValue></Signature>`
	}
	recipient := ""
	if options.RecipientName != "" {
		recipient = `<dest><CPF>11144477735</CPF><xNome>` + options.RecipientName + `</xNome></dest>`
	}
	return fmt.Sprintf(`<enviNFe versao="4.00" xmlns="http://www.portalfiscal.inf.br/nfe">`+
		`<idLote>1</idLote><indSinc>%d</indSinc>`+
		`<NFe><infNFe Id="NFe%s" versao="4.00">`+
		`<ide><cUF>35</cUF><cNF>12345678</cNF><natOp>Venda</natOp><mod>%s</mod><serie>1</serie><nNF>%d</nNF>`+
		`<dhEmi>2026-09-10T09:00:00-03:00</dhEmi><tpNF>1</tpNF><idDest>1</idDest><cMunFG>3550308</cMunFG>`+
		`<tpImp>4</tpImp><tpEmis>1</tpEmis><cDV>%s</cDV><tpAmb>%d</tpAmb><finNFe>1</finNFe><indFinal>1</indFinal>`+
		`<indPres>1</indPres><procEmi>0</procEmi><verProc>fake</verProc></ide>`+
		`<emit><CNPJ>%s</CNPJ><xNome>VENDER MAIS LTDA</xNome><IE>111111111111</IE></emit>`+
		`%s`+
		`<total><ICMSTot><vNF>10.00</vNF></ICMSTot></total>`+
		`</infNFe>%s</NFe></enviNFe>`,
		options.Sync, key, string(options.Model), options.Number, key[43:44], options.Environment, issuerTaxID, recipient, signature)
}

func statusOf(t *testing.T, body string) string {
	t.Helper()
	start := strings.Index(body, "<cStat>")
	if start < 0 {
		t.Fatalf("no cStat in response: %s", body)
	}
	end := strings.Index(body[start:], "</cStat>")
	return body[start+len("<cStat>") : start+end]
}

func TestServiceStatusAnswersRunning(t *testing.T) {
	testHarness := newHarness(t)
	body := testHarness.post(t, "/homologacao/SP/NFeStatusServico4",
		`<consStatServ versao="4.00" xmlns="http://www.portalfiscal.inf.br/nfe"><tpAmb>2</tpAmb><cUF>35</cUF><xServ>STATUS</xServ></consStatServ>`, nil)

	if got := statusOf(t, body); got != "107" {
		t.Fatalf("expected 107, got %s in %s", got, body)
	}
	if !strings.Contains(body, "retConsStatServ") {
		t.Fatalf("missing retConsStatServ: %s", body)
	}
}

func TestAuthorizationStoresDocumentAndAnswersProtocol(t *testing.T) {
	testHarness := newHarness(t)
	body := testHarness.post(t, "/homologacao/SP/NFeAutorizacao4", buildBatch(defaultDocument(1)), nil)

	if got := statusOf(t, body); got != "104" {
		t.Fatalf("expected batch processed 104, got %s in %s", got, body)
	}
	if !strings.Contains(body, "<cStat>100</cStat>") {
		t.Fatalf("expected document authorized: %s", body)
	}
	if !strings.Contains(body, "<nProt>") {
		t.Fatalf("expected a protocol number: %s", body)
	}
	stored := testHarness.documents.Documents(store.DocumentFilter{})
	if len(stored) != 1 {
		t.Fatalf("expected one stored document, got %d", len(stored))
	}
	if stored[0].Key != accessKey(1, dfe.ModelNFCe) {
		t.Fatalf("unexpected key stored: %s", stored[0].Key)
	}
}

func TestAuthorizationRejectsDuplicate(t *testing.T) {
	testHarness := newHarness(t)
	testHarness.post(t, "/homologacao/SP/NFeAutorizacao4", buildBatch(defaultDocument(2)), nil)
	body := testHarness.post(t, "/homologacao/SP/NFeAutorizacao4", buildBatch(defaultDocument(2)), nil)

	if !strings.Contains(body, "<cStat>204</cStat>") {
		t.Fatalf("expected duplicate rejection 204: %s", body)
	}
}

func TestAuthorizationRejectsHomologationRecipientName(t *testing.T) {
	testHarness := newHarness(t)
	options := defaultDocument(3)
	options.RecipientName = "JOAO DA SILVA"
	body := testHarness.post(t, "/homologacao/SP/NFeAutorizacao4", buildBatch(options), nil)

	if !strings.Contains(body, "<cStat>693</cStat>") {
		t.Fatalf("expected rejection 693: %s", body)
	}
}

func TestAuthorizationRejectsUnsignedDocument(t *testing.T) {
	testHarness := newHarness(t)
	options := defaultDocument(4)
	options.Signed = false
	body := testHarness.post(t, "/homologacao/SP/NFeAutorizacao4", buildBatch(options), nil)

	if !strings.Contains(body, "<cStat>297</cStat>") {
		t.Fatalf("expected rejection 297: %s", body)
	}
}

func TestAuthorizationRejectsBrokenAccessKey(t *testing.T) {
	testHarness := newHarness(t)
	options := defaultDocument(5)
	valid := accessKey(5, dfe.ModelNFCe)
	options.Key = valid[:43] + string(rune('0'+(int(valid[43]-'0')+1)%10))
	body := testHarness.post(t, "/homologacao/SP/NFeAutorizacao4", buildBatch(options), nil)

	if !strings.Contains(body, "<cStat>236</cStat>") {
		t.Fatalf("expected rejection 236: %s", body)
	}
}

func TestAuthorizationRejectsEnvironmentMismatch(t *testing.T) {
	testHarness := newHarness(t)
	body := testHarness.post(t, "/producao/SP/NFeAutorizacao4", buildBatch(defaultDocument(6)), nil)

	if !strings.Contains(body, "<cStat>252</cStat>") {
		t.Fatalf("expected rejection 252: %s", body)
	}
}

func TestAsynchronousBatchIsPolledUntilRelease(t *testing.T) {
	testHarness := newHarness(t)
	options := defaultDocument(7)
	options.Sync = 0
	body := testHarness.post(t, "/homologacao/SP/NFeAutorizacao4", buildBatch(options), nil)

	if got := statusOf(t, body); got != "103" {
		t.Fatalf("expected 103, got %s in %s", got, body)
	}
	receipt := between(t, body, "<nRec>", "</nRec>")

	polling := testHarness.post(t, "/homologacao/SP/NFeRetAutorizacao4",
		`<consReciNFe versao="4.00" xmlns="http://www.portalfiscal.inf.br/nfe"><tpAmb>2</tpAmb><nRec>`+receipt+`</nRec></consReciNFe>`, nil)
	if got := statusOf(t, polling); got != "105" {
		t.Fatalf("expected 105 while processing, got %s in %s", got, polling)
	}

	testHarness.clock = testHarness.clock.Add(time.Minute)
	processed := testHarness.post(t, "/homologacao/SP/NFeRetAutorizacao4",
		`<consReciNFe versao="4.00" xmlns="http://www.portalfiscal.inf.br/nfe"><tpAmb>2</tpAmb><nRec>`+receipt+`</nRec></consReciNFe>`, nil)
	if got := statusOf(t, processed); got != "104" {
		t.Fatalf("expected 104 after release, got %s in %s", got, processed)
	}
	if !strings.Contains(processed, "<cStat>100</cStat>") {
		t.Fatalf("expected authorized protocol in batch result: %s", processed)
	}
}

func TestProtocolQueryFindsAndMissesDocuments(t *testing.T) {
	testHarness := newHarness(t)
	key := accessKey(8, dfe.ModelNFCe)
	testHarness.post(t, "/homologacao/SP/NFeAutorizacao4", buildBatch(defaultDocument(8)), nil)

	found := testHarness.post(t, "/homologacao/SP/NFeConsultaProtocolo4",
		`<consSitNFe versao="4.00" xmlns="http://www.portalfiscal.inf.br/nfe"><tpAmb>2</tpAmb><xServ>CONSULTAR</xServ><chNFe>`+key+`</chNFe></consSitNFe>`, nil)
	if got := statusOf(t, found); got != "100" {
		t.Fatalf("expected 100, got %s in %s", got, found)
	}

	missingKey := accessKey(999, dfe.ModelNFCe)
	missing := testHarness.post(t, "/homologacao/SP/NFeConsultaProtocolo4",
		`<consSitNFe versao="4.00" xmlns="http://www.portalfiscal.inf.br/nfe"><tpAmb>2</tpAmb><xServ>CONSULTAR</xServ><chNFe>`+missingKey+`</chNFe></consSitNFe>`, nil)
	if got := statusOf(t, missing); got != "217" {
		t.Fatalf("expected 217, got %s in %s", got, missing)
	}
}

func TestCancellationEventFlow(t *testing.T) {
	testHarness := newHarness(t)
	key := accessKey(9, dfe.ModelNFCe)
	authorization := testHarness.post(t, "/homologacao/SP/NFeAutorizacao4", buildBatch(defaultDocument(9)), nil)
	protocol := between(t, authorization, "<nProt>", "</nProt>")

	cancellation := testHarness.post(t, "/homologacao/SP/NFeRecepcaoEvento4", cancelEvent(key, protocol, 1), nil)
	if !strings.Contains(cancellation, "<cStat>101</cStat>") {
		t.Fatalf("expected cancellation 101: %s", cancellation)
	}

	repeated := testHarness.post(t, "/homologacao/SP/NFeRecepcaoEvento4", cancelEvent(key, protocol, 1), nil)
	if !strings.Contains(repeated, "<cStat>573</cStat>") {
		t.Fatalf("expected duplicate event 573: %s", repeated)
	}

	query := testHarness.post(t, "/homologacao/SP/NFeConsultaProtocolo4",
		`<consSitNFe versao="4.00" xmlns="http://www.portalfiscal.inf.br/nfe"><tpAmb>2</tpAmb><xServ>CONSULTAR</xServ><chNFe>`+key+`</chNFe></consSitNFe>`, nil)
	if got := statusOf(t, query); got != "101" {
		t.Fatalf("expected cancelled document to answer 101, got %s in %s", got, query)
	}
	if !strings.Contains(query, "procEventoNFe") {
		t.Fatalf("expected the event attached to the query: %s", query)
	}
}

func TestCancellationRejectedAfterDeadline(t *testing.T) {
	testHarness := newHarness(t)
	key := accessKey(10, dfe.ModelNFCe)
	authorization := testHarness.post(t, "/homologacao/SP/NFeAutorizacao4", buildBatch(defaultDocument(10)), nil)
	protocol := between(t, authorization, "<nProt>", "</nProt>")

	testHarness.clock = testHarness.clock.Add(2 * time.Hour)
	late := testHarness.post(t, "/homologacao/SP/NFeRecepcaoEvento4", cancelEvent(key, protocol, 1), nil)
	if !strings.Contains(late, "<cStat>501</cStat>") {
		t.Fatalf("expected deadline rejection 501: %s", late)
	}
}

func TestCancellationOfUnknownDocumentIsRejected(t *testing.T) {
	testHarness := newHarness(t)
	body := testHarness.post(t, "/homologacao/SP/NFeRecepcaoEvento4", cancelEvent(accessKey(404, dfe.ModelNFCe), "135260000000001", 1), nil)

	if !strings.Contains(body, "<cStat>217</cStat>") {
		t.Fatalf("expected 217: %s", body)
	}
}

func TestVoidingNumberRange(t *testing.T) {
	testHarness := newHarness(t)
	body := testHarness.post(t, "/homologacao/SP/NFeInutilizacao4", voidRequest(20, 25), nil)

	if got := statusOf(t, body); got != "102" {
		t.Fatalf("expected 102, got %s in %s", got, body)
	}
	blocked := testHarness.post(t, "/homologacao/SP/NFeAutorizacao4", buildBatch(defaultDocument(22)), nil)
	if !strings.Contains(blocked, "<cStat>204</cStat>") {
		t.Fatalf("expected a voided number to be refused: %s", blocked)
	}
}

func TestVoidingRefusesRangeWithAuthorizedNumber(t *testing.T) {
	testHarness := newHarness(t)
	testHarness.post(t, "/homologacao/SP/NFeAutorizacao4", buildBatch(defaultDocument(31)), nil)
	body := testHarness.post(t, "/homologacao/SP/NFeInutilizacao4", voidRequest(30, 32), nil)

	if got := statusOf(t, body); got != "204" {
		t.Fatalf("expected 204, got %s in %s", got, body)
	}
}

func TestRegistrationQueryAnswersOneRecord(t *testing.T) {
	testHarness := newHarness(t)
	body := testHarness.post(t, "/homologacao/SP/CadConsultaCadastro4",
		`<ConsCad versao="2.00" xmlns="http://www.portalfiscal.inf.br/nfe"><infCons><xServ>CONS-CAD</xServ><UF>SP</UF><CNPJ>`+issuerTaxID+`</CNPJ></infCons></ConsCad>`, nil)

	if got := statusOf(t, body); got != "111" {
		t.Fatalf("expected 111, got %s in %s", got, body)
	}
	if !strings.Contains(body, "<infCad>") {
		t.Fatalf("expected one record: %s", body)
	}
}

func TestDistributionReturnsAuthorizedDocuments(t *testing.T) {
	testHarness := newHarness(t)
	testHarness.post(t, "/homologacao/SP/NFeAutorizacao4", buildBatch(defaultDocument(41)), nil)

	body := testHarness.post(t, "/homologacao/AN/NFeDistribuicaoDFe",
		`<distDFeInt versao="1.01" xmlns="http://www.portalfiscal.inf.br/nfe"><tpAmb>2</tpAmb><cUFAutor>35</cUFAutor><CNPJ>`+issuerTaxID+`</CNPJ><distNSU><ultNSU>000000000000000</ultNSU></distNSU></distDFeInt>`, nil)

	if got := statusOf(t, body); got != "138" {
		t.Fatalf("expected 138, got %s in %s", got, body)
	}
	if !strings.Contains(body, "docZip") {
		t.Fatalf("expected a compressed document: %s", body)
	}
	empty := testHarness.post(t, "/homologacao/AN/NFeDistribuicaoDFe",
		`<distDFeInt versao="1.01" xmlns="http://www.portalfiscal.inf.br/nfe"><tpAmb>2</tpAmb><cUFAutor>35</cUFAutor><CNPJ>11222333000181</CNPJ><distNSU><ultNSU>000000000000000</ultNSU></distNSU></distDFeInt>`, nil)
	if got := statusOf(t, empty); got != "137" {
		t.Fatalf("expected 137, got %s in %s", got, empty)
	}
}

func TestForcedStatusHeaderOverridesOutcome(t *testing.T) {
	testHarness := newHarness(t)
	body := testHarness.post(t, "/homologacao/SP/NFeAutorizacao4", buildBatch(defaultDocument(51)),
		map[string]string{ForcedStatusHeader: "539"})

	if !strings.Contains(body, "<cStat>539</cStat>") {
		t.Fatalf("expected forced 539: %s", body)
	}
}

func TestScenarioRuleAppliesOnceAndExpires(t *testing.T) {
	testHarness := newHarness(t)
	testHarness.admin(t, http.MethodPost, "/admin/scenarios",
		`{"operation":"enviNFe","issuerTaxId":"`+issuerTaxID+`","status":301,"remaining":1}`)

	denied := testHarness.post(t, "/homologacao/SP/NFeAutorizacao4", buildBatch(defaultDocument(61)), nil)
	if !strings.Contains(denied, "<cStat>301</cStat>") {
		t.Fatalf("expected denial 301: %s", denied)
	}
	authorized := testHarness.post(t, "/homologacao/SP/NFeAutorizacao4", buildBatch(defaultDocument(62)), nil)
	if !strings.Contains(authorized, "<cStat>100</cStat>") {
		t.Fatalf("expected the rule to be consumed: %s", authorized)
	}
}

func TestServiceCanBePausedThroughAdmin(t *testing.T) {
	testHarness := newHarness(t)
	testHarness.admin(t, http.MethodPut, "/admin/service", `{"status":108}`)

	body := testHarness.post(t, "/homologacao/SP/NFeAutorizacao4", buildBatch(defaultDocument(71)), nil)
	if got := statusOf(t, body); got != "108" {
		t.Fatalf("expected 108, got %s in %s", got, body)
	}
}

func TestOperationIsResolvedFromBodyForAnyPath(t *testing.T) {
	testHarness := newHarness(t)
	for _, path := range []string{"/", "/qualquer/coisa", "/ws/NFeStatusServico4.asmx"} {
		body := testHarness.post(t, path,
			`<consStatServ versao="4.00" xmlns="http://www.portalfiscal.inf.br/nfe"><tpAmb>2</tpAmb><cUF>35</cUF><xServ>STATUS</xServ></consStatServ>`, nil)
		if got := statusOf(t, body); got != "107" {
			t.Fatalf("path %s answered %s", path, got)
		}
	}
}

func TestEndpointCatalogueCoversEveryUnit(t *testing.T) {
	testHarness := newHarness(t)
	body := testHarness.admin(t, http.MethodGet, "/admin/endpoints", "")

	var catalogue []map[string]any
	if err := json.Unmarshal([]byte(body), &catalogue); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(catalogue) != 28 {
		t.Fatalf("expected 27 states plus the national environment, got %d", len(catalogue))
	}
	services, _ := catalogue[0]["services"].(map[string]any)
	if len(services) != len(nfe.WebServices()) {
		t.Fatalf("expected %d services per unit, got %d", len(nfe.WebServices()), len(services))
	}
}

func TestAdminResetClearsState(t *testing.T) {
	testHarness := newHarness(t)
	testHarness.post(t, "/homologacao/SP/NFeAutorizacao4", buildBatch(defaultDocument(81)), nil)
	testHarness.admin(t, http.MethodDelete, "/admin/state", "")

	if documents := testHarness.documents.Documents(store.DocumentFilter{}); len(documents) != 0 {
		t.Fatalf("expected an empty store, got %d documents", len(documents))
	}
}

func cancelEvent(key, protocol string, sequence int) string {
	return fmt.Sprintf(`<envEvento versao="1.00" xmlns="http://www.portalfiscal.inf.br/nfe"><idLote>1</idLote>`+
		`<evento versao="1.00"><infEvento Id="ID110111%s%02d"><cOrgao>35</cOrgao><tpAmb>2</tpAmb><CNPJ>%s</CNPJ>`+
		`<chNFe>%s</chNFe><dhEvento>2026-09-10T09:10:00-03:00</dhEvento><tpEvento>110111</tpEvento><nSeqEvento>%d</nSeqEvento>`+
		`<verEvento>1.00</verEvento><detEvento versao="1.00"><descEvento>Cancelamento</descEvento><nProt>%s</nProt>`+
		`<xJust>Cancelamento por erro de digitacao no pedido</xJust></detEvento></infEvento></evento></envEvento>`,
		key, sequence, issuerTaxID, key, sequence, protocol)
}

func voidRequest(first, last int64) string {
	return fmt.Sprintf(`<inutNFe versao="4.00" xmlns="http://www.portalfiscal.inf.br/nfe">`+
		`<infInut Id="ID3526%s65001%09d%09d"><tpAmb>2</tpAmb><xServ>INUTILIZAR</xServ><cUF>35</cUF><ano>26</ano>`+
		`<CNPJ>%s</CNPJ><mod>65</mod><serie>1</serie><nNFIni>%d</nNFIni><nNFFin>%d</nNFFin>`+
		`<xJust>Falha na numeracao sequencial do caixa</xJust></infInut></inutNFe>`,
		issuerTaxID, first, last, issuerTaxID, first, last)
}

func between(t *testing.T, body, open, close string) string {
	t.Helper()
	start := strings.Index(body, open)
	if start < 0 {
		t.Fatalf("missing %s in %s", open, body)
	}
	rest := body[start+len(open):]
	end := strings.Index(rest, close)
	if end < 0 {
		t.Fatalf("missing %s in %s", close, body)
	}
	return rest[:end]
}

func TestBarePayloadWithoutEnvelopeIsAccepted(t *testing.T) {
	testHarness := newHarness(t)
	payload := `<consStatServ versao="4.00" xmlns="http://www.portalfiscal.inf.br/nfe"><tpAmb>2</tpAmb><cUF>35</cUF><xServ>STATUS</xServ></consStatServ>`
	response, err := testHarness.server.Client().Post(testHarness.server.URL+"/homologacao/SP/NFeStatusServico4", "text/xml", strings.NewReader(payload))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)

	if strings.Contains(string(body), "Envelope") {
		t.Fatalf("a bare request must get a bare answer: %s", body)
	}
	if got := statusOf(t, string(body)); got != "107" {
		t.Fatalf("expected 107, got %s in %s", got, body)
	}
}

func TestAuthorizationIdentifiesProtocolByNumber(t *testing.T) {
	testHarness := newHarness(t)
	body := testHarness.post(t, "/homologacao/SP/NFeAutorizacao4", buildBatch(defaultDocument(91)), nil)

	protocol := between(t, body, "<nProt>", "</nProt>")
	if !strings.Contains(body, `<infProt Id="ID`+protocol+`">`) {
		t.Fatalf("infProt must be identified by the protocol number: %s", body)
	}
}
