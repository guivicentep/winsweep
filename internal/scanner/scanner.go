// Package scanner varre, de forma limitada e concorrente, os locais
// definidos pelo pacote rules, calculando o espaço ocupado por cada um sem
// travar a máquina do usuário.
package scanner

import (
	"context"
	"io/fs"
	"path/filepath"
	"sync"

	"winsweep/internal/rules"
)

// Finding é um local encontrado durante a varredura, pronto para ser exibido
// ao usuário e, mediante confirmação, removido.
type Finding struct {
	CategoryID   string
	CategoryName string
	Description  string
	Path         string
	SizeBytes    int64
	FileCount    int
	Permanent    bool
}

// RecycleBinProbe calcula o tamanho e a quantidade de itens atualmente na
// Lixeira do Windows. É injetado pelo chamador porque depende de uma chamada
// de sistema específica do Windows (mantendo este pacote livre disso).
type RecycleBinProbe func() (sizeBytes int64, itemCount int, err error)

// DefaultConcurrency é o número de varreduras de diretório feitas em
// paralelo. Mantido baixo de propósito para não competir por I/O de disco
// com o restante do sistema.
const DefaultConcurrency = 4

// Scan percorre todas as categorias curadas e retorna um Finding para cada
// local existente na máquina. onFinding, se não nil, é chamado (de forma
// serializada) assim que cada resultado fica pronto, permitindo reportar
// progresso incremental para a interface.
func Scan(ctx context.Context, categories []rules.Category, probeRecycleBin RecycleBinProbe, concurrency int, onFinding func(Finding)) ([]Finding, error) {
	if concurrency <= 0 {
		concurrency = DefaultConcurrency
	}

	type job struct {
		cat  rules.Category
		path string
	}

	var jobs []job
	for _, cat := range categories {
		if cat.Permanent {
			jobs = append(jobs, job{cat: cat, path: ""})
			continue
		}
		for _, pattern := range cat.PathPatterns {
			paths, err := ResolvePattern(pattern)
			if err != nil {
				continue
			}
			for _, p := range paths {
				jobs = append(jobs, job{cat: cat, path: p})
			}
		}
	}

	results := make([]Finding, 0, len(jobs))
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)

	for _, j := range jobs {
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(j job) {
			defer wg.Done()
			defer func() { <-sem }()

			var finding Finding
			if j.cat.Permanent {
				if probeRecycleBin == nil {
					return
				}
				size, count, err := probeRecycleBin()
				if err != nil || count == 0 {
					return
				}
				finding = Finding{
					CategoryID:   j.cat.ID,
					CategoryName: j.cat.Name,
					Description:  j.cat.Description,
					Path:         "SHELL:RecycleBinFolder",
					SizeBytes:    size,
					FileCount:    count,
					Permanent:    true,
				}
			} else {
				size, count, err := dirSize(ctx, j.path)
				if err != nil || (size == 0 && count == 0) {
					return
				}
				finding = Finding{
					CategoryID:   j.cat.ID,
					CategoryName: j.cat.Name,
					Description:  j.cat.Description,
					Path:         j.path,
					SizeBytes:    size,
					FileCount:    count,
				}
			}

			mu.Lock()
			results = append(results, finding)
			mu.Unlock()
			if onFinding != nil {
				onFinding(finding)
			}
		}(j)
	}

	wg.Wait()
	return results, ctx.Err()
}

// dirSize soma o tamanho de todos os arquivos regulares dentro de path.
// Erros de acesso a itens individuais (ex.: permissão negada) são ignorados
// para que um único arquivo protegido não impeça o restante da varredura.
//
// Links simbólicos e junctions NUNCA são seguidos: um item malicioso
// plantado dentro de um local varrido (ex.: %TEMP%) apontando para fora do
// escopo pretendido não deve influenciar o tamanho reportado nem levar a
// varredura para fora do local que o usuário pediu para analisar.
func dirSize(ctx context.Context, path string) (int64, int, error) {
	var size int64
	var count int

	err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			return nil // ignora entradas inacessíveis, segue a varredura
		}
		if d.Type()&fs.ModeSymlink != 0 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		size += info.Size()
		count++
		return nil
	})

	return size, count, err
}
