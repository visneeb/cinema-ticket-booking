<script setup lang="ts">
import { computed, ref, watch, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useShowtimeSocket, type WsStatus } from '@/composables/useShowtimeSocket'
import { useShowtimes } from '@/composables/useShowtimes'
import { useSeats } from '@/composables/useSeats'
import { useBooking } from '@/composables/useBooking'
import { formatShowtimeDate } from '@/utils/validate'
import SeatMap from '@/components/seats/SeatMap.vue'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const showtimeId = computed(() => (route.params.showtimeId as string) || '')
const isHome = computed(() => !showtimeId.value)

// ── Home page ──────────────────────────────────────────────────────────────
const {
  showtimes,
  loading: showtimesLoading,
  error: showtimesError,
  fetchShowtimes,
} = useShowtimes()

function goToShowtime(id: string) {
  router.push({ name: 'showtime', params: { showtimeId: id } })
}

// ── Showtime page ──────────────────────────────────────────────────────────
const { seats, loading: seatsLoading, error: fetchError, fetchSeats } = useSeats(showtimeId)

const {
  mySelectedSeatIds,
  selectedSeats,
  myReservedSeatIds,
  reservedSeats,
  reserveSecondsLeft,
  reserving,
  confirming,
  actionError,
  countdownDisplay,
  onSeatClick,
  confirmReserve,
  cancelSelection,
  cancelReservation,
  confirmBooking,
  onSeatEvent,
  restoreLockState,
  resetBookingState,
} = useBooking(showtimeId, seats, fetchSeats)

// WebSocket — tracked reactively so the banner updates correctly
const wsStatus = ref<WsStatus>('open')
let socketCleanup: (() => void) | null = null

function connectSocket(id: string) {
  if (socketCleanup) socketCleanup()
  if (!id) {
    wsStatus.value = 'open'
    return
  }
  const { lastEvent, wsStatus: sockStatus, cleanup } = useShowtimeSocket(id)
  socketCleanup = cleanup
  watch(
    sockStatus,
    (s) => {
      wsStatus.value = s
    },
    { immediate: true },
  )
  watch(lastEvent, (event) => {
    if (event) onSeatEvent(event)
  })
}

// Bootstrap + re-bootstrap on navigation (home ↔ showtime)
watch(
  showtimeId,
  (id) => {
    if (!id) {
      fetchShowtimes()
    } else {
      connectSocket(id)
      fetchSeats().then(() => restoreLockState())
    }
  },
  { immediate: true },
)

// Re-sync on auth change (showtime page only)
watch(
  () => authStore.isAuthenticated,
  (authenticated) => {
    if (isHome.value) return
    if (!authenticated) {
      resetBookingState()
      fetchSeats()
    } else {
      restoreLockState()
    }
  },
)

onUnmounted(() => {
  if (socketCleanup) socketCleanup()
})
</script>

<template>
  <!-- ══ HOME: showtime list ══════════════════════════════════════════════ -->
  <div v-if="isHome" class="home-view">
    <div class="view-header">
      <h1 class="title">🎬 Now Showing</h1>
      <p class="subtitle">Choose a showtime to pick your seat.</p>
    </div>

    <div v-if="showtimesLoading" class="state-msg">Loading showtimes…</div>
    <div v-else-if="showtimesError" class="state-msg state-msg--error">{{ showtimesError }}</div>

    <div v-else-if="showtimes.length === 0" class="state-msg">No showtimes available.</div>

    <div v-else class="showtime-list">
      <button
        v-for="st in showtimes"
        :key="st.id"
        class="showtime-card"
        @click="goToShowtime(st.id)"
      >
        <div class="card-movie">{{ st.title }}</div>
        <div class="card-desc">{{ st.description }}</div>
        <div class="card-time">🕐 {{ formatShowtimeDate(st.starts_at) }}</div>
        <div class="card-cta">Select Seats →</div>
      </button>
    </div>
  </div>

  <!-- ══ SHOWTIME: seat map ════════════════════════════════════════════════ -->
  <div v-else class="showtime-view">
    <div class="view-header">
      <button class="back-btn" @click="router.push({ name: 'home' })">← Back</button>
      <h1 class="title">Select Your Seat</h1>
    </div>

    <!-- WebSocket banner -->
    <div v-if="wsStatus !== 'open'" class="ws-banner" :class="`ws-banner--${wsStatus}`">
      <span v-if="wsStatus === 'connecting'">⟳ Connecting to live updates…</span>
      <span v-else-if="wsStatus === 'closed'">⚠ Reconnecting to live updates…</span>
      <span v-else>✕ Live update connection failed</span>
    </div>

    <div v-if="seatsLoading" class="state-msg">Loading seats…</div>
    <div v-else-if="fetchError" class="state-msg state-msg--error">{{ fetchError }}</div>
    <SeatMap
      v-else
      :seats="seats"
      :selected-seat-ids="mySelectedSeatIds"
      :my-locked-seat-ids="myReservedSeatIds"
      @seat-click="onSeatClick"
    />

    <div v-if="!authStore.isAuthenticated" class="signin-notice">
      Sign in to select and reserve seats.
    </div>

    <!-- Phase 1: seats selected, awaiting reservation -->
    <Transition name="slide-up">
      <div
        v-if="mySelectedSeatIds.length > 0 && myReservedSeatIds.length === 0"
        class="booking-panel"
      >
        <div class="panel-top">
          <span class="panel-seat">{{ mySelectedSeatIds.length }} seat(s) selected</span>
          <span class="panel-tag panel-tag--selected">Selected</span>
        </div>
        <div class="panel-seat-list">
          <span v-for="seat in selectedSeats" :key="seat.id" class="seat-chip seat-chip--selected">
            {{ seat.label }}
          </span>
        </div>
        <p class="panel-hint">Reserve these seats to start your 5-minute booking window.</p>
        <div class="panel-actions">
          <button class="btn btn-cancel" @click="cancelSelection">Cancel</button>
          <button class="btn btn-reserve" :disabled="reserving" @click="confirmReserve">
            {{ reserving ? 'Reserving…' : `Reserve ${mySelectedSeatIds.length} Seat(s)` }}
          </button>
        </div>
      </div>
    </Transition>

    <!-- Phase 2: seats reserved, awaiting booking -->
    <Transition name="slide-up">
      <div v-if="myReservedSeatIds.length > 0" class="booking-panel">
        <div class="panel-top">
          <span class="panel-seat">{{ myReservedSeatIds.length }} seat(s) reserved</span>
          <span class="panel-timer" :class="{ 'panel-timer--urgent': reserveSecondsLeft < 60 }">
            ⏱ {{ countdownDisplay }}
          </span>
        </div>
        <div class="panel-seat-list">
          <span v-for="seat in reservedSeats" :key="seat.id" class="seat-chip seat-chip--locked">
            {{ seat.label }}
          </span>
        </div>
        <p class="panel-hint">Complete your booking before the timer expires.</p>
        <div class="panel-actions">
          <button class="btn btn-cancel" @click="cancelReservation">Cancel Reservation</button>
          <button class="btn btn-confirm" :disabled="confirming" @click="confirmBooking">
            {{ confirming ? 'Confirming…' : 'Confirm Booking' }}
          </button>
        </div>
      </div>
    </Transition>

    <p v-if="actionError" class="action-error">⚠ {{ actionError }}</p>
  </div>
</template>

<style scoped src="./ShowtimeView.css" />
