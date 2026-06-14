<script setup lang="ts">
import type { BookingFilters, MovieItem, UserItem } from '@/types/admin'

const filters = defineModel<BookingFilters>('filters', { required: true })

defineProps<{
  movies: MovieItem[]
  users: UserItem[]
}>()

const emit = defineEmits<{
  apply: []
  reset: []
}>()
</script>

<template>
  <div class="filter-bar">
    <select v-model="filters.movie_id" class="filter-input">
      <option value="">All movies</option>
      <option v-for="m in movies" :key="m.id" :value="m.id">{{ m.title }}</option>
    </select>

    <select v-model="filters.user_id" class="filter-input">
      <option value="">All users</option>
      <option v-for="u in users" :key="u.uid" :value="u.uid">{{ u.email }}</option>
    </select>

    <input v-model="filters.date_from" type="date" class="filter-input" />
    <input v-model="filters.date_to" type="date" class="filter-input" />

    <select v-model="filters.status" class="filter-input">
      <option value="">All statuses</option>
      <option value="BOOKED">BOOKED</option>
      <option value="LOCKED">LOCKED</option>
    </select>

    <button class="btn-apply" @click="emit('apply')">Apply</button>
    <button class="btn-reset" @click="emit('reset')">Reset</button>
  </div>
</template>
