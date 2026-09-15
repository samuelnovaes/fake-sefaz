package scenario

import (
	"strconv"
	"strings"
	"sync"

	"github.com/vendermais/fake-sefaz/internal/status"
)

type Rule struct {
	Identifier  string      `json:"id"`
	Operation   string      `json:"operation,omitempty"`
	IssuerTaxID string      `json:"issuerTaxId,omitempty"`
	KeySuffix   string      `json:"keySuffix,omitempty"`
	Model       string      `json:"model,omitempty"`
	Issuance    string      `json:"issuance,omitempty"`
	Status      status.Code `json:"status,omitempty"`
	LoseAnswer  bool        `json:"loseAnswer,omitempty"`
	Remaining   int         `json:"remaining,omitempty"`
}

type Match struct {
	Operation   string
	IssuerTaxID string
	Key         string
	Model       string
	Issuance    string
}

type Engine struct {
	mutex         sync.Mutex
	rules         []Rule
	serviceStatus status.Code
	asynchronous  bool
	averageTime   int
}

func New() *Engine {
	return &Engine{serviceStatus: status.ServiceRunning, averageTime: 1}
}

func (e *Engine) Add(rule Rule) Rule {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	if rule.Identifier == "" {
		rule.Identifier = generateIdentifier(len(e.rules) + 1)
	}
	e.rules = append(e.rules, rule)
	return rule
}

func (e *Engine) Rules() []Rule {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	copied := make([]Rule, len(e.rules))
	copy(copied, e.rules)
	return copied
}

func (e *Engine) Remove(identifier string) bool {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	for index, rule := range e.rules {
		if rule.Identifier == identifier {
			e.rules = append(e.rules[:index], e.rules[index+1:]...)
			return true
		}
	}
	return false
}

func (e *Engine) Clear() {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.rules = nil
}

func (e *Engine) Resolve(match Match) (Rule, bool) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	for index := range e.rules {
		rule := e.rules[index]
		if !applies(rule, match) {
			continue
		}
		if rule.Remaining > 0 {
			e.rules[index].Remaining--
			if e.rules[index].Remaining == 0 {
				e.rules = append(e.rules[:index], e.rules[index+1:]...)
			}
		}
		return rule, true
	}
	return Rule{}, false
}

func applies(rule Rule, match Match) bool {
	if rule.Operation != "" && rule.Operation != match.Operation {
		return false
	}
	if rule.IssuerTaxID != "" && rule.IssuerTaxID != match.IssuerTaxID {
		return false
	}
	if rule.Model != "" && rule.Model != match.Model {
		return false
	}
	if rule.Issuance != "" && rule.Issuance != match.Issuance {
		return false
	}
	if rule.KeySuffix != "" && !strings.HasSuffix(match.Key, rule.KeySuffix) {
		return false
	}
	return true
}

func (e *Engine) ServiceStatus() status.Code {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	return e.serviceStatus
}

func (e *Engine) SetServiceStatus(code status.Code) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.serviceStatus = code
}

func (e *Engine) Asynchronous() bool {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	return e.asynchronous
}

func (e *Engine) SetAsynchronous(asynchronous bool) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.asynchronous = asynchronous
}

func (e *Engine) AverageTime() int {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	return e.averageTime
}

func (e *Engine) SetAverageTime(seconds int) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.averageTime = seconds
}

func generateIdentifier(sequence int) string {
	return "rule-" + strconv.Itoa(sequence)
}
