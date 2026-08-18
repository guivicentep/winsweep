package rules

import "testing"

func TestBuiltin_AllCategoriesAreWellFormed(t *testing.T) {
	categories := Builtin()
	if len(categories) == 0 {
		t.Fatal("Builtin() não retornou nenhuma categoria")
	}

	seenIDs := make(map[string]bool)
	for _, c := range categories {
		t.Run(c.ID, func(t *testing.T) {
			if c.ID == "" {
				t.Error("categoria sem ID")
			}
			if seenIDs[c.ID] {
				t.Errorf("ID duplicado: %s", c.ID)
			}
			seenIDs[c.ID] = true

			if c.Name == "" {
				t.Error("categoria sem Name")
			}
			if c.Description == "" {
				t.Errorf("categoria %s sem Description explicando para que serve o local", c.ID)
			}
			if len(c.Description) < 20 {
				t.Errorf("categoria %s tem Description curta demais para explicar o item ao usuário: %q", c.ID, c.Description)
			}
			if len(c.PathPatterns) == 0 {
				t.Errorf("categoria %s não define nenhum PathPatterns", c.ID)
			}
		})
	}
}

func TestBuiltin_OnlyRecycleBinIsPermanent(t *testing.T) {
	for _, c := range Builtin() {
		if c.Permanent && c.ID != "recycle_bin" {
			t.Errorf("categoria %s está marcada como Permanent, mas apenas recycle_bin deveria ser (exclusão definitiva exige cuidado extra)", c.ID)
		}
	}
}
