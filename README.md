# winsweep

Aplicativo desktop para Windows com duas funções:

- **Limpeza** — analisa locais conhecidos de arquivos desnecessários do
  sistema (cache de navegadores, temporários, backups de atualização,
  Lixeira, etc.), mostra para que serve cada um e só remove o que você
  confirmar, item a item.
- **Otimização** — ajustes de desempenho reversíveis (plano de energia,
  efeitos visuais do Windows) para computadores mais lentos ou antigos.

## Instalação (para usuários)

Não é preciso ter Go, Node.js ou qualquer outra linguagem instalada — o
instalador cuida de tudo, inclusive do único componente de que o app
depende para funcionar (o WebView2 Runtime da Microsoft, que já vem de
fábrica na grande maioria das instalações do Windows 10/11).

**Requisitos**: Windows 10 ou 11 (64 bits).

1. Baixe `winsweep-setup.msi` na [página de Releases](https://github.com/guivicentep/winsweep/releases/latest)
   (ou gere um novo, veja [Build](#build-gerar-o-instalador) abaixo).
2. Dê duplo clique no arquivo `.msi` e siga o assistente. Não é necessário
   ser administrador nem aparece nenhum aviso do Controle de Conta de
   Usuário (UAC) — a instalação é feita só para o seu usuário.
3. Ao final, o winsweep estará disponível no **Menu Iniciar** e um atalho
   será criado na sua **Área de Trabalho**.
4. Abra o app, escolha o módulo desejado (Limpeza ou Otimização) na barra
   lateral, e siga as instruções na tela — toda ação de exclusão ou ajuste
   pede confirmação individual antes de ser executada.

### Desinstalar

Pelo Windows: **Configurações → Aplicativos → Aplicativos instalados**,
procure por "winsweep" e clique em Desinstalar. Isso remove o executável,
os atalhos (Menu Iniciar e Área de Trabalho) e as configurações do
registro — nada fica para trás.

## Como funciona

- **Varredura eficiente**: o processo roda em modo de segundo plano do
  Windows (`PROCESS_MODE_BACKGROUND_BEGIN`), o que reduz automaticamente sua
  prioridade de CPU, disco e memória, e a varredura de cada local é feita de
  forma concorrente mas limitada (`internal/scanner`), para não competir por
  recursos com o resto do sistema.
- **Regras curadas**: apenas locais bem conhecidos e seguros são analisados
  (`internal/rules` e `internal/tweaks`) — nada de heurísticas sobre
  arquivos pessoais.
- **Exclusão segura**: cada item de limpeza, ao ser confirmado, é enviado
  para a Lixeira do Windows (reversível). Apenas esvaziar a própria Lixeira
  é uma ação definitiva, sinalizada com destaque na interface
  (`internal/cleaner`).
- **Ajustes reversíveis**: todo ajuste de otimização pode ser desfeito
  individualmente a qualquer momento (`internal/optimizer`), e nenhum deles
  mexe em nível de sistema (só configurações do usuário atual e o plano de
  energia ativo).

## Desenvolvimento

Pré-requisitos: Go 1.25+, Node.js, e a CLI do [Wails](https://wails.io)
(`go install github.com/wailsapp/wails/v2/cmd/wails@latest`).

```bash
wails dev
```

Abre a aplicação nativa e também um servidor em http://localhost:34115 para
testar o frontend no navegador.

## Build (gerar o executável)

```bash
wails build -clean
```

Gera o executável de produção em `build/bin/winsweep.exe`.

## Build (gerar o instalador .msi)

Requer o [WiX Toolset v3](https://wixtoolset.org/) instalado:

```bash
winget install --id WiXToolset.WiXToolset
```

Depois, um único comando builda o app e gera o instalador:

```powershell
build\windows\installer\wix\build-msi.ps1
```

Isso produz `build/bin/winsweep-setup.msi`, pronto para distribuir — já
inclui o bootstrapper do WebView2 Runtime embutido (baixado automaticamente
na primeira execução do script), então quem for instalar não precisa de
mais nada pré-instalado. Detalhes de por que o pacote é per-user (não exige
administrador) estão no [CLAUDE.md](CLAUDE.md).

## Testes

```bash
go test ./... -cover
```

Cobertura atual: 96% do total de statements. Por pacote: `rules`, `tweaks`,
`optimizer`, `cleaner` e `winexec` em 100%; `scanner` ~97%; pacote `main`
(bindings do Wails) ~91% — os únicos pontos não cobertos são o `main()` de
entrada (abre uma janela nativa de verdade, não faz sentido testar
isoladamente) e um branch defensivo de corrida em `dirSize` praticamente
impossível de reproduzir de forma confiável em teste automatizado.

O módulo de Otimização foi testado **apenas com um runner mockado** — as
ações reais (registro do Windows, `powercfg`) nunca são executadas contra
a máquina em `go test`, só quando você usa o app de verdade.
