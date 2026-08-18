# winsweep — notas técnicas para sessões futuras

Este arquivo existe para economizar tokens/tempo em sessões futuras: registra
decisões já tomadas (e por quê), para não precisar re-analisar do zero.

## O que é

App desktop Windows com duas funções independentes:

1. **Limpeza** — varre locais conhecidos de arquivos desnecessários do
   Windows (cache de navegadores, temporários, Lixeira, etc.) e exclui só o
   que o usuário confirmar, item a item.
2. **Otimização** — ajustes de desempenho reversíveis (plano de energia,
   efeitos visuais) para computadores lentos/antigos.

## Stack: Go + Wails (decisão tomada, não reabrir sem novo motivo)

Foi avaliado migrar para **Electron** e a decisão foi **não migrar**.
Motivos (registrados para não re-analisar):

- O Wails usa o **WebView2** no Windows, que já é Chromium por baixo dos
  panos — mesmo motor de renderização do Electron. Não há ganho visual/UX
  possível no Electron que não seja possível hoje no Wails.
- Electron embute Chromium+Node inteiros (instalador 150-200MB+, RAM em
  repouso 100-200MB+). O Wails usa o WebView2 já presente no Windows
  (instalador de poucos MB, RAM bem menor). Isso importa especialmente aqui:
  o próprio app tem um módulo de "otimizar PCs lentos", seria contraditório
  ele mesmo ser pesado.
- Backend (scanner, regras, exclusão segura, ajustes do registro) está todo
  em Go, testado. Migrar para Electron significaria reescrever em
  Node/TypeScript ou manter dois processos com IPC — sem ganho correspondente.
- `.msi` é possível nos dois casos (ambos usam WiX por baixo) — não é
  diferencial do Electron.

Se isso for reconsiderado no futuro, é porque surgiu um motivo novo e
específico — não repetir a análise acima, ela já foi feita.

## Arquitetura

Cada domínio segue o mesmo padrão de duas camadas, pensado para manter
100% (ou perto disso) de cobertura de teste sem tocar no sistema real:

- **Catálogo (dados puros, sem I/O)**: `internal/rules` (limpeza) e
  `internal/tweaks` (otimização). Cada entrada tem ID estável, nome,
  descrição em linguagem simples do que é/faz, e metadados de UI.
- **Execução (I/O real, via interface `Runner` injetável)**:
  `internal/scanner` + `internal/cleaner` (limpeza) e `internal/optimizer`
  (otimização). Testados com um `Runner`/fake que grava o script recebido
  e devolve saída pré-programada — nunca abrem processo real nos testes.
- `internal/winexec` — único ponto que de fato chama `powershell.exe`
  (`PowerShellRunner`). Compartilhado pelos dois domínios via *structural
  typing* do Go (cada pacote define sua própria interface `Runner` local;
  `winexec.PowerShellRunner{}` satisfaz as duas sem acoplamento entre elas).
- `app.go` — bindings do Wails, faz a ponte entre os pacotes `internal/*` e
  o frontend. Cada FindingDTO/TweakDTO tem `json` tags casando com o que o
  JS espera.

### Frontend

Vanilla JS + Vite (sem framework, sem Tailwind — decisão consciente para
não trazer dependências desnecessárias num app pequeno). Estrutura:

- `frontend/src/style.css` — design tokens (cor, espaçamento em escala de
  4px, tipografia, sombra, motion) em `:root`.
- `frontend/src/app.css` — componentes (shell/sidebar, cards, badges,
  pills de status, botões, empty states).
- `frontend/src/icons.js` — ícones inline SVG (sem CDN/fontes externas —
  o app não faz nenhuma requisição de rede para renderizar a UI).
- `frontend/src/format.js` — `formatBytes`, `escapeHtml`.
- `frontend/src/views/cleanup.js` — módulo de Limpeza (escuta eventos
  `scan:finding`/`scan:done` via `EventsOn`, exportado como
  `mountCleanupView(container) -> { unmount() }`).
