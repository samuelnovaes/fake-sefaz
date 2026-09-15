package server

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/vendermais/fake-sefaz/internal/dfe"
)

func (h *harness) authorize(t *testing.T, options documentOptions) string {
	t.Helper()
	return between(t, h.post(t, "/homologacao/SP/NFeAutorizacao4", buildBatch(options), nil), "<protNFe", "</protNFe>")
}

func (h *harness) consult(t *testing.T, key string) string {
	t.Helper()
	return h.post(t, "/homologacao/SP/NFeConsultaProtocolo4",
		`<consSitNFe versao="4.00" xmlns="http://www.portalfiscal.inf.br/nfe"><tpAmb>2</tpAmb><xServ>CONSULTAR</xServ><chNFe>`+key+`</chNFe></consSitNFe>`, nil)
}

func (h *harness) registerEvent(t *testing.T, envelope string) string {
	t.Helper()
	return between(t, h.post(t, "/homologacao/SP/NFeRecepcaoEvento4", envelope, nil), "<retEvento", "</retEvento>")
}

func offlineDocument(number int64) documentOptions {
	options := defaultDocument(number)
	options.IssuanceKind = dfe.IssuanceOffline
	return options
}

func issuedAt() time.Time {
	return withDefaults(documentOptions{}).IssuedAt
}

func substitutionEvent(key, protocol, substituteKey string, sequence int) string {
	return fmt.Sprintf(`<envEvento versao="1.00" xmlns="http://www.portalfiscal.inf.br/nfe"><idLote>1</idLote>`+
		`<evento versao="1.00"><infEvento Id="ID110112%s%02d"><cOrgao>35</cOrgao><tpAmb>2</tpAmb><CNPJ>%s</CNPJ>`+
		`<chNFe>%s</chNFe><dhEvento>2026-09-10T10:00:00-03:00</dhEvento><tpEvento>110112</tpEvento><nSeqEvento>%d</nSeqEvento>`+
		`<verEvento>1.00</verEvento><detEvento versao="1.00"><descEvento>Cancelamento por substituicao</descEvento>`+
		`<cOrgaoAutor>35</cOrgaoAutor><tpAutor>1</tpAutor><verAplic>vendermais</verAplic><nProt>%s</nProt>`+
		`<xJust>Cancelamento por emissao em contingencia</xJust><chNFeRef>%s</chNFeRef></detEvento></infEvento></evento></envEvento>`,
		key, sequence, issuerTaxID, key, sequence, protocol, substituteKey)
}

func TestScenarioRefusesOnlyTheIssuanceKindItNames(t *testing.T) {
	testHarness := newHarness(t)
	testHarness.admin(t, http.MethodPost, "/admin/scenarios", `{"operation":"enviNFe","issuance":"9","status":778}`)

	if got := statusOf(t, testHarness.authorize(t, defaultDocument(100))); got != "100" {
		t.Fatalf("expected the normal issuance to be authorized, got %s", got)
	}
	refused := testHarness.authorize(t, offlineDocument(101))
	if got := statusOf(t, refused); got != "778" {
		t.Fatalf("expected the off-line issuance to be refused with 778, got %s in %s", got, refused)
	}
	if !strings.Contains(refused, "Informado NCM inexistente") {
		t.Fatalf("expected the catalogue wording: %s", refused)
	}
}

func TestLostAnswerStillAuthorizesTheDocument(t *testing.T) {
	testHarness := newHarness(t)
	testHarness.admin(t, http.MethodPost, "/admin/scenarios", `{"operation":"enviNFe","loseAnswer":true,"remaining":1}`)
	options := defaultDocument(110)

	response, err := testHarness.send(t, "/homologacao/SP/NFeAutorizacao4", buildBatch(options), nil)
	if err == nil {
		body, _ := io.ReadAll(response.Body)
		response.Body.Close()
		t.Fatalf("expected the connection to drop, got http %d: %s", response.StatusCode, body)
	}

	query := testHarness.consult(t, keyOf(options))
	if got := statusOf(t, query); got != "100" {
		t.Fatalf("expected the consultation to find the authorization, got %s in %s", got, query)
	}
	if !strings.Contains(query, "<nProt>") {
		t.Fatalf("expected the protocol in the consultation: %s", query)
	}
	if got := statusOf(t, testHarness.authorize(t, options)); got != "204" {
		t.Fatalf("expected the retransmission to be a duplicate, got %s", got)
	}
}

