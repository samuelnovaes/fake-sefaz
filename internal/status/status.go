package status

type Code int

const (
	Authorized                 Code = 100
	CancellationAuthorized     Code = 101
	VoidingAuthorized          Code = 102
	BatchReceived              Code = 103
	BatchProcessed             Code = 104
	BatchInProcess             Code = 105
	BatchNotFound              Code = 106
	ServiceRunning             Code = 107
	ServicePausedBriefly       Code = 108
	ServicePausedIndefinite    Code = 109
	UseDenied                  Code = 110
	RegistrationOneMatch       Code = 111
	RegistrationNoMatch        Code = 217
	EventBatchProcessed        Code = 128
	EventLinked                Code = 135
	EventNotLinked             Code = 136
	NoDocumentFound            Code = 137
	DocumentFound              Code = 138
	AuthorizedLate             Code = 150
	CancellationAuthorizedLate Code = 155

	RejectedDuplicate                Code = 204
	RejectedDenied                   Code = 205
	RejectedVoided                   Code = 206
	RejectedSchema                   Code = 215
	RejectedNotFound                 Code = 217
	RejectedCancelled                Code = 218
	RejectedProtocolMismatch         Code = 222
	RejectedBatchSchema              Code = 225
	RejectedIssuerUF                 Code = 226
	RejectedCheckDigit               Code = 236
	RejectedVersionAhead             Code = 238
	RejectedVersionUnsupported       Code = 239
	RejectedRangeUsed                Code = 241
	RejectedEnvironment              Code = 252
	RejectedRangeVoided              Code = 256
	RejectedCertificate              Code = 280
	RejectedSignature                Code = 297
	RejectedIssuerIrregular          Code = 301
	RejectedRecipientIrregular       Code = 302
	RejectedKeyInexistent            Code = 494
	RejectedCancellationDeadline     Code = 501
	RejectedDiscountTotal            Code = 537
	RejectedDuplicateOtherKey        Code = 539
	RejectedMonthMismatch            Code = 561
	RejectedRandomCodeMismatch       Code = 562
	RejectedVoidingRepeated          Code = 563
	RejectedDuplicateEvent           Code = 573
	RejectedEventNeedsAuthorized     Code = 580
	RejectedKeyMismatch              Code = 613
	RejectedItemValue                Code = 629
	RejectedMisuse                   Code = 656
	RejectedHomologationName         Code = 693
	RejectedNCMUnknown               Code = 778
	RejectedSubstituteKeyInvalid     Code = 910
	RejectedSubstituteKeyIncorrect   Code = 911
	RejectedSubstituteMissing        Code = 912
	RejectedSubstituteUnavailable    Code = 913
	RejectedSubstituteIssuedLate     Code = 914
	RejectedSubstituteTotal          Code = 915
	RejectedSubstituteICMSTotal      Code = 916
	RejectedSubstituteRecipient      Code = 917
	RejectedSubstituteItemCount      Code = 918
	RejectedSubstituteItem           Code = 919
	RejectedSubstitutionIssuance     Code = 920
	RejectedSubstituteNotContingency Code = 921
	RejectedUncatalogued             Code = 999
)

