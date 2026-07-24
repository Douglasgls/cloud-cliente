<script setup lang="ts">
import { Loader2 } from '@lucide/vue'
import Card from '../ui/Card.vue'

defineProps<{
  stepMessage: string
}>()
</script>

<template>
  <div class="flex flex-col items-center justify-center min-h-[380px] max-w-sm mx-auto p-4 select-none">
    <Card class="w-full text-center space-y-6 p-8 border-indigo-500/30">
      <div class="relative inline-flex items-center justify-center">
        <div class="absolute inset-0 rounded-full bg-indigo-500/20 blur-md animate-pulse"></div>
        <div class="p-4 rounded-2xl bg-indigo-500/10 border border-indigo-500/30 text-indigo-400 relative">
          <Loader2 class="w-8 h-8 animate-spin" />
        </div>
      </div>

      <div class="space-y-2">
        <h3 class="text-sm font-bold text-zinc-100">Estabelecendo Conexão</h3>
        <p class="text-xs text-indigo-400 font-medium tracking-wide animate-pulse">
          {{ stepMessage || 'Carregando...' }}
        </p>
      </div>

      <div class="w-full bg-zinc-800/80 rounded-full h-1.5 overflow-hidden">
        <div class="bg-gradient-to-r from-indigo-500 to-emerald-400 h-full w-full animate-pulse"></div>
      </div>
    </Card>
  </div>
</template>
