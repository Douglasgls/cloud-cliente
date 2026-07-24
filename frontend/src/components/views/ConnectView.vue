<script setup lang="ts">
import { ref } from 'vue'
import { KeyRound, RotateCw, PlusCircle, ArrowRight, AlertCircle } from '@lucide/vue'
import Card from '../ui/Card.vue'
import Button from '../ui/Button.vue'
import Input from '../ui/Input.vue'

const props = defineProps<{
  hasSession: boolean
  errorMessage?: string
}>()

const emit = defineEmits<{
  (e: 'connect', token: string): void
  (e: 'reconnect'): void
  (e: 'newSession'): void
}>()

const token = ref('')
const showTokenForm = ref(!props.hasSession)

const handleConnectSubmit = () => {
  if (token.value.trim()) {
    emit('connect', token.value.trim())
  }
}
</script>

<template>
  <div class="flex flex-col items-center justify-center min-h-[420px] max-w-md mx-auto p-4 space-y-6">

    <!-- Error Alert -->
    <div v-if="errorMessage" class="w-full flex items-start gap-3 p-3.5 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-400 text-xs font-medium">
      <AlertCircle class="w-4 h-4 text-rose-400 flex-shrink-0 mt-0.5" />
      <span>{{ errorMessage }}</span>
    </div>

    <!-- State A: Saved Session Available -->
    <Card v-if="hasSession && !showTokenForm" class="w-full text-center space-y-5 border-indigo-500/30">
      <div class="inline-flex p-3 rounded-2xl bg-indigo-500/10 border border-indigo-500/20 text-indigo-400">
        <KeyRound class="w-7 h-7" />
      </div>

      <div class="space-y-1">
        <h2 class="text-base font-bold text-zinc-100">Última conexão encontrada</h2>
        <p class="text-xs text-zinc-400">Deseja reconectar usando as credenciais salvas?</p>
      </div>

      <div class="space-y-2.5 pt-2">
        <Button variant="primary" size="lg" class="w-full" @click="emit('reconnect')">
          <RotateCw class="w-4 h-4" />
          <span>Reconectar</span>
        </Button>

        <Button variant="ghost" size="md" class="w-full" @click="showTokenForm = true; emit('newSession')">
          <PlusCircle class="w-4 h-4" />
          <span>Nova Conexão</span>
        </Button>
      </div>
    </Card>

    <!-- State B: Token Form Input -->
    <Card v-else class="w-full space-y-5">
      <div class="flex items-center gap-3">
        <div class="p-2.5 rounded-xl bg-indigo-500/10 border border-indigo-500/20 text-indigo-400">
          <KeyRound class="w-5 h-5" />
        </div>
        <div>
          <h2 class="text-sm font-bold text-zinc-100">Nova Conexão</h2>
          <p class="text-xs text-zinc-400">Informe seu Access Token de autenticação</p>
        </div>
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
          v-if="hasSession"
          type="button"
          @click="showTokenForm = false"
          class="w-full text-center text-xs text-zinc-400 hover:text-zinc-200 py-1 transition-colors"
        >
          Voltar para sessão salva
        </button>
      </form>
    </Card>

  </div>
</template>
