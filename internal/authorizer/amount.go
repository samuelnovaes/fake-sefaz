package authorizer

import (
	"math/big"
	"strings"

	"github.com/vendermais/fake-sefaz/internal/status"
)

const PurposeNormal = 1

type decimal struct {
	digits *big.Int
	scale  int
}

var centTolerance = big.NewInt(1)

func amountRejection(submission Submission) (status.Code, bool) {
	if submission.Purpose == PurposeNormal && !itemValuesMatch(submission.Items) {
		return status.RejectedItemValue, true
	}
	if !discountTotalMatches(submission) {
		return status.RejectedDiscountTotal, true
	}
	return 0, false
}

func itemValuesMatch(items []Item) bool {
	for _, item := range items {
		if !itemValueMatches(item) {
			return false
		}
	}
	return true
}

func itemValueMatches(item Item) bool {
	quantity, quantityValid := parseDecimal(item.Quantity)
	unitValue, unitValueValid := parseDecimal(item.UnitValue)
	value, valueValid := parseCents(item.Value)
	if !quantityValid || !unitValueValid || !valueValid {
		return true
	}
	product := decimal{
		digits: new(big.Int).Mul(quantity.digits, unitValue.digits),
		scale:  quantity.scale + unitValue.scale,
	}
	return withinCent(product.cents(), value)
}

func discountTotalMatches(submission Submission) bool {
	total, totalValid := parseCents(submission.DiscountTotal)
	if !totalValid {
		return true
	}
	sum := new(big.Int)
	for _, item := range submission.Items {
		discount, valid := parseCents(item.Discount)
		if !valid {
			return true
		}
		sum.Add(sum, discount)
	}
	return withinCent(sum, total)
}

func sameDecimal(first, second string) bool {
	left, leftValid := parseDecimal(first)
	right, rightValid := parseDecimal(second)
	if !leftValid || !rightValid {
		return strings.TrimSpace(first) == strings.TrimSpace(second)
	}
	leftScaled := new(big.Int).Mul(left.digits, powerOfTen(right.scale))
	rightScaled := new(big.Int).Mul(right.digits, powerOfTen(left.scale))
	return leftScaled.Cmp(rightScaled) == 0
}

func withinCent(calculated, informed *big.Int) bool {
	difference := new(big.Int).Sub(calculated, informed)
	return difference.Abs(difference).Cmp(centTolerance) <= 0
}

func parseCents(text string) (*big.Int, bool) {
	value, valid := parseDecimal(text)
	if !valid {
		return nil, false
	}
	return value.cents(), true
}

func parseDecimal(text string) (decimal, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return decimal{digits: new(big.Int)}, true
	}
	whole, fraction, _ := strings.Cut(text, ".")
	if !onlyDigits(whole) || !onlyDigits(fraction) || whole+fraction == "" {
		return decimal{}, false
	}
	digits, _ := new(big.Int).SetString(whole+fraction, 10)
	return decimal{digits: digits, scale: len(fraction)}, true
}

func onlyDigits(text string) bool {
	for _, character := range text {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func (d decimal) cents() *big.Int {
	if d.scale <= 2 {
		return new(big.Int).Mul(d.digits, powerOfTen(2-d.scale))
	}
	divisor := powerOfTen(d.scale - 2)
	quotient, remainder := new(big.Int).QuoRem(d.digits, divisor, new(big.Int))
	if new(big.Int).Mul(remainder, big.NewInt(2)).Cmp(divisor) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	return quotient
}

func powerOfTen(exponent int) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exponent)), nil)
}
