// Package optimizer aplica, verifica e reverte os ajustes de desempenho
// definidos em internal/tweaks. Todo ajuste é feito por chave de registro
// do usuário atual (HKCU) ou por powercfg, nunca em nível de sistema — e
// todo ajuste tem um Revert correspondente que restaura o valor padrão do
// Windows.
package optimizer

import (
	"errors"
	"fmt"
	"strings"
)

// Runner executa um script PowerShell e retorna sua saída padrão. Existe
// como interface para permitir testar a lógica do Optimizer sem precisar
// abrir um processo powershell.exe (ou tocar no registro) de verdade a
// cada teste.
type Runner interface {
	RunPowerShell(script string) (string, error)
}

// State representa se um ajuste está atualmente aplicado na máquina.
type State int

const (
	StateUnknown State = iota
	StateApplied
	StateNotApplied
)

// ErrUnknownTweak é retornado quando o ID informado não corresponde a
// nenhum ajuste conhecido.
var ErrUnknownTweak = errors.New("ajuste desconhecido")

// Optimizer executa as ações de detecção, aplicação e reversão dos ajustes.
type Optimizer struct {
	runner Runner
}

// New cria um Optimizer que executa comandos através do Runner informado.
func New(runner Runner) *Optimizer {
	return &Optimizer{runner: runner}
}

// action agrupa os scripts PowerShell necessários para detectar, aplicar
// e reverter um único ajuste.
type action struct {
	detectScript string
	applyScript  string
	revertScript string
	parseDetect  func(output string) (State, error)
}

const (
	highPerformanceSchemeGUID = "8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c"
	balancedSchemeGUID        = "381b4222-f694-41f0-9685-ff5bb260df2e"
)

var actions = map[string]action{
	"power_high_performance": {
		detectScript: "powercfg /getactivescheme",
		applyScript:  "powercfg /setactive " + highPerformanceSchemeGUID,
		revertScript: "powercfg /setactive " + balancedSchemeGUID,
		parseDetect: func(out string) (State, error) {
			if strings.Contains(strings.ToLower(out), highPerformanceSchemeGUID) {
				return StateApplied, nil
			}
			return StateNotApplied, nil
		},
	},
	"visual_transparency": registryToggleAction(
		`HKCU:\Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, "EnableTransparency", kindDWord, "0", "1"),
	"visual_window_animations": registryToggleAction(
		`HKCU:\Control Panel\Desktop\WindowMetrics`, "MinAnimate", kindString, "0", "1"),
	"visual_drag_full_windows": registryToggleAction(
		`HKCU:\Control Panel\Desktop`, "DragFullWindows", kindString, "0", "1"),
	"visual_taskbar_animations": registryToggleAction(
		`HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced`, "TaskbarAnimations", kindDWord, "0", "1"),
}

// valueKind é o tipo de valor de registro manipulado por um toggle.
type valueKind string

const (
	kindDWord  valueKind = "DWord"
	kindString valueKind = "String"
)

// registryToggleAction constrói uma action para um ajuste que liga/desliga
// alternando um único valor de registro entre appliedValue (quando o ajuste
// está ativo) e defaultValue (o padrão de fábrica do Windows, usado ao reverter).
func registryToggleAction(path, name string, kind valueKind, appliedValue, defaultValue string) action {
	detect := fmt.Sprintf(
		`(Get-ItemProperty -Path '%s' -Name '%s' -ErrorAction SilentlyContinue).'%s'`,
		path, name, name,
	)
	apply := registrySetScript(path, name, kind, appliedValue)
	revert := registrySetScript(path, name, kind, defaultValue)

	return action{
		detectScript: detect,
		applyScript:  apply,
		revertScript: revert,
		parseDetect: func(out string) (State, error) {
			if strings.TrimSpace(out) == appliedValue {
				return StateApplied, nil
			}
			return StateNotApplied, nil
		},
	}
}

func registrySetScript(path, name string, kind valueKind, value string) string {
	psValue := value
	if kind == kindString {
		psValue = "'" + value + "'"
	}
	return fmt.Sprintf(
		"New-Item -Path '%s' -Force | Out-Null; Set-ItemProperty -Path '%s' -Name '%s' -Value %s -Type %s",
		path, path, name, psValue, kind,
	)
}

// Detect retorna se o ajuste identificado por id está atualmente aplicado
// na máquina.
func (o *Optimizer) Detect(id string) (State, error) {
	a, ok := actions[id]
	if !ok {
		return StateUnknown, fmt.Errorf("%w: %s", ErrUnknownTweak, id)
	}
	out, err := o.runner.RunPowerShell(a.detectScript)
	if err != nil {
		return StateUnknown, fmt.Errorf("falha ao verificar %q: %w", id, err)
	}
	return a.parseDetect(out)
}

// Apply aplica o ajuste identificado por id.
func (o *Optimizer) Apply(id string) error {
	a, ok := actions[id]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownTweak, id)
	}
	if _, err := o.runner.RunPowerShell(a.applyScript); err != nil {
		return fmt.Errorf("falha ao aplicar %q: %w", id, err)
	}
	return nil
}

// Revert desfaz o ajuste identificado por id, restaurando o padrão do Windows.
func (o *Optimizer) Revert(id string) error {
	a, ok := actions[id]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownTweak, id)
	}
	if _, err := o.runner.RunPowerShell(a.revertScript); err != nil {
		return fmt.Errorf("falha ao reverter %q: %w", id, err)
	}
	return nil
}
