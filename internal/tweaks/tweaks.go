// Package tweaks define o catálogo curado de ajustes de desempenho para
// deixar o Windows mais leve em computadores mais lentos ou antigos:
// plano de energia e efeitos visuais. Cada ajuste é individual e reversível.
package tweaks

// Category agrupa os ajustes por área, usado para organizar a interface.
type Category string

const (
	CategoryPower  Category = "energia"
	CategoryVisual Category = "visual"
)

// Tweak descreve um ajuste de desempenho individual: para que serve, o que
// muda na prática e se algo além de aplicá-lo é necessário para ter efeito.
type Tweak struct {
	// ID é um identificador estável usado internamente (não muda entre versões).
	ID string
	// Name é o nome amigável exibido na interface.
	Name string
	// Category agrupa o ajuste (energia ou visual) para organização na UI.
	Category Category
	// Description explica, em linguagem simples, o que o ajuste faz e por
	// que ele ajuda em máquinas mais lentas.
	Description string
	// Impact avisa sobre efeitos colaterais relevantes, como precisar
	// reiniciar algo ou o custo em outro cenário (ex.: autonomia de bateria).
	Impact string
}

// Builtin retorna o catálogo de ajustes curados e considerados seguros e
// reversíveis: nenhum deles desinstala software, mexe em serviços do
// Windows ou em programas de inicialização.
func Builtin() []Tweak {
	return []Tweak{
		{
			ID:       "power_high_performance",
			Name:     "Plano de energia: Alto desempenho",
			Category: CategoryPower,
			Description: "Impede que o Windows reduza a velocidade do processador para " +
				"economizar energia. Em computadores lentos ligados na tomada, isso evita " +
				"quedas de desempenho perceptíveis no dia a dia.",
			Impact: "Efeito imediato. Em notebooks na bateria, reduz a autonomia — prefira " +
				"usar apenas com o carregador conectado.",
		},
		{
			ID:       "visual_transparency",
			Name:     "Desativar transparência da interface",
			Category: CategoryVisual,
			Description: "Desliga o efeito de vidro fosco (acrylic) usado na barra de " +
				"tarefas, no menu Iniciar e nas janelas. Esse efeito consome a placa de " +
				"vídeo continuamente; desativar ajuda bastante em GPUs fracas ou integradas antigas.",
			Impact: "Efeito imediato.",
		},
		{
			ID:       "visual_window_animations",
			Name:     "Desativar animações de minimizar/maximizar",
			Category: CategoryVisual,
			Description: "Remove a animação de abrir, fechar e minimizar janelas, deixando " +
				"a resposta da interface mais instantânea em máquinas lentas.",
			Impact: "Efeito imediato.",
		},
		{
			ID:       "visual_drag_full_windows",
			Name:     "Não redesenhar o conteúdo ao arrastar janelas",
			Category: CategoryVisual,
			Description: "Ao mover uma janela, mostra apenas o contorno em vez de redesenhar " +
				"o conteúdo inteiro a cada movimento, reduzindo o uso de CPU/GPU ao organizar janelas.",
			Impact: "Efeito imediato.",
		},
		{
			ID:       "visual_taskbar_animations",
			Name:     "Desativar animações da barra de tarefas",
			Category: CategoryVisual,
			Description: "Remove as animações ao abrir e fechar programas na barra de tarefas.",
			Impact: "Pode exigir reiniciar o Explorer ou fazer logoff para ter efeito completo.",
		},
	}
}
