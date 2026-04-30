<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  modelValue: {
    art_style: string
    color_tone: string
    lighting: string
    quality_tags: string
    composition: string
    api_quality: string
    size: string
    extra_tokens: string[]
    reference_image_url: string
  }
}>()

const emit = defineEmits<{
  'update:modelValue': [value: any]
}>()

const form = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val),
})

const artStyles = [
  { value: 'anime illustration, cel shading', label: '赛璐珞动画', desc: '清晰的线条，色块分明' },
  { value: 'semi-realistic anime, soft shading', label: '半写实动画', desc: '介于动画与写实之间' },
  { value: 'concept art, digital painting', label: '概念艺术', desc: '厚涂风格，氛围感强' },
  { value: 'line art, monochrome manga style', label: '黑白漫画', desc: '纯线条，网点阴影' },
]

const lightingOptions = [
  { value: 'soft diffused lighting', label: '柔光' },
  { value: 'hard directional lighting', label: '硬光' },
  { value: 'backlight rim lighting', label: '逆光' },
  { value: 'flat lighting, no shadows', label: '无阴影' },
]

const qualityPresets = [
  { value: 'low', label: '低', delay: '~15s', cost: '低' },
  { value: 'medium', label: '中', delay: '~45s', cost: '中' },
  { value: 'high', label: '高', delay: '~120s', cost: '高' },
]

const sizePresets = [
  { value: 'square_1k', label: '1:1', desc: '头像/图标', dimensions: '1024×1024' },
  { value: 'landscape_hd', label: '16:9', desc: '场景/Banner', dimensions: '1536×1024' },
  { value: 'portrait_hd', label: '9:16', desc: '角色立绘', dimensions: '1024×1536' },
]

function addToken(event: Event) {
  const input = event.target as HTMLInputElement
  const value = input.value.trim()
  if (value && !form.value.extra_tokens.includes(value)) {
    const updated = { ...form.value, extra_tokens: [...form.value.extra_tokens, value] }
    emit('update:modelValue', updated)
  }
  input.value = ''
}

function removeToken(index: number) {
  const tokens = [...form.value.extra_tokens]
  tokens.splice(index, 1)
  emit('update:modelValue', { ...form.value, extra_tokens: tokens })
}

function handleImageUpload(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (file) {
    const url = URL.createObjectURL(file)
    emit('update:modelValue', { ...form.value, reference_image_url: url })
  }
}
</script>

