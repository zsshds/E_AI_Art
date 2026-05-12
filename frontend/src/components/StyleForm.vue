<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useAvailableModels } from '../composables/useAvailableModels'
import { listProjects, type Project } from '../api/project'
import ArtStyleSelector from './ArtStyleSelector.vue'
import GameTemplateSelector from './GameTemplateSelector.vue'

const props = defineProps<{
  modelValue: {
    name: string
    model: string
    art_style: string
    color_tone: string
    extra_tokens: string[]
    project_id: string
    reference_image_urls: string[]
  }
}>()

const emit = defineEmits<{
  'update:modelValue': [value: any]
}>()

const form = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val),
})

const { availableModels, loading: modelsLoading } = useAvailableModels()

const projects = ref<Project[]>([])
onMounted(async () => {
  try { projects.value = await listProjects() } catch { /* ignore */ }
})

function handleApplyTemplate(fields: Record<string, string>) {
  emit('update:modelValue', { ...form.value, ...fields })
}

function addToken(event: Event) {
  const input = event.target as HTMLInputElement
  const value = input.value.trim()
  const tokens = form.value.extra_tokens || []
  if (value && !tokens.includes(value)) {
    emit('update:modelValue', { ...form.value, extra_tokens: [...tokens, value] })
  }
  input.value = ''
}

function removeToken(index: number) {
  const tokens = [...(form.value.extra_tokens || [])]
  tokens.splice(index, 1)
  emit('update:modelValue', { ...form.value, extra_tokens: tokens })
}

function handleImageUpload(event: Event) {
  const input = event.target as HTMLInputElement
  const files = input.files
  if (!files || files.length === 0) return
  const urls = form.value.reference_image_urls || []
  const remaining = 10 - urls.length
  if (remaining <= 0) return
  const count = Math.min(files.length, remaining)
  for (let i = 0; i < count; i++) {
    const file = files[i]
    if (!file) continue
    const url = URL.createObjectURL(file)
    urls.push(url)
  }
  emit('update:modelValue', { ...form.value, reference_image_urls: [...urls] })
  input.value = ''
}

function removeReferenceImage(index: number) {
  const urls = [...(form.value.reference_image_urls || [])]
  urls.splice(index, 1)
  emit('update:modelValue', { ...form.value, reference_image_urls: urls })
}
</script>

<template>
  <form class="style-form" @submit.prevent>
    <!-- Style name -->
    <section class="form-section">
      <h3>风格名称 <span class="required">*</span></h3>
      <input
        type="text"
        :value="form.name"
        @input="(e) => emit('update:modelValue', { ...form, name: (e.target as HTMLInputElement).value })"
        placeholder="给你的风格起个名字"
        class="text-input"
      />
    </section>

    <!-- Project selector (admin only, used within StyleEditor which is admin-only) -->
    <section v-if="projects.length > 0" class="form-section">
      <h3>所属项目 <span class="hint">（可选，同项目成员可见此风格）</span></h3>
      <select
        :value="form.project_id"
        @change="(e) => emit('update:modelValue', { ...form, project_id: (e.target as HTMLSelectElement).value })"
        class="select-input"
      >
        <option value="">-- 无项目（全局可见） --</option>
        <option v-for="p in projects" :key="p.id" :value="p.id">{{ p.name }}</option>
      </select>
    </section>

    <!-- Game template quick apply -->
    <section class="form-section">
      <h3>快速模板 <span class="hint">（一键套用游戏场景预设）</span></h3>
      <GameTemplateSelector @apply="handleApplyTemplate" />
    </section>

    <!-- Model selector -->
    <section class="form-section">
      <h3>AI 模型 <span class="required">*</span></h3>
      <select
        :value="form.model"
        @change="(e) => emit('update:modelValue', { ...form, model: (e.target as HTMLSelectElement).value })"
        class="select-input"
        :disabled="modelsLoading"
      >
        <option value="">
          {{ modelsLoading ? '加载中...' : availableModels.length === 0 ? '无可用模型' : '-- 选择模型 --' }}
        </option>
        <option v-for="m in availableModels" :key="m.id" :value="m.id">{{ m.label }}</option>
      </select>
      <small v-if="!modelsLoading && availableModels.length === 0" class="model-hint">
        暂无可用模型，请先在系统设置中配置模型
      </small>
    </section>

    <!-- ArtStyle: 自由输入 + 预设库 -->
    <section class="form-section">
      <h3>画风 <span class="required">*</span></h3>
      <ArtStyleSelector
        :model-value="form.art_style"
        @update:model-value="(val: string) => emit('update:modelValue', { ...form, art_style: val })"
      />
    </section>

    <!-- ColorTone: 色盘 + 文字 + 游戏预设 -->
    <section class="form-section">
      <h3>色调</h3>
      <div class="row">
        <input
          type="color"
          :value="(form.color_tone || '').startsWith('#') ? form.color_tone : ''"
          @input="(e) => emit('update:modelValue', { ...form, color_tone: (e.target as HTMLInputElement).value })"
          class="color-picker"
          :title="(form.color_tone || '').startsWith('#') ? form.color_tone : '选择颜色'"
        />
        <input
          type="text"
          :value="form.color_tone"
          @input="(e) => emit('update:modelValue', { ...form, color_tone: (e.target as HTMLInputElement).value })"
          placeholder="无色调（留空即可）"
          class="text-input"
        />
        <button
          v-if="form.color_tone"
          type="button"
          class="clear-tone-btn"
          title="清除色调"
          @click="emit('update:modelValue', { ...form, color_tone: '' })"
        >✕</button>
      </div>
      <div class="preset-chips">
        <button
          v-for="preset in [
            'warm pastel color palette',
            'cool blue tones',
            'monochrome sepia',
            'vivid saturated colors',
            'muted earth tones',
            'dark moody dungeon palette',
            'cyberpunk neon glow',
            'medieval warm earth tones',
            'fantasy magical glow',
            'clean mobile game UI palette',
            'rich jewel tones',
            'cinematic teal and orange',
            'golden sunset warm hues',
            'dark mode with neon accents',
            'natural forest greens',
          ]"
          :key="preset"
          type="button"
          class="chip"
          :class="{ active: (form.color_tone || '') === preset }"
          @click="emit('update:modelValue', { ...form, color_tone: preset })"
        >{{ preset }}</button>
      </div>
    </section>

    <!-- ExtraTokens: Tag 输入 -->
    <section class="form-section">
      <h3>自定义追加词</h3>
      <div class="tag-container">
        <span v-for="(token, i) in form.extra_tokens" :key="i" class="tag">
          {{ token }}
          <button type="button" class="tag-remove" @click="removeToken(i)">&times;</button>
        </span>
        <input
          type="text"
          placeholder="输入后回车添加"
          class="tag-input"
          @keydown.enter.prevent="addToken"
        />
      </div>
    </section>

    <!-- ReferenceImageURL: 上传 -->
    <section class="form-section">
      <h3>风格参考图 <span class="hint">（最多 10 张，最强一致性保障）</span></h3>
      <div class="upload-area">
        <div class="ref-grid">
          <div
            v-for="(url, i) in form.reference_image_urls"
            :key="i"
            class="ref-thumb"
          >
            <img :src="url" :alt="`参考图 ${i + 1}`" />
            <button
              type="button"
              class="ref-remove-btn"
              @click="removeReferenceImage(i)"
              title="移除"
            >✕</button>
          </div>
          <label
            v-if="(form.reference_image_urls || []).length < 10"
            class="ref-add-btn"
            title="添加参考图"
          >
            <span>+</span>
            <input type="file" accept="image/*" multiple @change="handleImageUpload" class="ref-file-input" />
          </label>
        </div>
      </div>
    </section>
  </form>
