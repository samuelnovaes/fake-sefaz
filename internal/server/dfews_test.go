package server

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/vendermais/fake-sefaz/internal/dfe"
)

func modelKey(model dfe.Model, number int64) string {
	issued := time.Date(2026, time.September, 10, 9, 0, 0, 0, time.UTC)
	return dfe.BuildAccessKey("35", issued, issuerTaxID, model, 1, number, 1, "12345678")
}

func signature(prefix, key string) string {
	return `<Signature xmlns="http://www.w3.org/2000/09/xmldsig#"><SignedInfo><Reference URI="#` + prefix + key + `">` +
		`<DigestValue>t0Yl0Cq5UlY=</DigestValue></Reference></SignedInfo><SignatureValue>c2ln</SignatureValue></Signature>`
}

func buildCTe(number int64, recipientName string) (string, string) {
	key := modelKey(dfe.ModelCTe, number)
	payload := fmt.Sprintf(`<CTe xmlns="http://www.portalfiscal.inf.br/cte"><infCte Id="CTe%s" versao="4.00">`+
		`<ide><cUF>35</cUF><cCT>12345678</cCT><CFOP>5353</CFOP><natOp>Transporte</natOp><mod>57</mod><serie>1</serie>`+
		`<nCT>%d</nCT><dhEmi>2026-09-10T09:00:00-03:00</dhEmi><tpImp>1</tpImp><tpEmis>1</tpEmis><cDV>%s</cDV>`+
		`<tpAmb>2</tpAmb><tpCTe>0</tpCTe><procEmi>0</procEmi><verProc>fake</verProc></ide>`+
		`<emit><CNPJ>%s</CNPJ><IE>111111111111</IE><xNome>TRANSPORTADORA FAKE</xNome></emit>`+
		`<dest><CNPJ>11222333000181</CNPJ><xNome>%s</xNome></dest>`+
		`</infCte>%s</CTe>`,
		key, number, key[43:44], issuerTaxID, recipientName, signature("CTe", key))
	return key, payload
}

func buildMDFe(number int64) (string, string) {
	key := modelKey(dfe.ModelMDFe, number)
	payload := fmt.Sprintf(`<MDFe xmlns="http://www.portalfiscal.inf.br/mdfe"><infMDFe Id="MDFe%s" versao="3.00">`+
		`<ide><cUF>35</cUF><tpAmb>2</tpAmb><tpEmit>1</tpEmit><mod>58</mod><serie>1</serie><nMDF>%d</nMDF>`+
		`<cMDF>12345678</cMDF><cDV>%s</cDV><modal>1</modal><dhEmi>2026-09-10T09:00:00-03:00</dhEmi><tpEmis>1</tpEmis></ide>`+
		`<emit><CNPJ>%s</CNPJ><IE>111111111111</IE><xNome>TRANSPORTADORA FAKE</xNome></emit>`+
		`</infMDFe>%s</MDFe>`,
		key, number, key[43:44], issuerTaxID, signature("MDFe", key))
	return key, payload
}

func buildBPe(number int64) (string, string) {
	key := modelKey(dfe.ModelBPe, number)
	payload := fmt.Sprintf(`<BPe xmlns="http://www.portalfiscal.inf.br/bpe"><infBPe Id="BPe%s" versao="1.00">`+
		`<ide><cUF>35</cUF><tpAmb>2</tpAmb><mod>63</mod><serie>1</serie><nBP>%d</nBP><cBP>12345678</cBP>`+
		`<cDV>%s</cDV><modal>1</modal><dhEmi>2026-09-10T09:00:00-03:00</dhEmi><tpEmis>1</tpEmis></ide>`+
		`<emit><CNPJ>%s</CNPJ><IE>111111111111</IE><xNome>VIACAO FAKE</xNome></emit>`+
		`<comp><CPF>11144477735</CPF><xNome>PASSAGEIRO</xNome></comp>`+
		`</infBPe>%s</BPe>`,
		key, number, key[43:44], issuerTaxID, signature("BPe", key))
	return key, payload
}

