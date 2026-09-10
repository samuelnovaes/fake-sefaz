package status

import "github.com/vendermais/fake-sefaz/internal/dfe"

type naming struct {
	article string
	name    string
}

var namings = map[dfe.Model]naming{
	dfe.ModelNFe:   {"da", "NF-e"},
	dfe.ModelNFCe:  {"da", "NF-e"},
	dfe.ModelCTe:   {"do", "CT-e"},
	dfe.ModelCTeOS: {"do", "CT-e"},
	dfe.ModelMDFe:  {"do", "MDF-e"},
	dfe.ModelBPe:   {"do", "BP-e"},
	dfe.ModelNF3e:  {"da", "NF3e"},
	dfe.ModelCFe:   {"do", "CF-e"},
}

func MessageFor(model dfe.Model, code Code) string {
	document, known := namings[model]
	if !known {
		return Message(code)
	}
	switch code {
	case Authorized:
		return "Autorizado o uso " + document.article + " " + document.name
	case CancellationAuthorized:
		return "Cancelamento de " + document.name + " homologado"
	case RejectedDuplicate:
		return "Rejeicao: Duplicidade de " + document.name
	case RejectedNotFound:
		return "Rejeicao: " + document.name + " nao consta na base de dados da SEFAZ"
	case RejectedDuplicateOtherKey:
		return "Rejeicao: Duplicidade de " + document.name + " com diferenca na Chave de Acesso"
	case EventLinked:
		return "Evento registrado e vinculado a " + document.name
	case EventNotLinked:
		return "Evento registrado, mas nao vinculado a " + document.name
	}
	return Message(code)
}
