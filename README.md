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

### No Linux Pop_OS! ou qualquer outra distribuição que utilize WebKit2GTK 4.1.x
Gera o executável standalone do Linux em `build/bin/cloud-client`:
```bash
wails build -tags webkit2_41
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

### rodar no windows
- Baixe a pasta build-share
- Rode o app cloud-client.exe