package server

import (
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/vendermais/fake-sefaz/internal/authorizer"
	"github.com/vendermais/fake-sefaz/internal/scenario"
	"github.com/vendermais/fake-sefaz/internal/soap"
	"github.com/vendermais/fake-sefaz/internal/status"
	"github.com/vendermais/fake-sefaz/internal/store"
	"github.com/vendermais/fake-sefaz/internal/uf"
)

const ForcedStatusHeader = "X-Fake-Sefaz-Status"

type Handler interface {
	Operations() map[string]soap.Endpoint
	WebServices() []soap.Service
	Handle(request authorizer.Context, message soap.Message) ([]byte, error)
}

type registration struct {
	endpoint soap.Endpoint
	handler  Handler
}

type Server struct {
	engine     *authorizer.Engine
	documents  *store.Store
	scenarios  *scenario.Engine
	routes     map[string]registration
	operations map[string]bool
	services   []soap.Service
	extras     []http.Handler
	logger     *slog.Logger
	handler    http.Handler
}

func New(engine *authorizer.Engine, logger *slog.Logger, handlers ...Handler) *Server {
	server := &Server{
		engine:     engine,
		documents:  engine.Documents(),
		scenarios:  engine.Scenarios(),
		routes:     map[string]registration{},
		operations: map[string]bool{},
		logger:     logger,
	}
	for _, handler := range handlers {
		for operation, endpoint := range handler.Operations() {
			server.routes[operation] = registration{endpoint: endpoint, handler: handler}
			server.operations[operation] = true
		}
		server.services = append(server.services, handler.WebServices()...)
	}
	mux := http.NewServeMux()
	server.routeAdmin(mux)
	mux.HandleFunc("/", server.handleSOAP)
	server.handler = mux
	return server
}

func (s *Server) Mount(pattern string, handler http.Handler) {
	mux, ok := s.handler.(*http.ServeMux)
	if !ok {
		return
	}
	mux.Handle(pattern, handler)
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
	message, err := soap.Decode(body, s.operations)
	if err != nil {
		s.writeFault(writer, http.StatusBadRequest, "Sender", err.Error())
		return
	}
	route, known := s.routes[message.Operation]
	if !known {
		s.writeFault(writer, http.StatusBadRequest, "Sender", "operation "+message.Operation+" is not served")
		return
	}
	address := parseRoute(request.URL.Path)
	payload, err := route.handler.Handle(authorizer.Context{
		Operation:    message.Operation,
		UFCode:       address.UFCode,
		Environment:  address.Environment,
		ForcedStatus: forcedStatus(request),
	}, message)
	if err != nil {
		s.writeFault(writer, http.StatusInternalServerError, "Receiver", err.Error())
		return
	}
	s.logger.Info("operation served",
		"operation", message.Operation,
		"uf", address.UFCode,
		"environment", address.Environment,
		"path", request.URL.Path,
	)
	writer.Header().Set("Content-Type", soap.ContentType)
	writer.WriteHeader(http.StatusOK)
	writer.Write(soap.Encode(route.endpoint.WSDLNamespace, route.endpoint.ResultTag, payload, message.Enveloped))
}

func (s *Server) writeFault(writer http.ResponseWriter, code int, faultCode, reason string) {
	writer.Header().Set("Content-Type", soap.ContentType)
	writer.WriteHeader(code)
	writer.Write(soap.Fault(faultCode, reason))
}

type address struct {
	UFCode      string
	Environment int
}

func parseRoute(path string) address {
	resolved := address{}
	for _, segment := range strings.Split(strings.Trim(path, "/"), "/") {
		if unit, found := uf.ByAcronym(strings.ToUpper(segment)); found && resolved.UFCode == "" {
			resolved.UFCode = unit.Code
			continue
		}
		if uf.Known(segment) && resolved.UFCode == "" {
			resolved.UFCode = segment
			continue
		}
		switch strings.ToLower(segment) {
		case "producao", "production", "prod":
			resolved.Environment = authorizer.EnvironmentProduction
		case "homologacao", "homologation", "homolog", "sandbox":
			resolved.Environment = authorizer.EnvironmentHomologation
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
