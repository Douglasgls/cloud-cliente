# ☁️ Cloud Client

O **Cloud Client** é uma aplicação desktop desenvolvida em Go usando a framework **Wails v2** e **Vite/React/JS** no frontend. Ele é responsável por estabelecer um túnel VPN seguro de forma isolada (user-space) utilizando o motor do Tailscale/Headscale, facilitando o encaminhamento de serviços (TCP Proxy) e tráfego remoto de forma transparente.

---

## 🛠️ Requisitos de Ambiente

Antes de iniciar, garanta que você possui os seguintes itens instalados no seu sistema operacional (Windows ou Linux):

1. **Go (Golang)**: Versão 1.20 ou superior (Recomendado 1.22+).
2. **Node.js**: Versão 18 ou superior + npm.
3. **Wails CLI**: CLI oficial do Wails para gerenciar compilação e desenvolvimento.
   - Instale executando no terminal:
     ```bash
     go install github.com/wailsapp/wails/v2/cmd/wails@latest
     ```
4. **C++ Build Tools (Apenas para Windows)**: Necessário para a compilação do Wails no Windows.
5. **Bibliotecas de Desenvolvimento (Apenas para Linux)**:
   - No Ubuntu/Debian:
     ```bash
     sudo apt update && sudo apt install -y libgtk-3-dev libwebkit2gtk-4.0-dev pkg-config build-essential
     ```

---

## 🚀 Como Executar em Desenvolvimento (Modo Dev)

O modo de desenvolvimento habilita **Hot-Reload** (as alterações no código Go ou no Frontend são aplicadas na hora) e abre a janela do aplicativo com o inspetor Web (F12) ativo.

Execute na raiz do projeto:
```bash
wails dev
```

---

## 📦 Como Gerar o Executável Final (Build de Produção)

Para compilar a aplicação final pronta para distribuição:

### No Windows
Gera um arquivo executável standalone `.exe` em `build/bin/cloud-client.exe`:
```bash
wails build
```

### No Linux
Gera o executável standalone do Linux em `build/bin/cloud-client`:
```bash
wails build
```

### 🚀 Gerar Build de Compartilhamento (build-share)

Para gerar uma build atualizada e copiá-la automaticamente para a pasta de compartilhamento (`build-share`), execute o script de build na raiz do projeto:

#### No Windows (PowerShell ou Prompt de Comando)
```powershell
.\build-share.ps1
```
ou
```cmd
.\build-share.bat
```

Este script irá:
1. Compilar a aplicação usando `wails build`.
2. Criar a pasta `build-share` (se não existir).
3. Copiar e atualizar o executável `cloud-client.exe` dentro da pasta `build-share/`.

---

## 🐧 A Versão para Linux Ainda Funciona?

**Sim, funciona perfeitamente!**

### Como funciona nos bastidores:
* As correções estruturais que fizemos no Named Pipe do Windows estão protegidas pelas tags de compilação condicional do Go (`//go:build windows`). 
* No Linux, o sistema usa **Unix Sockets** em vez de Named Pipes. O Tailscaled no Linux não possui o comportamento do Windows de se auto-desligar quando o cliente fecha a conexão, portanto ele não precisa do serviço de "Anchor Connection" (que foi implementado como no-op no arquivo `anchor_other.go` para sistemas não-Windows).
* O monitor de saúde foi otimizado de forma genérica no arquivo Go principal, o que significa que o Linux também se beneficia de uma checagem de saúde de rede muito mais estável, sem falsos-negativos e sem reconexões agressivas.

---

## 🔍 Estrutura de Arquivos Importantes

* [`main.go`](file:///c:/Users/dougl/Downloads/cloud-cliente-main/cloud-cliente-main/main.go): Ponto de entrada que inicializa a aplicação desktop (Wails) ou a versão CLI de terminal.
* [`internal/runtime/`](file:///c:/Users/dougl/Downloads/cloud-cliente-main/cloud-cliente-main/internal/runtime): Gerenciamento do ciclo de vida dos binários `tailscale` e `tailscaled`.
* [`internal/gui/controller/connect_controller.go`](file:///c:/Users/dougl/Downloads/cloud-cliente-main/cloud-cliente-main/internal/gui/controller/connect_controller.go): Controlador do fluxo de conexão, autenticação Headscale e monitoramento de saúde de rede.
* [`internal/forwarding/`](file:///c:/Users/dougl/Downloads/cloud-cliente-main/cloud-cliente-main/internal/forwarding): Gerenciamento de proxies TCP locais e integração SOCKS5.
* [`frontend/`](file:///c:/Users/dougl/Downloads/cloud-cliente-main/cloud-cliente-main/frontend): Interface gráfica da aplicação (React/JS/Vite).
