import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { onAuthStateChanged, signInWithPopup, signOut, type User } from 'firebase/auth'
import { auth, googleProvider } from '@/firebase'
import api from '@/services/api/api'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const isReady = ref(false)
  const isAuthenticated = computed(() => user.value !== null)

  // Upsert user profile to MongoDB — called on login and on session restore
  async function syncUser() {
    try {
      await api.post('/api/users/me')
    } catch (e) {
      console.warn('[auth] syncUser failed:', e)
    }
  }

  const ready = new Promise<void>((resolve) => {
    const _unsubscribe = onAuthStateChanged(auth, async (u) => {
      user.value = u
      if (u) {
        // Persist/update user in cinema.users on every sign-in or page reload
        await syncUser()
      }
      if (!isReady.value) {
        isReady.value = true
        resolve()
      }
    })
  })

  async function loginWithGoogle() {
    const cred = await signInWithPopup(auth, googleProvider)
    return cred.user
  }

  async function logout() {
    await signOut(auth)
  }

  return { user, isReady, isAuthenticated, ready, loginWithGoogle, logout }
})
