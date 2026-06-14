<script setup lang="ts">
import { computed } from 'vue'
import type { Seat } from '@/types/seat'

const props = defineProps<{
  seat: Seat
  /** True when this seat is selected by the user (Phase 1 — not yet locked) */
  isSelected: boolean
  /** True when this seat is the one the current user has locked (Phase 2) */
  isMyLock: boolean
}>()

const emit = defineEmits<{
  (e: 'click', seatId: string): void
}>()

/** LOCKED-by-others and BOOKED seats cannot be clicked */
const isDisabled = computed(
  () => props.seat.status === 'BOOKED' || (props.seat.status === 'LOCKED' && !props.isMyLock),
)

const statusClass = computed(() => {
  if (props.isMyLock) return 'seat--mine'
  if (props.isSelected) return 'seat--selected'
  return `seat--${props.seat.status.toLowerCase()}`
})

const title = computed(() => {
  if (props.seat.status === 'BOOKED') return `${props.seat.label} — Booked`
  if (props.seat.status === 'LOCKED' && !props.isMyLock)
    return `${props.seat.label} — Reserved by someone`
  if (props.isMyLock) return `${props.seat.label} — Your reservation`
  if (props.isSelected) return `${props.seat.label} — Selected (click again to deselect)`
  return `${props.seat.label} — Available`
})
</script>

<template>
  <button
    class="seat"
    :class="statusClass"
    :disabled="isDisabled"
    :title="title"
    @click="emit('click', seat.id)"
  >
    {{ seat.label }}
  </button>
</template>

<style scoped>
.seat {
  width: 52px;
  height: 52px;
  border-radius: 8px 8px 4px 4px;
  border: 2px solid transparent;
  font-size: 11px;
  font-weight: 700;
  cursor: pointer;
  transition:
    transform 0.1s,
    box-shadow 0.1s;
  display: flex;
  align-items: center;
  justify-content: center;
}

.seat:not(:disabled):hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.15);
}

/* AVAILABLE — green */
.seat--available {
  background: #22c55e;
  color: #fff;
  border-color: #16a34a;
}

/* LOCKED by someone else — amber */
.seat--locked {
  background: #f59e0b;
  color: #fff;
  border-color: #d97706;
  cursor: not-allowed;
  opacity: 0.8;
}

/* Selected — purple (Phase 1: chosen but not yet locked) */
.seat--selected {
  background: #7c3aed;
  color: #fff;
  border-color: #6d28d9;
  cursor: pointer;
  box-shadow: 0 0 0 3px rgba(124, 58, 237, 0.3);
}

/* My own reservation — blue (Phase 2: lock acquired) */
.seat--mine {
  background: #3b82f6;
  color: #fff;
  border-color: #1d4ed8;
  cursor: pointer;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.35);
}

/* BOOKED — red/grey */
.seat--booked {
  background: #9ca3af;
  color: #fff;
  border-color: #6b7280;
  cursor: not-allowed;
  opacity: 0.7;
}
</style>
