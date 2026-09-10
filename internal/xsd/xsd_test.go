package xsd

import (
	"os"
	"strings"
	"testing"
)

const namespace = `xmlns="http://fake.sefaz/pedido"`

func load(t *testing.T) *Set {
	t.Helper()
	set, err := Load("testdata")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	return set
}

func validPedido() string {
	return `<pedido ` + namespace + ` versao="1.00">` +
		`<emitente><CNPJ>99999999000191</CNPJ><nome>VENDER</nome></emitente>` +
		`<ambiente>2</ambiente>` +
		`<item numero="1"><codigo>ABC</codigo><quantidade>2</quantidade></item>` +
		`<item numero="2"><codigo>12345</codigo><quantidade>3</quantidade></item>` +
		`</pedido>`
}

func validate(t *testing.T, set *Set, document string) []Failure {
	t.Helper()
	failures, err := set.Validate("pedido", "1.00", []byte(document))
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	return failures
}

func TestLoadIndexesRootsAndDocuments(t *testing.T) {
	set := load(t)

	if set.Documents() != 2 {
		t.Fatalf("expected two documents, got %d", set.Documents())
	}
	if !set.Knows("pedido") || !set.Knows("observacao") {
		t.Fatalf("expected both global elements to be indexed, got %v", set.Roots())
	}
	if set.Knows("inexistente") {
		t.Fatal("an unknown root must not be reported as known")
	}
}

func TestValidDocumentProducesNoFailure(t *testing.T) {
	set := load(t)

	if failures := validate(t, set, validPedido()); len(failures) != 0 {
		t.Fatalf("expected a clean document, got %v", failures)
	}
}

func TestNullableChoiceMatchesWhenAbsent(t *testing.T) {
	set := load(t)
	withBranch := strings.Replace(validPedido(), "<item numero=\"1\">",
		"<entrega>CASA</entrega><item numero=\"1\">", 1)

	if failures := validate(t, set, withBranch); len(failures) != 0 {
		t.Fatalf("the optional branch must be accepted, got %v", failures)
	}
	other := strings.Replace(validPedido(), "<item numero=\"1\">",
		"<retirada>LOJA</retirada><item numero=\"1\">", 1)
	if failures := validate(t, set, other); len(failures) != 0 {
		t.Fatalf("the second branch must be accepted, got %v", failures)
	}
}

func TestPatternsInOneRestrictionAreAlternatives(t *testing.T) {
	set := load(t)

	broken := strings.Replace(validPedido(), "<codigo>ABC</codigo>", "<codigo>AB</codigo>", 1)
	failures := validate(t, set, broken)
	if len(failures) == 0 {
		t.Fatal("expected a pattern failure")
	}
	if !strings.Contains(failures[0].Path, "codigo") {
		t.Fatalf("unexpected failure: %v", failures[0])
	}
}

func TestFacetsAreEnforced(t *testing.T) {
	set := load(t)
	cases := []struct {
		name     string
		from, to string
		expected string
	}{
		{"CNPJ com treze digitos", "<CNPJ>99999999000191</CNPJ>", "<CNPJ>9999999900019</CNPJ>", "formato"},
		{"nome curto demais", "<nome>VENDER</nome>", "<nome>V</nome>", "ao menos 2"},
		{"nome longo demais", "<nome>VENDER</nome>", "<nome>VENDER MAIS LTDA</nome>", "no maximo 10"},
		{"ambiente fora da lista", "<ambiente>2</ambiente>", "<ambiente>7</ambiente>", "fora da lista"},
		{"quantidade acima do maximo", "<quantidade>2</quantidade>", "<quantidade>250</quantidade>", "maior que o maximo"},
		{"quantidade abaixo do minimo", "<quantidade>2</quantidade>", "<quantidade>0</quantidade>", "menor que o minimo"},
	}
	for _, test := range cases {
		document := strings.Replace(validPedido(), test.from, test.to, 1)
		failures := validate(t, set, document)
		if len(failures) == 0 {
			t.Errorf("%s: expected a failure", test.name)
			continue
		}
		if !strings.Contains(failures[0].Message, test.expected) {
			t.Errorf("%s: expected %q in %q", test.name, test.expected, failures[0].Message)
		}
	}
}

