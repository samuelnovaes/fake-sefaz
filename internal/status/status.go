package status

type Code int

const (
	Authorized              Code = 100
	CancellationAuthorized  Code = 101
	VoidingAuthorized       Code = 102
	BatchReceived           Code = 103
	BatchProcessed          Code = 104
	BatchInProcess          Code = 105
	BatchNotFound           Code = 106
	ServiceRunning          Code = 107
	ServicePausedBriefly    Code = 108
	ServicePausedIndefinite Code = 109
	UseDenied               Code = 110
	RegistrationOneMatch    Code = 111
	RegistrationNoMatch     Code = 217
	EventBatchProcessed     Code = 128
	EventLinked             Code = 135
	EventNotLinked          Code = 136
	NoDocumentFound         Code = 137
	DocumentFound           Code = 138

	RejectedDuplicate            Code = 204
	RejectedSchema               Code = 215
	RejectedNotFound             Code = 217
	RejectedBatchSchema          Code = 225
	RejectedIssuerUF             Code = 226
	RejectedCheckDigit           Code = 236
	RejectedVersionAhead         Code = 238
	RejectedVersionUnsupported   Code = 239
	RejectedEnvironment          Code = 252
	RejectedCertificate          Code = 280
	RejectedSignature            Code = 297
	RejectedIssuerIrregular      Code = 301
	RejectedRecipientIrregular   Code = 302
	RejectedCancellationDeadline Code = 501
	RejectedDiscountTotal        Code = 537
	RejectedDuplicateOtherKey    Code = 539
	RejectedDuplicateEvent       Code = 573
	RejectedKeyMismatch          Code = 613
	RejectedItemValue            Code = 629
	RejectedHomologationName     Code = 693
	RejectedUncatalogued         Code = 999
)

var messages = map[Code]string{
	Authorized:              "Autorizado o uso da NF-e",
	CancellationAuthorized:  "Cancelamento de NF-e homologado",
	VoidingAuthorized:       "Inutilizacao de numero homologado",
	BatchReceived:           "Lote recebido com sucesso",
	BatchProcessed:          "Lote processado",
	BatchInProcess:          "Lote em processamento",
	BatchNotFound:           "Lote nao localizado",
	ServiceRunning:          "Servico em Operacao",
	ServicePausedBriefly:    "Servico Paralisado Momentaneamente (curto prazo)",
	ServicePausedIndefinite: "Servico Paralisado sem Previsao",
	UseDenied:               "Uso Denegado",
	RegistrationOneMatch:    "Consulta cadastro com uma ocorrencia",
	EventBatchProcessed:     "Lote de Evento Processado",
	EventLinked:             "Evento registrado e vinculado a NF-e",
	EventNotLinked:          "Evento registrado, mas nao vinculado a NF-e",
	NoDocumentFound:         "Nenhum documento localizado",
	DocumentFound:           "Documento(s) localizado(s)",

	RejectedDuplicate:            "Rejeicao: Duplicidade de NF-e",
	RejectedSchema:               "Rejeicao: Falha no schema XML",
	RejectedNotFound:             "Rejeicao: NF-e nao consta na base de dados da SEFAZ",
	RejectedBatchSchema:          "Rejeicao: Falha no Schema XML do lote de NFe",
	RejectedIssuerUF:             "Rejeicao: Codigo da UF do Emitente diverge da UF autorizadora",
	RejectedCheckDigit:           "Rejeicao: Chave de Acesso com digito verificador invalido",
	RejectedVersionAhead:         "Rejeicao: Cabecalho - Versao do arquivo XML superior a versao vigente",
	RejectedVersionUnsupported:   "Rejeicao: Cabecalho - Versao do arquivo XML nao suportada",
	RejectedEnvironment:          "Rejeicao: Ambiente informado diverge do Ambiente de recebimento",
	RejectedCertificate:          "Rejeicao: Certificado Transmissor invalido",
	RejectedSignature:            "Rejeicao: Assinatura difere do calculado",
	RejectedIssuerIrregular:      "Rejeicao: Uso Denegado: Irregularidade fiscal do emitente",
	RejectedRecipientIrregular:   "Rejeicao: Uso Denegado: Irregularidade fiscal do destinatario",
	RejectedCancellationDeadline: "Rejeicao: Prazo de cancelamento superior ao previsto na legislacao",
	RejectedDiscountTotal:        "Rejeicao: Total do Desconto difere do somatorio dos itens",
	RejectedDuplicateOtherKey:    "Rejeicao: Duplicidade de NF-e com diferenca na Chave de Acesso",
	RejectedDuplicateEvent:       "Rejeicao: Duplicidade de Evento",
	RejectedKeyMismatch:          "Rejeicao: Chave de Acesso difere da existente em BD",
	RejectedItemValue:            "Rejeicao: Valor do Produto difere do produto Valor Unitario de Comercializacao e Quantidade Comercial",
	RejectedHomologationName:     "Rejeicao: NF-e emitida em ambiente de homologacao com Razao Social do destinatario diferente de NF-E EMITIDA EM AMBIENTE DE HOMOLOGACAO - SEM VALOR FISCAL",
	RejectedUncatalogued:         "Rejeicao: Erro nao catalogado",
}

func Message(code Code) string {
	message, found := messages[code]
	if !found {
		return messages[RejectedUncatalogued]
	}
	return message
}

func Known(code Code) bool {
	_, found := messages[code]
	return found
}

func Catalogue() map[Code]string {
	copied := make(map[Code]string, len(messages))
	for code, message := range messages {
		copied[code] = message
	}
	return copied
}

func Denied(code Code) bool {
	return code == RejectedIssuerIrregular || code == RejectedRecipientIrregular || code == UseDenied
}
