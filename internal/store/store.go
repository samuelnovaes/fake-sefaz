package store

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vendermais/fake-sefaz/internal/dfe"
	"github.com/vendermais/fake-sefaz/internal/status"
)

type Event struct {
	Key          string      `json:"key"`
	Type         string      `json:"type"`
	Description  string      `json:"description"`
	Sequence     int         `json:"sequence"`
	Protocol     string      `json:"protocol"`
	Status       status.Code `json:"status"`
	RegisteredAt time.Time   `json:"registeredAt"`
	XML          string      `json:"xml"`
	NSU          int64       `json:"nsu"`
}

type Document struct {
	Key         string      `json:"key"`
	Model       dfe.Model   `json:"model"`
	Environment int         `json:"environment"`
	UFCode      string      `json:"ufCode"`
	IssuerTaxID string      `json:"issuerTaxId"`
	Series      int         `json:"series"`
	Number      int64       `json:"number"`
	DigestValue string      `json:"digestValue"`
	Protocol    string      `json:"protocol"`
	Status      status.Code `json:"status"`
	Reason      string      `json:"reason"`
	ReceivedAt  time.Time   `json:"receivedAt"`
	XML         string      `json:"xml"`
	Cancelled   bool        `json:"cancelled"`
	Events      []Event     `json:"events"`
	NSU         int64       `json:"nsu"`
}

type Batch struct {
	Receipt     string      `json:"receipt"`
	Environment int         `json:"environment"`
	UFCode      string      `json:"ufCode"`
	Keys        []string    `json:"keys"`
	Status      status.Code `json:"status"`
	ReceivedAt  time.Time   `json:"receivedAt"`
	ReleaseAt   time.Time   `json:"releaseAt"`
}

type Voiding struct {
	Identifier  string      `json:"identifier"`
	Environment int         `json:"environment"`
	UFCode      string      `json:"ufCode"`
	Year        int         `json:"year"`
	IssuerTaxID string      `json:"issuerTaxId"`
	Model       dfe.Model   `json:"model"`
	Series      int         `json:"series"`
	First       int64       `json:"first"`
	Last        int64       `json:"last"`
	Protocol    string      `json:"protocol"`
	Status      status.Code `json:"status"`
	ReceivedAt  time.Time   `json:"receivedAt"`
}

type Store struct {
	mutex      sync.RWMutex
	documents  map[string]*Document
	batches    map[string]*Batch
	voidings   map[string]*Voiding
	sequence   int64
	nsuCounter int64
}

func New() *Store {
	return &Store{
		documents: map[string]*Document{},
		batches:   map[string]*Batch{},
		voidings:  map[string]*Voiding{},
	}
}

func (s *Store) Document(key string) (Document, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	document, found := s.documents[key]
	if !found {
		return Document{}, false
	}
	return *document, true
}

func (s *Store) SaveDocument(document Document) Document {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.nsuCounter++
	document.NSU = s.nsuCounter
	stored := document
	s.documents[document.Key] = &stored
	return stored
}

func (s *Store) Documents(filter DocumentFilter) []Document {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	selected := make([]Document, 0, len(s.documents))
	for _, document := range s.documents {
		if filter.matches(*document) {
			selected = append(selected, *document)
		}
	}
	sort.Slice(selected, func(first, second int) bool { return selected[first].NSU < selected[second].NSU })
	return selected
}

type DocumentFilter struct {
	IssuerTaxID string
	Environment int
	AfterNSU    int64
	Model       dfe.Model
}

func (f DocumentFilter) matches(document Document) bool {
	if f.IssuerTaxID != "" && document.IssuerTaxID != f.IssuerTaxID {
		return false
	}
	if f.Environment != 0 && document.Environment != f.Environment {
		return false
	}
	if f.Model != "" && document.Model != f.Model {
		return false
	}
	return document.NSU > f.AfterNSU
}

func (s *Store) AppendEvent(key string, event Event) (Event, bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.nsuCounter++
	event.NSU = s.nsuCounter
	document, found := s.documents[key]
	if !found {
		return event, false
	}
	document.Events = append(document.Events, event)
	if event.Type == dfe.EventCancellation && event.Status == status.CancellationAuthorized {
		document.Cancelled = true
	}
	return event, true
}

func (s *Store) HasEvent(key, eventType string, sequence int) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	document, found := s.documents[key]
	if !found {
		return false
	}
	for _, event := range document.Events {
		if event.Type == eventType && event.Sequence == sequence {
			return true
		}
	}
	return false
}

func (s *Store) SaveBatch(batch Batch) Batch {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	stored := batch
	s.batches[batch.Receipt] = &stored
	return stored
}

func (s *Store) Batch(receipt string) (Batch, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	batch, found := s.batches[receipt]
	if !found {
		return Batch{}, false
	}
	return *batch, true
}

func (s *Store) SaveVoiding(voiding Voiding) Voiding {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	stored := voiding
	s.voidings[voiding.Identifier] = &stored
	return stored
}

func (s *Store) Voiding(identifier string) (Voiding, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	voiding, found := s.voidings[identifier]
	if !found {
		return Voiding{}, false
	}
	return *voiding, true
}

func (s *Store) Voidings() []Voiding {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	selected := make([]Voiding, 0, len(s.voidings))
	for _, voiding := range s.voidings {
		selected = append(selected, *voiding)
	}
	sort.Slice(selected, func(first, second int) bool {
		return selected[first].Identifier < selected[second].Identifier
	})
	return selected
}

func (s *Store) NextProtocol(ufCode string, now time.Time) string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.sequence++
	return fmt.Sprintf("%s%02d%011d", ufCode, now.Year()%100, s.sequence)
}

func (s *Store) LastNSU() int64 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.nsuCounter
}

func (s *Store) Reset() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.documents = map[string]*Document{}
	s.batches = map[string]*Batch{}
	s.voidings = map[string]*Voiding{}
	s.sequence = 0
	s.nsuCounter = 0
}

func VoidingIdentifier(environment int, ufCode string, year int, issuerTaxID string, model dfe.Model, series int, first, last int64) string {
	return strings.Join([]string{
		fmt.Sprint(environment), ufCode, fmt.Sprint(year), issuerTaxID, string(model),
		fmt.Sprint(series), fmt.Sprint(first), fmt.Sprint(last),
	}, "-")
}

func (s *Store) DocumentByNumber(environment int, issuerTaxID string, model dfe.Model, series int, number int64) (Document, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	for _, document := range s.documents {
		if document.Environment != environment || document.IssuerTaxID != issuerTaxID {
			continue
		}
		if document.Model != model || document.Series != series || document.Number != number {
			continue
		}
		return *document, true
	}
	return Document{}, false
}

func (s *Store) NumberVoided(environment int, issuerTaxID string, model dfe.Model, series int, number int64) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	for _, voiding := range s.voidings {
		if voiding.Environment != environment || voiding.IssuerTaxID != issuerTaxID {
			continue
		}
		if voiding.Model != model || voiding.Series != series {
			continue
		}
		if number >= voiding.First && number <= voiding.Last {
			return true
		}
	}
	return false
}
