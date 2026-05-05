<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listProjects, createProject, updateProject, deleteProject, type Project } from '../api/project'

const projects = ref<Project[]>([])
const loading = ref(false)
const message = ref('')

// Create/edit form
const showForm = ref(false)
const editingProject = ref<Project | null>(null)
const formName = ref('')
const memberInput = ref('')
const formMembers = ref<string[]>([])

onMounted(async () => {
  await refresh()
})

async function refresh() {
  loading.value = true
  try {
    projects.value = await listProjects()
  } catch {
    // handle silently
  } finally {
    loading.value = false
  }
}

function startCreate() {
  showForm.value = true
  editingProject.value = null
  formName.value = ''
  formMembers.value = []
  memberInput.value = ''
}

function startEdit(p: Project) {
  showForm.value = true
  editingProject.value = p
  formName.value = p.name
  formMembers.value = [...p.members]
  memberInput.value = ''
}

function addMember() {
  const name = memberInput.value.trim()
  if (name && !formMembers.value.includes(name)) {
    formMembers.value.push(name)
  }
  memberInput.value = ''
}

function removeMember(name: string) {
  formMembers.value = formMembers.value.filter(m => m !== name)
}

function cancelEdit() {
  showForm.value = false
  editingProject.value = null
  formName.value = ''
  formMembers.value = []
}

async function handleSave() {
  if (!formName.value.trim()) {
    message.value = '请输入项目名称'
    return
  }
  message.value = ''
  try {
    if (editingProject.value?.id) {
      await updateProject(editingProject.value.id, {
        name: formName.value.trim(),
        members: formMembers.value,
      })
      message.value = '更新成功'
    } else {
      await createProject(formName.value.trim(), formMembers.value)
      message.value = '创建成功'
    }
    cancelEdit()
    await refresh()
  } catch (e: any) {
    message.value = '保存失败: ' + (e.message || '未知错误')
  }
}

async function handleDelete(id: string) {
  if (!confirm('确定要删除该项目吗？')) return
  try {
    await deleteProject(id)
    message.value = '删除成功'
    await refresh()
  } catch (e: any) {
    message.value = '删除失败: ' + (e.message || '未知错误')
  }
}
</script>

<template>
  <div class="project-page">
    <div class="page-header">
      <h2>项目管理</h2>
      <button class="btn btn-primary" @click="startCreate">+ 新建项目</button>
    </div>
    <p class="subtitle">创建项目并将用户添加到项目中，同项目成员共享风格和任务</p>

    <div v-if="message" class="toast" :class="{ error: message.includes('失败') }">{{ message }}</div>

    <!-- Edit form -->
    <div v-if="showForm" class="edit-panel">
      <h3>{{ editingProject?.id ? '编辑项目' : '新建项目' }}</h3>
      <div class="form-group">
        <label>项目名称</label>
        <input v-model="formName" type="text" placeholder="输入项目名称" class="text-input" />
      </div>
      <div class="form-group">
        <label>项目成员</label>
        <div class="member-input-row">
          <input
            v-model="memberInput"
            type="text"
            placeholder="输入用户名后回车添加"
            class="text-input"
            @keydown.enter.prevent="addMember"
          />
          <button type="button" class="btn btn-secondary" @click="addMember">添加</button>
        </div>
        <div class="member-tags" v-if="formMembers.length > 0">
          <span v-for="m in formMembers" :key="m" class="member-tag">
            {{ m }}
            <button type="button" class="tag-remove" @click="removeMember(m)">&times;</button>
          </span>
        </div>
      </div>
      <div class="form-actions">
        <button class="btn btn-primary" @click="handleSave">保存</button>
        <button class="btn" @click="cancelEdit">取消</button>
      </div>
    </div>

    <!-- Project list -->
    <div v-if="loading" class="loading">加载中...</div>
    <div v-else-if="projects.length === 0" class="empty-state">
      暂无项目，点击"新建项目"创建第一个
    </div>
    <div v-else class="project-list">
      <div v-for="p in projects" :key="p.id" class="project-card">
        <div class="project-info">
          <strong>{{ p.name }}</strong>
          <small>创建者: {{ p.created_by }} | 成员: {{ p.members.length }} 人</small>
          <div class="member-list" v-if="p.members.length > 0">
            <span v-for="m in p.members" :key="m" class="member-badge">{{ m }}</span>
          </div>
        </div>
        <div class="project-actions">
          <button class="btn btn-secondary" @click="startEdit(p)">编辑</button>
          <button class="btn btn-danger" @click="handleDelete(p.id!)">删除</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.project-page {
  max-width: 700px;
  margin: 0 auto;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  padding: 24px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}
.page-header h2 { font-size: 20px; }

.subtitle {
  font-size: 13px;
  color: var(--color-text-muted);
  margin-bottom: 20px;
}

.toast {
  padding: 8px 12px;
  background: #d1fae5;
  color: #065f46;
  border-radius: var(--radius);
  font-size: 13px;
  margin-bottom: 16px;
}
.toast.error { background: #fee2e2; color: #991b1b; }

.edit-panel {
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  padding: 16px;
  margin-bottom: 20px;
}
.edit-panel h3 { font-size: 16px; margin-bottom: 12px; }

.form-group { margin-bottom: 12px; }
.form-group label { display: block; font-size: 13px; font-weight: 500; margin-bottom: 4px; }

.text-input {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  font-size: 14px;
  background: var(--color-surface);
}

.member-input-row { display: flex; gap: 8px; }
.member-input-row .text-input { flex: 1; }

.member-tags { display: flex; flex-wrap: wrap; gap: 6px; margin-top: 6px; }
.member-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 10px;
  background: rgba(99, 102, 241, 0.08);
  color: var(--color-primary);
  border-radius: 12px;
  font-size: 13px;
}
.tag-remove { border: none; background: none; color: inherit; cursor: pointer; font-size: 14px; }

.form-actions { display: flex; gap: 8px; margin-top: 8px; }

.loading, .empty-state {
  text-align: center;
  padding: 48px;
  color: var(--color-text-muted);
}

.project-list { display: flex; flex-direction: column; gap: 8px; }

.project-card {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 14px 16px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
}

.project-info strong { display: block; font-size: 15px; margin-bottom: 4px; }
.project-info small { font-size: 12px; color: var(--color-text-muted); display: block; margin-bottom: 6px; }

.member-list { display: flex; flex-wrap: wrap; gap: 4px; }
.member-badge {
  padding: 2px 8px;
  background: var(--color-bg);
  border-radius: 10px;
  font-size: 12px;
  color: var(--color-text-muted);
}

.project-actions { display: flex; gap: 8px; flex-shrink: 0; }

.btn {
  padding: 8px 16px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  font-size: 14px;
  cursor: pointer;
  background: var(--color-surface);
}
.btn-primary { background: var(--color-primary); color: white; border-color: var(--color-primary); }
.btn-primary:hover { background: var(--color-primary-hover); }
.btn-secondary:hover { background: var(--color-bg); }
.btn-danger { color: #991b1b; border-color: #fca5a5; }
.btn-danger:hover { background: #fee2e2; }
</style>
