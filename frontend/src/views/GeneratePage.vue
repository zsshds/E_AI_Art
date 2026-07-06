<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { listStyleProfiles, type StyleProfile } from '../api/style'
import { createTask, getTask, subscribeTask, type Task } from '../api/task'
import { useAvailableModels } from '../composables/useAvailableModels'
import PromptAssistant from '../components/PromptAssistant.vue'

const { availableModels, loading: modelsLoading } = useAvailableModels()

const sizeOptions = [
  { value: 'auto', label: 'auto (默认)' },
  { value: '1024x1024', label: '1024×1024 (1:1 正方形)' },
  { value: '1536x1024', label: '1536×1024 (3:2 横向)' },
  { value: '1024x1536', label: '1024×1536 (2:3 竖屏)' },
  { value: '2048x2048', label: '2048×2048 (1:1 2K)' },
  { value: '2048x1152', label: '2048×1152 (16:9 2K)' },
  { value: '3840x2160', label: '3840×2160 (16:9 4K)' },
  { value: '2160x3840', label: '2160×3840 (9:16 4K)' },
]

const qualityOptions = [
  { value: 'low', label: '低质量 (快速)' },
  { value: 'medium', label: '中等质量 (推荐)' },
  { value: 'high', label: '高质量 (较慢)' },
]

const selectedCount = ref(1)
const countOptions = Array.from({ length: 10 }, (_, i) => ({
  value: i + 1,
  label: `${i + 1} 张`,
}))

const profiles = ref<StyleProfile[]>([])
const selectedProfileId = ref('')
const selectedModel = ref('')
const selectedSize = ref('auto')
const selectedQuality = ref('medium')
const userInput = ref('')
const generating = ref(false)
const currentTask = ref<Task | null>(null)
const resultImage = ref('')

const MAX_SOURCE_IMAGES = 5

const sourceImageDatas = ref<string[]>([])
const sourceImagePreviews = ref<string[]>([])
const isDragging = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)
const uploadLimitDialogVisible = ref(false)
const uploadLimitDialogMessage = ref('')
const error = ref('')
let unsubscribe: (() => void) | null = null

onMounted(async () => {
  try {
    profiles.value = await listStyleProfiles()
  } catch {
    // Handle silently
  }
})

onUnmounted(() => {
  unsubscribe?.()
})

const downloading = ref(false)

function readFileAsBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(reader.result as string)
    reader.onerror = reject
    reader.readAsDataURL(file)
  })
}

async function handleFileSelect(event: Event) {
  const input = event.target as HTMLInputElement
  const files = input.files
  if (!files || files.length === 0) return
  await processFiles(Array.from(files))
  input.value = '' // reset so same file can be re-selected
}

async function handlePaste(e: ClipboardEvent) {
  if (generating.value) return
  const items = e.clipboardData?.items
  if (!items || items.length === 0) return

  const files: File[] = []
  for (let i = 0; i < items.length; i++) {
    const item = items[i]
    if (item.type.startsWith('image/')) {
      const blob = item.getAsFile()
      if (blob) {
        // Generate a filename for clipboard images
        const ext = item.type.split('/')[1] || 'png'
        const file = new File([blob], `clipboard.${ext}`, { type: item.type })
        files.push(file)
      }
    }
  }

  if (files.length > 0) {
    e.preventDefault()
    await processFiles(files)
  }
}

async function processFiles(files: File[]) {
  const remaining = MAX_SOURCE_IMAGES - sourceImageDatas.value.length
  if (remaining <= 0) {
    alert(`最多上传 ${MAX_SOURCE_IMAGES} 张源图`)
    return
  }
  const toAdd = files.slice(0, remaining)
  const oversizeFile = toAdd.find(file => file.size > 10 * 1024 * 1024)
  if (oversizeFile) {
    uploadLimitDialogMessage.value = `图片 ${oversizeFile.name} 超过 10MB，已取消本次上传。`
    uploadLimitDialogVisible.value = true
    return
  }
  for (const file of toAdd) {
    if (!file.type.startsWith('image/')) {
      alert(`文件 ${file.name} 不是图片格式`)
      continue
    }
    try {
      const data = await readFileAsBase64(file)
      sourceImageDatas.value.push(data)
      sourceImagePreviews.value.push(URL.createObjectURL(file))
    } catch {
      alert(`读取图片 ${file.name} 失败`)
    }
  }
}

