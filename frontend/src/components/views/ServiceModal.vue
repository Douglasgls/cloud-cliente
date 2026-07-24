<script setup lang="ts">
import { ref, watch } from 'vue'
import Modal from '../ui/Modal.vue'
import Input from '../ui/Input.vue'
import Button from '../ui/Button.vue'
import { bridge } from '../../wailsjs/go/models'

const props = defineProps<{
  show: boolean
  editingService?: bridge.ForwardingDTO | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'save', payload: { id?: string; name: string; remotePort: number; localPort: number; enabled?: boolean; isDefault?: boolean }): void
}>()

const name = ref('')
const remotePort = ref<number | string>('')
const localPort = ref<number | string>('')
const errorMessage = ref('')

watch(
  () => props.editingService,
  (srv) => {
    if (srv) {
      name.value = srv.name
      remotePort.value = srv.remote_port
      localPort.value = srv.local_port
    } else {
      name.value = ''
      remotePort.value = ''
      localPort.value = ''
    }
    errorMessage.value = ''
  },
  { immediate: true }
)

const handleSubmit = () => {
  errorMessage.value = ''
  const rPort = Number(remotePort.value)
  const lPort = Number(localPort.value)

  if (!name.value.trim()) {
    errorMessage.value = 'Informe um nome para o serviço'
    return
  }
  if (isNaN(rPort) || rPort < 1 || rPort > 65535) {
    errorMessage.value = 'Porta remota inválida (1 - 65535)'
    return
  }
  if (isNaN(lPort) || lPort < 1 || lPort > 65535) {
    errorMessage.value = 'Porta local inválida (1 - 65535)'
    return
  }

  emit('save', {
    id: props.editingService?.id,
    name: name.value.trim(),
    remotePort: rPort,
    localPort: lPort,
    enabled: props.editingService ? props.editingService.enabled : true,
    isDefault: props.editingService ? props.editingService.is_default : false
  })
  emit('close')
}
</script>

<template>
  <Modal :show="show" :title="editingService ? `Editar ${editingService.name}` : 'Novo Serviço Personalizado'" @close="emit('close')">
    <form @submit.prevent="handleSubmit" class="space-y-4">
      <div v-if="errorMessage" class="p-2.5 rounded-lg bg-rose-500/10 border border-rose-500/20 text-xs font-medium text-rose-400">
        {{ errorMessage }}
      </div>

      <Input
        v-model="name"
        label="Nome do Serviço"
        placeholder="Ex: Redis, Grafana, API"
        :disabled="editingService?.is_default"
        id="modal-service-name"
      />

      <div class="grid grid-cols-2 gap-3">
        <Input
          v-model="remotePort"
          label="Porta Remota"
          placeholder="Ex: 6379"
          type="number"
          :disabled="editingService?.is_default"
          id="modal-remote-port"
        />

        <Input
          v-model="localPort"
          label="Porta Local"
          placeholder="Ex: 6379"
          type="number"
          id="modal-local-port"
        />
      </div>

      <div class="flex items-center justify-end gap-2.5 pt-3 border-t border-zinc-800/80">
        <Button variant="ghost" size="md" @click="emit('close')">
          Cancelar
        </Button>
        <Button variant="primary" size="md" type="submit">
          Salvar
        </Button>
      </div>
    </form>
  </Modal>
</template>
