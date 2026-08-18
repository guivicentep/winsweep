package tweaks

import "testing"

func TestBuiltin_AllTweaksAreWellFormed(t *testing.T) {
	all := Builtin()
	if len(all) == 0 {
		t.Fatal("Builtin() não retornou nenhum ajuste")
	}

	seenIDs := make(map[string]bool)
	for _, tw := range all {
		t.Run(tw.ID, func(t *testing.T) {
			if tw.ID == "" {
				t.Error("ajuste sem ID")
			}
			if seenIDs[tw.ID] {
				t.Errorf("ID duplicado: %s", tw.ID)
			}
			seenIDs[tw.ID] = true

			if tw.Name == "" {
				t.Error("ajuste sem Name")
			}
			if tw.Category != CategoryPower && tw.Category != CategoryVisual && tw.Category != CategoryGaming {
				t.Errorf("categoria desconhecida: %q", tw.Category)
			}
			if len(tw.Description) < 20 {
				t.Errorf("ajuste %s tem Description curta demais para explicar o efeito ao usuário: %q", tw.ID, tw.Description)
			}
			if tw.Impact == "" {
				t.Errorf("ajuste %s sem Impact explicando efeitos colaterais", tw.ID)
			}
		})
	}
}