- `frontend/src/views/optimize.js` — módulo de Otimização (só
  request/response, sem eventos). **Nenhuma chamada ao backend acontece
  automaticamente**: `ListTweaks()` no mount só devolve o catálogo estático
  (sem tocar no SO); `DetectTweak`/`ApplyTweak`/`RevertTweak` só disparam
  com clique explícito do usuário.
- `frontend/src/main.js` — shell (sidebar com navegação entre módulos,
  monta/desmonta a view ativa).

## Padrões de segurança (não afrouxar sem discutir com o usuário)

- **Limpeza**: todo item vai para a **Lixeira do Windows**
  (`Microsoft.VisualBasic.FileIO.FileSystem`, opção `SendToRecycleBin`),
  nunca exclusão permanente direta. A única exceção é a categoria
  `recycle_bin` (esvaziar a própria Lixeira), sinalizada com `Permanent:
  true` e destacada em vermelho/aviso na UI.
- **Otimização**: todo ajuste é **HKCU** (usuário atual, nunca HKLM/sistema)
  ou `powercfg` (troca de plano, não cria/deleta planos). Todo `Apply` tem
  um `Revert` correspondente que restaura o valor padrão de fábrica do
  Windows. Deliberadamente **fora de escopo**: programas de inicialização,
  serviços do Windows, qualquer coisa exigindo HKLM — risco de deixar a
  máquina em estado ruim é maior que o ganho, revisar com o usuário antes
  de expandir por aí.
- O módulo de Otimização **nunca roda `Detect` automaticamente** (nem ao
  montar a tela) — só em resposta a clique explícito. Isso foi um pedido
  direto do usuário (não testar nada na máquina dele sem ele mandar) e virou
  também a política geral do módulo.
- Regra geral ao adicionar categorias/ajustes novos: só entram no catálogo
  locais/valores **bem conhecidos, documentados e reversíveis** — nunca
  heurística sobre arquivos pessoais do usuário.

## Bugs já encontrados e corrigidos (não reintroduzir)

- **`Clear-RecycleBin` (cmdlet do PowerShell) é instável** — falha
  intermitentemente com `Win32Exception: O sistema não pode encontrar o
  arquivo especificado`, mesmo em condições normais (reportado pelo usuário
  em uso real, não é só um caso de borda). Trocado em
  `internal/cleaner/cleaner.go` (`EmptyRecycleBin`) para chamar a API nativa
  `shell32.dll!SHEmptyRecycleBinW` via P/Invoke (`Add-Type` com
  `DllImport`) — é a mesma API que o Explorer usa no menu "Esvaziar
  Lixeira", bem mais confiável. HRESULT `0x80070002`
  (`ERROR_FILE_NOT_FOUND`, decimal `-2147024894`) é tratado como sucesso —
  é o que a API retorna quando a Lixeira já está vazia.
  **Não usar `Clear-RecycleBin` de novo neste projeto.**
- Essa mudança **não pôde ser testada de verdade nesta sessão**: esvaziar a
  Lixeira é uma ação permanente, e ações desse tipo (`emptying trash`) são
  proibidas para o Claude executar por conta própria, mesmo a pedido do
  usuário — só a lógica foi validada (testes com runner mockado + a
  assinatura do P/Invoke foi compilada isoladamente com `Add-Type` para
  garantir que não tem erro de sintaxe). Se aparecer um novo bug aqui, só
  vai ser possível confirmar com o usuário testando o botão "Esvaziar
  definitivamente" de verdade e reportando o resultado.

## Testes

`go test ./... -coverprofile=coverage && go tool cover -func=coverage`.
96% do total de statements na última verificação. Por pacote: `rules`,
`tweaks`, `optimizer`, `cleaner`, `winexec` em 100%; `scanner` ~97%; `main`
(bindings do Wails + `app.go`) ~91%.

Os únicos pontos que ficaram de fora, deliberadamente:
- `main.go:main` — abre uma janela nativa de verdade, não dá pra testar
  isoladamente sem um webview real.
