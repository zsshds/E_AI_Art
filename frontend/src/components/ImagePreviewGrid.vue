<script setup lang="ts">
defineProps<{
  images: string[]
  loading: boolean
}>()
</script>

<template>
  <div class="image-grid">
    <div v-if="loading" class="grid-loading">
      <div class="spinner"></div>
      <p>正在生成预览图...</p>
    </div>

    <div v-else-if="images.length === 0" class="grid-empty">
      <p>输入测试提示词，点击「生成预览」查看效果</p>
    </div>

    <div v-else class="grid-images">
      <div v-for="(url, i) in images" :key="i" class="grid-item">
        <img :src="url" :alt="`Preview ${i + 1}`" />
        <span class="grid-label">#{{ i + 1 }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.image-grid {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  padding: 16px;
  min-height: 300px;
}

.grid-loading, .grid-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 300px;
  color: var(--color-text-muted);
  font-size: 14px;
}

.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  margin-bottom: 12px;
}
@keyframes spin { to { transform: rotate(360deg); } }

.grid-images {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}

.grid-item {
  position: relative;
  border-radius: var(--radius);
  overflow: hidden;
  aspect-ratio: 1;
  background: var(--color-bg);
}

.grid-item img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.grid-label {
  position: absolute;
  bottom: 6px;
  left: 6px;
  padding: 2px 8px;
  background: rgba(0,0,0,0.6);
  color: white;
  border-radius: 4px;
  font-size: 12px;
}
</style>
