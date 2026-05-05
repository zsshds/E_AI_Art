<script setup lang="ts">
import { ref } from 'vue'

const emit = defineEmits<{
  insert: [text: string]
}>()

const collapsed = ref(true)
const activeTab = ref<'quality' | 'style' | 'negative' | 'game' | 'structure'>('quality')

const qualityTokens = [
  'highly detailed',
  '8K resolution',
  'sharp focus',
  'masterpiece',
  'AAA game quality',
  'intricate details',
  'ultra high quality',
  'game-ready asset',
  'production quality',
  'crisp details',
  'clean lineart',
  'pixel-perfect',
]

const styleTokens = [
  'character design sheet, concept art',
  'mobile game UI design, clean interface',
  'pixel art game style, 16-bit retro',
  'game card illustration, full art',
  'fantasy RPG character art',
  'cyberpunk game aesthetic, neon glow',
  'stylized game art, vibrant colors',
  'realistic AAA game render',
  'isometric game view, 3/4 perspective',
  'anime game art, cel shading',
  'game environment concept, matte painting',
  'game icon set, stylized icons',
  'dungeon environment, torch-lit',
  'game promotional key art, cinematic',
]

const negativeTokens = [
  'no plastic skin texture',
  'no digital over-sharpening',
  'no blurry details',
  'no distorted anatomy',
  'no bad composition',
  'no extra limbs',
  'no messy background',
  'no low quality artifacts',
  'no watermark',
  'no inconsistent style',
  'no jagged edges on sprites',
  'no text garbling',
]

const gameTemplates = [
  { label: '角色设定卡', value: 'character design sheet, turnaround reference, front side back views, facial expression variations, outfit breakdown, color swatches, organized grid layout on white background, concept art style' },
  { label: '游戏UI面板', value: 'mobile game UI design, clean game interface, dark mode with neon accents, centered layout, game HUD elements, professional game UI style' },
  { label: '4x4图标网格', value: 'organized 4x4 grid of game icons, each icon in its own cell, stylized skill icons, RPG ability icons, consistent style, clean silhouettes' },
  { label: '卡牌CG插画', value: 'game card illustration, full art collectible card style, gold ornamental border, legendary rarity, dramatic spotlight lighting, central character focus' },
  { label: '像素角色', value: 'pixel art character sprite, 16-bit retro game style, clean pixel edges, game-ready sprite sheet, vibrant pixel colors, SNES-era aesthetic' },
  { label: '关卡场景', value: 'game environment concept art, fantasy landscape, epic scale, dramatic volumetric lighting, matte painting quality, level design concept' },
]

const structureGuide = [
  { label: '主体', en: 'Subject', desc: '角色外观/UI元素/场景核心: 姿态、装备、表情' },
  { label: '环境', en: 'Environment', desc: '场景设定: 地牢、城市、森林、UI背景' },
  { label: '风格', en: 'Style', desc: '游戏美术风格: 像素、写实、动漫、概念艺术' },
  { label: '光照', en: 'Lighting', desc: '游戏光照: 体积光、点光源、环境光、霓虹' },
  { label: '构图', en: 'Composition', desc: '布局: 三视图、UI面板、图标网格、广角场景' },
  { label: '质量', en: 'Quality', desc: '游戏质量: 3A级、像素精准、生产就绪、8K' },
  { label: '负面', en: 'Negative', desc: '排除: 模糊、失真、不一致、水印、锯齿' },
]

function insertToken(token: string) {
  emit('insert', token)
}

function copyStructureTemplate() {
  const template = [
    '[游戏角色/UI/场景主体描述]',
    '[环境背景: 关卡场景、UI界面背景]',
    '[游戏美术风格: 像素/写实/动漫/概念艺术]',
    '[游戏光照: 体积光/点光源/霓虹/环境光]',
    '[构图布局: 三视图/UI面板/图标网格/广角]',
    '[质量要求: AAA game quality, highly detailed, 8K]',
    '[排除项: no blurry, no distorted anatomy, no watermark]',
  ].join(', ')
  emit('insert', template)
}
</script>

