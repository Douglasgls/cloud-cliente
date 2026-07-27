import { ref, onMounted, onUnmounted } from 'vue'
import {
  HasSession,
  ListSessions,
  Connect,
  ReconnectSession,
  ForgetSession,
  Disconnect,
  GetConnectionInfo,
  IsConnected
} from '../wailsjs/go/bridge/App'
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime'
import { bridge } from '../wailsjs/go/models'

export function useConnection() {
  const isConnected = ref(false)
  const isConnecting = ref(false)
  const isReconnecting = ref(false)
  const hasSession = ref(false)
  const sessions = ref<bridge.SessionDTO[]>([])
  const stepMessage = ref('')
  const errorMessage = ref('')
  const connectionInfo = ref<bridge.ConnectionInfoDTO>(new bridge.ConnectionInfoDTO())

  const fetchSessions = async () => {
    try {
      sessions.value = await ListSessions()
      hasSession.value = sessions.value.length > 0
    } catch (e) {
      sessions.value = []
      hasSession.value = false
    }
  }

  const checkStatus = async () => {
    try {
      isConnected.value = await IsConnected()
      if (isConnected.value) {
        connectionInfo.value = await GetConnectionInfo()
      }
    } catch (e) {
      isConnected.value = false
    }
  }

  const handleConnect = async (token: string) => {
    isConnecting.value = true
    isReconnecting.value = false
    errorMessage.value = ''
    stepMessage.value = 'Iniciando conexão...'

    try {
      await Connect(token)
      isConnected.value = true
      connectionInfo.value = await GetConnectionInfo()
      await fetchSessions()
    } catch (err: any) {
      errorMessage.value = typeof err === 'string' ? err : (err.message || 'Erro ao conectar')
      isConnected.value = false
      await fetchSessions()
    } finally {
      isConnecting.value = false
    }
  }

  const handleReconnectSession = async (id: string) => {
    isConnecting.value = true
    isReconnecting.value = false
    errorMessage.value = ''
    stepMessage.value = 'Reconectando...'

    try {
      await ReconnectSession(id)
      isConnected.value = true
      connectionInfo.value = await GetConnectionInfo()
      await fetchSessions()
    } catch (err: any) {
      errorMessage.value = typeof err === 'string' ? err : (err.message || 'Erro ao reconectar')
      isConnected.value = false
      await fetchSessions()
    } finally {
      isConnecting.value = false
    }
  }

  const handleForgetSession = async (id: string) => {
    try {
      await ForgetSession(id)
      await fetchSessions()
    } catch (e) {
      console.error('Error forgetting session:', e)
    }
  }

  const handleDisconnect = async () => {
    try {
      await Disconnect()
    } catch (e) {
      console.error(e)
    } finally {
      isConnected.value = false
      isReconnecting.value = false
      connectionInfo.value = new bridge.ConnectionInfoDTO()
      errorMessage.value = ''
      await fetchSessions()
    }
  }

  onMounted(() => {
    fetchSessions()
    checkStatus()

    EventsOn('connection_progress', (step: string) => {
      stepMessage.value = step
      if (step.startsWith('Reconectando')) {
        isReconnecting.value = true
      } else if (step.startsWith('✓') || step.includes('Conectado')) {
        isReconnecting.value = false
      }
    })

    EventsOn('connection_state_changed', async (connected: boolean) => {
      isConnected.value = connected
      if (connected) {
        connectionInfo.value = await GetConnectionInfo()
        isReconnecting.value = false
      } else {
        isReconnecting.value = false
      }
      await fetchSessions()
    })
  })

  onUnmounted(() => {
    EventsOff('connection_progress')
    EventsOff('connection_state_changed')
  })

  return {
    isConnected,
    isConnecting,
    isReconnecting,
    hasSession,
    sessions,
    stepMessage,
    errorMessage,
    connectionInfo,
    checkSession: fetchSessions,
    fetchSessions,
    connect: handleConnect,
    reconnectSession: handleReconnectSession,
    forgetSession: handleForgetSession,
    disconnect: handleDisconnect
  }
}
