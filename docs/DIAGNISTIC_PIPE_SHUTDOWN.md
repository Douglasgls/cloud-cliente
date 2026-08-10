# Diagnóstico Arquitetural: "client disconnected... disconnecting Tailscale"

**Data:** 2026-08-08  
**Repositório:** `cloud-cliente-main`  
**Investigador:** Antigravity (Arquiteto de Software Sênior — Go/Wails/Windows)

---

## Resumo Executivo

O daemon `tailscaled.exe` se auto-desliga com a mensagem `"client disconnected... disconnecting Tailscale"` porque o código Go abre uma conexão **probe/sondagem** com o Named Pipe (`\\.\pipe\cloud-client-tailscaled`) para validar se o daemon está vivo, e **fecha essa conexão imediatamente depois**. O Tailscale interpreta o fechamento dessa conexão como um sinal de que seu único cliente de controle (`tailscale.exe` CLI) se desconectou, e entra em shutdown.

Há **dois locais** no código que produzem esse comportamento defeituoso, sendo um deles o gatilho primário e o outro um gatilho secundário periódico.

---

## Localização dos Defeitos

### 🔴 DEFEITO PRIMÁRIO — `internal/runtime/daemon.go` (linhas 158–165)

**Função:** `validateDaemon(ctx context.Context)`

```go
// daemon.go, linhas 158–165
} else {
    if strings.HasPrefix(m.socketPath, `\\.\pipe\`) {
        f, err := os.OpenFile(m.socketPath, os.O_RDWR, 0)
        if err != nil {
            return fmt.Errorf("named pipe not reachable: %w", err)
        }
        f.Close()   // ← AQUI ESTÁ O CRIME ARQUITETURAL
    }
}
```

**O que acontece passo a passo:**

1. `validateDaemon()` é chamado no **polling loop** de `EnsureDaemonRunning()` a cada **150ms** para checar se o daemon subiu (linhas 113–140).
2. No Windows, ele abre o Named Pipe com `os.OpenFile(m.socketPath, os.O_RDWR, 0)`.
3. Ao abrir o pipe, o `tailscaled.exe` registra **uma conexão de cliente de controle** na sua camada de `LocalAPI`.
4. O código chama `f.Close()` imediatamente após.
5. O `tailscaled.exe` detecta que esse cliente de controle fechou a conexão e, como é o **único** cliente presente no momento, aciona o seu próprio shutdown com a mensagem `"client disconnected... disconnecting Tailscale"`.

> **Por que o Tailscale se mata quando a conexão fecha?**
> O `tailscaled` usa o padrão "owner-process" para a sua API local. Quando o pipe é aberto por um cliente, o daemon marca aquela conexão como a "sessão de controle ativa". Quando essa sessão fecha abruptamente (sem um handshake de desconexão adequado), o daemon assume que o processo controlador morreu e inicia o shutdown automaticamente. Isso é comportamento documentado no código fonte do Tailscale: `ipnserver.go` → `Run()`.

---

### 🔴 DEFEITO SECUNDÁRIO — `internal/runtime/daemon.go` (linhas 194–202)

**Função:** `isSocketAlive(ctx context.Context, socketPath string)`

```go
// daemon.go, linhas 194–202
if runtime.GOOS == "windows" {
    if strings.HasPrefix(socketPath, `\\.\pipe\`) {
        f, err := os.OpenFile(socketPath, os.O_RDWR, 0)
        if err == nil {
            f.Close()   // ← MESMO PROBLEMA
            return true
        }
        return false
    }
}
```

**Onde é chamada:**

- `manager.go` linha 91: `func (m *Manager) IsDaemonRunning() bool { return isSocketAlive(context.Background(), m.socketPath) }`
- `daemon.go` linha 131: No `PrintDebugInfo`
- `connect_controller.go` linha 460–466: Em `testConnectionHealth()` — chamado **a cada 4 segundos** pelo `startHealthMonitor`

