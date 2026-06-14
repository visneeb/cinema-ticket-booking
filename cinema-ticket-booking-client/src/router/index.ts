import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import ShowtimeView from '@/views/ShowtimeView.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'home',
      component: ShowtimeView,
    },
    {
      path: '/showtimes/:showtimeId',
      name: 'showtime',
      component: ShowtimeView,
      meta: { requiresAuth: true },
    },
  ],
})

router.beforeEach(async (to) => {
  const authStore = useAuthStore()
  // main.ts already awaits ready before mount, but guard here for safety
  await authStore.ready

  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    return { name: 'home' }
  }
})

export default router
