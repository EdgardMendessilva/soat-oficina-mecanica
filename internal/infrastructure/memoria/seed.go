package memoria

import (
	"context"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/application"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/cadastro"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/identidade"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/infrastructure/seguranca"
)

// SenhaDemo é a senha dos usuários de demonstração (política da RN12).
const SenhaDemo = "Teste@1234"

// Semear cadastra o primeiro Administrador (RN14) e, se demo for true, um usuário de cada perfil.
func Semear(u *Usuarios, adminDocumento, adminSenha string, demo bool) error {
	iniciais, err := UsuariosIniciais(adminDocumento, adminSenha, demo)
	if err != nil {
		return err
	}
	for _, x := range iniciais {
		u.Adicionar(x)
	}
	return nil
}

// UsuariosIniciais monta o Administrador (RN14) e, se demo for true, um usuário de cada perfil,
// com a senha já em hash. Os repositórios usam a lista para semear o armazenamento.
func UsuariosIniciais(adminDocumento, adminSenha string, demo bool) ([]identidade.Usuario, error) {
	var out []identidade.Usuario
	add := func(id, nome, doc string, p identidade.Perfil, clienteID, senha string) error {
		h, err := seguranca.HashSenha(senha)
		if err != nil {
			return err
		}
		out = append(out, identidade.Usuario{ID: id, Nome: nome, Documento: doc, Perfil: p, ClienteID: clienteID, SenhaHash: h})
		return nil
	}
	if adminDocumento != "" && adminSenha != "" {
		if err := add("adm-1", "Administrador", adminDocumento, identidade.Administrador, "", adminSenha); err != nil {
			return nil, err
		}
	}
	if !demo {
		return out, nil
	}
	demos := []struct {
		id, nome, doc string
		p             identidade.Perfil
		cli           string
	}{
		{"adm-demo", "Administrador Demo", "11144477735", identidade.Administrador, ""},
		{"ger-demo", "Gerente Demo", "52998224725", identidade.Gerente, ""},
		{"ate-demo", "Atendente Demo", "39053344705", identidade.Atendente, ""},
		{"mec-demo", "Mecânico Demo", "16899535009", identidade.Mecanico, ""},
		{"cli-demo", "Cliente Demo", "93541134780", identidade.Cliente, "cli-demo"},
		{"cli-outro", "Outro Cliente Demo", "86288366757", identidade.Cliente, "cli-outro"},
	}
	for _, d := range demos {
		if err := add(d.id, d.nome, d.doc, d.p, d.cli, SenhaDemo); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// SemearCadastro grava os clientes e veículos de demonstração que correspondem aos usuários Cliente.
// Funciona com qualquer repositório cujo Salvar atualize quando o ID já existe.
func SemearCadastro(ctx context.Context, clientes application.ClienteRepositorio, veiculos application.VeiculoRepositorio) error {
	demos := []struct {
		clienteID, doc, nome, veiculoID, placa, modelo string
		ano                                            int
	}{
		{"cli-demo", "93541134780", "Cliente Demo", "veic-demo", "ABC1D23", "Kwid", 2023},
		{"cli-outro", "86288366757", "Outro Cliente Demo", "veic-outro", "XYZ9K87", "Onix", 2021},
	}
	for _, d := range demos {
		c, err := cadastro.NovoCliente(d.clienteID, d.doc, d.nome, d.clienteID+"@oficina.local", "", "")
		if err != nil {
			return err
		}
		if err := clientes.Salvar(ctx, c); err != nil {
			return err
		}
		v, err := cadastro.NovoVeiculo(d.veiculoID, d.clienteID, d.placa, "Fabricante Demo", d.modelo, d.ano, "Prata")
		if err != nil {
			return err
		}
		if err := veiculos.Salvar(ctx, v); err != nil {
			return err
		}
	}
	return nil
}
