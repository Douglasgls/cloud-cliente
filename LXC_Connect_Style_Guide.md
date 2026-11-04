# LXC Connect — Design System & Style Guide

Este documento serve como referência oficial de identidade visual e UI/UX para a construção do **aplicativo Cliente** e futuras interfaces do ecossistema **LXC Connect**. O objetivo é manter uma estética consistente, premium, moderna e focada no conforto visual (dark mode first).

---

## 1. Identidade Principal
- **Nome do Produto**: LXC Connect
- **Proposta Visual**: Uma interface limpa, "dark mode nativa", com toques de glassmorphism e micro-interações que transmitem a ideia de tráfego de rede e segurança. Foge de templates genéricos, usando sombras sutis e cores frias.

---

## 2. Tipografia
As fontes devem transmitir clareza em textos corridos e aspecto técnico em dados e códigos.

- **Fonte Principal (Sans-Serif)**: `Inter`
  - *Uso*: Títulos, parágrafos, botões e navegação.
  - *Pesos*: Normal (400), Medium (500), Bold (700), ExtraBold (800).
- **Fonte Técnica (Monospace)**: `JetBrains Mono`
  - *Uso*: Badges, rótulos técnicos, endereços IP, nomes de containers e códigos.

---

## 3. Paleta de Cores

A paleta de cores foca em tons profundos de ardósia (slate/blue-gray) para o fundo, e um azul de marca vivo e confiável.

### Cores de Fundo (Dark Mode Base)
O fundo não é preto absoluto (`#000000`). Ele usa tons azulados super escuros para conforto visual.
- **Background Geral (`dark.bg`)**: `#0B1120`
- **Background de Cards (`dark.card`)**: `#111827` (slate-900 puro) ou `bg-slate-950`
- **Bordas (`dark.border`)**: `#1F2A3D`
- **Fundo de Hover / Elementos Secundários (`dark.hover`)**: `#1A2332`
- **Elementos Mutados / Linhas (`dark.muted`)**: `#334155`

### Cor da Marca (Brand)
Usada em botões principais, glows, e elementos em destaque.
- **Primária (`brand.500`)**: `#0066B2`
- **Apoio / Brilho (`brand.400`)**: `#3387d6`
- **Hover Botões (`brand.600`)**: `#005290`
- **Acento Metálico (`brand.accent`)**: `#0077D9`

### Cores de Estado e Técnicas (Tech)
- **Sucesso (Conectado, Ativo)**: `#10B981` (Emerald)
- **Aviso (Pendente, Atenção)**: `#F59E0B` (Amber)
- **Erro (Desconectado, Falha)**: `#EF4444` (Red)
- **Texto Técnico (`tech.subtle`)**: `#94A3B8` (Slate-400)

---

## 4. UI Patterns & Estilos

### Cards e Containers
- **Bordas Arredondadas**: Uso padronizado de `rounded-2xl` para cards grandes e modais, e `rounded-xl` ou `rounded-lg` para botões e inputs.
- **Bordas Sutis**: Sempre usar borda com 1px nas caixas. Ex: `border border-slate-800`.
- **Sombras de Profundidade**: Em vez de sombras grossas, usar sombras com glow da marca. Ex: `shadow-2xl shadow-brand-500/5` para elementos de destaque.
- **Ring / Glow**: Ao focar (hover ou state), usar rings sutis como `ring-1 ring-brand-500/10`.

### Botões e Interações
- **Feedback Tátil**: Todo botão deve ter uma escala reduzida ao clicar: `active:scale-[0.97]`.
- **Transições**: Sempre suavizadas. `transition-all duration-200`.

### Efeitos Especiais (Micro-interações)
O cliente LXC Connect deve incorporar efeitos "vivos" (usar com moderação para não pesar a performance):
1. **Glassmorphism (Vidro Fosco)**: Utilizar `backdrop-blur-md` ou `backdrop-blur-xl` acompanhado de fundos com muita transparência (ex: `bg-slate-950/40`) em modais, barras de topo ou barras de rodapé flutuantes.
2. **Textos em Shimmer (Gradiente Animado)**: Para textos que indicam estado "Conectando..." ou um destaque de feature principal. O texto ganha uma máscara transparente que passa um gradiente metálico `bg-gradient-to-r`.
3. **Flutuação (Levitation/Float)**: Elementos estáticos na tela que representam infraestrutura (ícones de servidores, cards de status) podem ficar transladando em Y levemente para dar a sensação de atividade no terminal (`translateY(-12px)` ao longo de 6 segundos).
4. **Malha de Galáxia (Background Mesh)**: Fundo com grid pontilhado (`radial-gradient`) usando a cor `#334155` com dots de `1.5px` para simular as constelações de rede/nodes.

---

## 5. Estrutura de Espaçamento (Whitespace)
- Evitar adensamento de informações. 
- Espaços amplos (Paddings `p-6` ou `p-8` dentro de modais/cards).
- Espaçamento de blocos (`space-y-6`) para respiro de conteúdo.
- Textos descritivos devem ter o `leading-relaxed` (maior espaçamento entrelinhas) e utilizar tamanhos legíveis como `text-[13px]` ou `text-sm`.