**Impacto:** O `startHealthMonitor` (linhas 400–452 de `connect_controller.go`) chama `testConnectionHealth()` a cada 4 segundos. Se o teste de `tsService.Status()` falhar (timeout de 2s), o código **não** cai no `isSocketAlive`, mas se a rota de `Status()` for lenta/indisponível, a chamada via CLI ao `tailscale status` também abre o pipe, coleta o status e fecha, mas esse caso é gerenciado pelo CLI interno, não pelo `os.OpenFile` bruto.

---

### 🟡 DEFEITO TERCIÁRIO — `internal/runtime/daemon.go` (linhas 44–49)

**Chamada inicial em `EnsureDaemonRunning()`:**

```go
// Antes de subir o daemon, o código já valida o socket:
if m.validateDaemon(ctx) == nil {
    // daemon já está rodando
    return nil
}
```

Se o daemon **já estava rodando** de uma sessão anterior, esta chamada abre e fecha o pipe — e pode matar o daemon que estava saudável.

---

## Mapa do Fluxo de Chamadas (Causa → Efeito)

```
bridge/app.go → ConnectAsync()
    └─ runtime.EnsureDaemonRunning(ctx)
         ├─ validateDaemon()          ← polling a cada 150ms
         │    └─ os.OpenFile(pipe) → f.Close()  ← MATA O DAEMON
         │
         └─ [daemon sobe e fica no estado Running]
              └─ Retorna nil (daemon "validado")

controller/connect_controller.go → startHealthMonitor()
    └─ [goroutine, ticker 4s]
         └─ testConnectionHealth()
              └─ tsService.Status(ctx)
                   └─ tailscale.exe --socket=pipe status  ← USA O CLI (OK)
                   (se falhar:)
                   └─ isSocketAlive()
                        └─ os.OpenFile(pipe) → f.Close()  ← MATA O DAEMON
```

---

## Por que o Daemon Cai "logo após o login"?

O timing é o seguinte:

1. `tailscale.exe up ...` é executado (linha 294 de `connect_controller.go`).
2. O `tailscale up` **abre internamente o Named Pipe** para enviar o comando `up` ao daemon e aguarda o resultado.
3. Quando o `tailscale up` retorna (o processo CLI termina), **ele fecha o pipe**.
4. Neste exato momento — ou nos 150ms seguintes — `validateDaemon()` abre o pipe novamente para confirmar que o daemon ainda está responsivo.
5. `validateDaemon()` fecha o pipe com `f.Close()`.
6. O daemon, que acabou de processar o `up`, recebe o sinal de desconexão do seu único cliente e faz shutdown.

Este timing é perfeitamente consistente com o sintoma relatado: _"o daemon conecta com sucesso ao Headscale e entra no estado Running. Porém, logo em seguida, o daemon se autodesliga."_

---

## Arquitetura Correta (Singleton de Conexão)

O modelo correto para interagir com o Named Pipe de um daemon no Windows **não é abrir e fechar o pipe para verificar se está vivo**. O modelo correto tem três pilares:

### 1. Verificação de Vida Sem Conexão

No Windows, para verificar se um Named Pipe existe sem se conectar a ele, use `os.Stat()` ou a API `WaitNamedPipe` + `CreateFile` com `OPEN_EXISTING` e flag `FILE_FLAG_OVERLAPPED`. Porém, **a forma mais segura** é rodar o `tailscale.exe status` (via `exec.Command`) — que gerencia o ciclo de vida da conexão corretamente através do protocolo LocalAPI do Tailscale.

```go
// CORRETO: Verificar via CLI (já implementado na outra metade do validateDaemon)
cmd := exec.CommandContext(valCtx, m.tailscalePath, "--socket="+m.socketPath, "status")
_ = cmd.Run()
output := stdoutBuf.String() + stderrBuf.String()
// Verificar se output NÃO contém erros de conexão
```

### 2. Remover os `os.OpenFile` Brutos em `validateDaemon` e `isSocketAlive`

