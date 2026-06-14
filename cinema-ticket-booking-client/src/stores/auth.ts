import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { onAuthStateChanged, signInWithPopup, signOut, type User } from 'firebase/auth'
import { auth, googleProvider } from '@/firebase'
import api from '@/services/api/api'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const role = ref<string | null>(null)
  const isReady = ref(false)
  const isAuthenticated = computed(() => user.value !== null)
  const isAdmin = computed(() => role.value === 'admin')

  // Upsert user profile to MongoDB and persist the returned role
  async function syncUser() {
    try {
      const res = await api.post('/api/users/me')
      role.value = res.data.role ?? null
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
      } else {
        role.value = null
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

  return { user, role, isReady, isAuthenticated, isAdmin, ready, loginWithGoogle, logout }
})