func TestOffLineNFCeAfterTheDeadlineIsAuthorizedOutOfTime(t *testing.T) {
	cases := []struct {
		name     string
		kind     int
		elapsed  time.Duration
		expected string
	}{
		{"off-line within the deadline", dfe.IssuanceOffline, 24 * time.Hour, "100"},
		{"off-line past the deadline", dfe.IssuanceOffline, 24*time.Hour + time.Second, "150"},
		{"normal issuance is not affected", dfe.IssuanceNormal, 25 * time.Hour, "100"},
	}
	for index, test := range cases {
		testHarness := newHarness(t)
		testHarness.clock = issuedAt().Add(test.elapsed)
		options := defaultDocument(int64(120 + index))
		options.IssuanceKind = test.kind
		if got := statusOf(t, testHarness.authorize(t, options)); got != test.expected {
			t.Errorf("%s: expected %s, got %s", test.name, test.expected, got)
			continue
		}
		if got := statusOf(t, testHarness.consult(t, keyOf(options))); got != test.expected {
			t.Errorf("%s: expected the consultation to answer %s, got %s", test.name, test.expected, got)
		}
	}
}

func TestConsultationByNumberReportsKeyDivergence(t *testing.T) {
	testHarness := newHarness(t)
	stored := defaultDocument(130)
	testHarness.authorize(t, stored)

	otherRandomCode := stored
	otherRandomCode.RandomCode = "87654321"
	otherMonth := stored
	otherMonth.IssuedAt = issuedAt().AddDate(0, -1, 0)
	otherIssuance := offlineDocument(130)
	otherNumber := defaultDocument(131)

	cases := []struct {
		name     string
		options  documentOptions
		expected string
	}{
		{"random code differs", otherRandomCode, "562"},
		{"month differs", otherMonth, "561"},
		{"issuance kind differs", otherIssuance, "613"},
		{"number never used", otherNumber, "217"},
	}
	for _, test := range cases {
		body := testHarness.consult(t, keyOf(test.options))
		if got := statusOf(t, body); got != test.expected {
			t.Errorf("%s: expected %s, got %s in %s", test.name, test.expected, got, body)
		}
	}
	divergent := testHarness.consult(t, keyOf(otherRandomCode))
	if !strings.Contains(divergent, "[chNFe:"+keyOf(stored)+"]") {
		t.Fatalf("expected the stored key in xMotivo: %s", divergent)
	}
}

func TestVoidingRefusesRepeatedAndOverlappingRanges(t *testing.T) {
	testHarness := newHarness(t)
	first := testHarness.post(t, "/homologacao/SP/NFeInutilizacao4", voidRequest(140, 145), nil)
	protocol := between(t, first, "<nProt>", "</nProt>")

	repeated := testHarness.post(t, "/homologacao/SP/NFeInutilizacao4", voidRequest(140, 145), nil)
	if got := statusOf(t, repeated); got != "563" {
		t.Fatalf("expected 563, got %s in %s", got, repeated)
	}
	if !strings.Contains(repeated, "<nProt>"+protocol+"</nProt>") {
		t.Fatalf("expected the earlier protocol in the answer: %s", repeated)
	}
	overlapping := testHarness.post(t, "/homologacao/SP/NFeInutilizacao4", voidRequest(144, 150), nil)
	if got := statusOf(t, overlapping); got != "256" {
		t.Fatalf("expected 256, got %s in %s", got, overlapping)
	}
}

