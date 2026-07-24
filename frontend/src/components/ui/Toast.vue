<script setup lang="ts">
import { CheckCircle2 } from '@lucide/vue'

defineProps<{
  show: boolean
  message: string
}>()
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition duration-200 ease-out transform"
      enter-from-class="translate-y-2 opacity-0"
      enter-to-class="translate-y-0 opacity-100"
      leave-active-class="transition duration-150 ease-in transform"
      leave-from-class="translate-y-0 opacity-100"
      leave-to-class="translate-y-2 opacity-0"
    >
      <div
        v-if="show"
        class="fixed bottom-5 right-5 z-50 flex items-center gap-2.5 px-4 py-2.5 bg-zinc-900 border border-zinc-700/80 rounded-xl shadow-2xl text-xs font-medium text-emerald-400"
      >
        <CheckCircle2 class="w-4 h-4 text-emerald-400" />
        <span>{{ message }}</span>
      </div>
    </Transition>
  </Teleport>
</template>