function handleDragOver(e: DragEvent) {
  e.preventDefault()
  isDragging.value = true
}

function handleDragLeave() {
  isDragging.value = false
}

async function handleDrop(e: DragEvent) {
  e.preventDefault()
  isDragging.value = false
  const files = e.dataTransfer?.files
  if (!files || files.length === 0) return
  await processFiles(Array.from(files))
}

function handleRemoveImage(index: number) {
  sourceImageDatas.value.splice(index, 1)
  const preview = sourceImagePreviews.value[index]
  if (preview) URL.revokeObjectURL(preview)
  sourceImagePreviews.value.splice(index, 1)
}

function closeUploadLimitDialog() {
  uploadLimitDialogVisible.value = false
  uploadLimitDialogMessage.value = ''
}

async function handleDownload() {
  if (!currentTask.value?.id) return
  downloading.value = true
  try {
    const token = localStorage.getItem('token')
    const resp = await fetch(`/api/v1/tasks/${currentTask.value.id}/download`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    if (!resp.ok) throw new Error('下载失败')
    const blob = await resp.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `E_AI_Art-${currentTask.value.id.slice(-8)}.png`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  } catch (e: any) {
    alert(e.message || '下载失败')
  } finally {
    downloading.value = false
  }
}

function handleInsertToken(token: string) {
  const current = userInput.value.trim()
  userInput.value = current ? `${current}, ${token}` : token
}

async function handleGenerate() {
  if (!selectedProfileId.value || !userInput.value.trim()) return

  generating.value = true
  error.value = ''
  resultImage.value = ''
  currentTask.value = null

  try {
    const task = await createTask(selectedProfileId.value, userInput.value.trim(), selectedModel.value || undefined, selectedSize.value, selectedQuality.value, sourceImageDatas.value.length > 0 ? sourceImageDatas.value : undefined, selectedCount.value)
    currentTask.value = task

    // Subscribe to WebSocket updates
    unsubscribe = subscribeTask(task.id!, (update) => {
      if (update.status === 'done') {
        resultImage.value = update.result_image_url
        generating.value = false
      } else if (update.status === 'failed') {
        error.value = update.error_message || '生图失败'
        generating.value = false
      }
      if (currentTask.value) {
        currentTask.value.status = update.status
      }
    })

    // Fallback polling
    pollTask(task.id!)
  } catch (e: any) {
    error.value = e.message
    generating.value = false
  }
}

async function pollTask(taskId: string) {
  const maxPolls = 60
  for (let i = 0; i < maxPolls; i++) {
    await new Promise(r => setTimeout(r, 2000))
    if (!generating.value) return
    try {
      const t = await getTask(taskId)
      currentTask.value = t
      if (t.status === 'done') {
        resultImage.value = t.result_image_url
        generating.value = false
        return
      }
      if (t.status === 'failed') {
        error.value = t.error_message || '生图失败'
        generating.value = false
        return
      }
    } catch {
      // Retry next poll
    }
  }
  generating.value = false
  error.value = '生图超时，请稍后重试'
}
</script>

<template>
  <div class="generate-page">
    <div class="generate-form">
      <h2>E_AI_Art 生图</h2>
      <p class="subtitle">选择风格，输入自然语言描述即可生图</p>

      <div class="form-group">
        <label>风格选择</label>
        <select v-model="selectedProfileId" class="select-input" :disabled="generating"
          @change="(e) => {
            const p = profiles.find(p => p.id === (e.target as HTMLSelectElement).value)
            if (p && p.model) selectedModel = p.model
          }">
          <option value="">-- 选择风格 --</option>
          <option v-for="p in profiles" :key="p.id" :value="p.id">{{ p.name }} ({{ p.model }})</option>
        </select>
      </div>

      <div class="form-group">
        <label>AI 模型</label>
        <select v-model="selectedModel" class="select-input" :disabled="generating || modelsLoading">
          <option value="">
            {{ modelsLoading ? '加载中...' : availableModels.length === 0 ? '无可用模型，请联系管理员配置' : '-- 选择模型 --' }}
          </option>
          <option v-for="m in availableModels" :key="m.id" :value="m.id">{{ m.label }}</option>
        </select>
      </div>

      <div class="form-group">
        <label>分辨率</label>
        <select v-model="selectedSize" class="select-input" :disabled="generating">
          <option v-for="s in sizeOptions" :key="s.value" :value="s.value">{{ s.label }}</option>
        </select>
      </div>

      <div class="form-group">
        <label>生图质量</label>
        <select v-model="selectedQuality" class="select-input" :disabled="generating">
          <option v-for="q in qualityOptions" :key="q.value" :value="q.value">{{ q.label }}</option>
        </select>
      </div>

      <div class="form-group">
        <label>生成数量 <span class="hint">（1-10 张）</span></label>
        <select v-model="selectedCount" class="select-input" :disabled="generating">
          <option v-for="c in countOptions" :key="c.value" :value="c.value">{{ c.label }}</option>
        </select>
      </div>

      <!-- 图生图：源图上传 -->
      <div class="form-group">
        <label>
          源图上传 (图生图)
          <span class="hint">— 可选，上传 1-{{ MAX_SOURCE_IMAGES }} 张图片作为生成基础</span>
        </label>
        <div
          class="upload-area"
          :class="{ dragging: isDragging }"
          @dragover="handleDragOver"
          @dragleave="handleDragLeave"
          @drop="handleDrop"
          @paste="handlePaste"
          tabindex="0"
        >
          <!-- Upload placeholder (always visible when under limit) -->
          <div v-if="sourceImagePreviews.length < MAX_SOURCE_IMAGES" class="upload-placeholder">
            <span class="upload-icon">📁</span>
            <p>拖拽图片到此处，点击选择，或 Ctrl+V 粘贴截图</p>
            <p class="upload-hint">支持 JPG / PNG / WebP，最大 10MB，最多 {{ MAX_SOURCE_IMAGES }} 张</p>
            <input
              ref="fileInput"
              type="file"
              accept="image/*"
              multiple
              class="file-input"
              @change="handleFileSelect"
              :disabled="generating"
            />
          </div>
          <!-- Preview grid -->
          <div v-if="sourceImagePreviews.length > 0" class="upload-previews">
            <div v-for="(preview, i) in sourceImagePreviews" :key="i" class="upload-preview-item">
              <img :src="preview" :alt="`源图 ${i + 1}`" />
              <button
                class="btn btn-remove"
                @click="handleRemoveImage(i)"
                :disabled="generating"
                title="移除这张图片"
              >✕</button>
              <span class="preview-index">{{ i + 1 }}</span>
            </div>
          </div>
        </div>
      </div>

      <div class="form-group">
        <label>画面描述</label>
        <textarea
          v-model="userInput"
          placeholder="用自然语言描述你想生成的画面..."
          rows="4"
          class="text-input"
          :disabled="generating"
        ></textarea>
        <PromptAssistant @insert="handleInsertToken" />
      </div>

      <button
        class="btn btn-primary btn-large"
        @click="handleGenerate"
        :disabled="generating || !selectedProfileId || !userInput.trim()"
      >
        {{ generating ? '生成中...' : '开始生成' }}
      </button>

      <div v-if="currentTask" class="task-status">
        <span class="status-badge" :class="currentTask.status">
          {{ { pending: '排队中', processing: '生成中', done: '已完成', failed: '失败' }[currentTask.status] }}
        </span>
      </div>

      <div v-if="error" class="error-msg">{{ error }}</div>
    </div>

    <div class="generate-result">
      <div v-if="resultImage" class="result-image">
        <img :src="resultImage" alt="生成结果" />
        <div class="result-actions">
          <button class="btn btn-primary" @click="handleDownload" :disabled="downloading">
            {{ downloading ? '下载中...' : '下载图片' }}
          </button>
        </div>
      </div>
      <div v-else class="result-placeholder">
        <p>生成的图片将显示在这里</p>
      </div>
    </div>

    <div v-if="uploadLimitDialogVisible" class="dialog-backdrop" @click.self="closeUploadLimitDialog">
      <div class="dialog-card" role="dialog" aria-modal="true" aria-labelledby="upload-limit-title">
        <h3 id="upload-limit-title">上传失败</h3>
        <p>{{ uploadLimitDialogMessage }}</p>
        <button class="btn btn-primary" @click="closeUploadLimitDialog">确认</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.generate-page {
  display: grid;
  grid-template-columns: 400px 1fr;
  gap: 24px;
  align-items: start;
}

