// Package winexec fornece a forma padrão do winsweep de executar comandos
// do sistema Windows (hoje, via PowerShell), compartilhada entre os módulos
// de limpeza e de otimização.
package winexec

import (
	"fmt"
	"os/exec"
)

// PowerShellRunner invoca o powershell.exe instalado no Windows para
// executar um script e devolver sua saída.
type PowerShellRunner struct{}

// RunPowerShell executa o script em um processo powershell.exe não
// interativo e retorna sua saída combinada (stdout+stderr).
func (PowerShellRunner) RunPowerShell(script string) (string, error) {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("%w: %s", err, string(out))
	}
	return string(out), nil
}
