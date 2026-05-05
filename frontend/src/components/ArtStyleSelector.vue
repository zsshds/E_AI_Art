<script setup lang="ts">
import { computed, ref } from 'vue'

const props = defineProps<{
  modelValue: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const showPresets = ref(false)

const inputValue = computed({
  get: () => props.modelValue,
  set: (val: string) => emit('update:modelValue', val),
})

interface PresetCategory {
  name: string
  presets: { label: string; value: string }[]
}

const categories: PresetCategory[] = [
  {
    name: '角色设计',
    presets: [
      { label: '角色设定卡(三视图)', value: 'character design sheet, turnaround reference, front side back views, outfit breakdown, color swatches, organized grid layout, concept art, professional game art style' },
      { label: '日系RPG角色', value: 'JRPG character art, anime style, cel shading, clean lineart, vibrant colors, fantasy armor design' },
      { label: '欧美写实角色', value: 'realistic game character, detailed armor, photorealistic rendering, dramatic pose, AAA game quality' },
      { label: 'Q版卡通角色', value: 'chibi game character, cute proportions, simple shading, colorful, stylized game art, kawaii' },
      { label: '怪物/敌人设计', value: 'game monster design, creature concept art, detailed anatomy, dark fantasy, horror game enemy' },
    ],
  },
  {
    name: '游戏UI / 图标',
    presets: [
      { label: '手游UI面板', value: 'mobile game UI design, clean game interface, dark mode with neon accents, game HUD elements' },
      { label: '游戏技能图标', value: 'game skill icon set, stylized icons, glowing magical effects, clean silhouettes, RPG ability icons' },
      { label: '道具图标集', value: 'game item icons, pixel-perfect rendering, detailed props, RPG inventory items, organized grid of icons' },
      { label: '游戏Logo/标题', value: 'game logo design, bold typography, metallic 3D text, epic fantasy title, cinematic game branding' },
    ],
  },
  {
    name: '场景 / 关卡',
    presets: [
      { label: '关卡场景概念', value: 'game environment concept art, fantasy landscape, epic scale, dramatic lighting, matte painting, level design' },
      { label: '赛博朋克场景', value: 'cyberpunk game environment, neon lights, rain-slicked streets, holographic billboards, high tech low life' },
      { label: '中世纪城堡', value: 'medieval castle game environment, stone architecture, torch-lit corridors, fantasy kingdom, isometric view' },
      { label: '像素游戏场景', value: 'pixel art game background, 16-bit retro style, parallax layers, platformer game environment, vibrant pixel colors' },
    ],
  },
  {
    name: '卡牌 / 插画',
    presets: [
      { label: '游戏卡牌CG', value: 'game card illustration, full art card, gold border, legendary rarity, epic fantasy scene, collectible card game style' },
      { label: '视觉小说立绘', value: 'visual novel character sprite, full body standing pose, transparent background ready, soft anime shading, dating sim style' },
      { label: '载入画面插画', value: 'game loading screen illustration, cinematic composition, dramatic lighting, epic storytelling moment, AAA game art' },
    ],
  },
  {
    name: '像素艺术',
    presets: [
      { label: '像素角色精灵', value: 'pixel art character sprite, 32x32 game sprite, clean pixel edges, game-ready, retro RPG style, sprite sheet' },
      { label: '等距像素场景', value: 'isometric pixel art, game environment, detailed tile set, diablo-style perspective, cozy pixel room' },
      { label: '像素动画帧', value: 'pixel art animation frames, sprite animation sheet, game character walk cycle, 8 frames, clean pixel art' },
    ],
  },
  {
    name: '宣传物料',
    presets: [
      { label: '游戏宣传海报', value: 'game promotional poster, cinematic key art, dramatic composition, movie poster style, main character centered' },
      { label: 'Steam商店图', value: 'Steam game capsule art, horizontal banner, eye-catching thumbnail, game title prominent, professional marketing art' },
      { label: '游戏截图美化', value: 'game screenshot enhancement, polished lighting, cinematic depth of field, enhanced colors, professional game photography' },
    ],
  },
]

function selectPreset(value: string) {
  emit('update:modelValue', value)
}
</script>

<template>
  <div class="art-style-selector">
    <textarea
      :value="inputValue"
      @input="(e) => emit('update:modelValue', (e.target as HTMLTextAreaElement).value)"
      class="text-input art-style-input"
      placeholder="输入自定义画风描述，例如：anime illustration, cel shading, clean lineart"
      rows="2"
    ></textarea>

    <button type="button" class="preset-toggle" @click="showPresets = !showPresets">
      <span class="toggle-arrow">{{ showPresets ? '▾' : '▸' }}</span>
      预设画风库 ({{ categories.reduce((sum, c) => sum + c.presets.length, 0) }} 种)
    </button>

    <div v-if="showPresets" class="preset-panel">
      <div v-for="cat in categories" :key="cat.name" class="preset-category">
        <h4 class="category-name">{{ cat.name }}</h4>
        <div class="preset-chips">
          <button
            type="button"
            v-for="p in cat.presets"
            :key="p.value"
            class="preset-chip"
            :class="{ active: props.modelValue === p.value }"
            @click="selectPreset(p.value)"
          >
            <span class="chip-label">{{ p.label }}</span>
            <span class="chip-preview">{{ p.value }}</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.art-style-selector {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.art-style-input {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  font-size: 14px;
  background: var(--color-surface);
  resize: vertical;
  font-family: inherit;
  line-height: 1.5;
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
  max-height: 320px;
  overflow-y: auto;
}

.preset-category {
  margin-bottom: 12px;
}
.preset-category:last-child {
  margin-bottom: 0;
}

.category-name {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-muted);
  margin-bottom: 6px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.preset-chips {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.preset-chip {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 10px;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-surface);
  cursor: pointer;
  text-align: left;
  transition: all 0.15s;
}
.preset-chip:hover {
  border-color: var(--color-primary);
  background: rgba(99, 102, 241, 0.04);
}
.preset-chip.active {
  border-color: var(--color-primary);
  background: rgba(99, 102, 241, 0.08);
}

.chip-label {
  font-size: 13px;
  font-weight: 500;
  white-space: nowrap;
  min-width: 80px;
  color: var(--color-text);
}

.chip-preview {
  font-size: 11px;
  color: var(--color-text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: monospace;
}
</style>