func buildNF3e(number int64) (string, string) {
	key := modelKey(dfe.ModelNF3e, number)
	payload := fmt.Sprintf(`<NF3e xmlns="http://www.portalfiscal.inf.br/nf3e"><infNF3e Id="NF3e%s" versao="1.00">`+
		`<ide><cUF>35</cUF><tpAmb>2</tpAmb><mod>66</mod><serie>1</serie><nNF>%d</nNF><cNF>12345678</cNF>`+
		`<cDV>%s</cDV><dhEmi>2026-09-10T09:00:00-03:00</dhEmi><tpEmis>1</tpEmis></ide>`+
		`<emit><CNPJ>%s</CNPJ><IE>111111111111</IE><xNome>DISTRIBUIDORA FAKE</xNome></emit>`+
		`<dest><CPF>11144477735</CPF><xNome>CONSUMIDOR</xNome></dest>`+
		`</infNF3e>%s</NF3e>`,
		key, number, key[43:44], issuerTaxID, signature("NF3e", key))
	return key, payload
}

func modelEvent(root, keyTag, key, eventType, protocol string, sequence int) string {
	return fmt.Sprintf(`<%s versao="4.00" xmlns="http://www.portalfiscal.inf.br/cte">`+
		`<infEvento Id="ID%s%s%02d"><cOrgao>35</cOrgao><tpAmb>2</tpAmb><CNPJ>%s</CNPJ><%s>%s</%s>`+
		`<dhEvento>2026-09-10T09:10:00-03:00</dhEvento><tpEvento>%s</tpEvento><nSeqEvento>%d</nSeqEvento>`+
		`<detEvento versao="4.00"><descEvento>Evento</descEvento><nProt>%s</nProt>`+
		`<xJust>Justificativa com mais de quinze caracteres</xJust></detEvento></infEvento></%s>`,
		root, eventType, key, sequence, issuerTaxID, keyTag, key, keyTag, eventType, sequence, protocol, root)
}

func TestCTeIsAuthorizedQueriedAndCancelled(t *testing.T) {
	testHarness := newHarness(t)
	key, payload := buildCTe(1, dfe.HomologationName(dfe.ModelCTe))

	authorization := testHarness.post(t, "/homologacao/SP/CTeRecepcaoSincV4", payload, nil)
	if got := statusOf(t, authorization); got != "100" {
		t.Fatalf("expected 100, got %s in %s", got, authorization)
	}
	if !strings.Contains(authorization, "<retCTe") || !strings.Contains(authorization, "<protCTe") {
		t.Fatalf("expected the CT-e response documents: %s", authorization)
	}
	if !strings.Contains(authorization, "Autorizado o uso do CT-e") {
		t.Fatalf("xMotivo must name the CT-e: %s", authorization)
	}
	if !strings.Contains(authorization, "<chCTe>"+key+"</chCTe>") {
		t.Fatalf("expected chCTe carrying the key: %s", authorization)
	}
	protocol := between(t, authorization, "<nProt>", "</nProt>")

	query := testHarness.post(t, "/homologacao/SP/CTeConsultaV4",
		`<consSitCTe versao="4.00" xmlns="http://www.portalfiscal.inf.br/cte"><tpAmb>2</tpAmb><xServ>CONSULTAR</xServ><chCTe>`+key+`</chCTe></consSitCTe>`, nil)
	if got := statusOf(t, query); got != "100" {
		t.Fatalf("expected 100 on query, got %s in %s", got, query)
	}

	cancellation := testHarness.post(t, "/homologacao/SP/CTeRecepcaoEventoV4",
		modelEvent("eventoCTe", "chCTe", key, dfe.EventCancellation, protocol, 1), nil)
	if got := statusOf(t, cancellation); got != "101" {
		t.Fatalf("expected 101, got %s in %s", got, cancellation)
	}
	if !strings.Contains(cancellation, "<retEventoCTe") {
		t.Fatalf("expected retEventoCTe: %s", cancellation)
	}

	cancelled := testHarness.post(t, "/homologacao/SP/CTeConsultaV4",
		`<consSitCTe versao="4.00" xmlns="http://www.portalfiscal.inf.br/cte"><tpAmb>2</tpAmb><xServ>CONSULTAR</xServ><chCTe>`+key+`</chCTe></consSitCTe>`, nil)
	if got := statusOf(t, cancelled); got != "101" {
		t.Fatalf("expected the CT-e to answer cancelled, got %s in %s", got, cancelled)
	}
	if !strings.Contains(cancelled, "procEventoCTe") {
		t.Fatalf("expected the event attached: %s", cancelled)
	}
}

