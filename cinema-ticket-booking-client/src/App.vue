<script setup lang="ts">
import LoginSession from '@/components/auth/LoginSession.vue'
</script>

<template>
  <header class="app-header">
    <span class="app-title"> Cinema Booking</span>
    <LoginSession />
  </header>

  <main class="app-main">
    <RouterView />
  </main>
</template>

<style scoped>
.app-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 24px;
  border-bottom: 1px solid #e0e0e0;
  background: #fff;
}

.app-title {
  font-size: 18px;
  font-weight: 700;
}

.app-main {
  padding: 24px;
}
</style>