- Um branch defensivo em `scanner.dirSize` (`d.Info()` retornando erro por
  uma condição de corrida entre `readdir` e `stat`) — praticamente
  impossível de reproduzir de forma confiável e determinística.
- Uma linha dentro do closure padrão de `runScan` em `NewApp` — só é
  exercitada se `scanner.Scan` rodar de verdade contra o sistema de
  arquivos e a Lixeira real; a lógica que essa closure chama já está 100%
  testada dentro do próprio pacote `scanner`, então não valeu a pena forçar
  isso aqui.

**Como `app.go` ficou testável**: `App` tem dois campos injetáveis,
`emit` (função que emite eventos pro frontend, default
`runtime.EventsEmit`) e `runScan` (função que efetivamente varre,
default chama `scanner.Scan`). Sem isso, `StartScan` chamaria
`runtime.EventsEmit` com um `context.Context` sem o runtime do Wails
embutido, e a própria lib do Wails faz `log.Fatalf` nesse caso — mataria o
processo de teste inteiro, não só o teste. Testes usam
`app.emit`/`app.runScan` trocados por fakes (ver `waitForEvent` em
`app_test.go` pra sincronizar com a goroutine sem sleep).

O módulo de Otimização continua testado **apenas com runner mockado** — as
ações reais (registro do Windows, `powercfg`) nunca são executadas contra
uma máquina de verdade em `go test`, só manualmente pelo usuário no app.
`internal/winexec.RunPowerShell` é a exceção: seus testes chamam o
`powershell.exe` de verdade (é o próprio objeto sob teste, não dá pra
mockar sem perder a cobertura), mas só executam comandos inofensivos
(`Write-Output`, `exit 1`) que não alteram nada no sistema.

## Build e dev

```powershell
wails dev              # app nativo + servidor em http://localhost:34115 para testar no navegador
wails build -clean     # build de produção -> build/bin/winsweep.exe
```

Pré-requisitos: Go 1.25+, Node.js, Wails CLI (`go install
github.com/wailsapp/wails/v2/cmd/wails@latest`).

### Empacotamento .msi (WiX Toolset) — testado e funcionando

WiX Toolset v3.14 instalado pelo usuário (exigia `NetFx3`, que exige
admin — Claude Code não roda elevado, então essa parte tem que ser manual:
`winget install --id WiXToolset.WiXToolset` numa janela de administrador).
A partir daí, todo o resto foi validado nesta máquina: build → candle/light
→ `msiexec /i` (instalação real) → app abrindo a partir do local instalado
→ `msiexec /x` (desinstalação) → checagem de que nada ficou para trás
(exe, pasta, atalhos do Menu Iniciar e Desktop, chave HKCU — tudo removido
limpo).

Arquivos em `build/windows/installer/wix/`:

- `winsweep.wxs` — definição do instalador (WiX v3). Dois GUIDs fixos que
  **nunca podem ser regenerados** (quebraria upgrades in-place):
  - `UpgradeCode` do `<Product>`: `23D793CC-94D6-4524-84C0-4C52D00353CD`
  - `Guid` do `Component Id="MainExecutable"`: `05F783B5-84BE-4569-8942-526221F0F790`
    (precisou ser fixo, não `Guid="*"`, porque o componente mistura um
    `<File>` com um `<RegistryValue KeyPath="yes">` — regra ICE38 do WiX
    para instalação per-user, ver abaixo).
- `build-msi.ps1` — builda o app (`wails build`), garante o bootstrapper do
  WebView2 baixado, roda `candle.exe -arch x64` / `light.exe`. Uso:
  `./build-msi.ps1 -Version 1.0.0` → gera `build/bin/winsweep-setup.msi`.
- `redist/MicrosoftEdgeWebview2Setup.exe` — bootstrapper "Evergreen" oficial
  da Microsoft (~2MB, baixado de `https://go.microsoft.com/fwlink/p/?LinkId=2124703`,
  mecanismo de distribuição documentado oficialmente). Embutido no .msi como
  `<Binary>`; só executa no destino se o WebView2 Runtime não for encontrado
  nem em HKLM nem em HKCU (`WEBVIEW2_MACHINE` / `WEBVIEW2_USER`).

