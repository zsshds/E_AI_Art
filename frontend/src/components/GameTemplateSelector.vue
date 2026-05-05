<script setup lang="ts">
import { ref } from 'vue'

const emit = defineEmits<{
  apply: [data: Record<string, string>]
}>()

interface GameTemplate {
  id: string
  icon: string
  name: string
  desc: string
  fields: Record<string, string>
}

const templates: GameTemplate[] = [
  {
    id: 'character_sheet',
    icon: '🧍',
    name: '角色设定卡',
    desc: '三视图 + 表情差分 + 装备拆解 + 色板',
    fields: {
      art_style: 'character design sheet, concept art, professional game art style, clean white background, turnaround reference, outfit breakdown, color swatches, organized grid layout',
      color_tone: 'clean studio lighting palette, natural skin tones',
    },
  },
  {
    id: 'ui_panel',
    icon: '📱',
    name: '游戏UI面板',
    desc: '背包/商城/设置 等游戏界面',
    fields: {
      art_style: 'mobile game UI design, clean game interface, professional game HUD, modern mobile game style',
      color_tone: 'dark mode with vibrant accent colors, deep navy background, gold and blue accents',
    },
  },
  {
    id: 'icon_grid',
    icon: '🎯',
    name: '技能/道具图标集',
    desc: '4x4 图标网格，适合 RPG 游戏',
    fields: {
      art_style: 'game icon set, stylized skill icons, detailed game items, RPG ability icons, professional game asset, organized grid layout',
      color_tone: 'vibrant elemental colors, fire red, ice blue, nature green, lightning yellow, shadow purple',
    },
  },
  {
    id: 'environment',
    icon: '🏰',
    name: '关卡场景概念',
    desc: '大型场景/关卡的概念设计图',
    fields: {
      art_style: 'game environment concept art, fantasy landscape, epic scale, matte painting, level design concept, dramatic cinematic lighting',
      color_tone: 'atmospheric environmental palette, natural earth tones with magical accent colors',
    },
  },
  {
    id: 'card_illustration',
    icon: '🃏',
    name: '卡牌CG插画',
    desc: '集换式卡牌游戏插画风格',
    fields: {
      art_style: 'game card illustration, full art collectible card style, legendary rarity, gold ornamental border, dramatic lighting, dynamic pose',
      color_tone: 'rich jewel tones, gold metallic accents, dramatic color contrast',
    },
  },
  {
    id: 'pixel_art',
    icon: '👾',
    name: '像素艺术',
    desc: '复古像素游戏风格',
    fields: {
      art_style: 'pixel art game style, 16-bit retro aesthetic, clean pixel edges, vibrant pixel colors, game sprite, limited palette',
      color_tone: 'vibrant retro game palette, bright saturated pixel colors, SNES-era colors',
    },
  },
]

const selectedId = ref<string | null>(null)

function applyTemplate(t: GameTemplate) {
  selectedId.value = t.id
  emit('apply', { ...t.fields })
}
</script>

<template>
  <div class="game-template-selector">
    <div class="template-grid">
      <button
        type="button"
        v-for="t in templates"
        :key="t.id"
        class="template-card"
        :class="{ active: selectedId === t.id }"
        @click="applyTemplate(t)"
      >
        <span class="template-icon">{{ t.icon }}</span>
        <div class="template-info">
          <strong>{{ t.name }}</strong>
          <small>{{ t.desc }}</small>
        </div>
      </button>
    </div>
  </div>
</template>

<style scoped>
.game-template-selector {
  margin-bottom: 0;
}

.template-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}

.template-card {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border: 2px solid var(--color-border);
  border-radius: var(--radius);
  background: var(--color-surface);
  cursor: pointer;
  text-align: left;
  transition: all 0.15s;
}
.template-card:hover {
  border-color: var(--color-primary);
  background: rgba(99, 102, 241, 0.03);
}
.template-card.active {
  border-color: var(--color-primary);
  background: rgba(99, 102, 241, 0.08);
}

.template-icon {
  font-size: 24px;
  flex-shrink: 0;
}

.template-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.template-info strong {
  font-size: 13px;
  font-weight: 600;
}
.template-info small {
  font-size: 11px;
  color: var(--color-text-muted);
}
</style>
