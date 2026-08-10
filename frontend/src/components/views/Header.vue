<script setup lang="ts">
import { Cloud, ShieldCheck, ShieldAlert } from '@lucide/vue'

defineProps<{
  isConnected: boolean
}>()
</script>

<template>
  <header class="flex items-center justify-between px-6 py-4 bg-zinc-900/80 backdrop-blur-md border-b border-zinc-800/80 select-none">
    <div class="flex items-center gap-3">
      <div class="p-2 rounded-xl bg-indigo-500/10 border border-indigo-500/20 text-indigo-400 shadow-sm">
        <Cloud class="w-5 h-5" />
      </div>
      <div>
        <h1 class="text-sm font-bold tracking-tight text-zinc-100">Cloud Client</h1>
        <p class="text-[11px] text-zinc-400 font-medium">Gestão de Forwardings TCP</p>
      </div>
    </div>

    <div>
      <div v-if="isConnected" class="flex items-center gap-2 px-3 py-1.5 rounded-full bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-xs font-semibold">
        <ShieldCheck class="w-4 h-4 text-emerald-400" />
        <span class="relative flex h-2 w-2">
          <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
          <span class="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
        </span>
        <span>Conectado</span>
      </div>
      <div v-else class="flex items-center gap-2 px-3 py-1.5 rounded-full bg-zinc-800/80 border border-zinc-700/50 text-zinc-400 text-xs font-medium">
        <ShieldAlert class="w-4 h-4 text-zinc-400" />
        <span>Desconectado</span>
      </div>
    </div>
  </header>
</template>
