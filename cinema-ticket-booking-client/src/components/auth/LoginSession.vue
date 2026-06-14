<script setup lang="ts">
import { ref } from 'vue'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const loading = ref(false)
const error = ref('')

async function handleLogin() {
  loading.value = true
  error.value = ''
  try {
    await auth.loginWithGoogle()
  } catch (e: unknown) {
    // user dismissed popup or network error — don't show error for popup-closed
    const code = (e as { code?: string })?.code
    if (code !== 'auth/popup-closed-by-user' && code !== 'auth/cancelled-popup-request') {
      error.value = 'Sign in failed. Please try again.'
    }
  } finally {
    loading.value = false
  }
}

async function handleLogout() {
  try {
    await auth.logout()
  } catch {
    error.value = 'Sign out failed. Please try again.'
  }
}
</script>

<template>
  <!-- Logged out -->
  <div v-if="!auth.isAuthenticated" class="session">
    <button class="btn-google" :disabled="loading" @click="handleLogin">
      <img
        src="https://www.gstatic.com/firebasejs/ui/2.0.0/images/auth/google.svg"
        alt="Google"
        class="google-icon"
      />
      {{ loading ? 'Signing in…' : 'Sign in with Google' }}
    </button>
    <span v-if="error" class="error-msg">{{ error }}</span>
  </div>

  <!-- Logged in -->
  <div v-else class="session">
    <img
      v-if="auth.user?.photoURL"
      class="avatar"
      :src="auth.user.photoURL"
      :alt="auth.user?.displayName ?? 'User'"
    />
    <span v-else class="avatar-fallback">{{ auth.user?.displayName?.charAt(0) ?? '?' }}</span>
    <span class="display-name">{{ auth.user?.displayName }}</span>
    <button class="btn-signout" @click="handleLogout">Sign out</button>
    <span v-if="error" class="error-msg">{{ error }}</span>
  </div>
</template>

<style scoped>
.session {
  display: flex;
  align-items: center;
  gap: 12px;
}

.btn-google {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border: 1px solid #dadce0;
  border-radius: 4px;
  background: #fff;
  font-size: 14px;
  font-weight: 500;
  color: #3c4043;
  cursor: pointer;
  transition: background 0.15s;
}

.btn-google:hover:not(:disabled) {
  background: #f7f8f8;
}

.btn-google:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.google-icon {
  width: 18px;
  height: 18px;
}

.avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  object-fit: cover;
}

.display-name {
  font-size: 14px;
  font-weight: 500;
}

.btn-signout {
  padding: 6px 12px;
  border: 1px solid #dadce0;
  border-radius: 4px;
  background: #fff;
  font-size: 13px;
  cursor: pointer;
  transition: background 0.15s;
}

.btn-signout:hover {
  background: #f7f8f8;
}

.avatar-fallback {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: #4285f4;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  font-weight: 600;
  flex-shrink: 0;
}

.error-msg {
  font-size: 12px;
  color: #d93025;
}
</style>
