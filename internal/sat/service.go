package sat

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/vendermais/fake-sefaz/internal/authorizer"
	"github.com/vendermais/fake-sefaz/internal/dfe"
	"github.com/vendermais/fake-sefaz/internal/status"
)

const (
	deviceSerial   = 900004019
	failureGeneric = "19000"
)

type request struct {
	Session          int    `json:"numeroSessao"`
	ActivationCode   string `json:"codigoDeAtivacao"`
	NewActivation    string `json:"novoCodigoDeAtivacao"`
	TaxID            string `json:"cnpj"`
	StateCode        string `json:"cUF"`
	SaleData         string `json:"dadosVenda"`
	CancellationData string `json:"dadosCancelamento"`
	QueryKey         string `json:"chaveConsulta"`
	Signature        string `json:"assinaturaAC"`
}

type response struct {
	Return  string   `json:"retorno"`
	Fields  []string `json:"campos"`
	Key     string   `json:"chaveConsulta,omitempty"`
	XML     string   `json:"xml,omitempty"`
	Command string   `json:"comando"`
}

type Service struct {
	engine   *authorizer.Engine
	mutex    sync.Mutex
	active   bool
	blocked  bool
	code     string
	sequence int64
	lastSale string
}

func NewService(engine *authorizer.Engine) *Service {
	return &Service{engine: engine, code: "12345678"}
}

func (s *Service) ServeHTTP(writer http.ResponseWriter, httpRequest *http.Request) {
	name := httpRequest.PathValue("command")
	command, known := Lookup(name)
	if !known {
		writeJSON(writer, http.StatusNotFound, map[string]string{"error": "unknown SAT command " + name})
		return
	}
	var body request
	if httpRequest.Body != nil {
		json.NewDecoder(httpRequest.Body).Decode(&body)
	}
	answer := s.run(command, body)
	answer.Command = command.Name
	answer.Fields = strings.Split(answer.Return, "|")
	writeJSON(writer, http.StatusOK, answer)
}

func (s *Service) run(command Command, body request) response {
	switch command.Name {
	case "EnviarDadosVenda":
		return s.sell(command, body)
	case "CancelarUltimaVenda":
		return s.cancel(command, body)
	case "ConsultarStatusOperacional":
		return s.operationalStatus(command, body)
	case "AtivarSAT":
		return s.activate(command, body)
	case "BloquearSAT":
		return s.setBlocked(command, body, true)
	case "DesbloquearSAT":
		return s.setBlocked(command, body, false)
	case "TrocarCodigoDeAtivacao":
		return s.changeCode(command, body)
	}
	return response{Return: plain(body.Session, command, "")}
}

func (s *Service) sell(command Command, body request) response {
	document := decodeContent(body.SaleData)
	if document == "" {
		return response{Return: failure(body.Session, "Dados de venda ausentes")}
	}
	parsed, err := dfe.ParseDocument([]byte(document))
	if err != nil && parsed.Key == "" {
		s.mutex.Lock()
		s.sequence++
		number := s.sequence
		s.mutex.Unlock()
		key := dfe.BuildSATAccessKey(
			firstNonEmpty(parsed.UFCode, "35"),
			s.engine.Now(),
			parsed.IssuerTaxID,
			deviceSerial,
			number,
			fmt.Sprintf("%06d", number),
		)
		document = injectIdentifier(document, key)
		parsed, err = dfe.ParseDocument([]byte(document))
		if err != nil {
			return response{Return: failure(body.Session, "CF-e invalido")}
		}
	}

	result := s.engine.Authorize(authorizer.Submission{
		Key:               parsed.Key,
		UFCode:            parsed.UFCode,
		Environment:       parsed.Environment,
		IssuerTaxID:       parsed.IssuerTaxID,
		DigestValue:       parsed.DigestValue,
		RecipientName:     parsed.RecipientName,
		RecipientDocument: parsed.RecipientDocument,
		Signed:            true,
		XML:               document,
	}, authorizer.Context{Operation: "EnviarDadosVenda"})
	if result.Status != status.Authorized {
		return response{Return: failure(body.Session, result.Reason)}
	}

	s.mutex.Lock()
	s.lastSale = result.Key
	s.mutex.Unlock()
	moment := s.engine.Now()
	line := strings.Join([]string{
		strconv.Itoa(body.Session), command.SuccessCode, command.Message, "0", "",
		base64.StdEncoding.EncodeToString([]byte(document)),
		moment.Format("20060102150405"),
		result.Key,
		"0.00",
		parsed.RecipientDocument,
		"fake-sefaz-qrcode",
	}, "|")
	return response{Return: line, Key: result.Key, XML: document}
}

