package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandWindowsEnv(t *testing.T) {
	t.Setenv("WINSWEEP_TEST_VAR", `C:\Somewhere`)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no variables", `C:\Windows\Temp`, `C:\Windows\Temp`},
		{"single variable", `%WINSWEEP_TEST_VAR%\Sub`, `C:\Somewhere\Sub`},
		{"unknown variable left untouched", `%DOES_NOT_EXIST_XYZ%\Sub`, `%DOES_NOT_EXIST_XYZ%\Sub`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := expandWindowsEnv(tc.input); got != tc.want {
				t.Errorf("expandWindowsEnv(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestResolvePattern_RecycleBinSentinel(t *testing.T) {
	got, err := ResolvePattern(recycleBinSentinel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0] != recycleBinSentinel {
		t.Errorf("got %v, want [%s]", got, recycleBinSentinel)
	}
}

func TestResolvePattern_NonExistentPathReturnsEmpty(t *testing.T) {
	tmp := t.TempDir()
	missing := filepath.Join(tmp, "does-not-exist")

	got, err := ResolvePattern(missing)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected no matches for missing path, got %v", got)
	}
}

func TestResolvePattern_ExistingPathIsReturned(t *testing.T) {
	tmp := t.TempDir()

	got, err := ResolvePattern(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0] != tmp {
		t.Errorf("got %v, want [%s]", got, tmp)
	}
}

func TestResolvePattern_WildcardExpandsToExistingDirs(t *testing.T) {
	tmp := t.TempDir()
	for _, name := range []string{"profileA", "profileB"} {
		if err := os.MkdirAll(filepath.Join(tmp, name, "cache2"), 0o755); err != nil {
			t.Fatalf("setup failed: %v", err)
		}
	}
	// A file (not a dir) sibling that must not match the "*/cache2" pattern.
	if err := os.WriteFile(filepath.Join(tmp, "not-a-profile.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	pattern := filepath.Join(tmp, "*", "cache2")
	got, err := ResolvePattern(pattern)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 matches, got %d: %v", len(got), got)
	}
}

func TestResolvePattern_BadGlobPatternReturnsError(t *testing.T) {
	// Uma classe de caracteres "[" nunca fechada é um padrão de glob
	// inválido (filepath.ErrBadPattern).
	_, err := ResolvePattern(`C:\temp\[*`)
	if err == nil {
		t.Fatal("esperava erro para padrão de glob inválido")
	}
}

func TestExpandWindowsEnv_UnterminatedPercentIsLeftUnchanged(t *testing.T) {
	tests := []string{
		"%TEMP",
		"prefixo %TEMP",
	}
	for _, in := range tests {
		if got := expandWindowsEnv(in); got != in {
			t.Errorf("expandWindowsEnv(%q) = %q, want unchanged", in, got)
		}
	}
}

func TestExpandWindowsEnv_MultipleVariablesInSameString(t *testing.T) {
	t.Setenv("WINSWEEP_TEST_A", "AAA")
	t.Setenv("WINSWEEP_TEST_B", "BBB")

	got := expandWindowsEnv(`%WINSWEEP_TEST_A%\%WINSWEEP_TEST_B%`)
	want := `AAA\BBB`
	if got != want {
		t.Errorf("expandWindowsEnv(...) = %q, want %q", got, want)
	}
}

func TestResolvePattern_EnvVarExpansion(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("WINSWEEP_TEST_ROOT", tmp)

	got, err := ResolvePattern(`%WINSWEEP_TEST_ROOT%`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0] != tmp {
		t.Errorf("got %v, want [%s]", got, tmp)
	}
}
