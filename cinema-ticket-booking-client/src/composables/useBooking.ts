import { ref, computed, onUnmounted } from 'vue'
import type { Ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { lockSeat, unlockSeat, bookSeat, getMyLocks } from '@/services/showtimeService'
import type { Seat, SeatEvent } from '@/types/seat'

export function useBooking(
  showtimeId: Ref<string>,
  seats: Ref<Seat[]>,
  fetchSeats: () => Promise<void>,
) {
  const authStore = useAuthStore()

  // ── Phase 1: seats selected (pending reservation) ──────────────────────────
  const mySelectedSeatIds = ref<string[]>([])
  const selectedSeats = computed(() =>
    seats.value.filter((s) => mySelectedSeatIds.value.includes(s.id)),
  )
  const selecting = ref<string | null>(null)
  const expectingLock = new Set<string>()

  // ── Phase 2: seats reserved (5-minute countdown) ───────────────────────────
  const myReservedSeatIds = ref<string[]>([])
  const reservedSeats = computed(() =>
    seats.value.filter((s) => myReservedSeatIds.value.includes(s.id)),
  )
  const reserveSecondsLeft = ref(0)
  const reserving = ref(false)
  const confirming = ref(false)
  const actionError = ref('')

  let countdownTimer: ReturnType<typeof setInterval> | null = null

  function startCountdown(seconds = 300) {
    stopCountdown()
    reserveSecondsLeft.value = seconds
    countdownTimer = setInterval(() => {
      reserveSecondsLeft.value--
      if (reserveSecondsLeft.value <= 0) clearReservation()
    }, 1000)
  }

  function stopCountdown() {
    if (countdownTimer) {
      clearInterval(countdownTimer)
      countdownTimer = null
    }
  }

  function clearReservation() {
    myReservedSeatIds.value = []
    reserveSecondsLeft.value = 0
    stopCountdown()
  }

  function resetBookingState() {
    mySelectedSeatIds.value = []
    clearReservation()
  }

  const countdownDisplay = computed(() => {
    const m = Math.floor(reserveSecondsLeft.value / 60)
    const s = reserveSecondsLeft.value % 60
    return `${m}:${s.toString().padStart(2, '0')}`
  })

  // ── WS event handler ───────────────────────────────────────────────────────
  function onSeatEvent(event: SeatEvent) {
    const seat = seats.value.find((s) => s.id === event.seat_id)
    if (seat) seat.status = event.status

    // Another client locked a seat we had selected — drop it from our selection.
    if (mySelectedSeatIds.value.includes(event.seat_id) && event.status !== 'AVAILABLE') {
      if (expectingLock.has(event.seat_id)) {
        expectingLock.delete(event.seat_id) // our own LOCKED echo — ignore
      } else {
        mySelectedSeatIds.value = mySelectedSeatIds.value.filter((id) => id !== event.seat_id)
        actionError.value = 'A selected seat was just taken. Please choose another.'
      }
    }

    // A reserved seat's TTL expired — remove it from our list.
    if (myReservedSeatIds.value.includes(event.seat_id) && event.status === 'AVAILABLE') {
      myReservedSeatIds.value = myReservedSeatIds.value.filter((id) => id !== event.seat_id)
      if (myReservedSeatIds.value.length === 0) clearReservation()
    }
  }

  // ── Restore lock state after page refresh ──────────────────────────────────
  async function restoreLockState() {
    if (!authStore.isAuthenticated || !showtimeId.value) return
    try {
      const locks = await getMyLocks(showtimeId.value)
      const active = locks.filter((l) => l.seconds_left > 0)
      if (active.length > 0) {
        myReservedSeatIds.value = active.map((l) => l.seat_id)
        const minSecs = Math.min(...active.map((l) => l.seconds_left))
        startCountdown(minSecs)
      }
    } catch {
      // Non-fatal — user simply won't see the booking panel restored
    }
  }

  // ── Phase 1 actions ────────────────────────────────────────────────────────
  async function onSeatClick(seatId: string) {
    if (!authStore.isAuthenticated) {
      actionError.value = 'Please sign in to book a seat.'
      return
    }
    if (myReservedSeatIds.value.length > 0) return
    if (selecting.value) return

    actionError.value = ''
    selecting.value = seatId

    try {
      if (mySelectedSeatIds.value.includes(seatId)) {
        await unlockSeat(showtimeId.value, seatId)
        mySelectedSeatIds.value = mySelectedSeatIds.value.filter((id) => id !== seatId)
        const seat = seats.value.find((s) => s.id === seatId)
        if (seat) seat.status = 'AVAILABLE'
      } else {
        await lockSeat(showtimeId.value, seatId)
        expectingLock.add(seatId)
        mySelectedSeatIds.value.push(seatId)
        const seat = seats.value.find((s) => s.id === seatId)
        if (seat) seat.status = 'SELECTED'
      }
    } catch {
      actionError.value = 'That seat was just taken. Please choose another.'
      await fetchSeats()
    } finally {
      selecting.value = null
    }
  }

  // ── Phase 1 → 2 actions ────────────────────────────────────────────────────
  async function confirmReserve() {
    if (mySelectedSeatIds.value.length === 0) return
    const seatIds = [...mySelectedSeatIds.value]
    reserving.value = true
    actionError.value = ''

    const results = await Promise.allSettled(
      seatIds.map((seatId) =>
        lockSeat(showtimeId.value, seatId).then((secondsLeft) => ({ seatId, secondsLeft })),
      ),
    )

    const succeeded: string[] = []
    let minSecs = 300
    const failed: string[] = []
    results.forEach((result, i) => {
      if (result.status === 'fulfilled') {
        succeeded.push(result.value.seatId)
        if (result.value.secondsLeft < minSecs) minSecs = result.value.secondsLeft
      } else {
        const id = seatIds[i]
        if (id) failed.push(id)
      }
    })

    mySelectedSeatIds.value = []
    myReservedSeatIds.value = succeeded
    if (succeeded.length > 0) startCountdown(minSecs)
    if (failed.length > 0)
      actionError.value = `${failed.length} seat(s) could not be reserved — already taken.`

    reserving.value = false
  }

  function cancelSelection() {
    if (mySelectedSeatIds.value.length === 0) return
    const seatIds = [...mySelectedSeatIds.value]

    for (const seatId of seatIds) {
      const seat = seats.value.find((s) => s.id === seatId)
      if (seat) seat.status = 'AVAILABLE'
    }
    mySelectedSeatIds.value = []
    actionError.value = ''

    for (const seatId of seatIds) {
      unlockSeat(showtimeId.value, seatId).catch(() => {})
    }
  }

  // ── Phase 2 actions ────────────────────────────────────────────────────────
  function cancelReservation() {
    if (myReservedSeatIds.value.length === 0) return
    const seatIds = [...myReservedSeatIds.value]

    for (const seatId of seatIds) {
      const seat = seats.value.find((s) => s.id === seatId)
      if (seat) seat.status = 'AVAILABLE'
    }
    clearReservation()

    for (const seatId of seatIds) {
      unlockSeat(showtimeId.value, seatId).catch(() => {})
    }
  }

  // ── Phase 3 actions ────────────────────────────────────────────────────────
  async function confirmBooking() {
    if (myReservedSeatIds.value.length === 0) return
    actionError.value = ''
    confirming.value = true

    const seatIds = [...myReservedSeatIds.value]
    const results = await Promise.allSettled(
      seatIds.map((seatId) => bookSeat(showtimeId.value, seatId)),
    )

    const failed: string[] = []
    results.forEach((result, i) => {
      if (result.status === 'rejected') {
        const id = seatIds[i]
        if (id) failed.push(id)
      }
    })

    myReservedSeatIds.value = failed
    if (failed.length === 0) clearReservation()
    else actionError.value = `${failed.length} seat(s) could not be confirmed. Please try again.`

    confirming.value = false
  }

  onUnmounted(stopCountdown)

  return {
    // Phase 1 state
    mySelectedSeatIds,
    selectedSeats,
    selecting,
    // Phase 2 state
    myReservedSeatIds,
    reservedSeats,
    reserveSecondsLeft,
    reserving,
    confirming,
    actionError,
    countdownDisplay,
    // Actions
    onSeatClick,
    confirmReserve,
    cancelSelection,
    cancelReservation,
    confirmBooking,
    // Hooks for the view
    onSeatEvent,
    restoreLockState,
    resetBookingState,
  }
}