func TestRetransmissionOfCancelledOrDeniedDocument(t *testing.T) {
	testHarness := newHarness(t)
	cancelled := defaultDocument(150)
	protocol := between(t, testHarness.authorize(t, cancelled), "<nProt>", "</nProt>")
	testHarness.registerEvent(t, cancelEvent(keyOf(cancelled), protocol, 1))
	if got := statusOf(t, testHarness.authorize(t, cancelled)); got != "218" {
		t.Fatalf("expected 218 for a cancelled document, got %s", got)
	}

	testHarness.admin(t, http.MethodPost, "/admin/scenarios", `{"operation":"enviNFe","status":301,"remaining":1}`)
	denied := defaultDocument(151)
	testHarness.authorize(t, denied)
	if got := statusOf(t, testHarness.authorize(t, denied)); got != "205" {
		t.Fatalf("expected 205 for a denied document, got %s", got)
	}
}

func TestMisuseRefusesTheWholeRequest(t *testing.T) {
	testHarness := newHarness(t)
	testHarness.admin(t, http.MethodPost, "/admin/scenarios", `{"operation":"enviNFe","status":656,"remaining":1}`)

	body := testHarness.post(t, "/homologacao/SP/NFeAutorizacao4", buildBatch(defaultDocument(160)), nil)
	if got := statusOf(t, body); got != "656" {
		t.Fatalf("expected 656 on retEnviNFe, got %s in %s", got, body)
	}
	if strings.Contains(body, "protNFe") || !strings.Contains(body, "Consumo indevido pelo aplicativo da empresa") {
		t.Fatalf("expected a request level refusal with the catalogue wording: %s", body)
	}
}

type substitution struct {
	harness            *harness
	key                string
	protocol           string
	substituteKey      string
	substituteProtocol string
}

func prepareSubstitution(t *testing.T, adjust func(*documentOptions)) substitution {
	t.Helper()
	testHarness := newHarness(t)
	cancelled := defaultDocument(500)
	cancelled.Items = item("1", "10.00", "10.00", "")
	substitute := offlineDocument(501)
	substitute.Items = cancelled.Items
	substitute.IssuedAt = issuedAt().Add(time.Hour)
	if adjust != nil {
		adjust(&substitute)
	}
	return substitution{
		harness:            testHarness,
		key:                keyOf(cancelled),
		protocol:           between(t, testHarness.authorize(t, cancelled), "<nProt>", "</nProt>"),
		substituteKey:      keyOf(substitute),
		substituteProtocol: between(t, testHarness.authorize(t, substitute), "<nProt>", "</nProt>"),
	}
}

func (s substitution) event() string {
	return substitutionEvent(s.key, s.protocol, s.substituteKey, 1)
}

func TestCancellationBySubstitutionIsLinkedAndConsulted(t *testing.T) {
	prepared := prepareSubstitution(t, nil)

	answer := prepared.harness.registerEvent(t, prepared.event())
	if got := statusOf(t, answer); got != "135" {
		t.Fatalf("expected 135, got %s in %s", got, answer)
	}
	query := prepared.harness.consult(t, prepared.key)
	if got := statusOf(t, query); got != "101" {
		t.Fatalf("expected the replaced document to answer 101, got %s in %s", got, query)
	}
	if !strings.Contains(query, "<protNFe") || !strings.Contains(query, "<tpEvento>110112</tpEvento>") {
		t.Fatalf("expected protNFe and procEventoNFe in the consultation: %s", query)
	}
	if got := statusOf(t, prepared.harness.registerEvent(t, prepared.event())); got != "573" {
		t.Fatalf("expected the repeated event to answer 573, got %s", got)
	}
}

