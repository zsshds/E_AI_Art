<script setup lang="ts">
import { computed, ref } from 'vue'

const props = defineProps<{
  modelValue: string
  placeholder?: string
  useTextarea?: boolean
  rows?: number
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const showPresets = ref(false)

const inputValue = computed({
  get: () => props.modelValue,
  set: (val: string) => emit('update:modelValue', val),
})
</script>

<template>
  <div class="preset-input">
    <textarea
      v-if="useTextarea"
      :value="inputValue"
      @input="(e) => emit('update:modelValue', (e.target as HTMLTextAreaElement).value)"
      class="text-input"
      :placeholder="placeholder || ''"
      :rows="rows || 2"
    ></textarea>
    <input
      v-else
      type="text"
      :value="inputValue"
      @input="(e) => emit('update:modelValue', (e.target as HTMLInputElement).value)"
      class="text-input"
      :placeholder="placeholder || ''"
    />

    <button v-if="$slots.presets" type="button" class="preset-toggle" @click="showPresets = !showPresets">
      <span class="toggle-arrow">{{ showPresets ? '▾' : '▸' }}</span>
      预设参考
    </button>

    <div v-if="showPresets && $slots.presets" class="preset-panel">
      <slot name="presets" />
    </div>
  </div>
</template>

<style scoped>
.preset-input {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.text-input {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  font-size: 14px;
  background: var(--color-surface);
  font-family: inherit;
  line-height: 1.5;
  resize: vertical;
}

.preset-toggle {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: var(--color-primary);
  background: none;
  border: none;
  cursor: pointer;
  padding: 2px 0;
  align-self: flex-start;
}
.preset-toggle:hover {
  text-decoration: underline;
}

.toggle-arrow {
  font-size: 10px;
  width: 12px;
}

.preset-panel {
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  background: var(--color-bg);
  padding: 12px;
  max-height: 280px;
  overflow-y: auto;
}
</style>
