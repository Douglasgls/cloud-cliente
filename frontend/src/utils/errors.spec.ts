import { describe, it, expect } from 'vitest'
import { formatConnectionError } from './errors'

describe('formatConnectionError', () => {
  it('formats invalid token error properly', () => {
    const res = formatConnectionError('Invalid token.')
    expect(res.title).toBe('Token inválido ou não encontrado')
    expect(res.hint).toContain('painel da Cloud')
  })

  it('formats expired token error properly', () => {
    const res = formatConnectionError('Connection expired.')
    expect(res.title).toBe('Este token de conexão expirou')
    expect(res.hint).toContain('Gere um novo token')
  })

  it('formats empty token error properly', () => {
    const res = formatConnectionError('informe um token de conexão válido')
    expect(res.title).toBe('Token não informado')
    expect(res.hint).toContain('insira ou cole o token')
  })

  it('formats network / unable to contact cloud properly', () => {
    const res = formatConnectionError('Unable to contact Cloud.: dial tcp: lookup no such host')
    expect(res.title).toBe('Não foi possível conectar à Cloud')
    expect(res.hint).toContain('conexão com a internet')
  })

  it('formats tailscale tunnel error properly', () => {
    const res = formatConnectionError('falha no Tailscale: login failed')
    expect(res.title).toBe('Falha ao estabelecer o túnel seguro')
    expect(res.hint).toContain('rede privada')
  })

  it('formats local port conflict error properly', () => {
    const res = formatConnectionError('bind: address already in use')
    expect(res.title).toBe('Conflito de portas locais')
    expect(res.hint).toContain('já está em uso')
  })

  it('handles null/undefined gracefully', () => {
    const res = formatConnectionError(undefined)
    expect(res.title).toBe('Erro inesperado na conexão')
  })
})