var messages = map[Code]string{
	Authorized:                 "Autorizado o uso da NF-e",
	CancellationAuthorized:     "Cancelamento de NF-e homologado",
	VoidingAuthorized:          "Inutilizacao de numero homologado",
	BatchReceived:              "Lote recebido com sucesso",
	BatchProcessed:             "Lote processado",
	BatchInProcess:             "Lote em processamento",
	BatchNotFound:              "Lote nao localizado",
	ServiceRunning:             "Servico em Operacao",
	ServicePausedBriefly:       "Servico Paralisado Momentaneamente (curto prazo)",
	ServicePausedIndefinite:    "Servico Paralisado sem Previsao",
	UseDenied:                  "Uso Denegado",
	RegistrationOneMatch:       "Consulta cadastro com uma ocorrencia",
	EventBatchProcessed:        "Lote de Evento Processado",
	EventLinked:                "Evento registrado e vinculado a NF-e",
	EventNotLinked:             "Evento registrado, mas nao vinculado a NF-e",
	NoDocumentFound:            "Nenhum documento localizado",
	DocumentFound:              "Documento(s) localizado(s)",
	AuthorizedLate:             "Autorizado o uso da NF-e, autorizacao fora de prazo",
	CancellationAuthorizedLate: "Cancelamento homologado fora de prazo",

	RejectedDuplicate:                "Rejeicao: Duplicidade de NF-e",
	RejectedDenied:                   "Rejeicao: NF-e esta denegada na base de dados da SEFAZ",
	RejectedVoided:                   "Rejeicao: NF-e ja esta inutilizada na Base de Dados da SEFAZ",
	RejectedSchema:                   "Rejeicao: Falha no schema XML",
	RejectedNotFound:                 "Rejeicao: NF-e nao consta na base de dados da SEFAZ",
	RejectedCancelled:                "Rejeicao: NF-e ja esta cancelada na base de dados da SEFAZ",
	RejectedProtocolMismatch:         "Rejeicao: Protocolo de Autorizacao de Uso difere do cadastrado",
	RejectedBatchSchema:              "Rejeicao: Falha no Schema XML do lote de NFe",
	RejectedIssuerUF:                 "Rejeicao: Codigo da UF do Emitente diverge da UF autorizadora",
	RejectedCheckDigit:               "Rejeicao: Chave de Acesso com digito verificador invalido",
	RejectedVersionAhead:             "Rejeicao: Cabecalho - Versao do arquivo XML superior a versao vigente",
	RejectedVersionUnsupported:       "Rejeicao: Cabecalho - Versao do arquivo XML nao suportada",
	RejectedRangeUsed:                "Rejeicao: Um numero da faixa ja foi utilizado",
	RejectedEnvironment:              "Rejeicao: Ambiente informado diverge do Ambiente de recebimento",
	RejectedRangeVoided:              "Rejeicao: Uma NF-e da faixa ja esta inutilizada na Base de dados da SEFAZ",
	RejectedCertificate:              "Rejeicao: Certificado Transmissor invalido",
	RejectedSignature:                "Rejeicao: Assinatura difere do calculado",
	RejectedIssuerIrregular:          "Rejeicao: Uso Denegado: Irregularidade fiscal do emitente",
	RejectedRecipientIrregular:       "Rejeicao: Uso Denegado: Irregularidade fiscal do destinatario",
	RejectedKeyInexistent:            "Rejeicao: Chave de Acesso inexistente",
	RejectedCancellationDeadline:     "Rejeicao: Prazo de cancelamento superior ao previsto na legislacao",
	RejectedDiscountTotal:            "Rejeicao: Total do Desconto difere do somatorio dos itens",
	RejectedDuplicateOtherKey:        "Rejeicao: Duplicidade de NF-e com diferenca na Chave de Acesso",
	RejectedMonthMismatch:            "Rejeicao: Mes de Emissao informado na Chave de Acesso difere do Mes de Emissao da NF-e",
	RejectedRandomCodeMismatch:       "Rejeicao: Codigo Numerico informado na Chave de Acesso difere do Codigo Numerico da NF-e",
	RejectedVoidingRepeated:          "Rejeicao: Ja existe pedido de Inutilizacao com a mesma faixa de inutilizacao",
	RejectedDuplicateEvent:           "Rejeicao: Duplicidade de Evento",
	RejectedEventNeedsAuthorized:     "Rejeicao: O evento exige uma NF-e autorizada",
	RejectedKeyMismatch:              "Rejeicao: Chave de Acesso difere da existente em BD",
	RejectedItemValue:                "Rejeicao: Valor do Produto difere do produto Valor Unitario de Comercializacao e Quantidade Comercial",
	RejectedMisuse:                   "Rejeicao: Consumo indevido pelo aplicativo da empresa",
	RejectedHomologationName:         "Rejeicao: NF-e emitida em ambiente de homologacao com Razao Social do destinatario diferente de NF-E EMITIDA EM AMBIENTE DE HOMOLOGACAO - SEM VALOR FISCAL",
	RejectedNCMUnknown:               "Rejeicao: Informado NCM inexistente",
	RejectedSubstituteKeyInvalid:     "Rejeicao: Chave de Acesso NF-e Substituta invalida",
	RejectedSubstituteKeyIncorrect:   "Rejeicao: Chave de Acesso NF-e Substituta incorreta",
	RejectedSubstituteMissing:        "Rejeicao: NF-e Substituta inexistente",
	RejectedSubstituteUnavailable:    "Rejeicao: NF-e Substituta Denegada ou Cancelada",
	RejectedSubstituteIssuedLate:     "Rejeicao: Data de emissao da NF-e Substituta maior que 2 horas da data de emissao da NF-e a ser cancelada",
	RejectedSubstituteTotal:          "Rejeicao: Valor total da NF-e Substituta difere do valor da NF-e a ser cancelada",
	RejectedSubstituteICMSTotal:      "Rejeicao: Valor total do ICMS da NF-e Substituta difere do valor da NF-e a ser cancelada",
	RejectedSubstituteRecipient:      "Rejeicao: Identificacao do destinatario da NF-e Substituta difere da identificacao do destinatario da NF-e a ser cancelada.",
	RejectedSubstituteItemCount:      "Rejeicao: Quantidade de itens da NF-e Substituta difere da quantidade de itens da NF-e a ser cancelada.",
	RejectedSubstituteItem:           "Rejeicao: Item da NF-e Substituta difere do mesmo item da NF-e a ser cancelada.",
	RejectedSubstitutionIssuance:     "Rejeicao: Tipo de Emissao invalido no Cancelamento por Substituicao",
	RejectedSubstituteNotContingency: "Rejeicao: Tipo de emissao da NF-e substituta deve ser de contingencia",
	RejectedUncatalogued:             "Rejeicao: Erro nao catalogado",
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

func InUse(code Code) bool {
	return code == Authorized || code == AuthorizedLate
}

func RefusesRequest(code Code) bool {
	return code == RejectedMisuse
}
