<script setup lang="ts">
import { ref, computed } from 'vue'
import {
  Server,
  Network,
  ExternalLink,
  Copy,
  Pencil,
  Trash2,
  Plus,
  LogOut,
  Terminal,
  Globe,
  Lock,
  Layers,
  RefreshCw,
  AlertCircle
} from '@lucide/vue'

import Card from '../ui/Card.vue'
import Button from '../ui/Button.vue'
import StatusBadge from '../ui/StatusBadge.vue'
import Switch from '../ui/Switch.vue'
import ServiceModal from './ServiceModal.vue'

import { bridge } from '../../wailsjs/go/models'

const props = defineProps<{
  connectionInfo: bridge.ConnectionInfoDTO
  forwardings: bridge.ForwardingDTO[]
  isReconnecting?: boolean
  stepMessage?: string
}>()

const emit = defineEmits<{
  (e: 'disconnect'): void
  (e: 'addForwarding', payload: { name: string; remotePort: number; localPort: number }): void
  (e: 'updateForwarding', payload: { id: string; name: string; remotePort: number; localPort: number; enabled: boolean }): void
  (e: 'deleteForwarding', id: string): void
  (e: 'toggleForwarding', payload: { id: string; enabled: boolean }): void
  (e: 'copyText', text: string, label?: string): void
  (e: 'openURL', url: string): void
}>()

const showModal = ref(false)
const editingService = ref<bridge.ForwardingDTO | null>(null)

const defaultServices = computed(() => props.forwardings.filter(f => f.is_default))
const customServices = computed(() => props.forwardings.filter(f => !f.is_default))

const openAddModal = () => {
  editingService.value = null
  showModal.value = true
}

const openEditModal = (srv: bridge.ForwardingDTO) => {
  editingService.value = srv
  showModal.value = true
}

const handleModalSave = (payload: { id?: string; name: string; remotePort: number; localPort: number; enabled?: boolean }) => {
  if (payload.id) {
    emit('updateForwarding', {
      id: payload.id,
      name: payload.name,
      remotePort: payload.remotePort,
      localPort: payload.localPort,
      enabled: payload.enabled ?? true
    })
  } else {
    emit('addForwarding', {
      name: payload.name,
      remotePort: payload.remotePort,
      localPort: payload.localPort
    })
  }
}

const getStatusType = (srv: bridge.ForwardingDTO) => {
  if (!srv.enabled) return 'disabled'
  if (srv.running) return 'running'
  return 'error'
}

const getSSHCommand = (localPort: number) => `ssh root@${props.connectionInfo.hostname}.interno -p ${localPort}`
const getHTTPURL = (localPort: number) => `http://${props.connectionInfo.hostname}.interno:${localPort}`
const getHTTPSURL = (localPort: number) => `https://${props.connectionInfo.hostname}.interno:${localPort}`
</script>

