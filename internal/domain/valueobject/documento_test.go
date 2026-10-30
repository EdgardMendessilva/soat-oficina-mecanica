package valueobject

import "testing"

func TestNovoDocumento(t *testing.T) {
	casos := []struct {
		nome    string
		entrada string
		tipo    TipoDocumento
		numero  string
		valido  bool
	}{
		{"cpf com máscara", "529.982.247-25", CPF, "52998224725", true},
		{"cpf só dígitos", "52998224725", CPF, "52998224725", true},
		{"cnpj com máscara", "11.222.333/0001-81", CNPJ, "11222333000181", true},
		{"cnpj só dígitos", "11222333000181", CNPJ, "11222333000181", true},
		{"cpf dígito verificador errado", "529.982.247-26", "", "", false},
		{"cnpj dígito verificador errado", "11.222.333/0001-82", "", "", false},
		{"cpf com todos dígitos iguais", "111.111.111-11", "", "", false},
		{"cnpj com todos dígitos iguais", "00.000.000/0000-00", "", "", false},
		{"tamanho errado", "1234", "", "", false},
		{"vazio", "", "", "", false},
		{"letras", "529.982.247-2A", "", "", false},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			d, err := NovoDocumento(c.entrada)
			if c.valido {
				if err != nil {
					t.Fatalf("esperava documento válido, veio erro: %v", err)
				}
				if d.Tipo() != c.tipo || d.Numero() != c.numero || d.String() != c.numero {
					t.Fatalf("documento inesperado: tipo=%s numero=%s", d.Tipo(), d.Numero())
				}
				return
			}
			if err != ErrDocumentoInvalido {
				t.Fatalf("esperava ErrDocumentoInvalido, veio %v", err)
			}
		})
	}
}
