import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      redirect: '/generate',
    },
    {
      path: '/style-editor',
      name: 'StyleEditor',
      component: () => import('./views/StyleEditor.vue'),
    },
    {
      path: '/generate',
      name: 'Generate',
      component: () => import('./views/GeneratePage.vue'),
    },
    {
      path: '/review',
      name: 'Review',
      component: () => import('./views/ReviewQueue.vue'),
    },
  ],
})

export default router
