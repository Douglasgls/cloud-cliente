export interface FormattedError {
  title: string
  hint: string
}

/**
 * Mapeia erros técnicos em mensagens amigáveis com título claro
 * e orientações práticas de como o usuário pode resolver o problema.
 */
export function formatConnectionError(rawError: unknown): FormattedError {
  if (!rawError) {
    return {
      title: 'Erro inesperado na conexão',
      hint: 'Tente novamente em alguns instantes.'
    }
  }

  const msg = typeof rawError === 'string' ? rawError : (rawError as any).message || String(rawError)
  const lower = msg.toLowerCase()

  // 1. Token inválido / não encontrado / unauthorized / forbidden
  if (
    lower.includes('invalid token') ||
    lower.includes('token inválido') ||
    lower.includes('não encontrado') ||
    lower.includes('unauthorized') ||
    lower.includes('forbidden')
  ) {
    return {
      title: 'Token inválido ou não encontrado',
      hint: 'Verifique se o token foi copiado corretamente do painel da Cloud e tente novamente.'
    }
  }

  // 2. Token / conexão expirada
  if (
    lower.includes('connection expired') ||
    lower.includes('expirou') ||
    lower.includes('expirada') ||
    lower.includes('status 410')
  ) {
    return {
      title: 'Este token de conexão expirou',
      hint: 'Por motivos de segurança, este token já venceu. Gere um novo token no painel da Cloud.'
    }
  }

  // 3. Token vazio ou não informado
  if (
    lower.includes('informe um token') ||
    lower.includes('token vazio') ||
    lower.includes('token de conexão')
  ) {
    return {
      title: 'Token não informado',
      hint: 'Por favor, insira ou cole o token de acesso para iniciar a conexão.'
    }
  }

  // 4. Sem conexão / falha ao contatar cloud / timeout / DNS
  if (
    lower.includes('unable to contact cloud') ||
    lower.includes('não foi possível conectar aos servidores') ||
    lower.includes('no such host') ||
    lower.includes('connection refused') ||
    lower.includes('timeout') ||
    lower.includes('deadline exceeded') ||
    lower.includes('internet')
  ) {
    return {
      title: 'Não foi possível conectar à Cloud',
      hint: 'Verifique sua conexão com a internet e se os servidores estão acessíveis na sua rede.'
    }
  }

  // 5. Tailscale / túnel VPN
  if (
    lower.includes('tailscale') ||
    lower.includes('túnel') ||
    lower.includes('vpn')
  ) {
    return {
      title: 'Falha ao estabelecer o túnel seguro',
      hint: 'Não foi possível conectar à rede privada. O servidor de controle pode estar indisponível no momento.'
    }
  }

  // 6. Conflito de portas locais (bind address already in use)
  if (
    lower.includes('already in use') ||
    lower.includes('endereço já em uso') ||
    lower.includes('portas') ||
    lower.includes('bind')
  ) {
    return {
      title: 'Conflito de portas locais',
      hint: 'Uma das portas necessárias para os serviços locais já está em uso por outro aplicativo no seu computador.'
    }
  }

  // 7. Daemon / serviço local / permissões
  if (
    lower.includes('daemon') ||
    lower.includes('permissão') ||
    lower.includes('permission') ||
    lower.includes('serviço de rede local') ||
    lower.includes('componentes do sistema')
  ) {
    return {
      title: 'Falha no serviço de rede local',
      hint: 'Certifique-se de que o aplicativo foi iniciado com as permissões necessárias para gerenciar conexões.'
    }
  }

  // 8. Confirmação da conexão
  if (lower.includes('confirm') || lower.includes('confirmação')) {
    return {
      title: 'Falha na confirmação da sessão',
      hint: 'A sessão não pôde ser confirmada pela Cloud. Aguarde alguns instantes e tente novamente.'
    }
  }

  // Se a mensagem já for composta com ponto ("Título. Orientação")
  const parts = msg.split('. ')
  if (parts.length >= 2) {
    return {
      title: parts[0].trim(),
      hint: parts.slice(1).join('. ').trim()
    }
  }

  return {
    title: 'Falha na conexão',
    hint: msg
  }
}