<template>
  <div class="h-full flex flex-col overflow-hidden bg-transparent">

    <!-- Header Connection Info Banner -->
    <div class="p-6 pb-4 space-y-3">
      <!-- Reconnecting Banner -->
      <div v-if="isReconnecting" class="p-3 bg-amber-500/10 border border-amber-500/30 rounded-xl flex items-center gap-3 text-amber-400 animate-pulse">
        <RefreshCw class="w-4 h-4 animate-spin text-amber-400 shrink-0" />
        <div class="text-xs">
          <p class="font-bold">Conexão instável com a internet</p>
          <p class="text-[11px] text-amber-300/80">{{ stepMessage || 'Reconectando ao container...' }}</p>
        </div>
      </div>

      <Card class="border-brand-500/20 bg-slate-900/50 backdrop-blur-md shadow-2xl shadow-brand-500/10">
        <div class="grid grid-cols-1 sm:grid-cols-4 gap-4 divide-y sm:divide-y-0 sm:divide-x divide-slate-800/80">

          <!-- Container Name -->
          <div class="flex items-center gap-3">
            <div class="p-2.5 rounded-xl bg-brand-500/10 border border-brand-500/20 text-brand-400">
              <Server class="w-5 h-5" />
            </div>
            <div>
              <p class="text-[11px] font-medium text-slate-400 uppercase tracking-wider" title="Nome do Container">Container</p>
              <h2 class="text-sm font-bold text-slate-100 truncate">
                {{ connectionInfo.hostname || connectionInfo.connection_id || 'Conectado' }}
              </h2>
              <div 
                v-if="connectionInfo.hostname"
                class="flex items-center gap-1 mt-0.5 group cursor-pointer" 
                @click="emit('copyText', `${connectionInfo.hostname}.interno`, 'URL Base copiada!')" 
                title="Copiar URL Base"
              >
                <p class="text-[11px] text-brand-400 font-mono">{{ connectionInfo.hostname }}.interno</p>
                <Copy class="w-3 h-3 text-brand-400 opacity-0 group-hover:opacity-100 transition-opacity" />
              </div>
            </div>
          </div>

          <!-- Tailscale IP -->
          <div class="flex items-center gap-3 sm:pl-4 pt-3 sm:pt-0">
            <div class="p-2.5 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-400">
              <Network class="w-5 h-5" />
            </div>
            <div>
              <p class="text-[11px] font-medium text-slate-400 uppercase tracking-wider" title="IP do Container Remoto">Container IP</p>
              <h2 class="text-sm font-mono font-bold text-emerald-400">
                {{ connectionInfo.tailscale_ip || '-' }}
              </h2>
            </div>
          </div>

          <!-- Local Tailscale IP -->
          <div class="flex items-center gap-3 sm:pl-4 pt-3 sm:pt-0">
            <div class="p-2.5 rounded-xl bg-brand-500/10 border border-brand-500/20 text-brand-400">
              <Globe class="w-5 h-5" />
            </div>
            <div>
              <p class="text-[11px] font-medium text-slate-400 uppercase tracking-wider" title="Seu IP Local na Rede Tailscale">Client IP</p>
              <h2 class="text-sm font-mono font-bold text-brand-400">
                {{ connectionInfo.client_tailscale_ip || '-' }}
              </h2>
            </div>
          </div>

          <!-- Connection Status / ID -->
          <div class="flex items-center justify-between sm:pl-4 pt-3 sm:pt-0">
            <div>
              <p class="text-[11px] font-medium text-slate-400 uppercase tracking-wider" title="ID único da conexão atual">ID da Conexão</p>
              <h2 class="text-xs font-mono font-semibold text-slate-300 truncate max-w-[140px]">
                {{ connectionInfo.connection_id || '-' }}
              </h2>
            </div>
            <Button variant="danger" size="sm" @click="emit('disconnect')" title="Desconectar">
              <LogOut class="w-3.5 h-3.5" />
              <span class="hidden sm:inline">Desconectar</span>
            </Button>
          </div>

        </div>
      </Card>
    </div>

    <!-- Scrollable Services List -->
    <div class="flex-1 overflow-y-auto px-6 space-y-6 pb-6">

      <!-- Default Services -->
      <div class="space-y-3">
        <div class="flex items-center gap-2">
          <Layers class="w-4 h-4 text-brand-400" />
          <h3 class="text-xs font-bold uppercase tracking-wider text-slate-400">Serviços Padrão</h3>
        </div>

        <div class="grid grid-cols-1 gap-3">
          <Card v-for="srv in defaultServices" :key="srv.id" class="p-4 bg-slate-900/40 backdrop-blur-sm border-slate-800/80 hover:border-brand-500/40 hover:shadow-brand-500/5 hover:shadow-lg transition-all duration-300">
            <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">

              <!-- Info & Status -->
              <div class="flex items-center gap-3">
                <Switch
                  :model-value="srv.enabled"
                  @update:model-value="(val) => emit('toggleForwarding', { id: srv.id, enabled: val })"
                />

                <div class="p-2 rounded-lg bg-slate-800/80 text-slate-300">
                  <Terminal v-if="srv.id === 'ssh'" class="w-4 h-4 text-emerald-400" />
                  <Globe v-else-if="srv.id === 'http'" class="w-4 h-4 text-sky-400" />
                  <Lock v-else-if="srv.id === 'https'" class="w-4 h-4 text-brand-400" />
                </div>

                <div class="space-y-1">
                  <div class="flex items-center gap-2">
                    <span class="text-sm font-bold text-slate-100">{{ srv.name }}</span>
                    <StatusBadge :status="getStatusType(srv)" :label="srv.last_error ? 'Erro ao iniciar' : undefined" />
                  </div>
                  <p class="text-xs text-slate-400 font-mono">
                    Remota: <span class="text-slate-200">{{ srv.remote_port }}</span>
                    <span class="text-slate-600 mx-1.5">➔</span>
                    Local: <span class="text-brand-300 font-semibold">{{ srv.local_port }}</span>
                  </p>
                  <p v-if="srv.last_error" class="text-[11px] text-rose-400 mt-1 flex items-center gap-1 font-medium">
                    <AlertCircle class="w-3.5 h-3.5 shrink-0" />
                    <span>{{ srv.last_error }}</span>
                  </p>
                </div>
              </div>

              <!-- Actions -->
              <div class="flex items-center gap-2 self-end sm:self-center">
                <!-- SSH Actions -->
                <template v-if="srv.id === 'ssh'">
                  <Button variant="secondary" size="sm" @click="emit('copyText', getSSHCommand(srv.local_port), 'Comando SSH copiado!')">
                    <Copy class="w-3.5 h-3.5" />
                    <span>Copiar comando</span>
                  </Button>
                </template>

                <!-- HTTP Actions -->
                <template v-else-if="srv.id === 'http'">
                  <Button variant="secondary" size="sm" @click="emit('copyText', getHTTPURL(srv.local_port), 'URL HTTP copiada!')">
                    <Copy class="w-3.5 h-3.5" />
                    <span>Copiar URL</span>
                  </Button>
                </template>

                <!-- HTTPS Actions -->
                <template v-else-if="srv.id === 'https'">
                  <Button variant="secondary" size="sm" @click="emit('copyText', getHTTPSURL(srv.local_port), 'URL HTTPS copiada!')">
                    <Copy class="w-3.5 h-3.5" />
                    <span>Copiar URL</span>
                  </Button>
                </template>

                <!-- Edit Button -->
                <Button variant="ghost" size="sm" @click="openEditModal(srv)">
                  <Pencil class="w-3.5 h-3.5" />
                </Button>
              </div>

            </div>
          </Card>
        </div>
      </div>

      <!-- Custom Services -->
      <div class="space-y-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <Layers class="w-4 h-4 text-emerald-400" />
            <h3 class="text-xs font-bold uppercase tracking-wider text-slate-400">Serviços Personalizados</h3>
          </div>
          <Button variant="secondary" size="sm" @click="openAddModal">
            <Plus class="w-3.5 h-3.5" />
            <span>Novo Serviço</span>
          </Button>
        </div>

        <div v-if="customServices.length === 0" class="p-6 text-center border border-dashed border-slate-800 rounded-xl bg-slate-900/30 backdrop-blur-sm">
          <p class="text-xs text-slate-400 font-medium">Nenhum serviço personalizado cadastrado</p>
          <p class="text-[11px] text-slate-500 mt-1">Clique em "+ Novo Serviço" para mapear portas adicionais</p>
        </div>

        <div v-else class="grid grid-cols-1 gap-3">
          <Card v-for="srv in customServices" :key="srv.id" class="p-4 bg-slate-900/40 backdrop-blur-sm border-slate-800/80 hover:border-brand-500/40 hover:shadow-brand-500/5 hover:shadow-lg transition-all duration-300">
            <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">

              <!-- Info & Status -->
              <div class="flex items-center gap-3">
                <Switch
                  :model-value="srv.enabled"
                  @update:model-value="(val) => emit('toggleForwarding', { id: srv.id, enabled: val })"
                />

                <div class="space-y-1">
                  <div class="flex items-center gap-2">
                    <span class="text-sm font-bold text-slate-100">{{ srv.name }}</span>
                    <StatusBadge :status="getStatusType(srv)" :label="srv.last_error ? 'Erro ao iniciar' : undefined" />
                  </div>
                  <p class="text-xs text-slate-400 font-mono">
                    Remota: <span class="text-slate-200">{{ srv.remote_port }}</span>
                    <span class="text-slate-600 mx-1.5">➔</span>
                    Local: <span class="text-emerald-300 font-semibold">{{ srv.local_port }}</span>
                  </p>
                  <p v-if="srv.last_error" class="text-[11px] text-rose-400 mt-1 flex items-center gap-1 font-medium">
                    <AlertCircle class="w-3.5 h-3.5 shrink-0" />
                    <span>{{ srv.last_error }}</span>
                  </p>
                </div>
              </div>

              <!-- Actions -->
              <div class="flex items-center gap-2 self-end sm:self-center">
                <Button variant="ghost" size="sm" @click="openEditModal(srv)">
                  <Pencil class="w-3.5 h-3.5" />
                  <span>Editar</span>
                </Button>
                <Button variant="danger" size="sm" @click="emit('deleteForwarding', srv.id)">
                  <Trash2 class="w-3.5 h-3.5" />
                </Button>
              </div>

            </div>
          </Card>
        </div>

      </div>

    </div>

    <!-- Service Modal Component -->
    <ServiceModal
      :show="showModal"
      :editing-service="editingService"
      @close="showModal = false"
      @save="handleModalSave"
    />

  </div>
</template>
