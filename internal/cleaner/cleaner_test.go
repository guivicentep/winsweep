package cleaner

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeRunner grava o último script recebido e devolve uma resposta
// pré-programada, permitindo testar a lógica do Cleaner sem abrir um
// processo powershell.exe real.
type fakeRunner struct {
	lastScript string
	output     string
	err        error
}

func (f *fakeRunner) RunPowerShell(script string) (string, error) {
	f.lastScript = script
	return f.output, f.err
}

func TestSendToRecycleBin_UsesDeleteFileForFiles(t *testing.T) {
	tmp := t.TempDir()
	file := filepath.Join(tmp, "a.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	runner := &fakeRunner{}
	c := New(runner)

	if err := c.SendToRecycleBin(file); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(runner.lastScript, "DeleteFile") {
		t.Errorf("expected script to call DeleteFile, got: %s", runner.lastScript)
	}
	if strings.Contains(runner.lastScript, "DeleteDirectory") {
		t.Errorf("did not expect DeleteDirectory in script for a file: %s", runner.lastScript)
	}
}

func TestSendToRecycleBin_UsesDeleteDirectoryForDirs(t *testing.T) {
	tmp := t.TempDir()

	runner := &fakeRunner{}
	c := New(runner)

	if err := c.SendToRecycleBin(tmp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(runner.lastScript, "DeleteDirectory") {
		t.Errorf("expected script to call DeleteDirectory, got: %s", runner.lastScript)
	}
}

func TestSendToRecycleBin_EscapesSingleQuotesInPath(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "it's a folder")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	runner := &fakeRunner{}
	c := New(runner)

	if err := c.SendToRecycleBin(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(runner.lastScript, "it''s a folder") {
		t.Errorf("expected escaped single quote in script, got: %s", runner.lastScript)
	}
}

func TestSendToRecycleBin_AdversarialCharactersStayLiteral(t *testing.T) {
	// Nenhum desses caracteres precisa ser tratado como especial dentro de
	// uma string de aspas simples do PowerShell — o teste prova que eles
	// chegam ao script como texto literal, nunca como comando executável.
	// Só usamos caracteres que o próprio NTFS aceita em nomes de pasta
	// (< > : " / \ | ? * são proibidos pelo Windows, então nem entram aqui).
	dangerous := []string{
		"$(calc)",
		"`whoami`",
		"a & echo pwned",
		"a; echo pwned",
		"a) } # comentario",
		"50% off",
	}

	for _, name := range dangerous {
		t.Run(name, func(t *testing.T) {
			tmp := t.TempDir()
			dir := filepath.Join(tmp, name)
			if err := os.Mkdir(dir, 0o755); err != nil {
				t.Skipf("nome de pasta não suportado neste sistema de arquivos: %v", err)
			}

			runner := &fakeRunner{}
			c := New(runner)

			if err := c.SendToRecycleBin(dir); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			wantQuoted := "'" + strings.ReplaceAll(dir, "'", "''") + "'"
			if !strings.Contains(runner.lastScript, wantQuoted) {
				t.Errorf("esperava %q embutido como texto literal, script gerado: %s", wantQuoted, runner.lastScript)
			}
		})
	}
}

func TestSendToRecycleBin_RefusesSymlinks(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	link := filepath.Join(tmp, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("sistema não permite criar link simbólico sem privilégio (Modo de Desenvolvedor desativado?): %v", err)
	}

	runner := &fakeRunner{}
	c := New(runner)

	err := c.SendToRecycleBin(link)
	if err == nil {
		t.Fatal("esperava erro ao tentar excluir um link simbólico ou junction")
	}
	if runner.lastScript != "" {
		t.Error("não deveria ter chamado o PowerShell para um link simbólico")
	}
}

func TestSendToRecycleBin_NonExistentPathFailsFast(t *testing.T) {
	runner := &fakeRunner{}
	c := New(runner)

	err := c.SendToRecycleBin(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("expected error for non-existent path")
	}
	if runner.lastScript != "" {
		t.Error("runner should not be invoked when the path does not exist")
	}
}

func TestSendToRecycleBin_PropagatesRunnerError(t *testing.T) {
	tmp := t.TempDir()
	runner := &fakeRunner{err: errors.New("access denied")}
	c := New(runner)

	err := c.SendToRecycleBin(tmp)
	if err == nil {
		t.Fatal("expected error to propagate from runner")
	}
}

func TestEmptyRecycleBin_CallsSHEmptyRecycleBinW(t *testing.T) {
	runner := &fakeRunner{}
	c := New(runner)

	if err := c.EmptyRecycleBin(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(runner.lastScript, "SHEmptyRecycleBinW") {
		t.Errorf("expected script to call SHEmptyRecycleBinW, got: %s", runner.lastScript)
	}
}

func TestEmptyRecycleBin_PropagatesRunnerError(t *testing.T) {
	runner := &fakeRunner{err: errors.New("boom")}
	c := New(runner)

	if err := c.EmptyRecycleBin(); err == nil {
		t.Fatal("expected error to propagate from runner")
	}
}

func TestRecycleBinInfo_ParsesCountAndSize(t *testing.T) {
	runner := &fakeRunner{output: "3|4096\n"}
	c := New(runner)

	size, count, err := c.RecycleBinInfo()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if size != 4096 || count != 3 {
		t.Errorf("got size=%d count=%d, want size=4096 count=3", size, count)
	}
}

func TestRecycleBinInfo_PropagatesRunnerError(t *testing.T) {
	runner := &fakeRunner{err: errors.New("no com")}
	c := New(runner)

	if _, _, err := c.RecycleBinInfo(); err == nil {
		t.Fatal("expected error to propagate from runner")
	}
}

func TestParseRecycleBinInfo(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantSize  int64
		wantCount int
		wantErr   bool
	}{
		{"valid", "5|1024", 1024, 5, false},
		{"valid with whitespace/newline", "  2 | 512 \n", 512, 2, false},
		{"missing separator", "no-separator-here", 0, 0, true},
		{"non-numeric count", "abc|1024", 0, 0, true},
		{"non-numeric size", "2|abc", 0, 0, true},
		{"empty", "", 0, 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			size, count, err := parseRecycleBinInfo(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if size != tc.wantSize || count != tc.wantCount {
				t.Errorf("got size=%d count=%d, want size=%d count=%d", size, count, tc.wantSize, tc.wantCount)
			}
		})
	}
}

func TestPsQuote(t *testing.T) {
	tests := []struct{ input, want string }{
		{`C:\Temp`, `'C:\Temp'`},
		{`it's a path`, `'it''s a path'`},
		{`''`, `''''''`},
	}
	for _, tc := range tests {
		if got := psQuote(tc.input); got != tc.want {
			t.Errorf("psQuote(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
