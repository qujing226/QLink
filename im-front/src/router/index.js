import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      redirect: '/blockchain'
    },
    {
      path: '/login',
      name: 'login',
      component: () => import('../views/Login.vue')
    },
    {
      path: '/install',
      name: 'install',
      component: () => import('../views/Install.vue')
    },
    {
      path: '/chat',
      name: 'chat',
      meta: { requiresAuth: true },
      component: () => import('../views/Chat.vue')
    },
    {
      path: '/blockchain',
      name: 'blockchain',
      component: () => import('../views/BlockchainPortal.vue')
    }
  ]
})

// 路由守卫
router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('qlink_token')
  
  if (to.meta.requiresAuth && !token) {
    next('/login')
  } else {
    next()
  }
})

export default router