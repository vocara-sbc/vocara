import { createRouter, createWebHashHistory, RouteRecordRaw } from 'vue-router'
import NProgress from 'nprogress'
import { useUserStore } from '@/store/modules/user'
import { useTabStore } from '@/store/modules/tab'

declare module 'vue-router' {
  interface RouteMeta {
    /** 搜索区为页面第一张卡片时，搜索卡片与标签栏直角拼接为一体 */
    searchJoined?: boolean
  }
}

NProgress.configure({ showSpinner: false, trickleSpeed: 100 })

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/login/index.vue'),
    meta: {
      title: '登录',
      requiresAuth: false,
      hidden: true,
    },
  },
  {
    path: '/403',
    name: 'Forbidden',
    component: () => import('@/views/error/403.vue'),
    meta: { title: '无权限访问', hidden: true },
  },
  {
    path: '/404',
    name: 'NotFound',
    component: () => import('@/views/error/404.vue'),
    meta: { title: '页面不存在', hidden: true },
  },
  {
    path: '/',
    component: () => import('@/layouts/default/index.vue'),
    // 根路径跳转在路由守卫中按角色 home_path 处理（座席默认 /workbench）
    meta: { requiresAuth: true },
    children: [
      // ===== 座席工作台（座席默认首页）=====
      {
        path: 'workbench',
        name: 'Workbench',
        component: () => import('@/views/workbench/index.vue'),
        meta: {
          title: '座席工作台',
          icon: 'workbench',
          permission: 'menu:workbench:view',
          affix: true,
          keepAlive: true,
        },
      },
      // ===== 仪表盘 =====
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('@/views/dashboard/index.vue'),
        meta: {
          title: '仪表盘',
          icon: 'dashboard',
          permission: 'menu:dashboard:view',
          affix: true,
          keepAlive: true,
        },
      },
      // ===== 座席管理 =====
      {
        path: 'agent',
        name: 'Agent',
        redirect: '/agent/list',
        meta: {
          title: '座席管理',
          icon: 'user',
          permission: 'menu:agent:view',
        },
        children: [
          {
            path: 'list',
            name: 'AgentList',
            component: () => import('@/views/agent/list.vue'),
            meta: {
              title: '座席管理',
              icon: 'user',
              permission: 'menu:agent:view',
              keepAlive: true,
              searchJoined: true,
            },
          },
          {
            path: 'org',
            name: 'Organization',
            component: () => import('@/views/agent/group.vue'),
            meta: {
              title: '组织机构',
              icon: 'group',
              permission: 'menu:agent_group:view',
              keepAlive: true,
              searchJoined: true,
            },
          },
          {
            path: 'skill',
            name: 'SkillGroup',
            component: () => import('@/views/agent/skill.vue'),
            meta: {
              title: '技能组',
              icon: 'skill',
              permission: 'menu:agent_skill:view',
              keepAlive: true,
              searchJoined: true,
            },
          },
          {
            path: 'dn',
            name: 'DnManage',
            component: () => import('@/views/agent/dn.vue'),
            meta: {
              title: '分机管理',
              icon: 'phone',
              permission: 'menu:agent_dn:view',
              keepAlive: true,
              searchJoined: true,
            },
          },
        ],
      },
      // ===== 队列管理 =====
      {
        path: 'queue',
        name: 'Queue',
        redirect: '/queue/list',
        meta: {
          title: '队列管理',
          icon: 'queue',
          permission: 'menu:queue:view',
        },
        children: [
          {
            path: 'list',
            name: 'QueueList',
            component: () => import('@/views/queue/list.vue'),
            meta: {
              title: '队列配置',
              icon: 'list',
              permission: 'menu:queue:view',
              keepAlive: true,
              searchJoined: true,
            },
          },
          {
            path: 'monitor',
            name: 'QueueMonitor',
            component: () => import('@/views/queue/monitor.vue'),
            meta: {
              title: '实时监控',
              icon: 'monitor',
              permission: 'menu:queue_monitor:view',
              keepAlive: true,
            },
          },
        ],
      },
      // ===== 呼入接待 =====
      {
        path: 'admit',
        name: 'Admit',
        component: () => import('@/views/admit/index.vue'),
        meta: {
          title: '呼入接待',
          icon: 'call-in',
          permission: 'menu:admit:view',
          keepAlive: true,
          searchJoined: true,
        },
      },
      // ===== 通话记录 =====
      {
        path: 'records',
        name: 'Records',
        redirect: '/records/inbound',
        meta: {
          title: '通话记录',
          icon: 'record',
          permission: 'menu:record:inbound',
        },
        children: [
          {
            path: 'inbound',
            name: 'InboundRecords',
            component: () => import('@/views/records/inbound.vue'),
            meta: {
              title: '呼入记录',
              icon: 'call-in',
              permission: 'menu:record:inbound',
              keepAlive: true,
              searchJoined: true,
            },
          },
          {
            path: 'outbound',
            name: 'OutboundRecords',
            component: () => import('@/views/records/outbound.vue'),
            meta: {
              title: '呼出记录',
              icon: 'call-out',
              permission: 'menu:record:outbound',
              keepAlive: true,
              searchJoined: true,
            },
          },
          {
            path: 'active',
            name: 'ActiveCalls',
            component: () => import('@/views/records/active.vue'),
            meta: {
              title: '实时通话',
              icon: 'phone',
              permission: 'menu:record:active',
              keepAlive: true,
            },
          },
        ],
      },
      // ===== 工单管理（所有角色可访问，不设权限码） =====
      {
        path: 'tickets',
        name: 'Tickets',
        component: () => import('@/views/tickets/index.vue'),
        meta: { title: '工单管理', icon: 'ticket', keepAlive: true, searchJoined: true },
      },
      // ===== 客户管理（所有角色可访问） =====
      {
        path: 'customers',
        name: 'Customers',
        component: () => import('@/views/customers/index.vue'),
        meta: { title: '客户管理', icon: 'user', keepAlive: true, searchJoined: true },
      },
      // ===== 线路管理 =====
      {
        path: 'lines',
        name: 'Lines',
        component: () => import('@/views/lines/index.vue'),
        meta: {
          title: '线路管理',
          icon: 'line',
          permission: 'menu:lines:view',
          keepAlive: true,
          searchJoined: true,
        },
      },
      // ===== 统计报表 =====
      {
        path: 'stats',
        name: 'Stats',
        redirect: '/stats/agent',
        meta: {
          title: '统计报表',
          icon: 'chart',
          permission: 'menu:stats:agent',
        },
        children: [
          {
            path: 'agent',
            name: 'StatsAgent',
            component: () => import('@/views/stats/agent.vue'),
            meta: {
              title: '座席统计',
              icon: 'user-chart',
              permission: 'menu:stats:agent',
              keepAlive: true,
              searchJoined: true,
            },
          },
          {
            path: 'queue',
            name: 'StatsQueue',
            component: () => import('@/views/stats/queue.vue'),
            meta: {
              title: '队列统计',
              icon: 'queue-chart',
              permission: 'menu:stats:queue',
              keepAlive: true,
              searchJoined: true,
            },
          },
          {
            path: 'line',
            name: 'StatsLine',
            component: () => import('@/views/stats/line.vue'),
            meta: {
              title: '线路统计',
              icon: 'line-chart',
              permission: 'menu:stats:line',
              keepAlive: true,
              searchJoined: true,
            },
          },
          {
            path: 'trend',
            name: 'StatsTrend',
            component: () => import('@/views/stats/trend.vue'),
            meta: {
              title: '趋势分析',
              icon: 'trend',
              permission: 'menu:stats:trend',
              keepAlive: true,
            },
          },
        ],
      },
      // ===== 系统管理 =====
      {
        path: 'system',
        name: 'System',
        redirect: '/system/param',
        meta: {
          title: '系统管理',
          icon: 'setting',
          permission: 'menu:system:param',
        },
        children: [
          {
            path: 'param',
            name: 'SystemParam',
            component: () => import('@/views/system/param.vue'),
            meta: {
              title: '参数配置',
              icon: 'setting',
              permission: 'menu:system:param',
              keepAlive: true,
              searchJoined: true,
            },
          },
          {
            path: 'license',
            name: 'SystemLicense',
            component: () => import('@/views/system/license.vue'),
            meta: {
              title: '授权许可',
              icon: 'license',
              permission: 'menu:system:license',
              keepAlive: true,
            },
          },
          {
            path: 'log',
            name: 'OperationLog',
            component: () => import('@/views/system/log.vue'),
            meta: {
              title: '操作日志',
              icon: 'log',
              permission: 'menu:system:log',
              keepAlive: true,
              searchJoined: true,
            },
          },
          {
            path: 'tenant',
            name: 'TenantManage',
            component: () => import('@/views/system/tenant.vue'),
            meta: {
              title: '租户管理',
              icon: 'building',
              permission: 'menu:system:tenant',
              keepAlive: true,
              searchJoined: true,
            },
          },
          {
            path: 'form-category',
            name: 'FormCategoryManage',
            component: () => import('@/views/system/form-category/index.vue'),
            meta: {
              title: '表单分类',
              icon: 'setting',
              permission: 'menu:system:param',
              keepAlive: true,
              searchJoined: true,
            },
          },
          {
            path: 'form-template',
            name: 'FormTemplate',
            component: () => import('@/views/form/template/index.vue'),
            meta: { title: '表单模板', icon: 'ticket', permission: 'menu:form:view', keepAlive: true, searchJoined: true },
          },
          {
            path: 'form-template/design',
            name: 'FormTemplateDesign',
            component: () => import('@/views/form/template/designer.vue'),
            meta: { title: '表单设计', icon: 'ticket', permission: 'menu:form:view', hidden: true },
          },
          {
            path: 'form',
            name: 'FormRecord',
            component: () => import('@/views/form/record/index.vue'),
            meta: { title: '表单管理', icon: 'list', permission: 'menu:form:view', keepAlive: true, searchJoined: true },
          },
        ],
      },

      // ===== 个人中心（右上角菜单进入，不显示在侧边栏） =====
      {
        path: 'profile',
        name: 'Profile',
        component: () => import('@/views/profile/index.vue'),
        meta: { title: '个人中心', hidden: true },
      },
    ],
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/404',
    meta: { hidden: true },
  },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

