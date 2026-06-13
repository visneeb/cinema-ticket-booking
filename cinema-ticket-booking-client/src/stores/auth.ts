import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { onAuthStateChanged, signInWithPopup, signOut, type User } from 'firebase/auth'
import { auth, googleProvider } from '@/firebase'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const isReady = ref(false)
  const isAuthenticated = computed(() => user.value !== null)

  const ready = new Promise<void>((resolve) => {
    const _unsubscribe = onAuthStateChanged(auth, (u) => {
      user.value = u
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
