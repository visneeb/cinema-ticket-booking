<script setup lang="ts">
import type { Seat } from '@/types/seat'
import SeatButton from './SeatButton.vue'

defineProps<{
  seats: Seat[]
  selectedSeatIds: string[]
  myLockedSeatIds: string[]
}>()

defineEmits<{
  (e: 'seat-click', seatId: string): void
}>()

const legend = [
  { label: 'Available', cls: 'swatch--available' },
  { label: 'Selected', cls: 'swatch--selected' },
  { label: 'Reserved', cls: 'swatch--locked' },
  { label: 'Your reservation', cls: 'swatch--mine' },
  { label: 'Booked', cls: 'swatch--booked' },
]
</script>

<template>
  <div class="seat-map">
    <!-- Screen indicator -->
    <div class="screen-wrap">
      <div class="screen" />
      <span class="screen-label">SCREEN</span>
    </div>

    <!-- Seat grid -->
    <div class="seat-grid">
      <SeatButton
        v-for="seat in seats"
        :key="seat.id"
        :seat="seat"
        :is-selected="selectedSeatIds.includes(seat.id)"
        :is-my-lock="myLockedSeatIds.includes(seat.id)"
        @click="$emit('seat-click', $event)"
      />
    </div>

    <!-- Legend -->
    <div class="legend">
      <div v-for="item in legend" :key="item.label" class="legend-item">
        <span class="swatch" :class="item.cls" />
        {{ item.label }}
      </div>
    </div>
  </div>
</template>

<style scoped>
.seat-map {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 32px;
  width: 100%;
}

/* Screen */
.screen-wrap {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  width: 55%;
}
.screen {
  width: 100%;
  height: 6px;
  background: linear-gradient(to bottom, #94a3b8, #cbd5e1);
  border-radius: 3px;
  box-shadow: 0 3px 12px rgba(0, 0, 0, 0.12);
}
.screen-label {
  font-size: 11px;
  letter-spacing: 0.25em;
  color: #94a3b8;
  font-weight: 600;
}

/* Seat grid */
.seat-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  justify-content: center;
  max-width: 640px;
  padding: 0 8px;
}

/* Legend */
.legend {
  display: flex;
  gap: 20px;
  flex-wrap: wrap;
  justify-content: center;
}
.legend-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #475569;
}
.swatch {
  width: 16px;
  height: 16px;
  border-radius: 4px;
  flex-shrink: 0;
}
.swatch--available {
  background: #22c55e;
}
.swatch--selected {
  background: #7c3aed;
}
.swatch--locked {
  background: #f59e0b;
}
.swatch--mine {
  background: #3b82f6;
}
.swatch--booked {
  background: #9ca3af;
}
</style>
