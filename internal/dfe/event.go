package dfe

const (
	EventCorrection            = "110110"
	EventCancellation          = "110111"
	EventCancellationBySwap    = "110112"
	EventOperationConfirmed    = "210200"
	EventOperationAcknowledged = "210210"
	EventOperationUnknown      = "210220"
	EventOperationNotPerformed = "210240"
)

var eventDescriptions = map[string]string{
	EventCorrection:            "Carta de Correcao",
	EventCancellation:          "Cancelamento",
	EventCancellationBySwap:    "Cancelamento por substituicao",
	EventOperationConfirmed:    "Confirmacao da Operacao",
	EventOperationAcknowledged: "Ciencia da Operacao",
	EventOperationUnknown:      "Desconhecimento da Operacao",
	EventOperationNotPerformed: "Operacao nao Realizada",
}

func EventDescription(eventType string) (string, bool) {
	description, found := eventDescriptions[eventType]
	return description, found
}

func EventLinksToDocument(eventType string) bool {
	return eventType == EventCorrection || eventType == EventCancellation || eventType == EventCancellationBySwap
}