func TestStructuralFailuresAreReported(t *testing.T) {
	set := load(t)
	cases := []struct {
		name     string
		document string
		expected string
	}{
		{
			"elemento obrigatorio ausente",
			strings.Replace(validPedido(), "<ambiente>2</ambiente>", "", 1),
			"esperava ambiente",
		},
		{
			"ordem trocada",
			strings.Replace(validPedido(),
				"<emitente><CNPJ>99999999000191</CNPJ><nome>VENDER</nome></emitente>",
				"<emitente><nome>VENDER</nome><CNPJ>99999999000191</CNPJ></emitente>", 1),
			"esperava CNPJ",
		},
		{
			"elemento desconhecido",
			strings.Replace(validPedido(), "</pedido>", "<inventado>1</inventado></pedido>", 1),
			"inesperado",
		},
		{
			"lista vazia",
			`<pedido ` + namespace + ` versao="1.00"><emitente><CNPJ>99999999000191</CNPJ><nome>VENDER</nome></emitente><ambiente>2</ambiente></pedido>`,
			"esperava",
		},
	}
	for _, test := range cases {
		failures := validate(t, set, test.document)
		if len(failures) == 0 {
			t.Errorf("%s: expected a failure", test.name)
			continue
		}
		if !strings.Contains(failures[0].String(), test.expected) {
			t.Errorf("%s: expected %q in %q", test.name, test.expected, failures[0])
		}
	}
}

func TestRequiredAttributeIsEnforced(t *testing.T) {
	set := load(t)

	missing := strings.Replace(validPedido(), ` versao="1.00"`, "", 1)
	failures := validate(t, set, missing)
	if len(failures) == 0 || !strings.Contains(failures[0].Message, "versao") {
		t.Fatalf("expected the missing attribute to be reported, got %v", failures)
	}

	broken := strings.Replace(validPedido(), `numero="1"`, `numero="um"`, 1)
	failures = validate(t, set, broken)
	if len(failures) == 0 || !strings.Contains(failures[0].Message, "inteiro") {
		t.Fatalf("expected the attribute type to be checked, got %v", failures)
	}
}

func TestReferencedElementKeepsItsOwnType(t *testing.T) {
	set := load(t)
	document := strings.Replace(validPedido(), "</pedido>", "<observacao>X</observacao></pedido>", 1)

	failures := validate(t, set, document)
	if len(failures) == 0 || !strings.Contains(failures[0].Path, "observacao") {
		t.Fatalf("the referenced element must be validated with its global type, got %v", failures)
	}
}

func TestUnknownRootIsReported(t *testing.T) {
	set := load(t)

	if _, err := set.Validate("naoExiste", "1.00", []byte("<naoExiste/>")); err != ErrRootUnknown {
		t.Fatalf("expected ErrRootUnknown, got %v", err)
	}
}

func TestMalformedDocumentIsReported(t *testing.T) {
	set := load(t)

	failures, err := set.Validate("pedido", "1.00", []byte(`<pedido `+namespace+`><emitente>`))
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if len(failures) != 1 || !strings.Contains(failures[0].Message, "malformado") {
		t.Fatalf("expected a malformed document failure, got %v", failures)
	}
}

func TestOfficialSchemasWhenAvailable(t *testing.T) {
	directory := os.Getenv("FAKE_SEFAZ_SCHEMA_DIR")
	if directory == "" {
		t.Skip("set FAKE_SEFAZ_SCHEMA_DIR to run against the published schemas")
	}
	set, err := Load(directory)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	for _, root := range []string{"NFe", "enviNFe", "consSitNFe", "CTe", "MDFe"} {
		if !set.Knows(root) {
			t.Errorf("expected the published schemas to declare %s", root)
		}
	}
}