// 路由守卫
router.beforeEach(async (to, from, next) => {
  NProgress.start()
  const userStore = useUserStore()

  // 设置页面标题
  const title = (to.meta?.title as string) || 'Vocara 管理系统'
  document.title = `${title} - Vocara 呼叫中心管理系统`

  // 不需要登录的页面
  if (to.meta.requiresAuth === false) {
    next()
    return
  }

  // 未登录 → 跳登录
  if (!userStore.token) {
    next({
      path: '/login',
      query: { redirect: to.fullPath },
    })
    return
  }

  // 已登录但没有用户信息，获取一次
  if (!userStore.userInfo) {
    try {
      await userStore.fetchUserInfo()
    } catch (e) {
      userStore.logout()
      next({
        path: '/login',
        query: { redirect: to.fullPath },
      })
      return
    }
  }

  // 权限校验：路由 meta.permission 存在时检查
  const perm = to.meta?.permission as string | undefined
  if (perm && !userStore.hasPermission(perm)) {
    next('/403')
    return
  }

  // 根路径跳转到角色对应的首页
  if (to.path === '/') {
    next(userStore.homePath)
    return
  }

  next()
})

router.afterEach((to) => {
  NProgress.done()
  // 添加标签页
  if (to.meta && to.meta.title && !to.meta.hidden) {
    const tabStore = useTabStore()
    tabStore.addTab(to)
  }
})

export default router
