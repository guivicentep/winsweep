package scanner

import (
	"os"
	"path/filepath"
	"strings"
)

// recycleBinSentinel é o valor especial usado nas regras para representar a
// Lixeira do Windows, que não é um caminho de sistema de arquivos comum.
const recycleBinSentinel = "SHELL:RecycleBinFolder"

// ResolvePattern expande variáveis de ambiente do Windows (%TEMP%, etc.) e um
// único "*" de coringa de diretório em um padrão de caminho, retornando todos
// os caminhos existentes que correspondem. Padrões sem coringa retornam no
// máximo um caminho (se existir).
func ResolvePattern(pattern string) ([]string, error) {
	if pattern == recycleBinSentinel {
		return []string{recycleBinSentinel}, nil
	}

	expanded := expandWindowsEnv(pattern)

	if !strings.Contains(expanded, "*") {
		if info, err := os.Stat(expanded); err == nil && info != nil {
			return []string{expanded}, nil
		}
		return nil, nil
	}

	matches, err := filepath.Glob(expanded)
	if err != nil {
		return nil, err
	}
	return matches, nil
}

// expandWindowsEnv substitui ocorrências de %NOME% pelo valor da variável de
// ambiente correspondente. Variáveis desconhecidas são deixadas como estão.
func expandWindowsEnv(s string) string {
	var b strings.Builder
	for {
		start := strings.IndexByte(s, '%')
		if start == -1 {
			b.WriteString(s)
			break
		}
		end := strings.IndexByte(s[start+1:], '%')
		if end == -1 {
			b.WriteString(s)
			break
		}
		end += start + 1
		name := s[start+1 : end]
		if val, ok := os.LookupEnv(name); ok {
			b.WriteString(s[:start])
			b.WriteString(val)
		} else {
			b.WriteString(s[:end+1])
		}
		s = s[end+1:]
	}
	return b.String()
}
