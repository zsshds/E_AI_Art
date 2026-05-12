import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from './stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      redirect: '/generate',
    },
    {
      path: '/login',
      name: 'Login',
      component: () => import('./views/LoginPage.vue'),
    },
    {
      path: '/style-editor',
      name: 'StyleEditor',
      component: () => import('./views/StyleEditor.vue'),
      meta: { requiresAuth: true, requiresAdmin: true },
    },
    {
      path: '/generate',
      name: 'Generate',
      component: () => import('./views/GeneratePage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/tasks/:id',
      name: 'TaskDetail',
      component: () => import('./views/TaskDetailPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/tasks',
      name: 'Tasks',
      component: () => import('./views/TaskQueue.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/projects',
      name: 'Projects',
      component: () => import('./views/ProjectPage.vue'),
      meta: { requiresAuth: true, requiresAdmin: true },
    },
    {
      path: '/settings',
      name: 'Settings',
      component: () => import('./views/SettingsPage.vue'),
      meta: { requiresAuth: true, requiresAdmin: true },
    },
    {
      path: '/users',
      name: 'Users',
      component: () => import('./views/UserManagePage.vue'),
      meta: { requiresAuth: true, requiresAdmin: true },
    },
  ],
})

router.beforeEach((to, _from, next) => {
  // Pinia needs to be accessed after app creation, so we initialize on demand
  const auth = useAuthStore()

  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    next('/login')
  } else if (to.meta.requiresAdmin && !auth.isAdmin) {
    next('/generate')
  } else if (to.path === '/login' && auth.isAuthenticated) {
    // Already logged in, redirect based on role
    next(auth.isAdmin ? '/style-editor' : '/generate')
  } else {
    next()
  }
})

export default router