func TestCTeRejectsHomologationRecipientName(t *testing.T) {
	testHarness := newHarness(t)
	_, payload := buildCTe(2, "CLIENTE QUALQUER")

	body := testHarness.post(t, "/homologacao/SP/CTeRecepcaoSincV4", payload, nil)
	if got := statusOf(t, body); got != "693" {
		t.Fatalf("expected 693, got %s in %s", got, body)
	}
}

func TestCTeOSRootIsAccepted(t *testing.T) {
	testHarness := newHarness(t)
	key := modelKey(dfe.ModelCTeOS, 3)
	payload := fmt.Sprintf(`<CTeOS xmlns="http://www.portalfiscal.inf.br/cte"><infCte Id="CTe%s" versao="4.00">`+
		`<ide><cUF>35</cUF><mod>67</mod><serie>1</serie><nCT>3</nCT><cDV>%s</cDV><tpAmb>2</tpAmb><tpEmis>1</tpEmis></ide>`+
		`<emit><CNPJ>%s</CNPJ><xNome>TRANSPORTADORA FAKE</xNome></emit></infCte>%s</CTeOS>`,
		key, key[43:44], issuerTaxID, signature("CTe", key))

	body := testHarness.post(t, "/homologacao/SP/CTeRecepcaoOSV4", payload, nil)
	if got := statusOf(t, body); got != "100" {
		t.Fatalf("expected 100, got %s in %s", got, body)
	}
}

func TestMDFeIsAuthorizedClosedAndListed(t *testing.T) {
	testHarness := newHarness(t)
	key, payload := buildMDFe(1)

	authorization := testHarness.post(t, "/homologacao/SP/MDFeRecepcaoSinc", payload, nil)
	if got := statusOf(t, authorization); got != "100" {
		t.Fatalf("expected 100, got %s in %s", got, authorization)
	}
	if !strings.Contains(authorization, "Autorizado o uso do MDF-e") {
		t.Fatalf("xMotivo must name the MDF-e: %s", authorization)
	}
	protocol := between(t, authorization, "<nProt>", "</nProt>")

	open := testHarness.post(t, "/homologacao/SP/MDFeConsNaoEnc",
		`<consMDFeNaoEnc versao="3.00" xmlns="http://www.portalfiscal.inf.br/mdfe"><tpAmb>2</tpAmb><xServ>CONSULTAR NAO ENCERRADOS</xServ><CNPJ>`+issuerTaxID+`</CNPJ></consMDFeNaoEnc>`, nil)
	if !strings.Contains(open, "<chMDFe>"+key+"</chMDFe>") {
		t.Fatalf("expected the open MDF-e to be listed: %s", open)
	}

	closing := testHarness.post(t, "/homologacao/SP/MDFeRecepcaoEvento",
		strings.ReplaceAll(modelEvent("eventoMDFe", "chMDFe", key, dfe.EventClosing, protocol, 1),
			"http://www.portalfiscal.inf.br/cte", "http://www.portalfiscal.inf.br/mdfe"), nil)
	if got := statusOf(t, closing); got != "135" {
		t.Fatalf("expected 135, got %s in %s", got, closing)
	}
	if !strings.Contains(closing, "Encerramento") {
		t.Fatalf("expected the closing description: %s", closing)
	}

	after := testHarness.post(t, "/homologacao/SP/MDFeConsNaoEnc",
		`<consMDFeNaoEnc versao="3.00" xmlns="http://www.portalfiscal.inf.br/mdfe"><tpAmb>2</tpAmb><xServ>CONSULTAR NAO ENCERRADOS</xServ><CNPJ>`+issuerTaxID+`</CNPJ></consMDFeNaoEnc>`, nil)
	if got := statusOf(t, after); got != "137" {
		t.Fatalf("expected 137 once closed, got %s in %s", got, after)
	}
}

