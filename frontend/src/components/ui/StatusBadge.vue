<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  status: 'running' | 'error' | 'disabled' | 'connecting'
  label?: string
}>()

const statusConfig = computed(() => {
  switch (props.status) {
    case 'running':
      return {
        bg: 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20',
        dot: 'bg-emerald-400 animate-pulse',
        text: props.label || 'Executando'
      }
    case 'error':
      return {
        bg: 'bg-rose-500/10 text-rose-400 border-rose-500/20',
        dot: 'bg-rose-400',
        text: props.label || 'Erro ao iniciar'
      }
    case 'connecting':
      return {
        bg: 'bg-amber-500/10 text-amber-400 border-amber-500/20',
        dot: 'bg-amber-400 animate-ping',
        text: props.label || 'Conectando...'
      }
    case 'disabled':
    default:
      return {
        bg: 'bg-zinc-800/80 text-zinc-400 border-zinc-700/50',
        dot: 'bg-zinc-500',
        text: props.label || 'Desabilitado'
      }
  }
})
</script>

<template>
  <span :class="['inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium border transition-all duration-200', statusConfig.bg]">
    <span :class="['w-1.5 h-1.5 rounded-full', statusConfig.dot]"></span>
    {{ statusConfig.text }}
  </span>
</template>
