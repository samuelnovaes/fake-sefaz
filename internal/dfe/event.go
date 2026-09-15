package dfe

const (
	IssuanceNormal  = 1
	IssuanceEPEC    = 4
	IssuanceOffline = 9
)

const (
	EventCorrection            = "110110"
	EventCancellation          = "110111"
	EventCancellationBySwap    = "110112"
	EventClosing               = "110112"
	EventPriorEmission         = "110113"
	EventDriverAdded           = "110114"
	EventDocumentAdded         = "110115"
	EventBoardingMissed        = "110115"
	EventSeatChanged           = "110116"
	EventDeliveryReceipt       = "110160"
	EventOperationConfirmed    = "210200"
	EventOperationAcknowledged = "210210"
	EventOperationUnknown      = "210220"
	EventOperationNotPerformed = "210240"
	EventServiceDisagreement   = "610110"
)

var eventCatalogue = map[Model]map[string]string{
	ModelNFe: {
		EventCorrection:            "Carta de Correcao",
		EventCancellation:          "Cancelamento",
		EventOperationConfirmed:    "Confirmacao da Operacao",
		EventOperationAcknowledged: "Ciencia da Operacao",
		EventOperationUnknown:      "Desconhecimento da Operacao",
		EventOperationNotPerformed: "Operacao nao Realizada",
	},
	ModelCTe: {
		EventCorrection:          "Carta de Correcao",
		EventCancellation:        "Cancelamento",
		EventPriorEmission:       "EPEC",
		EventDeliveryReceipt:     "Comprovante de Entrega do CT-e",
		EventServiceDisagreement: "Prestacao de Servico em Desacordo",
	},
	ModelMDFe: {
		EventCancellation:  "Cancelamento",
		EventClosing:       "Encerramento",
		EventDriverAdded:   "Inclusao de Condutor",
		EventDocumentAdded: "Inclusao de DF-e",
	},
	ModelBPe: {
		EventCancellation:   "Cancelamento",
		EventBoardingMissed: "Nao Embarque",
		EventSeatChanged:    "Alteracao de Poltrona",
	},
	ModelNF3e: {
		EventCancellation: "Cancelamento",
	},
	ModelCFe: {
		EventCancellation: "Cancelamento",
	},
}

func init() {
	eventCatalogue[ModelNFCe] = extendedEvents(eventCatalogue[ModelNFe], EventCancellationBySwap, "Cancelamento por substituicao")
	eventCatalogue[ModelCTeOS] = eventCatalogue[ModelCTe]
}

func extendedEvents(events map[string]string, eventType, description string) map[string]string {
	extended := make(map[string]string, len(events)+1)
	for existing, text := range events {
		extended[existing] = text
	}
	extended[eventType] = description
	return extended
}

func EventDescription(model Model, eventType string) (string, bool) {
	events, known := eventCatalogue[model]
	if !known {
		return "", false
	}
	description, found := events[eventType]
	return description, found
}

func EventsOf(model Model) map[string]string {
	events, known := eventCatalogue[model]
	if !known {
		return map[string]string{}
	}
	copied := make(map[string]string, len(events))
	for eventType, description := range events {
		copied[eventType] = description
	}
	return copied
}

func EventLinksToDocument(eventType string) bool {
	switch eventType {
	case EventOperationConfirmed, EventOperationAcknowledged, EventOperationUnknown, EventOperationNotPerformed:
		return false
	}
	return true
}
