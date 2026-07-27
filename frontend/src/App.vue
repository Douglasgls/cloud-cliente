<script setup lang="ts">
import Header from './components/views/Header.vue'
import ConnectView from './components/views/ConnectView.vue'
import LoadingView from './components/views/LoadingView.vue'
import ConnectedView from './components/views/ConnectedView.vue'
import Toast from './components/ui/Toast.vue'

import { useConnection } from './composables/useConnection'
import { useForwardings } from './composables/useForwardings'

const {
  isConnected,
  isConnecting,
  hasSession,
  sessions,
  stepMessage,
  errorMessage,
  connectionInfo,
  connect,
  reconnectSession,
  forgetSession,
  disconnect
} = useConnection()

const {
  forwardings,
  toastMessage,
  showToast,
  addForwarding,
  updateForwarding,
  deleteForwarding,
  toggleForwarding,
  copyText,
  openAppURL
} = useForwardings()
</script>

<template>
  <div class="h-screen w-screen flex flex-col bg-zinc-950 text-zinc-100 overflow-hidden select-none">
    <!-- Header -->
    <Header :is-connected="isConnected" />

    <!-- Main View Switch -->
    <main class="flex-1 overflow-hidden relative">
      <!-- Loading View -->
      <LoadingView
        v-if="isConnecting"
        :step-message="stepMessage"
      />

      <!-- Connected View -->
      <ConnectedView
        v-else-if="isConnected"
        :connection-info="connectionInfo"
        :forwardings="forwardings"
        @disconnect="disconnect"
        @add-forwarding="(p) => addForwarding(p.name, p.remotePort, p.localPort)"
        @update-forwarding="(p) => updateForwarding(p.id, p.name, p.remotePort, p.localPort, p.enabled)"
        @delete-forwarding="deleteForwarding"
        @toggle-forwarding="(p) => toggleForwarding(p.id, p.enabled)"
        @copy-text="copyText"
        @open-u-r-l="openAppURL"
      />

      <!-- Connect View -->
      <ConnectView
        v-else
        :has-session="hasSession"
        :sessions="sessions"
        :error-message="errorMessage"
        @connect="connect"
        @reconnect-session="reconnectSession"
        @forget-session="forgetSession"
      />
    </main>

    <!-- Toast Notification -->
    <Toast :show="showToast" :message="toastMessage" />
  </div>
</template>
