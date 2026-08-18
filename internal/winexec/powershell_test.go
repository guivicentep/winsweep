package winexec

import (
	"strings"
	"testing"
)

// Estes testes invocam o powershell.exe de verdade (não há o que mockar
// aqui — este é o único lugar do projeto que faz isso). São inofensivos:
// só imprimem texto ou retornam um código de saída, sem alterar nada no
// sistema.

func TestRunPowerShell_ReturnsStdout(t *testing.T) {
	r := PowerShellRunner{}

	out, err := r.RunPowerShell("Write-Output 'hello-winsweep'")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !strings.Contains(out, "hello-winsweep") {
		t.Errorf("saída não contém o texto esperado: %q", out)
	}
}

func TestRunPowerShell_PropagatesNonZeroExit(t *testing.T) {
	r := PowerShellRunner{}

	_, err := r.RunPowerShell("exit 1")
	if err == nil {
		t.Fatal("esperava erro para um script que termina com exit 1")
	}
}
