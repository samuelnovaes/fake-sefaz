package authorizer

import (
	"strings"

	"github.com/vendermais/fake-sefaz/internal/dfe"
)

const lastCNPJSeries = 909

func validIssuerDocument(key dfe.AccessKey) bool {
	if key.Series <= lastCNPJSeries {
		return validCNPJ(key.IssuerTaxID)
	}
	return strings.HasPrefix(key.IssuerTaxID, "000") && validCPF(key.IssuerTaxID[3:])
}

func validCNPJ(digits string) bool {
	if len(digits) != 14 || zeroed(digits) {
		return false
	}
	return dfe.CheckDigit(digits[:12]) == digitAt(digits, 12) && dfe.CheckDigit(digits[:13]) == digitAt(digits, 13)
}

func validCPF(digits string) bool {
	if len(digits) != 11 || zeroed(digits) {
		return false
	}
	return cpfDigit(digits[:9]) == digitAt(digits, 9) && cpfDigit(digits[:10]) == digitAt(digits, 10)
}

func cpfDigit(body string) int {
	sum := 0
	weight := len(body) + 1
	for _, character := range body {
		sum += int(character-'0') * weight
		weight--
	}
	remainder := sum % 11
	if remainder < 2 {
		return 0
	}
	return 11 - remainder
}

func digitAt(digits string, index int) int {
	return int(digits[index] - '0')
}

func zeroed(digits string) bool {
	return strings.Trim(digits, "0") == ""
}
