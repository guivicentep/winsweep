// Package rules define o catálogo curado de locais conhecidos do Windows que
// costumam acumular arquivos desnecessários (cache, temporários, backups de
// atualização, etc). Cada categoria documenta para que serve o local, para
// que o usuário final saiba exatamente o que está prestes a excluir.
package rules

// Category descreve um local conhecido do sistema que pode ser analisado e,
// mediante confirmação do usuário, limpo.
type Category struct {
	// ID é um identificador estável usado internamente (não muda entre versões).
	ID string
	// Name é o nome amigável exibido na interface.
	Name string
	// Description explica, em linguagem simples, para que serve o conteúdo
	// desse local e por que costuma ser seguro removê-lo.
	Description string
	// PathPatterns são caminhos (podendo conter variáveis de ambiente do
	// Windows como %TEMP% e um único "*" como coringa de diretório, usado
	// por exemplo para percorrer múltiplos perfis de navegador) que serão
	// expandidos para os caminhos reais na máquina do usuário.
	PathPatterns []string
	// Permanent indica que a "exclusão" desta categoria não é um envio de
	// arquivos para a Lixeira, e sim uma ação de limpeza definitiva própria
	// (hoje usada apenas para esvaziar a própria Lixeira do Windows).
	Permanent bool
}

// Builtin retorna o catálogo de categorias curadas e consideradas seguras.
// Cada uma foi escolhida por ser (a) bem conhecida como "lixo" do sistema,
// (b) recriada automaticamente pelo Windows/aplicativos quando necessário, e
// (c) não conter, em condições normais, documentos pessoais do usuário.
func Builtin() []Category {
	return []Category{
		{
			ID:   "temp_user",
			Name: "Temporários do usuário",
			Description: "Arquivos temporários criados por programas e instaladores " +
				"(pasta %TEMP%). São recriados automaticamente quando necessário; " +
				"acumulam-se porque muitos programas não os apagam sozinhos.",
			PathPatterns: []string{`%TEMP%`},
		},
		{
			ID:   "temp_windows",
			Name: "Temporários do Windows",
			Description: "Arquivos temporários internos do próprio sistema operacional " +
				"(C:\\Windows\\Temp), usados durante instalações e atualizações.",
			PathPatterns: []string{`%WINDIR%\Temp`},
		},
		{
			ID:   "windows_old",
			Name: "Windows.old (backup da instalação anterior)",
			Description: "Cópia da instalação anterior do Windows, guardada automaticamente " +
				"após uma grande atualização para permitir reverter o sistema. O Windows " +
				"só permite reverter por cerca de 10 dias após a atualização; depois disso " +
				"esta pasta só ocupa espaço (geralmente muitos GB).",
			PathPatterns: []string{`C:\Windows.old`},
		},
		{
			ID:   "windows_update_cache",
			Name: "Cache de Atualizações do Windows",
			Description: "Instaladores de atualizações do Windows já aplicadas, guardados " +
				"em C:\\Windows\\SoftwareDistribution\\Download. Se algum dia forem " +
				"necessários de novo, o Windows Update baixa tudo novamente.",
			PathPatterns: []string{`%WINDIR%\SoftwareDistribution\Download`},
		},
		{
			ID:   "chrome_cache",
			Name: "Cache do Google Chrome",
			Description: "Arquivos de cache (imagens, scripts, páginas) guardados pelo Chrome " +
				"para acelerar o carregamento de sites. É reconstruído automaticamente; " +
				"não contém favoritos, senhas ou histórico.",
			PathPatterns: []string{`%LOCALAPPDATA%\Google\Chrome\User Data\Default\Cache`},
		},
		{
			ID:   "edge_cache",
			Name: "Cache do Microsoft Edge",
			Description: "Mesma função do cache do Chrome, mas para o navegador Edge. " +
				"É reconstruído automaticamente e não contém favoritos ou senhas.",
			PathPatterns: []string{`%LOCALAPPDATA%\Microsoft\Edge\User Data\Default\Cache`},
		},
		{
			ID:   "firefox_cache",
			Name: "Cache do Firefox",
			Description: "Cache de navegação do Firefox (um por perfil). É reconstruído " +
				"automaticamente e não contém favoritos ou senhas.",
			PathPatterns: []string{`%LOCALAPPDATA%\Mozilla\Firefox\Profiles\*\cache2`},
		},
		{
			ID:   "explorer_thumbnails",
			Name: "Cache de miniaturas do Explorer",
			Description: "Miniaturas (thumbnails) de imagens e vídeos que o Windows Explorer " +
				"guarda para exibir pastas mais rápido. São geradas de novo automaticamente " +
				"na próxima vez que a pasta for aberta.",
			PathPatterns: []string{`%LOCALAPPDATA%\Microsoft\Windows\Explorer`},
		},
		{
			ID:   "windows_inet_cache",
			Name: "Cache de Internet do Windows",
			Description: "Arquivos temporários de internet usados por componentes internos " +
				"do Windows (INetCache). Reconstruído automaticamente conforme necessário.",
			PathPatterns: []string{`%LOCALAPPDATA%\Microsoft\Windows\INetCache`},
		},
		{
			ID:   "crash_dumps",
			Name: "Relatórios de erro e dumps de memória",
			Description: "Arquivos gerados quando programas travam (Windows Error Reporting " +
				"e CrashDumps), usados apenas para diagnóstico técnico. Seguros de remover " +
				"se você não precisa investigar uma falha específica.",
			PathPatterns: []string{
				`%LOCALAPPDATA%\CrashDumps`,
				`%PROGRAMDATA%\Microsoft\Windows\WER`,
			},
		},
		{
			ID:   "recycle_bin",
			Name: "Lixeira do Windows",
			Description: "Itens que você (ou programas) já enviaram para a Lixeira. " +
				"Esvaziar aqui é uma exclusão definitiva e não pode ser desfeita — " +
				"diferente das demais categorias, que só movem itens para a Lixeira.",
			PathPatterns: []string{`SHELL:RecycleBinFolder`},
			Permanent:    true,
		},
	}
}
