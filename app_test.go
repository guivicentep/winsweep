package main

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"winsweep/internal/cleaner"
	"winsweep/internal/optimizer"
	"winsweep/internal/scanner"
)

// fakeRunner grava o último script PowerShell recebido, evitando abrir um
// processo powershell.exe real durante os testes.
type fakeRunner struct {
	script string
	output string
	err    error
}

func (f *fakeRunner) RunPowerShell(script string) (string, error) {
	f.script = script
	if f.output != "" {
		return f.output, f.err
	}
	return "0|0", f.err
}

func newTestApp(runner *fakeRunner) *App {
	return &App{
		cleaner:   cleaner.New(runner),
		optimizer: optimizer.New(runner),
		findings:  make(map[string]scanner.Finding),
		ctx:       context.Background(),
		emit:      func(ctx context.Context, eventName string, data ...interface{}) {},
		runScan: func(ctx context.Context, onFinding func(scanner.Finding)) ([]scanner.Finding, error) {
			return nil, nil
		},
	}
}

func TestNewApp_StartsWithEmptyState(t *testing.T) {
	app := NewApp()
	if app.cleaner == nil {
		t.Error("cleaner não deveria ser nil")
	}
	if app.optimizer == nil {
		t.Error("optimizer não deveria ser nil")
	}
	if len(app.findings) != 0 {
		t.Error("findings deveria começar vazio")
	}
}

func TestRegisterFinding_StoresAndReturnsDTO(t *testing.T) {
	app := newTestApp(&fakeRunner{})
	f := scanner.Finding{
		CategoryID: "cat", CategoryName: "Categoria", Description: "desc",
		Path: `C:\tmp\x`, SizeBytes: 10, FileCount: 1,
	}

	dto := app.registerFinding(f)

	if dto.ID != f.Path || dto.CategoryName != f.CategoryName || dto.SizeBytes != f.SizeBytes {
		t.Errorf("dto inesperado: %+v", dto)
	}
	if _, ok := app.findings[f.Path]; !ok {
		t.Error("finding não foi armazenado")
	}
}

func TestDeleteFinding_UnknownIDReturnsError(t *testing.T) {
	app := newTestApp(&fakeRunner{})
	if err := app.DeleteFinding("nao-existe"); err == nil {
		t.Error("esperava erro para id desconhecido")
	}
}

func TestDeleteFinding_RemovesFromMapOnSuccess(t *testing.T) {
	tmp := t.TempDir()
	runner := &fakeRunner{}
	app := newTestApp(runner)
	app.registerFinding(scanner.Finding{Path: tmp, CategoryName: "x", Description: "d"})

	if err := app.DeleteFinding(tmp); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if _, ok := app.findings[tmp]; ok {
		t.Error("finding deveria ter sido removido após exclusão bem-sucedida")
	}
	if !strings.Contains(runner.script, "DeleteDirectory") {
		t.Errorf("esperava chamada a DeleteDirectory, obtido: %s", runner.script)
	}
}

func TestDeleteFinding_PermanentUsesEmptyRecycleBin(t *testing.T) {
	runner := &fakeRunner{}
	app := newTestApp(runner)
	app.registerFinding(scanner.Finding{Path: "SHELL:RecycleBinFolder", Permanent: true})

	if err := app.DeleteFinding("SHELL:RecycleBinFolder"); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !strings.Contains(runner.script, "SHEmptyRecycleBinW") {
		t.Errorf("esperava chamada a SHEmptyRecycleBinW, obtido: %s", runner.script)
	}
}

func TestDeleteFinding_KeepsEntryOnFailure(t *testing.T) {
	tmp := t.TempDir()
	runner := &fakeRunner{err: errors.New("boom")}
	app := newTestApp(runner)
	app.registerFinding(scanner.Finding{Path: tmp})

	if err := app.DeleteFinding(tmp); err == nil {
		t.Fatal("esperava que o erro se propagasse")
	}
	if _, ok := app.findings[tmp]; !ok {
		t.Error("finding deveria permanecer quando a exclusão falha")
	}
}

func TestListTweaks_ReturnsCatalogWithoutTouchingTheRunner(t *testing.T) {
	runner := &fakeRunner{}
	app := newTestApp(runner)

	dtos := app.ListTweaks()

	if len(dtos) == 0 {
		t.Fatal("esperava ao menos um ajuste no catálogo")
	}
	if runner.script != "" {
		t.Error("ListTweaks não deveria executar nenhum comando na máquina")
	}
}

