package server

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/vendermais/fake-sefaz/internal/dfe"
	"github.com/vendermais/fake-sefaz/internal/sat"
	"github.com/vendermais/fake-sefaz/internal/scenario"
	"github.com/vendermais/fake-sefaz/internal/status"
	"github.com/vendermais/fake-sefaz/internal/store"
	"github.com/vendermais/fake-sefaz/internal/uf"
)

func (s *Server) routeAdmin(mux *http.ServeMux) {
	mux.HandleFunc("GET /admin/health", s.health)
	mux.HandleFunc("GET /admin/units", s.units)
	mux.HandleFunc("GET /admin/models", s.models)
	mux.HandleFunc("GET /admin/status-codes", s.statusCodes)
	mux.HandleFunc("GET /admin/sat/commands", s.satCommands)
	mux.HandleFunc("GET /admin/schemas", s.schemaCatalogue)
	mux.HandleFunc("GET /admin/endpoints", s.endpoints)
	mux.HandleFunc("GET /admin/documents", s.listDocuments)
	mux.HandleFunc("GET /admin/documents/{key}", s.showDocument)
	mux.HandleFunc("GET /admin/voidings", s.listVoidings)
	mux.HandleFunc("GET /admin/scenarios", s.listScenarios)
	mux.HandleFunc("POST /admin/scenarios", s.createScenario)
	mux.HandleFunc("DELETE /admin/scenarios", s.clearScenarios)
	mux.HandleFunc("DELETE /admin/scenarios/{id}", s.deleteScenario)
	mux.HandleFunc("GET /admin/service", s.showService)
	mux.HandleFunc("PUT /admin/service", s.updateService)
	mux.HandleFunc("DELETE /admin/state", s.resetState)
}

func (s *Server) health(writer http.ResponseWriter, _ *http.Request) {
	write(writer, http.StatusOK, map[string]any{"status": "ok", "documents": len(s.documents.Documents(store.DocumentFilter{}))})
}

func (s *Server) units(writer http.ResponseWriter, _ *http.Request) {
	write(writer, http.StatusOK, uf.All())
}

func (s *Server) models(writer http.ResponseWriter, _ *http.Request) {
	write(writer, http.StatusOK, dfe.Models())
}

func (s *Server) statusCodes(writer http.ResponseWriter, _ *http.Request) {
	catalogue := s.statusCatalogue()
	write(writer, http.StatusOK, catalogue)
}

func (s *Server) statusCatalogue() []map[string]any {
	entries := make([]map[string]any, 0)
	for code, message := range status.Catalogue() {
		entries = append(entries, map[string]any{"code": int(code), "message": message})
	}
	return entries
}

func (s *Server) schemaCatalogue(writer http.ResponseWriter, _ *http.Request) {
	if s.schemas == nil {
		write(writer, http.StatusOK, map[string]any{"loaded": false, "documents": 0, "roots": []string{}})
		return
	}
	roots := s.schemas.Roots()
	sort.Strings(roots)
	write(writer, http.StatusOK, map[string]any{
		"loaded":    true,
		"documents": s.schemas.Documents(),
		"roots":     roots,
	})
}

func (s *Server) satCommands(writer http.ResponseWriter, _ *http.Request) {
	write(writer, http.StatusOK, sat.Commands())
}

func (s *Server) endpoints(writer http.ResponseWriter, request *http.Request) {
	base := request.URL.Query().Get("base")
	if base == "" {
		base = "http://" + request.Host
	}
	base = strings.TrimRight(base, "/")
	catalogue := make([]map[string]any, 0)
	for _, unit := range uf.All() {
		services := make(map[string]map[string]string, len(s.services))
		for _, service := range s.services {
			services[service.Name] = map[string]string{
				"homologacao": base + "/homologacao/" + unit.Acronym + "/" + service.Name,
				"producao":    base + "/producao/" + unit.Acronym + "/" + service.Name,
			}
		}
		catalogue = append(catalogue, map[string]any{
			"acronym":        unit.Acronym,
			"code":           unit.Code,
			"name":           unit.Name,
			"nfeAuthorizer":  unit.NFeAuthorizer,
			"nfceAuthorizer": unit.NFCeAuthorizer,
			"contingency":    unit.ContingencyRoute,
			"services":       services,
		})
	}
	write(writer, http.StatusOK, catalogue)
}

