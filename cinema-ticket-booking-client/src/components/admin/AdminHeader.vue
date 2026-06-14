<script setup lang="ts">
defineProps<{ total: number }>()
</script>

<template>
  <div class="admin-header">
    <h1 class="admin-title">Admin Dashboard</h1>
    <p class="admin-sub">All bookings · {{ total }} total</p>
  </div>
</template>

<style scoped>
.admin-header {
  text-align: center;
}
.admin-title {
  font-size: 26px;
  font-weight: 700;
  margin: 0 0 4px;
}
.admin-sub {
  margin: 0;
  color: #64748b;
  font-size: 14px;
}
</style>
