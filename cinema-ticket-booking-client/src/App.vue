<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'
import LoginSession from '@/components/auth/LoginSession.vue'

const authStore = useAuthStore()
</script>

<template>
  <header class="app-header">
    <div class="header-left">
      <RouterLink :to="{ name: 'home' }" class="app-title">Cinema Booking</RouterLink>
      <RouterLink v-if="authStore.isAdmin" :to="{ name: 'admin' }" class="admin-link"
        >Admin Dashboard</RouterLink
      >
    </div>
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

.header-left {
  display: flex;
  align-items: center;
  gap: 20px;
}

.app-title {
  font-size: 18px;
  font-weight: 700;
  color: inherit;
  text-decoration: none;
}

.admin-link {
  font-size: 13px;
  font-weight: 600;
  color: #2563eb;
  text-decoration: none;
  padding: 4px 12px;
  border: 1px solid #bfdbfe;
  border-radius: 6px;
  background: #eff6ff;
}

.admin-link:hover {
  background: #dbeafe;
}

.app-main {
  padding: 24px;
}
</style>
