<script setup lang="ts">
import { ref, onMounted } from 'vue'
import StyleForm from '../components/StyleForm.vue'
import PromptPreview from '../components/PromptPreview.vue'
import ImagePreviewGrid from '../components/ImagePreviewGrid.vue'
import { listStyleProfiles, createStyleProfile, updateStyleProfile, previewStyleProfile, type StyleProfile } from '../api/style'

const profiles = ref<StyleProfile[]>([])
const selectedId = ref<string | null>(null)
const saving = ref(false)
const saveError = ref('')
const previewing = ref(false)
const previewImages = ref<string[]>([])
const testInput = ref('')

const emptyProfile = (): StyleProfile => ({
  name: '',
  created_by: '',
  version: 1,
  model: 'gpt-4o-image',
  art_style: '',
  color_tone: '',
  extra_tokens: [],
  project_id: '',
  reference_image_urls: [],
})

const form = ref<StyleProfile>(emptyProfile())
const editedForm = ref<StyleProfile>({ ...emptyProfile() })

onMounted(async () => {
  try {
    profiles.value = await listStyleProfiles()
  } catch {
    // Handle error silently
  }
})

function selectProfile(profile: StyleProfile) {
  selectedId.value = profile.id!
  // Normalize fields that may be null from the API (Go nil slice → JSON null)
  const normalized = {
    ...profile,
    extra_tokens: profile.extra_tokens || [],
    reference_image_urls: profile.reference_image_urls || [],
  }
  form.value = { ...normalized }
  editedForm.value = { ...normalized }
}

function createNew() {
  selectedId.value = null
  form.value = emptyProfile()
  editedForm.value = emptyProfile()
}

const importing = ref(false)

