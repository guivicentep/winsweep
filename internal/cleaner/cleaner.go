// Package cleaner remove, de forma segura, os locais encontrados pelo
// scanner. Por padrão os itens são enviados para a Lixeira do Windows (ação
// reversível); apenas esvaziar a própria Lixeira é uma exclusão definitiva.
package cleaner

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Runner executa um script PowerShell e retorna sua saída padrão. Existe
// como interface para permitir testar a lógica do Cleaner sem precisar
// abrir um processo powershell.exe de verdade a cada teste.
type Runner interface {
	RunPowerShell(script string) (string, error)
}

// Cleaner realiza as operações de limpeza sobre os locais identificados
// pelo scanner.
type Cleaner struct {
	runner Runner
}

// New cria um Cleaner que executa comandos através do Runner informado.
func New(runner Runner) *Cleaner {
	return &Cleaner{runner: runner}
}

// SendToRecycleBin move um arquivo ou pasta para a Lixeira do Windows, uma
// ação reversível pelo próprio usuário.
func (c *Cleaner) SendToRecycleBin(path string) error {
	// os.Lstat (em vez de os.Stat) para NÃO seguir links simbólicos/junctions:
	// se o caminho em si foi trocado por um redirecionamento entre a
	// varredura e a exclusão (ou plantado por algo malicioso), recusamos em
	// vez de arriscar agir sobre um alvo diferente do que foi mostrado ao
	// usuário.
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("não foi possível acessar %q: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%q é um link simbólico ou junction; por segurança, não será excluído automaticamente", path)
	}

	method := "DeleteFile"
	if info.IsDir() {
		method = "DeleteDirectory"
	}

	script := fmt.Sprintf(
		"Add-Type -AssemblyName Microsoft.VisualBasic; "+
			"[Microsoft.VisualBasic.FileIO.FileSystem]::%s(%s, 'OnlyErrorDialogs', 'SendToRecycleBin')",
		method, psQuote(path),
	)

	if _, err := c.runner.RunPowerShell(script); err != nil {
		return fmt.Errorf("falha ao mover %q para a lixeira: %w", path, err)
	}
	return nil
}

// emptyRecycleBinScript chama diretamente a API nativa do Windows
// (shell32.dll!SHEmptyRecycleBinW) — a mesma usada pelo próprio Explorer no
// menu "Esvaziar Lixeira". O cmdlet `Clear-RecycleBin` do PowerShell é
// conhecido por falhar de forma intermitente com "O sistema não pode
// encontrar o arquivo especificado" (Win32Exception) mesmo em condições
// normais; a API nativa não tem esse problema.
//
// HRESULT 0x80070002 (ERROR_FILE_NOT_FOUND) é tolerado como sucesso: é o
// que a própria API retorna quando a Lixeira já está vazia.
const emptyRecycleBinScript = `
$sig = '[DllImport("shell32.dll", CharSet = CharSet.Unicode)] public static extern int SHEmptyRecycleBinW(IntPtr hwnd, string pszRootPath, uint dwFlags);'
Add-Type -MemberDefinition $sig -Name Shell32 -Namespace WinSweep -ErrorAction Stop
$SHERB_NOCONFIRMATION = 0x00000001
$SHERB_NOPROGRESSUI = 0x00000002
$SHERB_NOSOUND = 0x00000004
$flags = $SHERB_NOCONFIRMATION -bor $SHERB_NOPROGRESSUI -bor $SHERB_NOSOUND
$hr = [WinSweep.Shell32]::SHEmptyRecycleBinW([IntPtr]::Zero, $null, $flags)
if ($hr -ne 0 -and $hr -ne -2147024894) {
    throw "SHEmptyRecycleBinW falhou com codigo $hr"
}
`

// EmptyRecycleBin remove definitivamente todo o conteúdo da Lixeira do
// Windows. Diferente de SendToRecycleBin, esta ação não pode ser desfeita.
func (c *Cleaner) EmptyRecycleBin() error {
	if _, err := c.runner.RunPowerShell(emptyRecycleBinScript); err != nil {
		return fmt.Errorf("falha ao esvaziar a lixeira: %w", err)
	}
	return nil
}

// RecycleBinInfo retorna o tamanho total em bytes e a quantidade de itens
// atualmente na Lixeira do Windows.
func (c *Cleaner) RecycleBinInfo() (sizeBytes int64, itemCount int, err error) {
	const script = `
$shell = New-Object -ComObject Shell.Application
$bin = $shell.Namespace(10)
$items = $bin.Items()
$size = 0
foreach ($item in $items) { $size += [int64]$item.ExtendedProperty('System.Size') }
"$($items.Count)|$size"
`
	out, err := c.runner.RunPowerShell(script)
	if err != nil {
		return 0, 0, fmt.Errorf("falha ao consultar a lixeira: %w", err)
	}
	return parseRecycleBinInfo(out)
}

func parseRecycleBinInfo(out string) (int64, int, error) {
	parts := strings.SplitN(strings.TrimSpace(out), "|", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("saída inesperada ao consultar a lixeira: %q", out)
	}
	count, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, fmt.Errorf("quantidade de itens inválida: %w", err)
	}
	size, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("tamanho inválido: %w", err)
	}
	return size, count, nil
}

// psQuote escapa uma string para uso seguro dentro de uma string literal
// entre aspas simples do PowerShell (evitando injeção de comandos mesmo
// com caminhos contendo aspas).
func psQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