</template>

<style scoped>
.style-form {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.form-section h3 {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 8px;
  color: var(--color-text);
}

.required { color: #ef4444; }
.hint { font-weight: 400; font-size: 12px; color: var(--color-text-muted); }
.model-hint { display: block; margin-top: 6px; font-size: 12px; color: #f59e0b; }

.row { display: flex; gap: 8px; align-items: center; }
.color-picker { width: 40px; height: 40px; border: none; border-radius: var(--radius); cursor: pointer; }

.text-input, .select-input {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  font-size: 14px;
  background: var(--color-surface);
}

.preset-chips { display: flex; flex-wrap: wrap; gap: 6px; margin-top: 8px; }
.chip {
  padding: 4px 12px;
  border: 1px solid var(--color-border);
  border-radius: 16px;
  background: var(--color-surface);
  font-size: 12px;
  cursor: pointer;
}
.chip.active { border-color: var(--color-primary); background: rgba(99, 102, 241, 0.08); color: var(--color-primary); }

.tag-container {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  padding: 8px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  background: var(--color-surface);
  min-height: 40px;
}
.tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 10px;
  background: rgba(99, 102, 241, 0.08);
  color: var(--color-primary);
  border-radius: 12px;
  font-size: 13px;
}
.tag-remove { border: none; background: none; color: inherit; cursor: pointer; font-size: 16px; line-height: 1; padding: 0; }
.tag-input { border: none; outline: none; flex: 1; min-width: 120px; font-size: 14px; }

.upload-area { width: 100%; }
.ref-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: flex-start;
}
.ref-thumb {
  position: relative;
  width: 80px;
  height: 80px;
  border-radius: var(--radius);
  border: 1px solid var(--color-border);
  overflow: hidden;
}
.ref-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.ref-remove-btn {
  position: absolute;
  top: 2px;
  right: 2px;
  width: 18px;
  height: 18px;
  border: none;
  border-radius: 50%;
  background: rgba(0,0,0,0.6);
  color: white;
  font-size: 11px;
  cursor: pointer;
  line-height: 1;
  padding: 0;
}
.ref-remove-btn:hover { background: #ef4444; }
.ref-add-btn {
  width: 80px;
  height: 80px;
  border: 2px dashed var(--color-border);
  border-radius: var(--radius);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: border-color 0.2s;
}
.ref-add-btn:hover { border-color: var(--color-primary); }
.ref-add-btn span { font-size: 28px; color: var(--color-text-muted); }
.ref-file-input {
  position: absolute;
  width: 0;
  height: 0;
  opacity: 0;
}

.preset-category {
  margin-bottom: 10px;
}
.preset-category:last-child {
  margin-bottom: 0;
}

.category-name {
  font-size: 11px;
  font-weight: 600;
  color: var(--color-text-muted);
  margin-bottom: 6px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.clear-tone-btn {
  width: 32px;
  height: 32px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  background: var(--color-surface);
  cursor: pointer;
  font-size: 14px;
  color: var(--color-text-muted);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.clear-tone-btn:hover {
  border-color: #ef4444;
  color: #ef4444;
}
</style>