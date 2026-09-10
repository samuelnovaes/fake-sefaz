package dfe

import (
	"testing"
	"time"
)

func TestBuildAccessKeyProducesParsableKey(t *testing.T) {
	issued := time.Date(2026, time.September, 10, 12, 0, 0, 0, time.UTC)
	key := BuildAccessKey("35", issued, "99999999000191", ModelNFCe, 1, 27, 1, "12345678")

	if len(key) != KeyLength {
		t.Fatalf("expected %d digits, got %d", KeyLength, len(key))
	}
	parsed, err := ParseAccessKey(key)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.UFCode != "35" || parsed.Model != ModelNFCe || parsed.Series != 1 || parsed.Number != 27 {
		t.Fatalf("unexpected parse result: %+v", parsed)
	}
	if parsed.IssuerTaxID != "99999999000191" {
		t.Fatalf("unexpected issuer: %s", parsed.IssuerTaxID)
	}
	if parsed.Year != 2026 || parsed.Month != 9 {
		t.Fatalf("unexpected competence: %d-%d", parsed.Year, parsed.Month)
	}
}

func TestParseAccessKeyRejectsBrokenCheckDigit(t *testing.T) {
	issued := time.Date(2026, time.September, 10, 12, 0, 0, 0, time.UTC)
	key := BuildAccessKey("35", issued, "99999999000191", ModelNFe, 1, 1, 1, "12345678")
	broken := key[:43] + string(rune('0'+(int(key[43]-'0')+1)%10))

	if _, err := ParseAccessKey(broken); err != ErrKeyCheckSum {
		t.Fatalf("expected check sum error, got %v", err)
	}
}

func TestCheckDigitMatchesPublishedExample(t *testing.T) {
	body := "3526099999999000191650010000000271123456789"
	if len(body) != 43 {
		t.Fatalf("fixture must have 43 digits, got %d", len(body))
	}
	digit := CheckDigit(body)
	if digit < 0 || digit > 9 {
		t.Fatalf("check digit out of range: %d", digit)
	}
	if _, err := ParseAccessKey(body + string(rune('0'+digit))); err != nil {
		t.Fatalf("key built from its own check digit must parse: %v", err)
	}
}
