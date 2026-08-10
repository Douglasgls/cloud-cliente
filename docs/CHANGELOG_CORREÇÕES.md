# 🛠️ Changelog de Correções — Cloud Client

**Período:** Agosto 2026  
**Repositório:** `cloud-cliente-main`  
**Investigador:** Antigravity (Arquiteto de Software Sênior — Go/Wails/Windows)

---

## Índice

1. [Runtime Autocontido e Isolado (Windows & Linux)](#1-runtime-autocontido-e-isolado)
2. [Estabilidade do Daemon — Grupo de Processos Windows](#2-estabilidade-do-daemon--grupo-de-processos-windows)
3. [Permissões de Named Pipe (SDDL) para Usuário Comum](#3-permissões-de-named-pipe-sddl-para-usuário-comum)
4. [Protocolo TLS/HTTPS no Headscale](#4-protocolo-tlshttps-no-headscale)
5. [Daemon se auto-desligando — os.OpenFile no Named Pipe](#5-daemon-se-auto-desligando--osopenfile-no-named-pipe)
6. [Anchor Connection — mask=0 no watch-ipn-bus](#6-anchor-connection--mask0-no-watch-ipn-bus)
7. [Health Monitor — Falso-negativo derrubando a conexão](#7-health-monitor--falso-negativo-derrubando-a-conexão)

---

## 1. Runtime Autocontido e Isolado

**Arquivo:** `internal/runtime/windows.go`  
**Severidade:** 🔴 Crítica  
**Status:** ✅ Corrigido

### Problema

No Windows, o aplicativo tentava localizar ou instalar o Tailscale em `C:\Program Files\Tailscale`, exigindo permissões de Administrador e criando dependência do sistema operacional.

### Correção

O aplicativo agora é 100% autocontido. Ele cria a pasta `%USERPROFILE%\.cloud-client\runtime\` e copia seus próprios executáveis standalone (`tailscale.exe` e `tailscaled.exe`), exatamente como no Linux. Nenhuma instalação externa é necessária.

---

## 2. Estabilidade do Daemon — Grupo de Processos Windows

**Arquivo:** `internal/runtime/daemon.go` (linha 97–101)  
**Severidade:** 🔴 Crítica  
**Status:** ✅ Corrigido

### Problema

O `tailscaled.exe` encerrava sozinho após a execução do `tailscale up` com a mensagem:

```
tailscaled got signal interrupt; shutting down
```

No Windows, executáveis sem grupo de processos próprio compartilham o grupo de sinais de console do pai. Quando o contexto CLI encerrava, um sinal `SIGINT` de console era propagado para o daemon.

### Correção

Adicionado `CreationFlags` no `SysProcAttr` do processo do daemon:

```go
cmd.SysProcAttr = &syscall.SysProcAttr{
    CreationFlags: 0x00000200 | 0x08000000, // CREATE_NEW_PROCESS_GROUP | CREATE_NO_WINDOW
}
```

O daemon agora permanece rodando de forma estável e independente do processo pai.

---

## 3. Permissões de Named Pipe (SDDL) para Usuário Comum

**Arquivo:** Configuração SDDL do `tailscaled` (pipe `\\.\pipe\cloud-client-tailscaled`)  
**Severidade:** 🔴 Crítica  
**Status:** ✅ Corrigido

### Problema

A criação da Named Pipe falhava em usuários não-Administradores com o erro `0x51B` (`ERROR_INVALID_OWNER`).

### Correção

A Security Descriptor (SDDL) foi ajustada para `D:PAI(A;OICI;GWGR;;;BU)(A;OICI;GWGR;;;SY)`, permitindo a criação de pipes por qualquer usuário sem necessidade de elevação de privilégios.

---

## 4. Protocolo TLS/HTTPS no Headscale

**Componente:** Servidor Headscale (infraestrutura)  
**Severidade:** 🔴 Crítica  
**Status:** ✅ Corrigido

### Problema

O Headscale rodava em HTTP plano (`http://201.23.87.235:8080`). O motor de rede moderno do Tailscale (`controlhttp`) fazia upgrade automático para HTTPS no meio do handshake (`POST https://201.23.87.235:8080/machine/register`), resultando em `context deadline exceeded`.

### Correção

Configurado domínio com certificado SSL/TLS válido no servidor (`https://cloud-headscale.duckdns.org`). A autenticação entre o aplicativo e o Headscale passou a funcionar com Status 200 OK.

---

## 5. Daemon se Auto-Desligando — os.OpenFile no Named Pipe

**Arquivo:** `internal/runtime/daemon.go` (funções `validateDaemon` e `isSocketAlive`)  
**Severidade:** 🔴 Crítica — causa raiz do bug "client disconnected... disconnecting Tailscale"  
**Status:** ✅ Corrigido

### Problema

O código Go abria uma conexão com o Named Pipe (`\\.\pipe\cloud-client-tailscaled`) usando `os.OpenFile` para validar se o daemon estava vivo, e **fechava essa conexão imediatamente**. O `tailscaled` interpreta o fechamento dessa conexão como um sinal de que seu único cliente de controle se desconectou e entra em shutdown automático.

Havia **dois locais** no código com este defeito:

| Função | Frequência | Impacto |
|---|---|---|
| `validateDaemon()` | Polling a cada 150ms durante startup | **Causa primária** — mata o daemon logo após o `tailscale up` |
| `isSocketAlive()` | Chamada pelo health monitor a cada 4s | **Causa secundária** — mata o daemon durante operação normal |

### Mecanismo Detalhado

```
1. validateDaemon() → os.OpenFile(pipe, O_RDWR)
2. tailscaled registra "sessão de controle ativa"
3. validateDaemon() → f.Close()   ← FECHAMENTO ABRUPTO
4. tailscaled: "último cliente desconectou" → shutdown
```

### Correção

1. **Removidos os blocos `os.OpenFile` / `f.Close()`** de ambas as funções no Windows
2. Substituído por `os.Stat(socketPath)` — que chama `GetFileAttributesW` internamente e **não abre** uma conexão no Named Pipe
3. Implementada uma **Anchor Connection** persistente (ver item 6 abaixo) para manter o daemon vivo indefinidamente

```go
// ANTES (causava o problema):
f, err := os.OpenFile(m.socketPath, os.O_RDWR, 0)
if err == nil {
    f.Close()  // ← MATA O DAEMON
}

// DEPOIS (correto — verificação passiva):
_, err := os.Stat(m.socketPath)
if err != nil {
    return fmt.Errorf("named pipe not yet available: %w", err)
}
```

---

## 6. Anchor Connection — mask=0 no watch-ipn-bus

**Arquivo:** `internal/runtime/anchor_windows.go` (linha 138–139)  
**Severidade:** 🔴 Crítica — a anchor ficava caindo em loop infinito de EOF  
**Status:** ✅ Corrigido

### Contexto

Para resolver o bug #5, foi implementada uma "anchor connection" — uma conexão HTTP persistente ao endpoint `GET /localapi/v0/watch-ipn-bus` do daemon. Esta conexão funciona como um "cliente de controle permanente" que mantém o daemon vivo enquanto o app estiver rodando, permitindo que chamadas CLI (`tailscale up`, `tailscale status`) abram e fechem o pipe livremente sem derrubar o daemon.

### Problema

A anchor era criada com `mask=0` na query string:

```
GET /localapi/v0/watch-ipn-bus?mask=0
```

Análise do código-fonte do Tailscale v1.98.9 (`ipn/ipnlocal/local.go`, função `WatchNotificationsAs`) revelou que com `mask=0`:

1. **Nenhum estado inicial é enviado** — o `if mask & initialBits != 0` (linha 3310) é falso, então `ini` permanece `nil`
2. **Nenhum polling de engine é iniciado** — o `if mask & NotifyWatchEngineUpdates != 0` (linha 3387) é falso, então `pollRequestEngineStatus` nunca é invocado
3. **O canal `ch` fica vazio para sempre** — `sender.Run(ctx, ch)` bloqueia indefinidamente sem receber nenhum evento
4. **O body HTTP fica em branco** — sem eventos JSON sendo escritos, nenhum `\n` é produzido
5. **`ReadString('\n')` recebe EOF** — o transport HTTP eventualmente fecha a conexão ociosa

O resultado era o loop infinito observado nos logs:

```
[anchor] stream dropped: stream closed by daemon (EOF) — reconnecting in 300ms
[anchor] stream dropped: stream closed by daemon (EOF) — reconnecting in 600ms
[anchor] stream dropped: stream closed by daemon (EOF) — reconnecting in 1.2s
...
```

### Correção

Alterado `mask=0` para `mask=3`:

```
GET /localapi/v0/watch-ipn-bus?mask=3
```

Onde `3` = `NotifyWatchEngineUpdates (1)` | `NotifyInitialState (2)`:

| Bit | Valor | Efeito |
|---|---|---|
| `NotifyInitialState` | 2 | Daemon envia imediatamente um JSON com o estado atual (State, SessionID, BrowseToURL) — confirma que a conexão está viva |
| `NotifyWatchEngineUpdates` | 1 | Daemon inicia `pollRequestEngineStatus` em background, enviando um evento JSON a cada ~2 segundos — mantém a conexão ativa indefinidamente |

Com esta correção, a anchor recebe eventos contínuos, o `ReadString('\n')` retorna regularmente, e a conexão permanece estável sem reconexões.

---

## 7. Health Monitor — Falso-negativo Derrubando a Conexão

**Arquivo:** `internal/gui/controller/connect_controller.go`  
**Severidade:** 🟠 Alta — causava desconexão desnecessária e re-registro no Headscale  
**Status:** ✅ Corrigido

### Problema

O `startHealthMonitor` iniciava verificações de saúde **4 segundos** após a conexão, com intervalos de **4 segundos**. Quando `testConnectionHealth` falhava, o `autoReconnectLoop` chamava `tailscale up --reset` **na primeira tentativa**, derrubando o nó do Headscale e reiniciando o handshake WireGuard do zero.

O problema tinha três camadas:

#### 7a. Sem grace period após conexão

O handshake WireGuard/DERP leva de 5 a 30 segundos para completar (especialmente via DERP relay). O health check começava após apenas 4 segundos — antes do túnel estar operacional.

#### 7b. Timeout de Status() muito curto

O timeout do `tailscale status` era de apenas **2 segundos**. No Windows, spawnar um subprocess pode levar mais do que isso, causando timeouts transientes que eram interpretados como "daemon morto".

#### 7c. Reconnect agressivo na tentativa 1

O `autoReconnectLoop` executava `tailscale up --reset` na primeira tentativa de reconexão, o que:
- Faz `down` do nó atual → **offline no Headscale**
- Registra um novo nó → novo handshake WireGuard
- Reseta completamente o progresso de conexão

### Correções

| O quê | Antes | Depois |
|---|---|---|
| Grace period antes do 1º check | 0s | **20 segundos** |
| Intervalo entre checks | 4s | **10 segundos** |
| Timeout do `tailscale status` | 2s | **5 segundos** |
| Timeout do TCP dial (fallback) | 2s | **8 segundos** |
| `tailscale up --reset` na tentativa 1 | ✅ Sim | ❌ Removido — só a cada 5ª tentativa |

```go
// ANTES:
ticker := time.NewTicker(4 * time.Second)

// DEPOIS:
// Grace period de 20s antes do primeiro check
select {
case <-ctx.Done():
    return
case <-time.After(20 * time.Second):
}
ticker := time.NewTicker(10 * time.Second)
```

```go
// ANTES:
if (attempt == 1 || attempt%5 == 0) && c.tsService != nil && loginServer != "" {

// DEPOIS:
if attempt%5 == 0 && c.tsService != nil && loginServer != "" {
```

---

## Resumo Visual da Arquitetura de Correções

```
┌─────────────────────────────────────────────────────────────┐
│                    Cloud Client (Wails)                      │
│                                                             │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │   Runtime     │    │   Connect    │    │  Forwarding  │  │
│  │   Manager     │    │  Controller  │    │   (Proxy)    │  │
│  │              │    │              │    │              │  │
│  │ ✅ os.Stat    │    │ ✅ Grace 20s  │    │  SOCKS5 →    │  │
│  │ ✅ No OpenFile│    │ ✅ Status 5s  │    │  tailscaled  │  │
│  │ ✅ ProcGroup  │    │ ✅ No reset#1 │    │              │  │
│  └──────┬───────┘    └──────────────┘    └──────┬───────┘  │
│         │                                        │          │
│  ┌──────┴───────┐                        ┌──────┴───────┐  │
│  │   Anchor     │◄── Named Pipe ──►│  tailscaled  │  │
│  │  Connection  │   (persistent)    │    (daemon)   │  │
│  │              │                   │              │  │
│  │ ✅ mask=3     │                   │  WireGuard/  │  │
│  │ ✅ Keepalive  │                   │  DERP tunnel │  │
│  └──────────────┘                   └──────┬───────┘  │
│                                            │          │
└────────────────────────────────────────────┼──────────┘
                                             │
                                    ┌────────┴────────┐
                                    │   Headscale     │
                                    │  (HTTPS/TLS)    │
                                    │ ✅ duckdns.org   │
                                    └─────────────────┘
```

---

## Arquivos Modificados (Resumo)

| Arquivo | Alterações |
|---|---|
| `internal/runtime/daemon.go` | Removido `os.OpenFile` do pipe, adicionado `CREATE_NEW_PROCESS_GROUP`, validação via `os.Stat` |
| `internal/runtime/manager.go` | Adicionado campo `cancelAnchor`, `startAnchorAndWait()`, `StopDaemon()` com cleanup do anchor |
| `internal/runtime/anchor_windows.go` | **NOVO** — Anchor connection persistente via `watch-ipn-bus`, `mask=0→3` |
| `internal/runtime/anchor_other.go` | **NOVO** — Stub no-op para Linux (anchor não necessário em Unix sockets) |
| `internal/runtime/windows.go` | Runtime autocontido em `~/.cloud-client/runtime/` |
| `internal/gui/controller/connect_controller.go` | Grace period 20s, timeout 5s, intervalo 10s, sem reset na tentativa 1 |
