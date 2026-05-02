<script setup lang="ts">
</script>

<template>
  <div class="app-container">
    <header class="app-header">
      <h1>AI 生图平台</h1>
      <nav>
        <router-link to="/style-editor">风格配置</router-link>
        <router-link to="/generate">生图</router-link>
        <router-link to="/review">审核队列</router-link>
      </nav>
    </header>
    <main class="app-main">
      <router-view />
    </main>
  </div>
</template>

<style>
:root {
  --color-primary: #6366f1;
  --color-primary-hover: #4f46e5;
  --color-bg: #f8fafc;
  --color-surface: #ffffff;
  --color-border: #e2e8f0;
  --color-text: #1e293b;
  --color-text-muted: #64748b;
  --radius: 8px;
  --shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  background: var(--color-bg);
  color: var(--color-text);
}

.app-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  height: 56px;
  background: var(--color-surface);
  border-bottom: 1px solid var(--color-border);
}

.app-header h1 {
  font-size: 18px;
  font-weight: 600;
}

.app-header nav {
  display: flex;
  gap: 16px;
}

.app-header nav a {
  color: var(--color-text-muted);
  text-decoration: none;
  font-size: 14px;
  padding: 6px 12px;
  border-radius: var(--radius);
}

.app-header nav a:hover,
.app-header nav a.router-link-exact-active {
  color: var(--color-primary);
  background: rgba(99, 102, 241, 0.08);
}

.app-main {
  max-width: 1200px;
  margin: 0 auto;
  padding: 24px;
}
</style>