As seções Windows de ambas as funções devem ser **removidas ou substituídas** por uma verificação passiva:

```go
// ANTES (causa o problema):
f, err := os.OpenFile(m.socketPath, os.O_RDWR, 0)
if err == nil {
    f.Close()  // MATA O DAEMON
    return true
}

// DEPOIS (correto — apenas verifica existência do pipe):
// No Windows, a existência de um Named Pipe pode ser verificada com 
// CreateFile em modo peek, mas a forma mais pragmática é:
// Remova o bloco Windows de validateDaemon e deixe APENAS o exec.Command abaixo.
// isSocketAlive() no Windows deve fazer uma chamada HTTP ao endpoint LocalAPI,
// não um open/close do pipe.
```

### 3. Usar o LocalAPI HTTP do Tailscale (Abordagem Enterprise)

O `tailscaled` expõe uma API HTTP sobre o Named Pipe. A forma correta de verificar se está vivo é fazer uma requisição HTTP ao endpoint `/localapi/v0/status`, **usando um `http.Client` configurado com um transport que dialoga com o Named Pipe**:

```go
// Exemplo de verificação correta e não-destrutiva:
client := &http.Client{
    Transport: &http.Transport{
        DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
            return winio.DialPipeContext(ctx, socketPath)
        },
    },
    Timeout: 1 * time.Second,
}
resp, err := client.Get("http://local-tailscaled.sock/localapi/v0/status")
// Esta abordagem abre e fecha a conexão HTTP de forma adequada,
// sem disparar o shutdown do daemon.
```

---

## Resumo dos Arquivos e Linhas a Corrigir

| Arquivo | Função | Linhas | Problema |
|---|---|---|---|
| [`internal/runtime/daemon.go`](file:///c:/Users/dougl/Downloads/cloud-cliente-main/cloud-cliente-main/internal/runtime/daemon.go#L158-L165) | `validateDaemon()` | 158–165 | `os.OpenFile` + `f.Close()` no pipe — **causa primária** |
| [`internal/runtime/daemon.go`](file:///c:/Users/dougl/Downloads/cloud-cliente-main/cloud-cliente-main/internal/runtime/daemon.go#L194-L202) | `isSocketAlive()` | 194–202 | Mesmo padrão — **causa secundária** |
| [`internal/runtime/manager.go`](file:///c:/Users/dougl/Downloads/cloud-cliente-main/cloud-cliente-main/internal/runtime/manager.go#L90-L92) | `IsDaemonRunning()` | 90–92 | Chama `isSocketAlive()` sem contexto — pode ser invocado a qualquer momento |
| [`internal/gui/controller/connect_controller.go`](file:///c:/Users/dougl/Downloads/cloud-cliente-main/cloud-cliente-main/internal/gui/controller/connect_controller.go#L454-L467) | `testConnectionHealth()` | 454–467 | Invoca `tsService.Status()` (correto), mas path de fallback pode chegar em `isSocketAlive()` |

---

## Conclusão

**O bug não está na lógica de conexão com o Headscale, nem no proxy SOCKS5, nem no ciclo de vida das goroutines.** O bug está em **dois `os.OpenFile` + `f.Close()` em funções de validação** que tratam o Named Pipe do Tailscale como um arquivo comum, sem respeitar o protocolo de sessão de controle do daemon.

A correção cirúrgica é:
1. **Remover os blocos `os.OpenFile` / `f.Close()` do Windows** em `validateDaemon()` e `isSocketAlive()`.
2. Deixar que a validação seja feita **exclusivamente** pelo `exec.Command("tailscale status")`, que já está implementado na segunda metade de `validateDaemon()` (linhas 168–186) e funciona corretamente.
3. Opcionalmente, substituir as sondagens por uma chamada HTTP à LocalAPI do Tailscale via Named Pipe, usando a biblioteca `github.com/Microsoft/go-winio`.
