package main

import (
	"context"
	"fmt"
	"sync"

	"winsweep/internal/cleaner"
	"winsweep/internal/optimizer"
	"winsweep/internal/rules"
	"winsweep/internal/scanner"
	"winsweep/internal/tweaks"
	"winsweep/internal/winexec"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// FindingDTO é a representação de um achado da varredura exposta ao
// frontend via bindings do Wails.
type FindingDTO struct {
	ID           string `json:"id"`
	CategoryName string `json:"categoryName"`
	Description  string `json:"description"`
	Path         string `json:"path"`
	SizeBytes    int64  `json:"sizeBytes"`
	FileCount    int    `json:"fileCount"`
	Permanent    bool   `json:"permanent"`
}

// TweakDTO é a representação de um ajuste de desempenho exposta ao frontend.
type TweakDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
}

// App é a estrutura principal ligada ao frontend pelo Wails.
type App struct {
	ctx       context.Context
	cleaner   *cleaner.Cleaner
	optimizer *optimizer.Optimizer

	// emit e runScan existem como pontos de injeção para tornar StartScan
	// testável sem depender do runtime real do Wails (que mata o processo
	// com log.Fatal se chamado fora de uma aplicação Wails em execução) nem
	// varrer o sistema de arquivos de verdade a cada teste.
	emit    func(ctx context.Context, eventName string, data ...interface{})
	runScan func(ctx context.Context, onFinding func(scanner.Finding)) ([]scanner.Finding, error)

	mu       sync.Mutex
	findings map[string]scanner.Finding
	scanning bool
}

// NewApp cria a estrutura da aplicação.
func NewApp() *App {
	c := cleaner.New(winexec.PowerShellRunner{})
	a := &App{
		cleaner:   c,
		optimizer: optimizer.New(winexec.PowerShellRunner{}),
		findings:  make(map[string]scanner.Finding),
		emit:      runtime.EventsEmit,
	}
	a.runScan = func(ctx context.Context, onFinding func(scanner.Finding)) ([]scanner.Finding, error) {
		return scanner.Scan(ctx, rules.Builtin(), c.RecycleBinInfo, scanner.DefaultConcurrency, onFinding)
	}
	return a
}

// startup é chamado pelo Wails quando a aplicação inicia.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// StartScan inicia, em segundo plano, a varredura dos locais conhecidos de
// arquivos desnecessários do Windows. O progresso é emitido pelo evento
// "scan:finding" (um FindingDTO por vez) e a conclusão pelo evento
// "scan:done".
func (a *App) StartScan() {
	a.mu.Lock()
	if a.scanning {
		a.mu.Unlock()
		return
	}
	a.scanning = true
	a.findings = make(map[string]scanner.Finding)
	a.mu.Unlock()

	go func() {
		defer func() {
			a.mu.Lock()
			a.scanning = false
			a.mu.Unlock()
			a.emit(a.ctx, "scan:done")
		}()

		_, _ = a.runScan(a.ctx, func(f scanner.Finding) {
			dto := a.registerFinding(f)
			a.emit(a.ctx, "scan:finding", dto)
		})
	}()
}

// registerFinding guarda o Finding internamente (para permitir excluí-lo
// depois por ID) e devolve o DTO correspondente.
func (a *App) registerFinding(f scanner.Finding) FindingDTO {
	id := f.Path
	a.mu.Lock()
	a.findings[id] = f
	a.mu.Unlock()
	return FindingDTO{
		ID:           id,
		CategoryName: f.CategoryName,
		Description:  f.Description,
		Path:         f.Path,
		SizeBytes:    f.SizeBytes,
		FileCount:    f.FileCount,
		Permanent:    f.Permanent,
	}
}

// DeleteFinding exclui o item identificado por id, sempre mediante
// confirmação prévia do usuário na interface: envia para a Lixeira do
// Windows (ação reversível), exceto quando o próprio item já é a Lixeira,
// caso em que ela é esvaziada de forma definitiva.
func (a *App) DeleteFinding(id string) error {
	a.mu.Lock()
	f, ok := a.findings[id]
	a.mu.Unlock()
	if !ok {
		return fmt.Errorf("item não encontrado ou já removido: %s", id)
	}

	var err error
	if f.Permanent {
		err = a.cleaner.EmptyRecycleBin()
	} else {
		err = a.cleaner.SendToRecycleBin(f.Path)
	}
	if err != nil {
		return err
	}

	a.mu.Lock()
	delete(a.findings, id)
	a.mu.Unlock()
	return nil
}

// ListTweaks retorna o catálogo de ajustes de desempenho disponíveis. Não
// consulta nem altera nada na máquina — apenas descreve o que existe.
func (a *App) ListTweaks() []TweakDTO {
	all := tweaks.Builtin()
	dtos := make([]TweakDTO, 0, len(all))
	for _, tw := range all {
		dtos = append(dtos, TweakDTO{
			ID:          tw.ID,
			Name:        tw.Name,
			Category:    string(tw.Category),
			Description: tw.Description,
			Impact:      tw.Impact,
		})
	}
	return dtos
}

// DetectTweak verifica, na máquina do usuário, se o ajuste identificado por
// id está atualmente aplicado. Retorna "applied", "not_applied" ou
// "unknown". Só é executado quando o usuário pede explicitamente pela
// interface — nunca automaticamente.
func (a *App) DetectTweak(id string) (string, error) {
	state, err := a.optimizer.Detect(id)
	if err != nil {
		return "unknown", err
	}
	return tweakStateString(state), nil
}

// ApplyTweak aplica o ajuste identificado por id, mediante confirmação
// prévia do usuário na interface.
func (a *App) ApplyTweak(id string) error {
	return a.optimizer.Apply(id)
}

// RevertTweak desfaz o ajuste identificado por id, restaurando o
// comportamento padrão do Windows.
func (a *App) RevertTweak(id string) error {
	return a.optimizer.Revert(id)
}

func tweakStateString(s optimizer.State) string {
	switch s {
	case optimizer.StateApplied:
		return "applied"
	case optimizer.StateNotApplied:
		return "not_applied"
	default:
		return "unknown"
	}
}