func (s *Service) cancel(command Command, body request) response {
	s.mutex.Lock()
	key := firstNonEmpty(body.QueryKey, s.lastSale)
	s.mutex.Unlock()
	if key == "" {
		return response{Return: failure(body.Session, "Nenhum CF-e para cancelar")}
	}
	result := s.engine.RegisterEvent(authorizer.EventSubmission{
		Key:      key,
		Type:     dfe.EventCancellation,
		Sequence: 1,
		XML:      decodeContent(body.CancellationData),
	}, authorizer.Context{Operation: "CancelarUltimaVenda"})
	if result.Status != status.CancellationAuthorized {
		return response{Return: failure(body.Session, result.Reason)}
	}
	moment := s.engine.Now()
	line := strings.Join([]string{
		strconv.Itoa(body.Session), command.SuccessCode, command.Message, "0", "",
		base64.StdEncoding.EncodeToString([]byte("<CFeCanc/>")),
		moment.Format("20060102150405"), key, "0.00", "", "fake-sefaz-qrcode",
	}, "|")
	return response{Return: line, Key: key}
}

func (s *Service) operationalStatus(command Command, body request) response {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	content := strings.Join([]string{
		strconv.Itoa(deviceSerial),
		"DHCP", "192.168.0.10", "00:11:22:33:44:55", "255.255.255.0", "192.168.0.1", "8.8.8.8", "8.8.4.4",
		statusText(s.active, s.blocked),
		s.engine.Now().Format("20060102150405"),
		"0", "0", "0",
		s.engine.Now().Format("20060102150405"),
		"fake-sefaz", "0.08",
	}, "|")
	return response{Return: strings.Join([]string{
		strconv.Itoa(body.Session), command.SuccessCode, command.Message, "0", "", content,
	}, "|")}
}

func (s *Service) activate(command Command, body request) response {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if body.ActivationCode != "" {
		s.code = body.ActivationCode
	}
	s.active = true
	return response{Return: plain(body.Session, command, base64.StdEncoding.EncodeToString([]byte("fake-sefaz-csr")))}
}

func (s *Service) setBlocked(command Command, body request, blocked bool) response {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.blocked = blocked
	return response{Return: plain(body.Session, command, "")}
}

func (s *Service) changeCode(command Command, body request) response {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if body.ActivationCode != s.code {
		return response{Return: failure(body.Session, "Codigo de ativacao invalido")}
	}
	s.code = body.NewActivation
	return response{Return: plain(body.Session, command, "")}
}

func statusText(active, blocked bool) string {
	if blocked {
		return "BLOQUEADO"
	}
	if !active {
		return "DESBLOQUEADO_SEM_ATIVACAO"
	}
	return "ATIVO"
}

func plain(session int, command Command, extra string) string {
	line := strings.Join([]string{strconv.Itoa(session), command.SuccessCode, command.Message, "0", ""}, "|")
	if extra == "" {
		return line
	}
	return line + "|" + extra
}

func failure(session int, reason string) string {
	return strings.Join([]string{strconv.Itoa(session), failureGeneric, reason, "0", ""}, "|")
}

func decodeContent(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "<") {
		return trimmed
	}
	decoded, err := base64.StdEncoding.DecodeString(trimmed)
	if err != nil {
		return trimmed
	}
	return string(decoded)
}

func injectIdentifier(document, key string) string {
	index := strings.Index(document, "<infCFe")
	if index < 0 {
		return document
	}
	return document[:index+len("<infCFe")] + ` Id="CFe` + key + `"` + document[index+len("<infCFe"):]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func writeJSON(writer http.ResponseWriter, code int, body any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(code)
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	encoder.Encode(body)
}
