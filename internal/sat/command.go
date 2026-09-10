package sat

type Command struct {
	Name        string `json:"name"`
	SuccessCode string `json:"successCode"`
	Message     string `json:"message"`
}

var commands = []Command{
	{"AtivarSAT", "04000", "Ativado corretamente"},
	{"ComunicarCertificadoICPBRASIL", "05000", "Certificado transmitido com sucesso"},
	{"EnviarDadosVenda", "06000", "Emitido com sucesso"},
	{"CancelarUltimaVenda", "07000", "Cancelamento efetuado com sucesso"},
	{"ConsultarSAT", "08000", "SAT em Operacao"},
	{"TesteFimAFim", "09000", "Emitido com sucesso"},
	{"ConsultarStatusOperacional", "10000", "Resposta com sucesso"},
	{"ConfigurarInterfaceDeRede", "11000", "Rede configurada com sucesso"},
	{"AssociarAssinatura", "12000", "Assinatura do aplicativo comercial registrada"},
	{"AtualizarSoftwareSAT", "13000", "Software atualizado com sucesso"},
	{"ExtrairLogs", "14000", "Transferencia completa"},
	{"BloquearSAT", "15000", "Equipamento bloqueado com sucesso"},
	{"DesbloquearSAT", "16000", "Equipamento desbloqueado com sucesso"},
	{"TrocarCodigoDeAtivacao", "17000", "Codigo de ativacao alterado com sucesso"},
	{"ConsultarNumeroSessao", "18000", "Resposta com sucesso"},
}

var catalogue = map[string]Command{}

func init() {
	for _, command := range commands {
		catalogue[command.Name] = command
	}
}

func Commands() []Command {
	copied := make([]Command, len(commands))
	copy(copied, commands)
	return copied
}

func Lookup(name string) (Command, bool) {
	command, found := catalogue[name]
	return command, found
}