**Instalação é per-user (`InstallScope="perUser"`), não per-machine —
decisão consciente**, não só limitação técnica:
- Instala em `%LOCALAPPDATA%\Programs\winsweep`, sem precisar de admin/UAC
  — melhor encaixe para uma ferramenta pessoal de um usuário só.
- Testar `perMachine` de verdade exigiria admin (deu erro 1925 na primeira
  tentativa, sem elevação); com `perUser` deu pra testar o ciclo completo
  install→run→uninstall de ponta a ponta nesta sessão.
- Se no futuro fizer sentido instalar para todos os usuários de uma máquina
  compartilhada, mudar `InstallScope` para `perMachine`, trocar
  `LocalAppDataFolder\Programs` por `ProgramFiles64Folder`, voltar o `File`
  a ser `KeyPath="yes"` (não precisa mais do truque do ICE38 nesse caso) e
  testar de novo com admin — não é uma mudança grande, mas precisa de
  privilégios que esta sessão não tinha para validar.

O template NSIS padrão do Wails (`build/windows/installer/project.nsi`)
continua existindo e funcional (`wails build -nsis` gera um `.exe`
instalador) — não foi removido, só não é o caminho usado para `.msi`.

## Notas de ambiente (armadilhas já descobertas, não perder tempo de novo)

- O `PowerShell`/`Bash` tools deste ambiente **não herdam PATH atualizado**
  depois de instalar algo novo (ex.: Go, Wails) — cada chamada é um processo
  novo, mas aparentemente com um PATH "congelado" de antes da instalação.
  Sempre prefixar comandos com:
  `$env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User") + ";" + "$env:USERPROFILE\go\bin"`
- A ferramenta de preview do navegador (`mcp__Claude_Browser__preview_start`)
  roda o comando configurado em `.claude/launch.json` (na raiz
  `C:\Users\guivi\dev`, não dentro do projeto) num processo cujo `cd`
  **não gruda** (o `%CD%` volta pro diretório original mesmo depois de
  `cd /d`) — provavelmente algum reset de sandbox por comando. Workaround
  que funcionou: um `.cmd` físico (`winsweep/dev.cmd`) que usa `%~dp0` para
  resolver o próprio diretório e só então chama `wails.exe dev`, apontado
  como `runtimeExecutable` direto (sem passar por `cmd /c cd && ...`).
- `wails.exe` (não `wails`, sem extensão) é necessário nesse mesmo processo
  de preview — resolução de comando sem extensão via PATH não funcionou lá
  mesmo com `where wails` encontrando o binário.
- WiX + instalação **per-user**: um componente que instala num diretório do
  perfil do usuário (`LocalAppDataFolder`, `DesktopFolder`, etc.) não pode
  usar o `<File>` como `KeyPath` (regra ICE38) — precisa de um
  `<RegistryValue Root="HKCU" ... KeyPath="yes">` no lugar, e nesse caso o
  `Guid` do `<Component>` também não pode mais ser `Guid="*"` (regra
  CNDL0230: componente com File + RegistryValue KeyPath exige GUID fixo).
  Precisa também de `<RemoveFolder On="uninstall">` para cada diretório sob
  o perfil do usuário (regra ICE64), senão o `light.exe` recusa a build.
- `candle.exe` sem `-arch x64` gera instalador x86 por padrão; ao instalar
  em `ProgramFiles64Folder` (perMachine) isso quebra com ICE80. Sempre usar
  `candle.exe -arch x64` já que o binário do Go é amd64.

## Escopo deliberadamente de fora (não implementar sem alinhar antes)

- Gerenciar programas de inicialização (Task Scheduler / registro Run).
- Mexer em serviços do Windows (start/stop/disable).
- Qualquer ajuste em nível de sistema (HKLM) no módulo de Otimização.
- Heurística de "arquivo desnecessário" fora dos locais curados —
  criação de uma lista dinâmica com scan de tamanho/data é ideia
  arriscada, discutir antes de tentar.