function handleExport() {
  if (!selectedId.value) return
  const profile = profiles.value.find(p => p.id === selectedId.value)
  if (!profile) return
  const exportData = { ...profile }
  delete (exportData as any).id
  delete (exportData as any).created_at
  delete (exportData as any).updated_at
  delete (exportData as any).created_by
  const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${profile.name || 'style-profile'}.json`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

async function handleImport(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  try {
    const text = await file.text()
    const data = JSON.parse(text)
    importing.value = true
    const created = await createStyleProfile(data)
    profiles.value = await listStyleProfiles()
    selectProfile(created)
  } catch {
    alert('导入失败，请检查 JSON 格式')
  } finally {
    importing.value = false
  }
}

async function handleSave() {
  if (!editedForm.value.name.trim()) {
    saveError.value = '请输入风格名称'
    return
  }
  saveError.value = ''
  saving.value = true
  try {
    // Strip server-set fields before sending
    const { created_at, updated_at, id, ...data } = editedForm.value as any
    if (selectedId.value) {
      const updated = await updateStyleProfile(selectedId.value, data)
      form.value = { ...updated }
    } else {
      const created = await createStyleProfile(data)
      selectedId.value = created.id!
      form.value = { ...created }
      profiles.value = await listStyleProfiles()
    }
  } catch (e: any) {
    saveError.value = e.message || '保存失败'
  } finally {
    saving.value = false
  }
}

async function handlePreview() {
  if (!selectedId.value || !testInput.value.trim()) return
  previewing.value = true
  previewImages.value = []
  try {
    const result = await previewStyleProfile(selectedId.value, testInput.value)
    // Preview returns the API request; in production, it generates 4 images
    // For now, show placeholder images
    previewImages.value = result.data?.map((_: any) => _.url) || []
  } finally {
    previewing.value = false
  }
}
</script>

<template>
  <div class="style-editor">
    <div class="editor-sidebar">
      <h2>风格配置</h2>
      <button class="btn btn-primary" @click="createNew">+ 新建风格</button>
      <div class="sidebar-actions">
        <button
          class="btn btn-sm"
          @click="handleExport"
          :disabled="!selectedId"
          title="导出选中风格为 JSON"
        >📤 导出</button>
        <label class="btn btn-sm import-btn" :class="{ disabled: importing }" title="从 JSON 文件导入风格">
          {{ importing ? '导入中...' : '📥 导入' }}
          <input
            type="file"
            accept=".json"
            class="import-file-input"
            @change="handleImport"
            :disabled="importing"
          />
        </label>
      </div>
      <div class="profile-list">
        <div
          v-for="p in profiles"
          :key="p.id"
          class="profile-item"
          :class="{ active: p.id === selectedId }"
          @click="selectProfile(p)"
        >
          <span>{{ p.name }}</span>
          <small>v{{ p.version }}</small>
        </div>
      </div>
    </div>

    <!-- 区域 A: 语义配置表单 -->
    <div class="editor-form">
      <StyleForm v-if="selectedId || !selectedId" v-model="editedForm" />
      <div v-if="saveError" class="save-error">{{ saveError }}</div>
      <div class="form-actions">
        <button class="btn btn-primary" @click="handleSave" :disabled="saving">
          {{ saving ? '保存中...' : '保存' }}
        </button>
      </div>
    </div>

    <!-- 区域 B: Prompt 实时预览 + 区域 C: 测试生成 -->
    <div class="editor-preview">
      <PromptPreview :profile="editedForm" :user-input="testInput" />

      <div class="test-panel">
        <h3>测试生成</h3>
        <div class="test-input-row">
          <input
            v-model="testInput"
            type="text"
            placeholder="输入测试提示词..."
            class="text-input"
            :disabled="!selectedId"
          />
          <button class="btn btn-primary" @click="handlePreview" :disabled="!selectedId || previewing">
            {{ previewing ? '生成中...' : '生成预览' }}
          </button>
        </div>
        <ImagePreviewGrid :images="previewImages" :loading="previewing" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.style-editor {
  display: grid;
  grid-template-columns: 240px 1fr 360px;
  gap: 24px;
  align-items: start;
}

.editor-sidebar h2 {
  font-size: 16px;
  margin-bottom: 12px;
}

.sidebar-actions {
  display: flex;
  gap: 6px;
  margin-top: 8px;
}
.sidebar-actions .btn-sm {
  padding: 5px 10px;
  font-size: 12px;
  flex: 1;
  text-align: center;
}
.import-btn {
  position: relative;
  overflow: hidden;
}
.import-btn.disabled { opacity: 0.5; }
.import-file-input {
  position: absolute;
  inset: 0;
  opacity: 0;
  cursor: pointer;
}

.profile-list {
  margin-top: 12px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.profile-item {
  padding: 10px 12px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  cursor: pointer;
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.profile-item:hover { border-color: var(--color-primary); }
.profile-item.active { border-color: var(--color-primary); background: rgba(99, 102, 241, 0.04); }
.profile-item small { font-size: 11px; color: var(--color-text-muted); }

.editor-form {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  padding: 20px;
}

.save-error {
  margin-top: 12px;
  padding: 8px 12px;
  background: #fee2e2;
  color: #991b1b;
  border-radius: var(--radius);
  font-size: 13px;
}

.form-actions {
  display: flex;
  gap: 8px;
  margin-top: 12px;
  padding-top: 16px;
  border-top: 1px solid var(--color-border);
}

.editor-preview {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.test-panel {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  padding: 16px;
}

.test-panel h3 {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 12px;
}

.test-input-row {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}

.btn {
  padding: 8px 16px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  font-size: 14px;
  cursor: pointer;
  background: var(--color-surface);
  white-space: nowrap;
}
.btn:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-primary { background: var(--color-primary); color: white; border-color: var(--color-primary); }
.btn-primary:hover:not(:disabled) { background: var(--color-primary-hover); }
.btn-secondary:hover:not(:disabled) { background: var(--color-bg); }

.text-input {
  flex: 1;
  padding: 8px 12px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  font-size: 14px;
}
</style>