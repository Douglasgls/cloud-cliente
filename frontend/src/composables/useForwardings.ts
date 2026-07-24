import { ref, onMounted, onUnmounted } from 'vue'
import {
  ListForwardings,
  AddForwarding,
  UpdateForwarding,
  DeleteForwarding,
  ToggleForwarding,
  CopyToClipboard,
  OpenURL
} from '../wailsjs/go/bridge/App'
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime'
import { bridge } from '../wailsjs/go/models'

export function useForwardings() {
  const forwardings = ref<bridge.ForwardingDTO[]>([])
  const toastMessage = ref('')
  const showToast = ref(false)

  const triggerToast = (msg: string) => {
    toastMessage.value = msg
    showToast.value = true
    setTimeout(() => {
      showToast.value = false
    }, 2500)
  }

  const fetchForwardings = async () => {
    try {
      forwardings.value = await ListForwardings()
    } catch (e) {
      console.error('Failed to list forwardings:', e)
    }
  }

  const handleAdd = async (name: string, remotePort: number, localPort: number) => {
    await AddForwarding(name, remotePort, localPort)
    await fetchForwardings()
  }

  const handleUpdate = async (id: string, name: string, remotePort: number, localPort: number, enabled: boolean) => {
    await UpdateForwarding(id, name, remotePort, localPort, enabled)
    await fetchForwardings()
  }

  const handleDelete = async (id: string) => {
    await DeleteForwarding(id)
    await fetchForwardings()
  }

  const handleToggle = async (id: string, enabled: boolean) => {
    await ToggleForwarding(id, enabled)
    await fetchForwardings()
  }

  const copyText = async (text: string, label = 'Copiado para a área de transferência!') => {
    try {
      await CopyToClipboard(text)
      triggerToast(label)
    } catch (e) {
      navigator.clipboard?.writeText(text)
      triggerToast(label)
    }
  }

  const openAppURL = async (url: string) => {
    try {
      await OpenURL(url)
    } catch (e) {
      window.open(url, '_blank')
    }
  }

  onMounted(() => {
    fetchForwardings()

    EventsOn('forwardings_changed', (updatedList: bridge.ForwardingDTO[]) => {
      if (Array.isArray(updatedList)) {
        forwardings.value = updatedList
      } else {
        fetchForwardings()
      }
    })
  })

  onUnmounted(() => {
    EventsOff('forwardings_changed')
  })

  return {
    forwardings,
    toastMessage,
    showToast,
    fetchForwardings,
    addForwarding: handleAdd,
    updateForwarding: handleUpdate,
    deleteForwarding: handleDelete,
    toggleForwarding: handleToggle,
    copyText,
    openAppURL
  }
}
