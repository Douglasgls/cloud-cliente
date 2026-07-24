import { ref, onMounted, onUnmounted } from 'vue'
import {
  HasSession,
  Connect,
  Reconnect,
  Disconnect,
  GetConnectionInfo,
  IsConnected
} from '../wailsjs/go/bridge/App'
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime'
import { bridge } from '../wailsjs/go/models'

export function useConnection() {
  const isConnected = ref(false)
  const isConnecting = ref(false)
  const hasSession = ref(false)
  const stepMessage = ref('')
  const errorMessage = ref('')
  const connectionInfo = ref<bridge.ConnectionInfoDTO>(new bridge.ConnectionInfoDTO())

  const checkSession = async () => {
    try {
      hasSession.value = await HasSession()
    } catch (e) {
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
    errorMessage.value = ''
    stepMessage.value = 'Iniciando conexão...'

    try {
      await Connect(token)
      isConnected.value = true
      connectionInfo.value = await GetConnectionInfo()
      hasSession.value = true
    } catch (err: any) {
      errorMessage.value = typeof err === 'string' ? err : (err.message || 'Erro ao conectar')
      isConnected.value = false
      await checkSession()
    } finally {
      isConnecting.value = false
    }
  }

  const handleReconnect = async () => {
    isConnecting.value = true
    errorMessage.value = ''
    stepMessage.value = 'Reconectando...'

    try {
      await Reconnect()
      isConnected.value = true
      connectionInfo.value = await GetConnectionInfo()
      hasSession.value = true
    } catch (err: any) {
      errorMessage.value = typeof err === 'string' ? err : (err.message || 'Erro ao reconectar')
      isConnected.value = false
      await checkSession()
    } finally {
      isConnecting.value = false
    }
  }

  const handleDisconnect = async () => {
    try {
      await Disconnect()
    } catch (e) {
      console.error(e)
    } finally {
      isConnected.value = false
      connectionInfo.value = new bridge.ConnectionInfoDTO()
      hasSession.value = false
      errorMessage.value = ''
    }
  }

  onMounted(() => {
    checkSession()
    checkStatus()

    EventsOn('connection_progress', (step: string) => {
      stepMessage.value = step
    })

    EventsOn('connection_state_changed', async (connected: boolean) => {
      isConnected.value = connected
      if (connected) {
        connectionInfo.value = await GetConnectionInfo()
      }
    })
  })

  onUnmounted(() => {
    EventsOff('connection_progress')
    EventsOff('connection_state_changed')
  })

  return {
    isConnected,
    isConnecting,
    hasSession,
    stepMessage,
    errorMessage,
    connectionInfo,
    checkSession,
    connect: handleConnect,
    reconnect: handleReconnect,
    disconnect: handleDisconnect
  }
}