.generate-form {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  padding: 24px;
}

.generate-form h2 {
  font-size: 20px;
  margin-bottom: 4px;
}

.subtitle {
  font-size: 14px;
  color: var(--color-text-muted);
  margin-bottom: 20px;
}

.form-group {
  margin-bottom: 16px;
}

.form-group label {
  display: block;
  font-size: 14px;
  font-weight: 500;
  margin-bottom: 6px;
}

.select-input, .text-input {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  font-size: 14px;
  background: var(--color-surface);
}

textarea.text-input {
  resize: vertical;
  font-family: inherit;
}

.btn-large {
  width: 100%;
  padding: 12px;
  font-size: 16px;
}

.task-status {
  margin-top: 12px;
}

.status-badge {
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 13px;
}
.status-badge.pending { background: #fef3c7; color: #92400e; }
.status-badge.processing { background: #dbeafe; color: #1e40af; }
.status-badge.done { background: #d1fae5; color: #065f46; }
.status-badge.failed { background: #fee2e2; color: #991b1b; }

.error-msg {
  margin-top: 12px;
  padding: 8px 12px;
  background: #fee2e2;
  color: #991b1b;
  border-radius: var(--radius);
  font-size: 13px;
}

.generate-result {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  min-height: 500px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.result-placeholder {
  color: var(--color-text-muted);
  font-size: 14px;
}

.result-image {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 16px;
}
.result-image img {
  max-width: 100%;
  border-radius: var(--radius);
  margin-bottom: 12px;
}
.result-actions {
  display: flex;
  gap: 8px;
}

.btn {
  padding: 8px 16px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  font-size: 14px;
  cursor: pointer;
  background: var(--color-surface);
}
.btn:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-primary { background: var(--color-primary); color: white; border-color: var(--color-primary); }
.btn-primary:hover:not(:disabled) { background: var(--color-primary-hover); }

.hint { font-weight: 400; font-size: 12px; color: var(--color-text-muted); }

.dialog-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(15, 23, 42, 0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  z-index: 1000;
}

.dialog-card {
  width: min(420px, 100%);
  background: var(--color-surface);
  border-radius: var(--radius);
  box-shadow: 0 20px 40px rgba(15, 23, 42, 0.18);
  padding: 24px;
  border: 1px solid var(--color-border);
}

.dialog-card h3 {
  font-size: 18px;
  margin-bottom: 12px;
}

.dialog-card p {
  font-size: 14px;
  color: var(--color-text);
  line-height: 1.6;
  margin-bottom: 16px;
}

.upload-area {
  border: 2px dashed var(--color-border);
  border-radius: var(--radius);
  padding: 20px;
  text-align: center;
  cursor: pointer;
  transition: border-color 0.2s, background 0.2s;
  min-height: 100px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
}
.upload-area.dragging {
  border-color: var(--color-primary);
  background: rgba(99, 102, 241, 0.04);
}
.upload-placeholder {
  position: relative;
}
.upload-icon {
  font-size: 32px;
  display: block;
  margin-bottom: 8px;
}
.upload-placeholder p {
  margin: 4px 0;
  font-size: 13px;
  color: var(--color-text-muted);
}
.upload-hint {
  font-size: 11px !important;
  opacity: 0.7;
}
.file-input {
  position: absolute;
  inset: 0;
  opacity: 0;
  cursor: pointer;
}
.upload-previews {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: center;
  padding-top: 8px;
}
.upload-preview-item {
  position: relative;
  width: 90px;
  height: 90px;
  border-radius: var(--radius);
  border: 1px solid var(--color-border);
  overflow: hidden;
}
.upload-preview-item img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.preview-index {
  position: absolute;
  bottom: 2px;
  left: 2px;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: rgba(0,0,0,0.6);
  color: white;
  font-size: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.btn-remove {
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
  display: flex;
  align-items: center;
  justify-content: center;
}
.btn-remove:hover:not(:disabled) { background: rgba(239, 68, 68, 0.8); }
</style>
