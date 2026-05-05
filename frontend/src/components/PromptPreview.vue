<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  profile: {
    art_style: string
    color_tone: string
    extra_tokens: string[]
  }
  userInput?: string
}>()

const fullPrompt = computed(() => {
  const parts: string[] = []
  const push = (s: string) => { if (s?.trim()) parts.push(s.trim()) }

  push(props.profile.art_style)
  push(props.profile.color_tone)
  parts.push(...(props.profile.extra_tokens || []))

  const stylePart = parts.join(', ')
  const userPart = props.userInput?.trim() || ''

  if (!stylePart) return userPart || '(等待填写风格配置...)'
  if (!userPart) return `${stylePart}. [用户输入]`
  return `${stylePart}. ${userPart}`
})
</script>

<template>
  <div class="prompt-preview">
    <h3>Prompt 实时预览</h3>
    <div class="preview-box">
      <pre>{{ fullPrompt }}</pre>
    </div>
    <div class="prompt-meta">
      <span>风格段: {{ profile.art_style ? '✓ 已配置' : '✗ 未配置' }}</span>
      <span>预估长度: {{ fullPrompt.length }} 字符</span>
    </div>
  </div>
</template>

<style scoped>
.prompt-preview {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  padding: 16px;
  position: sticky;
  top: 24px;
}

.prompt-preview h3 {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 12px;
}

.preview-box {
  background: #1e293b;
  color: #e2e8f0;
  padding: 16px;
  border-radius: var(--radius);
  min-height: 80px;
  font-size: 13px;
  line-height: 1.6;
  word-break: break-word;
}

.preview-box pre {
  white-space: pre-wrap;
  font-family: 'SF Mono', 'Fira Code', monospace;
}

.prompt-meta {
  display: flex;
  justify-content: space-between;
  margin-top: 8px;
  font-size: 12px;
  color: var(--color-text-muted);
}
</style>
