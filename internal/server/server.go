package server

import (
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/vendermais/fake-sefaz/internal/nfe"
	"github.com/vendermais/fake-sefaz/internal/scenario"
	"github.com/vendermais/fake-sefaz/internal/soap"
	"github.com/vendermais/fake-sefaz/internal/status"
	"github.com/vendermais/fake-sefaz/internal/store"
	"github.com/vendermais/fake-sefaz/internal/uf"
)

const ForcedStatusHeader = "X-Fake-Sefaz-Status"

type Server struct {
	documents *store.Store
	scenarios *scenario.Engine
	service   *nfe.Service
	logger    *slog.Logger
	handler   http.Handler
}

func New(documents *store.Store, scenarios *scenario.Engine, service *nfe.Service, logger *slog.Logger) *Server {
	server := &Server{documents: documents, scenarios: scenarios, service: service, logger: logger}
	mux := http.NewServeMux()
	server.routeAdmin(mux)
	mux.HandleFunc("/", server.handleSOAP)
	server.handler = mux
	return server
}

func (s *Server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	s.handler.ServeHTTP(writer, request)
}

func (s *Server) handleSOAP(writer http.ResponseWriter, request *http.Request) {
	if request.Method == http.MethodGet {
		s.describe(writer, request)
		return
	}
	if request.Method != http.MethodPost {
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(request.Body)
	if err != nil {
		s.writeFault(writer, http.StatusBadRequest, "Sender", "request body could not be read")
		return
	}
	message, err := soap.Decode(body, nfe.Operations())
	if err != nil {
		s.writeFault(writer, http.StatusBadRequest, "Sender", err.Error())
		return
	}
	endpoint, known := nfe.Endpoint(message.Operation)
	if !known {
		s.writeFault(writer, http.StatusBadRequest, "Sender", "operation "+message.Operation+" is not served")
		return
	}
	route := parseRoute(request.URL.Path)
	payload, err := s.service.Handle(nfe.Request{
		Operation:    message.Operation,
		Version:      message.Version,
		Payload:      message.Payload,
		UFCode:       route.UFCode,
		Environment:  route.Environment,
		ForcedStatus: forcedStatus(request),
	})
	if err != nil {
		s.writeFault(writer, http.StatusInternalServerError, "Receiver", err.Error())
		return
	}
	s.logger.Info("operation served",
		"operation", message.Operation,
		"uf", route.UFCode,
		"environment", route.Environment,
		"path", request.URL.Path,
	)
	writer.Header().Set("Content-Type", soap.ContentType)
	writer.WriteHeader(http.StatusOK)
	writer.Write(soap.Encode(endpoint.WSDLNamespace, endpoint.ResultTag, payload, message.Enveloped))
}

func (s *Server) writeFault(writer http.ResponseWriter, code int, faultCode, reason string) {
	writer.Header().Set("Content-Type", soap.ContentType)
	writer.WriteHeader(code)
	writer.Write(soap.Fault(faultCode, reason))
}

type route struct {
	UFCode      string
	Environment int
}

func parseRoute(path string) route {
	resolved := route{}
	for _, segment := range strings.Split(strings.Trim(path, "/"), "/") {
		normalized := strings.ToUpper(segment)
		if unit, found := uf.ByAcronym(normalized); found && resolved.UFCode == "" {
			resolved.UFCode = unit.Code
			continue
		}
		if uf.Known(segment) && resolved.UFCode == "" {
			resolved.UFCode = segment
			continue
		}
		switch strings.ToLower(segment) {
		case "producao", "production", "prod":
			resolved.Environment = nfe.EnvironmentProduction
		case "homologacao", "homologation", "homolog", "sandbox":
			resolved.Environment = nfe.EnvironmentHomologation
		}
	}
	return resolved
}

func forcedStatus(request *http.Request) status.Code {
	value := request.Header.Get(ForcedStatusHeader)
	if value == "" {
		return 0
	}
	code, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return status.Code(code)
}
