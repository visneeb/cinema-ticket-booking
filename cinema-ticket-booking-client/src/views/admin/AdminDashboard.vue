<script setup lang="ts">
import { onMounted } from 'vue'
import { useAdminBookings } from '@/composables/useAdminBookings'
import AdminHeader from '@/components/admin/AdminHeader.vue'
import BookingFilterBar from '@/components/admin/BookingFilterBar.vue'
import BookingTable from '@/components/admin/BookingTable.vue'
import BookingPagination from '@/components/admin/BookingPagination.vue'

const {
  filters,
  bookings,
  total,
  movies,
  users,
  loading,
  error,
  totalPages,
  fetchData,
  applyFilters,
  resetFilters,
  prevPage,
  nextPage,
} = useAdminBookings()

onMounted(fetchData)
</script>

<template>
  <div class="admin-view">
    <AdminHeader :total="total" />

    <BookingFilterBar
      v-model:filters="filters"
      :movies="movies"
      :users="users"
      @apply="applyFilters"
      @reset="resetFilters"
    />

    <p v-if="loading" class="state-msg">Loading…</p>
    <p v-else-if="error" class="state-msg state-msg--error">{{ error }}</p>

    <template v-else>
      <BookingTable :bookings="bookings" :movies="movies" :users="users" />
      <BookingPagination
        :page="filters.page"
        :total-pages="totalPages"
        @prev="prevPage"
        @next="nextPage"
      />
    </template>
  </div>
</template>

<style scoped>
.admin-view {
  max-width: 1100px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 24px;
}
.state-msg {
  color: #64748b;
  text-align: center;
}
.state-msg--error {
  color: #ef4444;
}
</style>
