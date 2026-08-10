# 📝 Guia de Diagnóstico e Correções da VPN (Named Pipe & Status Offline)

Este documento explica as correções finais de arquitetura aplicadas ao **Cloud Client** para resolver o problema de desconexão e queda constante da VPN (`offline` no Headscale / loop de `EOF` na Anchor / falhas de SOCKS5).

---

## 1. O Que Foi Corrigido (Causa Raiz & Solução)

O problema de conectividade residia em três pontos do código e ciclo de vida:

### A. O Loop de EOF na Anchor Connection
* **Onde:** [`internal/runtime/anchor_windows.go`](file:///c:/Users/dougl/Downloads/cloud-cliente-main/cloud-cliente-main/internal/runtime/anchor_windows.go)
* **Causa:** O endpoint `/localapi/v0/watch-ipn-bus` do Tailscale exige duas coisas para se manter aberto:
  1. A conexão Named Pipe deve ser aberta com o nível de impersonação correto do Windows (`PipeImpLevelIdentification`). Caso contrário, o `tailscaled.exe` recusa a chamada imediatamente por não conseguir identificar as credenciais do usuário.
  2. A query parameter `mask` não podia ser `0`. Com `mask=0`, o Tailscale não envia o estado inicial e não agenda o polling interno de status, fazendo com que o canal HTTP fique silencioso e sofra timeout/EOF.
* **Solução:** 
  - Alterado o dial do named pipe de `DialPipeContext` para `DialPipeAccessImpLevel` com impersonação identificável.
  - Alterada a URL da API para usar `mask=3` (`NotifyInitialState` + `NotifyWatchEngineUpdates`), forçando eventos de batimento cardíaco contínuos do daemon que impedem o timeout da conexão.

### B. O Health Monitor Derrubando o Nó (Falsos-Negativos)
* **Onde:** [`internal/gui/controller/connect_controller.go`](file:///c:/Users/dougl/Downloads/cloud-cliente-main/cloud-cliente-main/internal/gui/controller/connect_controller.go)
* **Causa:** O monitor de saúde testava a integridade da VPN fazendo pings TCP diretos nas portas `80/443/22` do host de destino remotamente logo nos primeiros 4 segundos. Como o handshake WireGuard/DERP pode demorar até 20 segundos para estabilizar, o monitor falhava e, na primeira tentativa de reconexão, chamava `tailscale up --reset`, que matava a autenticação atual e deixava o nó offline no Headscale.
* **Solução:** 
  - Adicionado um **grace period de 20 segundos** antes do monitor de saúde realizar qualquer checagem.
  - O monitor agora verifica a saúde do daemon consultando se a CLI responde (`tsService.Status`) em vez de forçar tráfego TCP bruto na máquina remota.
  - O loop de auto-reconexão não realiza mais o `--reset` destrutivo na primeira falha transitória.

---

## 2. Onde as Mudanças Foram Feitas (Arquivos)

### 📌 [1] [internal/runtime/anchor_windows.go](file:///c:/Users/dougl/Downloads/cloud-cliente-main/cloud-cliente-main/internal/runtime/anchor_windows.go#L38-L46)
Substituição da conexão crua de Named Pipe pela conexão com impersonação de segurança do Windows exigida pela API do Tailscale:

```go
func (d *namedPipeDialer) DialContext(ctx context.Context, _, _ string) (net.Conn, error) {
	// IMPORTANTE: Impersonation nível "Identification" é mandatório!
	return winio.DialPipeAccessImpLevel(ctx, d.pipePath, windows.GENERIC_READ|windows.GENERIC_WRITE, winio.PipeImpLevelIdentification)
}
```

E alteração da query string para inscrever a anchor nos eventos de status:
```go
req, err := http.NewRequestWithContext(ctx, http.MethodGet,
	"http://local-tailscaled.sock/localapi/v0/watch-ipn-bus?mask=3", nil)
```

---

### 📌 [2] [internal/gui/controller/connect_controller.go](file:///c:/Users/dougl/Downloads/cloud-cliente-main/cloud-cliente-main/internal/gui/controller/connect_controller.go#L400-L540)
Aumento da estabilidade e remoção do comportamento agressivo de reconexão:

1. **Grace Period de Inicialização:**
   ```go
   // Espera 20s para o WireGuard fechar o handshake antes de iniciar testes
   select {
   case <-ctx.Done():
       return
   case <-time.After(20 * time.Second):
   }
   ticker := time.NewTicker(10 * time.Second) // Check a cada 10s
   ```

2. **Validação Não-Destrutiva:**
   Checagem baseada no status local do daemon (se o binário responde e se a porta está disponível) ao invés de depender puramente do tráfego TCP de destino imediato.

3. **Reconexão Conservadora:**
   ```go
   // Apenas roda o "tailscale up --reset" a cada 5 tentativas, nunca na primeira!
   if attempt%5 == 0 && c.tsService != nil && loginServer != "" {
       _ = c.tsService.Up(ctx, loginServer, authKey, clientHostname)
   }
   ```

---

## 3. Como Executar Corretamente para Funcionar

Para garantir que os binários antigos não interfiram nos novos, siga este procedimento:

### Passo 1: Fechar instâncias duplicadas em execução
Se houver processos do cliente rodando em segundo plano ocultos no Windows, eles tentarão controlar o mesmo Named Pipe e causarão conflito. 

Abra o PowerShell como Administrador ou Comum e rode:
```powershell
Stop-Process -Name "main", "main_debug", "tailscaled" -Force -ErrorAction SilentlyContinue
```

### Passo 2: Rodar o Wails em Desenvolvimento
Execute o comando padrão do Wails na pasta raiz do projeto. Ele vai compilar a versão mais recente com todas as correções acima aplicadas:
```bash
wails dev
```

### Passo 3: Teste de Conexão
1. Insira sua chave de acesso e realize a conexão no app.
2. Acompanhe a saída do console no terminal do `wails dev`. O log `[anchor] persistent connection established.` deve aparecer uma única vez e **nunca mais cair** ou dar `EOF`.
3. Verifique no Headscale se o nó (`client-test-win`) mantém o status `online` de forma contínua.

---

## 4. Comandos Úteis de Diagnóstico (Caso precise testar)

Se você quiser validar se a VPN do Tailscale está de fato conectada na rede e respondendo pings (de forma pura, isolando os proxies locais HTTP/Socks5 do app), rode no terminal do Windows:

```bash
# Pinga o nó remoto "test-win" diretamente pela VPN local
C:\Users\dougl\.cloud-client\runtime\tailscale.exe --socket=\\.\pipe\cloud-client-tailscaled ping 100.64.0.8
```

```bash
# Verifica a tabela de rotas e conexões ativas do Tailscale local
C:\Users\dougl\.cloud-client\runtime\tailscale.exe --socket=\\.\pipe\cloud-client-tailscaled status
```
