package optimizer

import (
	"errors"
	"strings"
	"testing"

	"winsweep/internal/tweaks"
)

// fakeRunner grava o último script recebido e devolve uma resposta
// pré-programada, permitindo testar a lógica do Optimizer sem tocar no
// registro do Windows ou no plano de energia de verdade.
type fakeRunner struct {
	lastScript string
	output     string
	err        error
}

func (f *fakeRunner) RunPowerShell(script string) (string, error) {
	f.lastScript = script
	return f.output, f.err
}

func TestEveryBuiltinTweakHasAnAction(t *testing.T) {
	runner := &fakeRunner{output: ""}
	o := New(runner)

	for _, tw := range tweaks.Builtin() {
		t.Run(tw.ID, func(t *testing.T) {
			if _, err := o.Detect(tw.ID); errors.Is(err, ErrUnknownTweak) {
				t.Errorf("tweak %q do catálogo não tem action registrada no optimizer", tw.ID)
			}
		})
	}
}

func TestDetect_UnknownIDReturnsError(t *testing.T) {
	o := New(&fakeRunner{})
	if _, err := o.Detect("does-not-exist"); !errors.Is(err, ErrUnknownTweak) {
		t.Errorf("esperava ErrUnknownTweak, obtido: %v", err)
	}
}

func TestApply_UnknownIDReturnsError(t *testing.T) {
	o := New(&fakeRunner{})
	if err := o.Apply("does-not-exist"); !errors.Is(err, ErrUnknownTweak) {
		t.Errorf("esperava ErrUnknownTweak, obtido: %v", err)
	}
}

func TestRevert_UnknownIDReturnsError(t *testing.T) {
	o := New(&fakeRunner{})
	if err := o.Revert("does-not-exist"); !errors.Is(err, ErrUnknownTweak) {
		t.Errorf("esperava ErrUnknownTweak, obtido: %v", err)
	}
}

func TestDetect_PropagatesRunnerError(t *testing.T) {
	o := New(&fakeRunner{err: errors.New("boom")})
	if _, err := o.Detect("power_high_performance"); err == nil {
		t.Fatal("esperava erro do runner se propagando")
	}
}

func TestApply_PropagatesRunnerError(t *testing.T) {
	o := New(&fakeRunner{err: errors.New("boom")})
	if err := o.Apply("power_high_performance"); err == nil {
		t.Fatal("esperava erro do runner se propagando")
	}
}

func TestPowerHighPerformance_DetectParsesActiveScheme(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   State
	}{
		{"high performance active", "Esquema de energia ativo: (Alto desempenho) 8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c", StateApplied},
		{"case insensitive guid", "GUID: 8C5E7FDA-E8BF-4A96-9A85-A6E23A8C635C", StateApplied},
		{"balanced active", "Esquema de energia ativo: (Equilibrado) 381b4222-f694-41f0-9685-ff5bb260df2e", StateNotApplied},
		{"empty output", "", StateNotApplied},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o := New(&fakeRunner{output: tc.output})
			got, err := o.Detect("power_high_performance")
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if got != tc.want {
				t.Errorf("Detect() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPowerHighPerformance_ApplyAndRevertUsePowercfg(t *testing.T) {
	runner := &fakeRunner{}
	o := New(runner)

	if err := o.Apply("power_high_performance"); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !strings.Contains(runner.lastScript, "powercfg") || !strings.Contains(runner.lastScript, highPerformanceSchemeGUID) {
		t.Errorf("script de apply inesperado: %s", runner.lastScript)
	}

	if err := o.Revert("power_high_performance"); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !strings.Contains(runner.lastScript, "powercfg") || !strings.Contains(runner.lastScript, balancedSchemeGUID) {
		t.Errorf("script de revert inesperado: %s", runner.lastScript)
	}
}

func TestRegistryToggle_DetectParsesCurrentValue(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   State
	}{
		{"applied", "0", StateApplied},
		{"default windows value", "1", StateNotApplied},
		{"missing registry value", "", StateNotApplied},
		{"value with surrounding whitespace", "  0  \n", StateApplied},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o := New(&fakeRunner{output: tc.output})
			got, err := o.Detect("visual_transparency")
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if got != tc.want {
				t.Errorf("Detect() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRegistryToggle_ApplySetsAppliedValue(t *testing.T) {
	runner := &fakeRunner{}
	o := New(runner)

	if err := o.Apply("visual_transparency"); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !strings.Contains(runner.lastScript, "EnableTransparency") || !strings.Contains(runner.lastScript, "-Value 0") {
		t.Errorf("script de apply inesperado: %s", runner.lastScript)
	}
	if !strings.Contains(runner.lastScript, "-Type DWord") {
		t.Errorf("esperava tipo DWord no script: %s", runner.lastScript)
	}
}

func TestRegistryToggle_RevertRestoresDefaultValue(t *testing.T) {
	runner := &fakeRunner{}
	o := New(runner)

	if err := o.Revert("visual_transparency"); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !strings.Contains(runner.lastScript, "-Value 1") {
		t.Errorf("script de revert inesperado: %s", runner.lastScript)
	}
}

func TestRegistryToggle_StringKindIsQuoted(t *testing.T) {
	runner := &fakeRunner{}
	o := New(runner)

	if err := o.Apply("visual_window_animations"); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !strings.Contains(runner.lastScript, "-Value '0'") {
		t.Errorf("esperava valor de string entre aspas simples, obtido: %s", runner.lastScript)
	}
	if !strings.Contains(runner.lastScript, "-Type String") {
		t.Errorf("esperava tipo String no script: %s", runner.lastScript)
	}
}

func TestRegistryToggle_RevertPropagatesRunnerError(t *testing.T) {
	o := New(&fakeRunner{err: errors.New("access denied")})
	if err := o.Revert("visual_taskbar_animations"); err == nil {
		t.Fatal("esperava erro do runner se propagando")
	}
}
