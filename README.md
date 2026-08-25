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

---

## 🏷️ Como publicar e instalar no Linux e Windows via GitHub Releases

Como o **Wails** utiliza a interface nativa de cada sistema (WebKitGTK no Linux e WebView2 no Windows), o executável do Linux não roda no Windows e vice-versa. Por isso, geramos binários específicos para cada plataforma.

### 1. Publicação Automática (GitHub Actions - Recomendado)
O projeto já conta com um workflow automatizado em `.github/workflows/release.yml`. Para gerar os pacotes `.zip` de **Linux** e **Windows** automaticamente:

1. Faça commit das suas alterações:
   ```bash
   git add .
   git commit -m "feat: preparando versão v1.0.0"
   ```
2. Crie uma tag de versão e envie para o GitHub:
   ```bash
   git tag v1.0.0
   git push origin main --tags
   ```
3. O GitHub irá compilar o app em um servidor Linux e em um servidor Windows e publicará os arquivos na aba **Releases** do repositório:
   - `cloud-client-linux.zip`
   - `cloud-client-windows.zip`

### 2. Instruções de Instalação para Usuários

#### 🐧 No Linux:
1. Baixe o `cloud-client-linux.zip` na página de **Releases** do repositório.
2. Descompacte o arquivo:
   ```bash
   unzip cloud-client-linux.zip -d cloud-client-linux
   cd cloud-client-linux
   ```
3. Garanta permissão de execução e execute:
   ```bash
   chmod +x cloud-client
   ./cloud-client
   ```

#### 🪟 No Windows:
1. Baixe o `cloud-client-windows.zip` na página de **Releases** do repositório.
2. Clique com o botão direito e selecione **Extrair Tudo...**.
3. Abra a pasta extraída e dê duplo clique em `cloud-client.exe`.


#### comando especial linux
sudo setcap 'cap_net_bind_service=+ep' build/bin/cloud-client
resolvectl status -> links de rede dns
sudo resolvectl dns enp37s0 1.1.1.1 8.8.8.8
resolvectl flush-caches