func TestMDFeAcceptsGzippedPayload(t *testing.T) {
	testHarness := newHarness(t)
	_, payload := buildMDFe(2)

	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	writer.Write([]byte(payload))
	writer.Close()
	packed := base64.StdEncoding.EncodeToString(buffer.Bytes())

	envelope := `<?xml version="1.0" encoding="utf-8"?>` +
		`<soap12:Envelope xmlns:soap12="http://www.w3.org/2003/05/soap-envelope"><soap12:Body>` +
		`<mdfeDadosMsg xmlns="http://www.portalfiscal.inf.br/mdfe/wsdl/MDFeRecepcaoSinc">` + packed +
		`</mdfeDadosMsg></soap12:Body></soap12:Envelope>`
	response, err := testHarness.server.Client().Post(testHarness.server.URL+"/homologacao/SP/MDFeRecepcaoSinc",
		"application/soap+xml", strings.NewReader(envelope))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer response.Body.Close()
	body := readAll(t, response)

	if got := statusOf(t, body); got != "100" {
		t.Fatalf("expected 100 from a gzipped payload, got %s in %s", got, body)
	}
	if !strings.Contains(body, "Envelope") {
		t.Fatalf("an enveloped request must get an enveloped answer: %s", body)
	}
}

func TestBPeAndNF3eAreAuthorized(t *testing.T) {
	testHarness := newHarness(t)

	_, bpe := buildBPe(1)
	bpeBody := testHarness.post(t, "/homologacao/SP/BPeRecepcao", bpe, nil)
	if got := statusOf(t, bpeBody); got != "100" {
		t.Fatalf("BP-e expected 100, got %s in %s", got, bpeBody)
	}
	if !strings.Contains(bpeBody, "<retBPe") || !strings.Contains(bpeBody, "Autorizado o uso do BP-e") {
		t.Fatalf("unexpected BP-e answer: %s", bpeBody)
	}

	_, nf3e := buildNF3e(1)
	nf3eBody := testHarness.post(t, "/homologacao/SP/NF3eRecepcao", nf3e, nil)
	if got := statusOf(t, nf3eBody); got != "100" {
		t.Fatalf("NF3e expected 100, got %s in %s", got, nf3eBody)
	}
	if !strings.Contains(nf3eBody, "<retNF3e") || !strings.Contains(nf3eBody, "Autorizado o uso da NF3e") {
		t.Fatalf("unexpected NF3e answer: %s", nf3eBody)
	}
}

func TestEveryModelStatusServiceAnswers(t *testing.T) {
	testHarness := newHarness(t)
	requests := []struct{ root, namespace, response string }{
		{"consStatServCte", "http://www.portalfiscal.inf.br/cte", "retConsStatServCte"},
		{"consStatServMDFe", "http://www.portalfiscal.inf.br/mdfe", "retConsStatServMDFe"},
		{"consStatServBPe", "http://www.portalfiscal.inf.br/bpe", "retConsStatServBPe"},
		{"consStatServNF3e", "http://www.portalfiscal.inf.br/nf3e", "retConsStatServNF3e"},
	}
	for _, request := range requests {
		body := testHarness.post(t, "/homologacao/SP/status",
			`<`+request.root+` versao="4.00" xmlns="`+request.namespace+`"><tpAmb>2</tpAmb><xServ>STATUS</xServ></`+request.root+`>`, nil)
		if got := statusOf(t, body); got != "107" {
			t.Fatalf("%s answered %s", request.root, got)
		}
		if !strings.Contains(body, "<"+request.response+" ") {
			t.Fatalf("%s answered an unexpected document: %s", request.root, body)
		}
	}
}