func (s *Server) listDocuments(writer http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()
	environment, _ := strconv.Atoi(query.Get("environment"))
	afterNSU, _ := strconv.ParseInt(query.Get("afterNsu"), 10, 64)
	documents := s.documents.Documents(store.DocumentFilter{
		IssuerTaxID: query.Get("issuer"),
		Environment: environment,
		AfterNSU:    afterNSU,
		Model:       dfe.Model(query.Get("model")),
	})
	summaries := make([]map[string]any, 0, len(documents))
	for _, document := range documents {
		summaries = append(summaries, map[string]any{
			"key":         document.Key,
			"model":       document.Model,
			"environment": document.Environment,
			"issuerTaxId": document.IssuerTaxID,
			"series":      document.Series,
			"number":      document.Number,
			"protocol":    document.Protocol,
			"status":      document.Status,
			"reason":      document.Reason,
			"cancelled":   document.Cancelled,
			"receivedAt":  document.ReceivedAt,
			"events":      len(document.Events),
			"nsu":         document.NSU,
		})
	}
	write(writer, http.StatusOK, summaries)
}

func (s *Server) showDocument(writer http.ResponseWriter, request *http.Request) {
	document, found := s.documents.Document(request.PathValue("key"))
	if !found {
		write(writer, http.StatusNotFound, map[string]string{"error": "document not found"})
		return
	}
	write(writer, http.StatusOK, document)
}

func (s *Server) listVoidings(writer http.ResponseWriter, _ *http.Request) {
	write(writer, http.StatusOK, s.documents.Voidings())
}

func (s *Server) listScenarios(writer http.ResponseWriter, _ *http.Request) {
	write(writer, http.StatusOK, s.scenarios.Rules())
}

func (s *Server) createScenario(writer http.ResponseWriter, request *http.Request) {
	var rule scenario.Rule
	if err := json.NewDecoder(request.Body).Decode(&rule); err != nil {
		write(writer, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if rule.Status == 0 {
		write(writer, http.StatusBadRequest, map[string]string{"error": "status is required"})
		return
	}
	write(writer, http.StatusCreated, s.scenarios.Add(rule))
}

func (s *Server) deleteScenario(writer http.ResponseWriter, request *http.Request) {
	if !s.scenarios.Remove(request.PathValue("id")) {
		write(writer, http.StatusNotFound, map[string]string{"error": "rule not found"})
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (s *Server) clearScenarios(writer http.ResponseWriter, _ *http.Request) {
	s.scenarios.Clear()
	writer.WriteHeader(http.StatusNoContent)
}

type serviceState struct {
	Status       *int  `json:"status,omitempty"`
	Asynchronous *bool `json:"asynchronous,omitempty"`
	AverageTime  *int  `json:"averageTime,omitempty"`
}

func (s *Server) showService(writer http.ResponseWriter, _ *http.Request) {
	write(writer, http.StatusOK, map[string]any{
		"status":       s.scenarios.ServiceStatus(),
		"reason":       status.Message(s.scenarios.ServiceStatus()),
		"asynchronous": s.scenarios.Asynchronous(),
		"averageTime":  s.scenarios.AverageTime(),
	})
}

func (s *Server) updateService(writer http.ResponseWriter, request *http.Request) {
	var state serviceState
	if err := json.NewDecoder(request.Body).Decode(&state); err != nil {
		write(writer, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if state.Status != nil {
		s.scenarios.SetServiceStatus(status.Code(*state.Status))
	}
	if state.Asynchronous != nil {
		s.scenarios.SetAsynchronous(*state.Asynchronous)
	}
	if state.AverageTime != nil {
		s.scenarios.SetAverageTime(*state.AverageTime)
	}
	s.showService(writer, request)
}

func (s *Server) resetState(writer http.ResponseWriter, _ *http.Request) {
	s.documents.Reset()
	s.scenarios.Clear()
	s.scenarios.SetServiceStatus(status.ServiceRunning)
	s.scenarios.SetAsynchronous(false)
	writer.WriteHeader(http.StatusNoContent)
}

func (s *Server) describe(writer http.ResponseWriter, request *http.Request) {
	write(writer, http.StatusOK, map[string]any{
		"service":     "fake-sefaz",
		"path":        request.URL.Path,
		"post":        "send the SEFAZ SOAP envelope to any path; the operation is read from the body",
		"operations":  s.services,
		"admin":       "/admin/health, /admin/endpoints, /admin/models, /admin/documents, /admin/scenarios, /admin/service, /admin/state",
		"sat":         "POST /sat/{command} for the CF-e SAT equipment commands",
		"forceHeader": ForcedStatusHeader,
	})
}

func write(writer http.ResponseWriter, code int, body any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(code)
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	encoder.Encode(body)
}
