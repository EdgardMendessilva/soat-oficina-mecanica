// Package ordemservico reúne as regras do contexto Ordem de Serviço (core):
// OS, orçamento, status, desconto e decisão do cliente.
package ordemservico

import "errors"

// Status é a etapa em que a OS se encontra (RN03).
type Status string

// Status possíveis da OS, na ordem do fluxo normal.
const (
	Recebida            Status = "Recebida"
	EmDiagnostico       Status = "Em diagnóstico"
	AguardandoAprovacao Status = "Aguardando aprovação"
	EmExecucao          Status = "Em execução"
	Finalizada          Status = "Finalizada"
	Entregue            Status = "Entregue"
)

// ErrTransicaoInvalida indica uma mudança de status que pula etapa ou não existe.
var ErrTransicaoInvalida = errors.New("transição de status inválida: a OS não pode pular etapas")

// transicoes lista de onde a OS pode ir. Na reprovação do orçamento a OS vai de
// Aguardando aprovação direto para Finalizada, sem execução (RN05).
var transicoes = map[Status][]Status{
	Recebida:            {EmDiagnostico},
	EmDiagnostico:       {AguardandoAprovacao},
	AguardandoAprovacao: {EmExecucao, Finalizada},
	EmExecucao:          {Finalizada},
	Finalizada:          {Entregue},
}

func (s Status) podeIrPara(destino Status) bool {
	for _, d := range transicoes[s] {
		if d == destino {
			return true
		}
	}
	return false
}
