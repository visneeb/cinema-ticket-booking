<script setup lang="ts">
import type { BookingItem, MovieItem, UserItem } from '@/types/admin'
import { movieTitle, userEmail, fmtDate } from '@/utils/formatters'

defineProps<{
  bookings: BookingItem[]
  movies: MovieItem[]
  users: UserItem[]
}>()
</script>

<template>
  <div class="table-wrap">
    <table class="booking-table">
      <thead>
        <tr>
          <th>Booking ID</th>
          <th>Movie</th>
          <th>User</th>
          <th>Seat ID</th>
          <th>Status</th>
          <th>Created At</th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="bookings.length === 0">
          <td colspan="6" class="empty-row">No bookings found.</td>
        </tr>
        <tr v-for="b in bookings" :key="b.id">
          <td class="id-cell">{{ b.id }}</td>
          <td>{{ movieTitle(movies, b.showtime_id) }}</td>
          <td>{{ userEmail(users, b.user_id) }}</td>
          <td class="id-cell">{{ b.seat_id }}</td>
          <td>
            <span :class="['status-badge', `status-badge--${b.status.toLowerCase()}`]">
              {{ b.status }}
            </span>
          </td>
          <td>{{ fmtDate(b.created_at) }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.table-wrap {
  overflow-x: auto;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
}
.booking-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.booking-table th {
  background: #f1f5f9;
  padding: 10px 14px;
  text-align: left;
  font-weight: 600;
  color: #475569;
  white-space: nowrap;
}
.booking-table td {
  padding: 9px 14px;
  border-top: 1px solid #f1f5f9;
  color: #1e293b;
}
.booking-table tr:hover td {
  background: #f8fafc;
}
.id-cell {
  font-family: monospace;
  font-size: 11px;
  color: #64748b;
}
.empty-row {
  text-align: center;
  color: #94a3b8;
  padding: 24px;
}
.status-badge {
  display: inline-block;
  padding: 2px 10px;
  border-radius: 99px;
  font-size: 11px;
  font-weight: 600;
}
.status-badge--booked {
  background: #dcfce7;
  color: #166534;
}
.status-badge--locked {
  background: #dbeafe;
  color: #1d4ed8;
}
</style>
