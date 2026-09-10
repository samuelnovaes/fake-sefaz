package dfe

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const KeyLength = 44

var (
	ErrKeyLength    = errors.New("access key must have 44 digits")
	ErrKeyNotDigits = errors.New("access key must contain digits only")
	ErrKeyCheckSum  = errors.New("access key check digit does not match")
)

type AccessKey struct {
	Raw          string
	UFCode       string
	Year         int
	Month        int
	IssuerTaxID  string
	Model        Model
	Series       int
	Number       int64
	IssuanceKind int
	RandomCode   string
	CheckDigit   int
}

func ParseAccessKey(value string) (AccessKey, error) {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) != KeyLength {
		return AccessKey{}, ErrKeyLength
	}
	if !isDigits(trimmed) {
		return AccessKey{}, ErrKeyNotDigits
	}
	year, _ := strconv.Atoi(trimmed[2:4])
	month, _ := strconv.Atoi(trimmed[4:6])
	series, _ := strconv.Atoi(trimmed[22:25])
	number, _ := strconv.ParseInt(trimmed[25:34], 10, 64)
	issuance, _ := strconv.Atoi(trimmed[34:35])
	check, _ := strconv.Atoi(trimmed[43:44])
	key := AccessKey{
		Raw:          trimmed,
		UFCode:       trimmed[0:2],
		Year:         2000 + year,
		Month:        month,
		IssuerTaxID:  trimmed[6:20],
		Model:        Model(trimmed[20:22]),
		Series:       series,
		Number:       number,
		IssuanceKind: issuance,
		RandomCode:   trimmed[35:43],
		CheckDigit:   check,
	}
	if CheckDigit(trimmed[:43]) != check {
		return key, ErrKeyCheckSum
	}
	return key, nil
}

func BuildAccessKey(ufCode string, issued time.Time, issuerTaxID string, model Model, series int, number int64, issuanceKind int, randomCode string) string {
	body := fmt.Sprintf("%s%02d%02d%s%s%03d%09d%01d%08s",
		ufCode,
		issued.Year()%100,
		int(issued.Month()),
		issuerTaxID,
		string(model),
		series,
		number,
		issuanceKind,
		randomCode,
	)
	return body + strconv.Itoa(CheckDigit(body))
}

func CheckDigit(body string) int {
	weight := 2
	sum := 0
	for index := len(body) - 1; index >= 0; index-- {
		digit := int(body[index] - '0')
		sum += digit * weight
		weight++
		if weight > 9 {
			weight = 2
		}
	}
	remainder := sum % 11
	if remainder < 2 {
		return 0
	}
	return 11 - remainder
}

func isDigits(value string) bool {
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return len(value) > 0
}
