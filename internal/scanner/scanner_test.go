package scanner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"winsweep/internal/rules"
)

func writeFile(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
}

func TestScan_ComputesSizeAndCountForExistingCategory(t *testing.T) {
	tmp := t.TempDir()
	writeFile(t, filepath.Join(tmp, "a.tmp"), 100)
	writeFile(t, filepath.Join(tmp, "sub", "b.tmp"), 250)

	categories := []rules.Category{
		{ID: "test_cat", Name: "Categoria de teste", Description: "usada apenas em teste automatizado",
			PathPatterns: []string{tmp}},
	}

	findings, err := Scan(context.Background(), categories, nil, 2, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(findings), findings)
	}

	f := findings[0]
	if f.SizeBytes != 350 {
		t.Errorf("SizeBytes = %d, want 350", f.SizeBytes)
	}
	if f.FileCount != 2 {
		t.Errorf("FileCount = %d, want 2", f.FileCount)
	}
	if f.CategoryID != "test_cat" {
		t.Errorf("CategoryID = %q, want test_cat", f.CategoryID)
	}
}

func TestScan_SkipsCategoriesWhosePathDoesNotExist(t *testing.T) {
	categories := []rules.Category{
		{ID: "missing", Name: "Não existe", Description: "caminho inexistente usado em teste",
			PathPatterns: []string{filepath.Join(t.TempDir(), "nope")}},
	}

	findings, err := Scan(context.Background(), categories, nil, 2, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected no findings for a missing path, got %+v", findings)
	}
}

func TestScan_SkipsEmptyDirectories(t *testing.T) {
	tmp := t.TempDir()
	categories := []rules.Category{
		{ID: "empty", Name: "Vazia", Description: "pasta vazia usada em teste automatizado",
			PathPatterns: []string{tmp}},
	}

	findings, err := Scan(context.Background(), categories, nil, 2, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected no findings for an empty dir, got %+v", findings)
	}
}

func TestScan_PermanentCategoryUsesProbe(t *testing.T) {
	categories := []rules.Category{
		{ID: "recycle_bin", Name: "Lixeira", Description: "categoria permanente usada em teste",
			PathPatterns: []string{"SHELL:RecycleBinFolder"}, Permanent: true},
	}

	probe := func() (int64, int, error) { return 4096, 3, nil }

	findings, err := Scan(context.Background(), categories, probe, 2, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].SizeBytes != 4096 || findings[0].FileCount != 3 || !findings[0].Permanent {
		t.Errorf("unexpected finding: %+v", findings[0])
	}
}

func TestScan_PermanentCategoryWithoutProbeIsSkipped(t *testing.T) {
	categories := []rules.Category{
		{ID: "recycle_bin", Name: "Lixeira", Description: "categoria permanente usada em teste",
			PathPatterns: []string{"SHELL:RecycleBinFolder"}, Permanent: true},
	}

	findings, err := Scan(context.Background(), categories, nil, 2, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected no findings without a probe, got %+v", findings)
	}
}

func TestScan_PermanentCategoryProbeErrorIsSkipped(t *testing.T) {
	categories := []rules.Category{
		{ID: "recycle_bin", Name: "Lixeira", Description: "categoria permanente usada em teste",
			PathPatterns: []string{"SHELL:RecycleBinFolder"}, Permanent: true},
	}

	probe := func() (int64, int, error) { return 0, 0, errors.New("boom") }

	findings, err := Scan(context.Background(), categories, probe, 2, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected no findings when probe fails, got %+v", findings)
	}
}

func TestScan_CallsOnFindingForEachResult(t *testing.T) {
	tmp1 := t.TempDir()
	tmp2 := t.TempDir()
	writeFile(t, filepath.Join(tmp1, "a.tmp"), 10)
	writeFile(t, filepath.Join(tmp2, "b.tmp"), 20)

	categories := []rules.Category{
		{ID: "cat1", Name: "Cat 1", Description: "primeira categoria usada em teste", PathPatterns: []string{tmp1}},
		{ID: "cat2", Name: "Cat 2", Description: "segunda categoria usada em teste", PathPatterns: []string{tmp2}},
	}

	var mu sync.Mutex
	var seen []string
	_, err := Scan(context.Background(), categories, nil, 2, func(f Finding) {
		mu.Lock()
		seen = append(seen, f.CategoryID)
		mu.Unlock()
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(seen) != 2 {
		t.Errorf("expected onFinding called twice, got %d: %v", len(seen), seen)
	}
}

func TestDirSize_NonExistentPathReturnsZero(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")

	size, count, err := dirSize(context.Background(), missing)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if size != 0 || count != 0 {
		t.Errorf("dirSize(caminho inexistente) = (%d, %d), want (0, 0)", size, count)
	}
}

func TestDirSize_CountsNestedFiles(t *testing.T) {
	tmp := t.TempDir()
	writeFile(t, filepath.Join(tmp, "a.tmp"), 5)
	writeFile(t, filepath.Join(tmp, "sub", "deep", "b.tmp"), 7)

	size, count, err := dirSize(context.Background(), tmp)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if size != 12 || count != 2 {
		t.Errorf("dirSize() = (%d, %d), want (12, 2)", size, count)
	}
}

func TestScan_SkipsCategoryWhenPathPatternIsInvalid(t *testing.T) {
	categories := []rules.Category{
		{ID: "bad_pattern", Name: "Padrão inválido", Description: "categoria usada em teste automatizado",
			PathPatterns: []string{`C:\temp\[*`}},
	}

	findings, err := Scan(context.Background(), categories, nil, 2, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("esperava nenhum finding para um padrão de caminho inválido, obtido: %+v", findings)
	}
}

func TestScan_PermanentCategoryWithZeroCountIsSkipped(t *testing.T) {
	categories := []rules.Category{
		{ID: "recycle_bin", Name: "Lixeira", Description: "categoria permanente usada em teste",
			PathPatterns: []string{"SHELL:RecycleBinFolder"}, Permanent: true},
	}

	// Tamanho > 0 mas contagem 0 não deveria acontecer na prática, mas o
	// código trata como "nada para mostrar" mesmo assim.
	probe := func() (int64, int, error) { return 1024, 0, nil }

	findings, err := Scan(context.Background(), categories, probe, 2, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("esperava nenhum finding quando a contagem é zero, obtido: %+v", findings)
	}
}

func TestScan_RespectsCancelledContext(t *testing.T) {
	tmp := t.TempDir()
	writeFile(t, filepath.Join(tmp, "a.tmp"), 10)

	categories := []rules.Category{
		{ID: "cat", Name: "Cat", Description: "categoria usada em teste de cancelamento", PathPatterns: []string{tmp}},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := Scan(ctx, categories, nil, 2, nil)
	if err == nil {
		t.Error("expected context cancellation error, got nil")
	}
}