func TestDetectTweak_ReturnsAppliedState(t *testing.T) {
	runner := &fakeRunner{}
	runner.output = "0"
	app := newTestApp(runner)

	state, err := app.DetectTweak("visual_transparency")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if state != "applied" {
		t.Errorf("state = %q, want applied", state)
	}
}

func TestDetectTweak_UnknownIDReturnsError(t *testing.T) {
	app := newTestApp(&fakeRunner{})
	if _, err := app.DetectTweak("nao-existe"); err == nil {
		t.Error("esperava erro para id desconhecido")
	}
}

func TestApplyTweak_DelegatesToOptimizer(t *testing.T) {
	runner := &fakeRunner{}
	app := newTestApp(runner)

	if err := app.ApplyTweak("power_high_performance"); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !strings.Contains(runner.script, "powercfg") {
		t.Errorf("esperava chamada a powercfg, obtido: %s", runner.script)
	}
}

func TestRevertTweak_DelegatesToOptimizer(t *testing.T) {
	runner := &fakeRunner{}
	app := newTestApp(runner)

	if err := app.RevertTweak("power_high_performance"); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !strings.Contains(runner.script, "powercfg") {
		t.Errorf("esperava chamada a powercfg, obtido: %s", runner.script)
	}
}

type ctxMarkerKey struct{}

func TestStartup_SetsContext(t *testing.T) {
	app := newTestApp(&fakeRunner{})
	ctx := context.WithValue(context.Background(), ctxMarkerKey{}, "x")

	app.startup(ctx)

	if app.ctx != ctx {
		t.Error("startup não armazenou o contexto recebido em a.ctx")
	}
}

// waitForEvent bloqueia até que o evento `name` seja emitido (ou falha o
// teste após um timeout curto), sem depender de sleeps arbitrários.
func waitForEvent(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout esperando o evento")
	}
}

func TestStartScan_EmitsFindingAndDoneEvents(t *testing.T) {
	app := newTestApp(&fakeRunner{})

	var mu sync.Mutex
	var events []string
	done := make(chan struct{})

	app.emit = func(ctx context.Context, eventName string, data ...interface{}) {
		mu.Lock()
		events = append(events, eventName)
		mu.Unlock()
		if eventName == "scan:done" {
			close(done)
		}
	}
	app.runScan = func(ctx context.Context, onFinding func(scanner.Finding)) ([]scanner.Finding, error) {
		f := scanner.Finding{CategoryID: "cat", Path: `C:\fake`, SizeBytes: 10, FileCount: 1}
		onFinding(f)
		return []scanner.Finding{f}, nil
	}

	app.StartScan()
	waitForEvent(t, done)

	mu.Lock()
	defer mu.Unlock()
	if len(events) != 2 || events[0] != "scan:finding" || events[1] != "scan:done" {
		t.Errorf("eventos inesperados: %v", events)
	}
	if _, ok := app.findings[`C:\fake`]; !ok {
		t.Error("finding emitido deveria ter sido registrado em a.findings")
	}
	if app.scanning {
		t.Error("scanning deveria voltar a false após StartScan terminar")
	}
}

func TestStartScan_IgnoresConcurrentCalls(t *testing.T) {
	app := newTestApp(&fakeRunner{})

	var mu sync.Mutex
	callCount := 0
	started := make(chan struct{})
	proceed := make(chan struct{})
	done := make(chan struct{})

	app.runScan = func(ctx context.Context, onFinding func(scanner.Finding)) ([]scanner.Finding, error) {
		mu.Lock()
		callCount++
		mu.Unlock()
		close(started)
		<-proceed
		return nil, nil
	}
	app.emit = func(ctx context.Context, eventName string, data ...interface{}) {
		if eventName == "scan:done" {
			close(done)
		}
	}

	app.StartScan()
	waitForEvent(t, started)
	app.StartScan() // deve ser ignorado: já há uma varredura em andamento
	close(proceed)
	waitForEvent(t, done)

	mu.Lock()
	defer mu.Unlock()
	if callCount != 1 {
		t.Errorf("runScan foi chamado %d vezes, esperava 1", callCount)
	}
}

func TestTweakStateString(t *testing.T) {
	tests := []struct {
		name string
		in   optimizer.State
		want string
	}{
		{"applied", optimizer.StateApplied, "applied"},
		{"not applied", optimizer.StateNotApplied, "not_applied"},
		{"unknown", optimizer.StateUnknown, "unknown"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tweakStateString(tc.in); got != tc.want {
				t.Errorf("tweakStateString(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