func TestCancellationBySubstitutionRules(t *testing.T) {
	cases := []struct {
		name     string
		adjust   func(*documentOptions)
		arrange  func(*testing.T, substitution) substitution
		expected string
		reason   string
	}{
		{name: "cancelled document issued off-line", expected: "920", arrange: func(_ *testing.T, s substitution) substitution {
			s.key, s.substituteKey, s.protocol = s.substituteKey, s.key, s.substituteProtocol
			return s
		}},
		{name: "substitute key check digit", expected: "910", reason: "(Digito)", arrange: func(_ *testing.T, s substitution) substitution {
			s.substituteKey = s.substituteKey[:43] + string(rune('0'+(int(s.substituteKey[43]-'0')+1)%10))
			return s
		}},
		{name: "substitute is the cancelled key", expected: "911", reason: "(mesma Chave de Acesso)", arrange: func(_ *testing.T, s substitution) substitution {
			s.substituteKey = s.key
			return s
		}},
		{name: "substitute issued in a later month", expected: "911", reason: "(Ano-Mes)", adjust: func(o *documentOptions) {
			o.IssuedAt = issuedAt().AddDate(0, 1, 0)
		}},
		{name: "cancelled document unknown", expected: "494", arrange: func(_ *testing.T, s substitution) substitution {
			s.key = keyOf(defaultDocument(777))
			return s
		}},
		{name: "past 168 hours", expected: "501", arrange: func(_ *testing.T, s substitution) substitution {
			s.harness.clock = s.harness.clock.Add(168*time.Hour + time.Second)
			return s
		}},
		{name: "within 168 hours", expected: "135", arrange: func(_ *testing.T, s substitution) substitution {
			s.harness.clock = s.harness.clock.Add(168 * time.Hour)
			return s
		}},
		{name: "cancelled document already cancelled", expected: "580", arrange: func(t *testing.T, s substitution) substitution {
			s.harness.registerEvent(t, cancelEvent(s.key, s.protocol, 1))
			return s
		}},
		{name: "protocol differs", expected: "222", arrange: func(_ *testing.T, s substitution) substitution {
			s.protocol = "135260000009999"
			return s
		}},
		{name: "substitute missing", expected: "912", arrange: func(_ *testing.T, s substitution) substitution {
			s.substituteKey = keyOf(offlineDocument(999))
			return s
		}},
		{name: "substitute cancelled", expected: "913", arrange: func(t *testing.T, s substitution) substitution {
			s.harness.registerEvent(t, cancelEvent(s.substituteKey, s.substituteProtocol, 1))
			return s
		}},
		{name: "substitute issued two hours after", expected: "135", adjust: func(o *documentOptions) {
			o.IssuedAt = issuedAt().Add(2 * time.Hour)
		}},
		{name: "substitute issued more than two hours after", expected: "914", adjust: func(o *documentOptions) {
			o.IssuedAt = issuedAt().Add(2*time.Hour + time.Second)
		}},
		{name: "total differs", expected: "915", adjust: func(o *documentOptions) { o.Total = "10.01" }},
		{name: "same total written differently", expected: "135", adjust: func(o *documentOptions) { o.Total = "10.0" }},
		{name: "ICMS total differs", expected: "916", adjust: func(o *documentOptions) { o.ICMSTotal = "1.80" }},
		{name: "recipient differs", expected: "917", adjust: func(o *documentOptions) { o.RecipientDocument = "52998224725" }},
		{name: "item count differs", expected: "918", adjust: func(o *documentOptions) { o.Items += item("1", "10.00", "10.00", "") }},
		{name: "item differs", expected: "919", adjust: func(o *documentOptions) { o.Items = item("2", "5.00", "10.00", "") }},
		{name: "substitute issued normally", expected: "921", adjust: func(o *documentOptions) { o.IssuanceKind = dfe.IssuanceNormal }},
	}
	for _, test := range cases {
		prepared := prepareSubstitution(t, test.adjust)
		if test.arrange != nil {
			prepared = test.arrange(t, prepared)
		}
		answer := prepared.harness.registerEvent(t, prepared.event())
		if got := statusOf(t, answer); got != test.expected {
			t.Errorf("%s: expected %s, got %s in %s", test.name, test.expected, got, answer)
			continue
		}
		if !strings.Contains(answer, test.reason) {
			t.Errorf("%s: expected %q in xMotivo: %s", test.name, test.reason, answer)
		}
	}
}

func TestCancellationBySubstitutionIsNotAnNFeEvent(t *testing.T) {
	testHarness := newHarness(t)
	options := defaultDocument(170)
	options.Model = dfe.ModelNFe
	options.RecipientName = dfe.HomologationName(dfe.ModelNFe)
	protocol := between(t, testHarness.authorize(t, options), "<nProt>", "</nProt>")

	answer := testHarness.registerEvent(t, substitutionEvent(keyOf(options), protocol, keyOf(offlineDocument(171)), 1))
	if got := statusOf(t, answer); got != "215" {
		t.Fatalf("expected model 55 to refuse the event with 215, got %s in %s", got, answer)
	}
}
