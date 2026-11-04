import { describe, it, expect, vi, beforeEach } from 'vitest'
import { defineComponent } from 'vue'
import { mount } from '@vue/test-utils'
import { useConnection } from './useConnection'

// Mock Wails Go Bridge bindings
vi.mock('../wailsjs/go/bridge/App', () => {
  return {
    HasSession: vi.fn().mockResolvedValue(true),
    ListSessions: vi.fn().mockResolvedValue([
      {
        id: 'sess-1',
        access_token: 'token-123',
        container_name: 'dev-container',
        hostname: 'dev.interno',
        tailscale_ip: '100.64.0.2',
        is_active: false,
      }
    ]),
    Connect: vi.fn(),
    ReconnectSession: vi.fn(),
    ForgetSession: vi.fn().mockResolvedValue(undefined),
    Disconnect: vi.fn().mockResolvedValue(undefined),
    GetConnectionInfo: vi.fn().mockResolvedValue({
      connection_id: 'conn-1',
      hostname: 'dev.interno',
      tailscale_ip: '100.64.0.2',
      client_tailscale_ip: '100.64.0.5',
    }),
    IsConnected: vi.fn().mockResolvedValue(false),
  }
})

// Mock Wails Runtime Events
vi.mock('../wailsjs/runtime/runtime', () => {
  return {
    EventsOn: vi.fn(),
    EventsOff: vi.fn(),
  }
})

// Helper to run composable inside an active Vue component context
function mountComposable() {
  let composableInstance: ReturnType<typeof useConnection>
  const TestComponent = defineComponent({
    setup() {
      composableInstance = useConnection()
      return () => null
    }
  })
  mount(TestComponent)
  return composableInstance!
}

describe('useConnection Composable', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('initializes with default disconnected state', () => {
    const c = mountComposable()

    expect(c.isConnected.value).toBe(false)
    expect(c.isConnecting.value).toBe(false)
    expect(c.errorMessage.value).toBe('')
  })

  it('updates state successfully on connect', async () => {
    const { Connect } = await import('../wailsjs/go/bridge/App')
    vi.mocked(Connect).mockResolvedValueOnce(undefined as any)

    const c = mountComposable()

    const connectPromise = c.connect('valid-token-123')
    expect(c.isConnecting.value).toBe(true)

    await connectPromise

    expect(c.isConnecting.value).toBe(false)
    expect(c.isConnected.value).toBe(true)
    expect(c.connectionInfo.value.hostname).toBe('dev.interno')
    expect(Connect).toHaveBeenCalledWith('valid-token-123')
  })

  it('handles connect failure and sets error message', async () => {
    const { Connect } = await import('../wailsjs/go/bridge/App')
    vi.mocked(Connect).mockRejectedValueOnce('Falha de autenticação com o token')

    const c = mountComposable()

    await c.connect('bad-token')

    expect(c.isConnecting.value).toBe(false)
    expect(c.isConnected.value).toBe(false)
    expect(c.errorMessage.value).toBe('Falha de autenticação com o token')
  })

  it('disconnects and resets connection state', async () => {
    const { Disconnect } = await import('../wailsjs/go/bridge/App')
    const c = mountComposable()

    c.isConnected.value = true

    await c.disconnect()

    expect(Disconnect).toHaveBeenCalled()
    expect(c.isConnected.value).toBe(false)
  })
})