func TestModelsAreAllReportedAsServed(t *testing.T) {
	testHarness := newHarness(t)
	body := testHarness.admin(t, http.MethodGet, "/admin/models", "")

	for _, name := range []string{"NF-e", "NFC-e", "CT-e", "CT-e OS", "MDF-e", "BP-e", "NF3e", "CF-e SAT"} {
		if !strings.Contains(body, `"name": "`+name+`"`) {
			t.Fatalf("model %s missing from the catalogue: %s", name, body)
		}
	}
}

func TestSATSellsAndCancels(t *testing.T) {
	testHarness := newHarness(t)
	sale := `<CFe><infCFe versao="0.08"><ide><cUF>35</cUF><cNF>123456</cNF><mod>59</mod><nserieSAT>900004019</nserieSAT>` +
		`<nCFe>1</nCFe><dEmi>20260910</dEmi><hEmi>090000</hEmi><tpAmb>2</tpAmb><CNPJ>16716114000172</CNPJ><numeroCaixa>1</numeroCaixa></ide>` +
		`<emit><CNPJ>` + issuerTaxID + `</CNPJ><IE>111111111111</IE></emit>` +
		`<dest><CPF>11144477735</CPF></dest><total><vCFe>10.00</vCFe></total></infCFe></CFe>`

	body := testHarness.admin(t, http.MethodPost, "/sat/EnviarDadosVenda",
		`{"numeroSessao":12345,"dadosVenda":`+quote(sale)+`}`)
	if !strings.Contains(body, `"06000"`) {
		t.Fatalf("expected the SAT success code 06000: %s", body)
	}
	key := between(t, body, `"chaveConsulta": "`, `"`)
	if len(key) != dfe.KeyLength {
		t.Fatalf("the SAT must mint a 44 digit key, got %q", key)
	}
	parsed, err := dfe.ParseAccessKey(key)
	if err != nil {
		t.Fatalf("minted key must be valid: %v", err)
	}
	if parsed.Model != dfe.ModelCFe {
		t.Fatalf("expected model 59, got %s", parsed.Model)
	}
	if _, found := testHarness.documents.Document(key); !found {
		t.Fatalf("the CF-e must be stored like any other document")
	}

	cancellation := testHarness.admin(t, http.MethodPost, "/sat/CancelarUltimaVenda", `{"numeroSessao":12346}`)
	if !strings.Contains(cancellation, `"07000"`) {
		t.Fatalf("expected the SAT cancellation code 07000: %s", cancellation)
	}
	document, _ := testHarness.documents.Document(key)
	if !document.Cancelled {
		t.Fatalf("the CF-e must be marked cancelled")
	}
}

func TestSATReportsOperationalStatus(t *testing.T) {
	testHarness := newHarness(t)
	body := testHarness.admin(t, http.MethodPost, "/sat/ConsultarStatusOperacional", `{"numeroSessao":1}`)

	if !strings.Contains(body, `"10000"`) {
		t.Fatalf("expected 10000: %s", body)
	}
	if !strings.Contains(body, "DESBLOQUEADO_SEM_ATIVACAO") {
		t.Fatalf("expected the equipment to start unactivated: %s", body)
	}
	testHarness.admin(t, http.MethodPost, "/sat/AtivarSAT", `{"numeroSessao":2,"codigoDeAtivacao":"12345678"}`)
	activated := testHarness.admin(t, http.MethodPost, "/sat/ConsultarStatusOperacional", `{"numeroSessao":3}`)
	if !strings.Contains(activated, "ATIVO") {
		t.Fatalf("expected the equipment to report itself active: %s", activated)
	}
}

func TestSATRejectsUnknownCommand(t *testing.T) {
	testHarness := newHarness(t)
	response, err := testHarness.server.Client().Post(testHarness.server.URL+"/sat/NaoExiste", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", response.StatusCode)
	}
}
