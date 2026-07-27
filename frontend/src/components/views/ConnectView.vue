<script setup lang="ts">
import { ref, watch } from 'vue'
import {
  KeyRound,
  RotateCw,
  PlusCircle,
  ArrowRight,
  ArrowLeft,
  AlertCircle,
  Server,
  Network,
  Clock,
  Trash2,
  Play
} from '@lucide/vue'
import Card from '../ui/Card.vue'
import Button from '../ui/Button.vue'
import Input from '../ui/Input.vue'
import { bridge } from '../../wailsjs/go/models'

const props = defineProps<{
  hasSession: boolean
  sessions?: bridge.SessionDTO[]
  errorMessage?: string
}>()

const emit = defineEmits<{
  (e: 'connect', token: string): void
  (e: 'reconnectSession', id: string): void
  (e: 'forgetSession', id: string): void
}>()

const token = ref('')
const showTokenForm = ref(!props.hasSession || (props.sessions && props.sessions.length === 0))

watch(
  () => props.sessions,
  (newSessions) => {
    if (!newSessions || newSessions.length === 0) {
      showTokenForm.value = true
    }
  },
  { immediate: true }
)

const handleConnectSubmit = () => {
  if (token.value.trim()) {
    emit('connect', token.value.trim())
  }
}

const formatDate = (isoString?: string) => {
  if (!isoString) return 'Nunca'
  try {
    const d = new Date(isoString)
    if (isNaN(d.getTime())) return isoString
    return d.toLocaleString('pt-BR', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    })
  } catch (e) {
    return isoString
  }
}
</script>

<template>
  <div class="flex flex-col h-full max-w-2xl mx-auto p-6 space-y-6 overflow-y-auto custom-scrollbar">

    <!-- Error Alert -->
    <div v-if="errorMessage" class="w-full flex items-start gap-3 p-3.5 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-400 text-xs font-medium shrink-0">
      <AlertCircle class="w-4 h-4 text-rose-400 flex-shrink-0 mt-0.5" />
      <span>{{ errorMessage }}</span>
    </div>

    <!-- Mode A: Session Manager Cards List -->
    <div v-if="!showTokenForm && sessions && sessions.length > 0" class="flex flex-col space-y-5">
      <!-- Section Header -->
      <div class="flex items-center justify-between">
        <div>
          <h2 class="text-base font-bold text-zinc-100 flex items-center gap-2">
            <Server class="w-5 h-5 text-indigo-400" />
            Sessões Salvas
          </h2>
          <p class="text-xs text-zinc-400">Selecione um container para conectar ou gerencie suas conexões salvas</p>
        </div>
        <Button variant="primary" size="md" @click="showTokenForm = true" class="shrink-0">
          <PlusCircle class="w-4 h-4" />
          <span>Nova Conexão</span>
        </Button>
      </div>

      <!-- Cards Grid -->
      <div class="grid grid-cols-1 gap-4">
        <Card
          v-for="sess in sessions"
          :key="sess.id"
          class="space-y-4 border-zinc-800/80 hover:border-zinc-700/80 transition-all duration-200"
          :class="{ 'border-emerald-500/30 bg-emerald-950/10': sess.is_active }"
        >
          <!-- Card Header -->
          <div class="flex items-start justify-between">
            <div class="flex items-center gap-3">
              <div class="p-2.5 rounded-xl bg-indigo-500/10 border border-indigo-500/20 text-indigo-400">
                <Server class="w-5 h-5" />
              </div>
              <div>
                <h3 class="text-sm font-bold text-zinc-100 flex items-center gap-2">
                  {{ sess.container_name || sess.hostname || 'Container' }}
                  <span v-if="sess.hostname" class="px-2 py-0.5 text-[10px] font-mono rounded bg-zinc-800 text-zinc-400 border border-zinc-700/50">
                    {{ sess.hostname }}
                  </span>
                </h3>
                <p class="text-xs text-zinc-400 flex items-center gap-2 mt-0.5">
                  <span class="flex items-center gap-1">
                    <Network class="w-3.5 h-3.5 text-zinc-500" />
                    <span class="font-mono text-zinc-300">{{ sess.tailscale_ip || 'IP indisponível' }}</span>
                  </span>
                </p>
              </div>
            </div>

            <!-- Status Badge -->
            <div
              v-if="sess.is_active"
              class="px-2.5 py-1 rounded-full text-[11px] font-semibold bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 flex items-center gap-1.5"
            >
              <span class="h-1.5 w-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
              Conectado
            </div>
            <div
              v-else
              class="px-2.5 py-1 rounded-full text-[11px] font-medium bg-zinc-800/60 border border-zinc-700/50 text-zinc-400"
            >
              Salvo
            </div>
          </div>

          <!-- Card Details -->
          <div class="flex items-center justify-between text-xs text-zinc-400 pt-2 border-t border-zinc-800/60">
            <div class="flex items-center gap-1.5 text-zinc-400">
              <Clock class="w-3.5 h-3.5 text-zinc-500" />
              <span>Última conexão: {{ formatDate(sess.last_used_at) }}</span>
            </div>

            <!-- Card Actions -->
            <div class="flex items-center gap-2">
              <Button
                variant="ghost"
                size="sm"
                class="text-rose-400 hover:text-rose-300 hover:bg-rose-500/10 px-2.5"
                title="Remover completamente esta sessão"
                @click="emit('forgetSession', sess.id)"
              >
                <Trash2 class="w-3.5 h-3.5" />
                <span>Esquecer</span>
              </Button>

              <Button
                variant="primary"
                size="sm"
                class="px-3"
                @click="emit('reconnectSession', sess.id)"
              >
                <Play class="w-3.5 h-3.5 fill-current" />
                <span>Reconectar</span>
              </Button>
            </div>
          </div>
        </Card>
      </div>
    </div>

    <!-- Mode B: Token Form Input -->
    <div v-else class="flex flex-col items-center justify-center my-auto">
      <Card class="w-full max-w-md space-y-5">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="p-2.5 rounded-xl bg-indigo-500/10 border border-indigo-500/20 text-indigo-400">
              <KeyRound class="w-5 h-5" />
            </div>
            <div>
              <h2 class="text-sm font-bold text-zinc-100">Nova Conexão</h2>
              <p class="text-xs text-zinc-400">Informe seu Access Token de autenticação</p>
            </div>
          </div>

          <button
            v-if="sessions && sessions.length > 0"
            type="button"
            @click="showTokenForm = false"
            class="p-1.5 rounded-lg text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800 transition-colors"
            title="Voltar para sessões salvas"
          >
            <ArrowLeft class="w-4 h-4" />
          </button>
        </div>

        <form @submit.prevent="handleConnectSubmit" class="space-y-4 pt-1">
          <Input
            v-model="token"
            label="Access Token"
            placeholder="Cole seu token de acesso aqui..."
            type="password"
            id="token-input"
          />

          <Button variant="primary" size="lg" class="w-full" type="submit" :disabled="!token.trim()">
            <span>Conectar</span>
            <ArrowRight class="w-4 h-4" />
          </Button>

          <button
            v-if="sessions && sessions.length > 0"
            type="button"
            @click="showTokenForm = false"
            class="w-full text-center text-xs text-zinc-400 hover:text-zinc-200 py-1 transition-colors flex items-center justify-center gap-1.5"
          >
            <ArrowLeft class="w-3.5 h-3.5" />
            <span>Voltar para sessões salvas</span>
          </button>
        </form>
      </Card>
    </div>

  </div>
</template>