<template>
  <div class="prompt-assistant">
    <button class="assistant-toggle" @click="collapsed = !collapsed">
      <span class="toggle-icon">{{ collapsed ? '▸' : '▾' }}</span>
      提示词助手
      <span class="toggle-hint">{{ collapsed ? '展开获取优化建议' : '' }}</span>
    </button>

    <div v-if="!collapsed" class="assistant-body">
      <div class="assistant-tabs">
        <button
          v-for="tab in [
            { key: 'quality' as const, label: '质量增强' },
            { key: 'style' as const, label: '风格修饰' },
            { key: 'negative' as const, label: '负面提示' },
            { key: 'game' as const, label: '游戏模板' },
            { key: 'structure' as const, label: '结构指南' },
          ]"
          :key="tab.key"
          class="tab-btn"
          :class="{ active: activeTab === tab.key }"
          @click="activeTab = tab.key"
        >{{ tab.label }}</button>
      </div>

      <!-- Quality Tab -->
      <div v-if="activeTab === 'quality'" class="tab-content">
        <p class="tab-desc">点击标签追加到输入框末尾，提升生成质量：</p>
        <div class="token-chips">
          <button
            v-for="token in qualityTokens"
            :key="token"
            class="token-chip quality-chip"
            @click="insertToken(token)"
          >{{ token }}</button>
        </div>
      </div>

      <!-- Style Tab -->
      <div v-if="activeTab === 'style'" class="tab-content">
        <p class="tab-desc">风格化修饰词，定义视觉风格：</p>
        <div class="token-chips">
          <button
            v-for="token in styleTokens"
            :key="token"
            class="token-chip style-chip"
            @click="insertToken(token)"
          >{{ token }}</button>
        </div>
      </div>

      <!-- Negative Tab -->
      <div v-if="activeTab === 'negative'" class="tab-content">
        <p class="tab-desc">负面提示词，排除不希望出现的元素（追加到末尾）：</p>
        <div class="token-chips">
          <button
            v-for="token in negativeTokens"
            :key="token"
            class="token-chip negative-chip"
            @click="insertToken(token)"
          >{{ token }}</button>
        </div>
      </div>

      <!-- Game Tab -->
      <div v-if="activeTab === 'game'" class="tab-content">
        <p class="tab-desc">游戏场景专用模板，点击一键填入提示词：</p>
        <div class="token-chips">
          <button
            v-for="t in gameTemplates"
            :key="t.label"
            class="token-chip game-chip"
            @click="insertToken(t.value)"
          >{{ t.label }}</button>
        </div>
      </div>

      <!-- Structure Tab -->
      <div v-if="activeTab === 'structure'" class="tab-content">
        <p class="tab-desc">GPT-Image-2 最佳 prompt 结构（7层递进）：</p>
        <div class="structure-list">
          <div v-for="s in structureGuide" :key="s.en" class="structure-item">
            <strong>{{ s.label }}</strong>
            <small>{{ s.en }}</small>
            <span class="structure-desc">{{ s.desc }}</span>
          </div>
        </div>
        <button class="btn-copy-structure" @click="copyStructureTemplate">
          复制结构模板
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.prompt-assistant {
  margin-top: 8px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  background: var(--color-surface);
  overflow: hidden;
}

.assistant-toggle {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  padding: 8px 12px;
  font-size: 13px;
  font-weight: 500;
  background: none;
  border: none;
  cursor: pointer;
  color: var(--color-text);
}
.assistant-toggle:hover {
  background: var(--color-bg);
}

.toggle-icon {
  font-size: 10px;
  width: 14px;
  color: var(--color-text-muted);
}

.toggle-hint {
  font-size: 12px;
  color: var(--color-text-muted);
  font-weight: 400;
  margin-left: 4px;
}

.assistant-body {
  border-top: 1px solid var(--color-border);
}

.assistant-tabs {
  display: flex;
  gap: 0;
  border-bottom: 1px solid var(--color-border);
}

.tab-btn {
  flex: 1;
  padding: 8px 4px;
  font-size: 12px;
  border: none;
  background: none;
  cursor: pointer;
  color: var(--color-text-muted);
  border-bottom: 2px solid transparent;
  transition: all 0.15s;
}
.tab-btn:hover {
  color: var(--color-text);
}
.tab-btn.active {
  color: var(--color-primary);
  border-bottom-color: var(--color-primary);
  font-weight: 500;
}

.tab-content {
  padding: 10px;
}

.tab-desc {
  font-size: 12px;
  color: var(--color-text-muted);
  margin-bottom: 8px;
}

.token-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.token-chip {
  padding: 3px 10px;
  font-size: 12px;
  border: 1px solid var(--color-border);
  border-radius: 14px;
  background: var(--color-bg);
  cursor: pointer;
  color: var(--color-text);
  transition: all 0.15s;
}
.token-chip:hover {
  border-color: var(--color-primary);
  background: rgba(99, 102, 241, 0.08);
  color: var(--color-primary);
}

.quality-chip:hover {
  border-color: #10b981;
  background: rgba(16, 185, 129, 0.08);
  color: #065f46;
}

.style-chip:hover {
  border-color: #8b5cf6;
  background: rgba(139, 92, 246, 0.08);
  color: #6d28d9;
}

.negative-chip:hover {
  border-color: #ef4444;
  background: rgba(239, 68, 68, 0.08);
  color: #991b1b;
}

.game-chip:hover {
  border-color: #f59e0b;
  background: rgba(245, 158, 11, 0.08);
  color: #92400e;
}

.structure-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 10px;
}

.structure-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border-radius: 6px;
  background: var(--color-bg);
  font-size: 12px;
}
.structure-item strong {
  min-width: 36px;
  color: var(--color-primary);
}
.structure-item small {
  min-width: 70px;
  color: var(--color-text-muted);
  font-family: monospace;
}

.structure-desc {
  color: var(--color-text-muted);
}

.btn-copy-structure {
  width: 100%;
  padding: 6px 12px;
  font-size: 12px;
  border: 1px dashed var(--color-border);
  border-radius: var(--radius);
  background: var(--color-bg);
  cursor: pointer;
  color: var(--color-primary);
}
.btn-copy-structure:hover {
  background: rgba(99, 102, 241, 0.08);
  border-style: solid;
}
</style>