<template>
  <form class="style-form">
    <!-- ArtStyle: 卡片单选 -->
    <section class="form-section">
      <h3>画风 <span class="required">*</span></h3>
      <div class="card-grid">
        <label
          v-for="style in artStyles"
          :key="style.value"
          class="card-option"
          :class="{ active: form.art_style === style.value }"
        >
          <input
            type="radio"
            :value="style.value"
            v-model="form.art_style"
          />
          <div class="card-content">
            <strong>{{ style.label }}</strong>
            <small>{{ style.desc }}</small>
          </div>
        </label>
      </div>
    </section>

    <!-- ColorTone: 色盘 + 文字预设 -->
    <section class="form-section">
      <h3>色调</h3>
      <div class="row">
        <input
          type="color"
          :value="form.color_tone.startsWith('#') ? form.color_tone : '#6366f1'"
          @input="(e) => emit('update:modelValue', { ...form, color_tone: (e.target as HTMLInputElement).value })"
          class="color-picker"
        />
        <input
          type="text"
          :value="form.color_tone"
          @input="(e) => emit('update:modelValue', { ...form, color_tone: (e.target as HTMLInputElement).value })"
          placeholder="e.g. warm pastel color palette"
          class="text-input"
        />
      </div>
      <div class="preset-chips">
        <button
          v-for="preset in ['warm pastel color palette', 'cool blue tones', 'monochrome sepia', 'vivid saturated colors', 'muted earth tones']"
          :key="preset"
          type="button"
          class="chip"
          :class="{ active: form.color_tone === preset }"
          @click="emit('update:modelValue', { ...form, color_tone: preset })"
        >{{ preset }}</button>
      </div>
    </section>

    <!-- Lighting: 下拉 -->
    <section class="form-section">
      <h3>光照</h3>
      <select v-model="form.lighting" class="select-input">
        <option value="">-- 选择光照 --</option>
        <option v-for="opt in lightingOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
      </select>
    </section>

    <!-- QualityTags: 文本输入 -->
    <section class="form-section">
      <h3>质量标签</h3>
      <input
        type="text"
        :value="form.quality_tags"
        @input="(e) => emit('update:modelValue', { ...form, quality_tags: (e.target as HTMLInputElement).value })"
        placeholder="e.g. highly detailed, clean lineart"
        class="text-input"
      />
    </section>

    <!-- Composition: 文本输入 -->
    <section class="form-section">
      <h3>构图</h3>
      <input
        type="text"
        :value="form.composition"
        @input="(e) => emit('update:modelValue', { ...form, composition: (e.target as HTMLInputElement).value })"
        placeholder="e.g. centered composition, full body"
        class="text-input"
      />
    </section>

    <!-- APIQuality: Slider -->
    <section class="form-section">
      <h3>生图质量</h3>
      <div class="quality-slider">
        <input
          type="range"
          min="0"
          max="2"
          step="1"
          :value="qualityPresets.findIndex(p => p.value === form.api_quality)"
          @input="(e) => {
            const idx = parseInt((e.target as HTMLInputElement).value)
            emit('update:modelValue', { ...form, api_quality: qualityPresets[idx].value })
          }"
        />
        <div class="quality-labels">
          <span v-for="q in qualityPresets" :key="q.value">
            {{ q.label }} ({{ q.delay }}, {{ q.cost }}费用)
          </span>
        </div>
      </div>
    </section>

    <!-- Size: 比例图标单选 -->
    <section class="form-section">
      <h3>分辨率</h3>
      <div class="card-grid cols-3">
        <label
          v-for="size in sizePresets"
          :key="size.value"
          class="card-option size-card"
          :class="{ active: form.size === size.value }"
        >
          <input type="radio" :value="size.value" v-model="form.size" />
          <div class="card-content">
            <div class="size-ratio" :class="size.value">{{ size.label }}</div>
            <strong>{{ size.desc }}</strong>
            <small>{{ size.dimensions }}</small>
          </div>
        </label>
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
      <h3>风格参考图 <span class="hint">（最强一致性保障，使用 Edits 端点）</span></h3>
      <div class="upload-area">
        <input type="file" accept="image/*" @change="handleImageUpload" />
        <img v-if="form.reference_image_url" :src="form.reference_image_url" class="ref-preview" />
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

.card-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}
.card-grid.cols-3 {
  grid-template-columns: 1fr 1fr 1fr;
}

.card-option {
  display: block;
  cursor: pointer;
}
.card-option input { display: none; }
.card-option .card-content {
  border: 2px solid var(--color-border);
  border-radius: var(--radius);
  padding: 12px;
  text-align: center;
}
.card-option.active .card-content {
  border-color: var(--color-primary);
  background: rgba(99, 102, 241, 0.04);
}
.card-content strong { display: block; font-size: 14px; margin-bottom: 4px; }
.card-content small { font-size: 12px; color: var(--color-text-muted); }

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

.quality-slider input { width: 100%; }
.quality-labels { display: flex; justify-content: space-between; font-size: 12px; color: var(--color-text-muted); }

.size-ratio {
  width: 48px; height: 32px;
  margin: 0 auto 8px;
  border: 2px solid var(--color-border);
  border-radius: 4px;
}
.size-ratio.square_1k { width: 32px; }
.size-ratio.landscape_hd { width: 48px; }
.size-ratio.portrait_hd { width: 24px; }

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

.upload-area { display: flex; gap: 12px; align-items: flex-start; }
.ref-preview { width: 120px; height: 120px; object-fit: cover; border-radius: var(--radius); border: 1px solid var(--color-border); }
</style>
