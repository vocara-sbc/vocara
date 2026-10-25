import { h } from 'vue'
import { useUserStore } from '@/store/modules/user'

// 侧边栏菜单配置（与 docs/09-管理后台菜单设计.md 一致）
// 菜单项结构：{ label, key, icon?, permission?, hidden?, children? }
// permission: 访问该菜单需要的权限码，不填表示登录即可见
// hidden: 规划占位菜单（功能未实现），默认不渲染

export const menuOptions = [
  {
    label: '仪表盘',
    key: '/dashboard',
    icon: () => menuIcon('dashboard'),
    permission: 'menu:dashboard:view',
  },
  {
    label: '座席工作台',
    key: '/workbench',
    icon: () => menuIcon('workbench'),
    permission: 'menu:workbench:view',
  },
  {
    label: '通话记录',
    key: '/records',
    icon: () => menuIcon('record'),
    permission: 'menu:record:inbound',
    children: [
      {
        label: '呼入通话',
        key: '/records/inbound',
        icon: () => menuIcon('call-in'),
        permission: 'menu:record:inbound',
      },
      {
        label: '呼出通话',
        key: '/records/outbound',
        icon: () => menuIcon('call-out'),
        permission: 'menu:record:outbound',
      },
    ],
  },
  {
    label: '工单管理',
    key: '/tickets',
    icon: () => menuIcon('ticket'),
  },
  {
    label: '客户管理',
    key: '/customers',
    icon: () => menuIcon('user'),
  },
  {
    label: '系统监控',
    key: '/monitor',
    icon: () => menuIcon('monitor'),
    permission: 'menu:queue_monitor:view',
    children: [
      {
        label: '实时监控',
        key: '/queue/monitor',
        icon: () => menuIcon('monitor'),
        permission: 'menu:queue_monitor:view',
      },
      {
        label: '操作日志',
        key: '/system/log',
        icon: () => menuIcon('log'),
        permission: 'menu:system:log',
      },
    ],
  },
  {
    label: '统计报表',
    key: '/stats',
    icon: () => menuIcon('chart'),
    permission: 'menu:stats:agent',
    children: [
      {
        label: '座席统计',
        key: '/stats/agent',
        icon: () => menuIcon('user-chart'),
        permission: 'menu:stats:agent',
      },
      {
        label: '队列统计',
        key: '/stats/queue',
        icon: () => menuIcon('queue-chart'),
        permission: 'menu:stats:queue',
      },
      {
        label: '线路统计',
        key: '/stats/line',
        icon: () => menuIcon('line-chart'),
        permission: 'menu:stats:line',
      },
      {
        label: '趋势分析',
        key: '/stats/trend',
        icon: () => menuIcon('trend'),
        permission: 'menu:stats:trend',
      },
    ],
  },
  {
    label: '系统管理',
    key: '/system',
    icon: () => menuIcon('setting'),
    permission: 'menu:agent:view',
    children: [
      {
        label: '座席管理',
        key: '/agent/list',
        icon: () => menuIcon('user'),
        permission: 'menu:agent:view',
      },
      {
        label: '分机管理',
        key: '/agent/dn',
        icon: () => menuIcon('phone'),
        permission: 'menu:agent_dn:view',
      },
      {
        label: '技能组管理',
        key: '/agent/skill',
        icon: () => menuIcon('skill'),
        permission: 'menu:agent_skill:view',
      },
      {
        label: '队列管理',
        key: '/queue/list',
        icon: () => menuIcon('queue'),
        permission: 'menu:queue:view',
      },
      {
        label: '线路管理',
        key: '/lines',
        icon: () => menuIcon('line'),
        permission: 'menu:lines:view',
      },
      {
        label: '呼入接待',
        key: '/admit',
        icon: () => menuIcon('call-in'),
        permission: 'menu:admit:view',
      },
      {
        label: '组织机构',
        key: '/agent/org',
        icon: () => menuIcon('group'),
        permission: 'menu:agent_group:view',
      },
      {
        label: '租户管理',
        key: '/system/tenant',
        icon: () => menuIcon('building'),
        permission: 'menu:system:tenant',
      },
      {
        label: '参数配置',
        key: '/system/param',
        icon: () => menuIcon('setting'),
        permission: 'menu:system:param',
      },
      {
        label: '授权许可',
        key: '/system/license',
        icon: () => menuIcon('license'),
        permission: 'menu:system:license',
      },
      {
        label: '表单分类',
        key: '/system/form-category',
        icon: () => menuIcon('setting'),
        permission: 'menu:system:param',
      },
      {
        label: '表单模板',
        key: '/system/form-template',
        icon: () => menuIcon('ticket'),
        permission: 'menu:form:view',
      },
      {
        label: '表单管理',
        key: '/system/form',
        icon: () => menuIcon('list'),
        permission: 'menu:form:view',
      },
    ],
  },
]

// 获取过滤后的菜单（根据当前用户权限）
export function getFilteredMenus() {
  const userStore = useUserStore()
  return filterMenuByPermission(menuOptions, userStore.permissions)
}

/**
 * 根据当前路由路径返回完整的菜单层级标题（如 ['系统管理', '座席管理']）。
 * 菜单中"系统管理"等分组项本身不对应路由，route.matched 拿不到，
 * 因此从菜单配置的分组结构中查找；找不到时返回 null，由调用方回退。
 */
export function getMenuTrail(path: string): string[] | null {
  return findTrail(menuOptions, path, [])
}

function findTrail(nodes: any[], path: string, ancestors: string[]): string[] | null {
  for (const node of nodes) {
    const trail = [...ancestors, node.label]
    // 精确命中该节点
    if (node.key === path) {
      return trail
    }
    if (node.children) {
      const hit = findTrail(node.children, path, trail)
      if (hit) return hit
    }
  }
  return null
}

// 递归过滤菜单（权限码 + hidden 占位过滤）
function filterMenuByPermission(menus: any[], permissions: string[]): any[] {
  return menus
    .filter((item) => {
      // hidden 为规划占位，不渲染
      if (item.hidden) return false
      // 没有 permission 字段表示所有人可见
      if (!item.permission) return true
      return permissions.includes(item.permission)
    })
    .map((item) => {
      // 有子菜单的话递归过滤子菜单
      if (item.children && item.children.length > 0) {
        const filteredChildren = filterMenuByPermission(item.children, permissions)
        // 如果子菜单全被过滤掉了，父菜单也不显示
        if (filteredChildren.length === 0) {
          return null
        }
        // 子级菜单项不显示图标：仅一级菜单保留图标
        const childrenWithoutIcon = filteredChildren.map((child: any) => {
          const { icon, ...rest } = child
          return rest
        })
        return { ...item, children: childrenWithoutIcon }
      }
      return item
    })
    .filter(Boolean)
}

// 渲染 SVG 图标为 VNode
function menuIcon(name: string) {
  return h('span', {
    class: 'menu-svg-icon',
    innerHTML: iconSvgs[name] || iconSvgs.dashboard,
  })
}

// SVG 图标集合
const iconSvgs: Record<string, string> = {
  dashboard: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:18px;height:18px"><rect x="3" y="3" width="7" height="9" rx="1"/><rect x="14" y="3" width="7" height="5" rx="1"/><rect x="14" y="12" width="7" height="9" rx="1"/><rect x="3" y="16" width="7" height="5" rx="1"/></svg>',
  workbench: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:18px;height:18px"><path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72c.127.96.361 1.903.7 2.81a2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45c.907.339 1.85.573 2.81.7A2 2 0 0 1 22 16.92z"/><circle cx="12" cy="12" r="3"/></svg>',
  user: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:18px;height:18px"><path d="M20 21v-2a4 4 0 00-4-4H8a4 4 0 00-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>',
  skill: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:18px;height:18px"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>',
  phone: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:18px;height:18px"><path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 014.11 2h3a2 2 0 012 1.72c.127.96.361 1.903.7 2.81a2 2 0 01-.45 2.11L8.09 9.91a16 16 0 006 6l1.27-1.27a2 2 0 012.11-.45c.907.339 1.85.573 2.81.7A2 2 0 0122 16.92z"/></svg>',
  queue: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:18px;height:18px"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/></svg>',
  list: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:18px;height:18px"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/></svg>',
  monitor: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:18px;height:18px"><rect x="2" y="3" width="20" height="14" rx="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/></svg>',
  'call-in': '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:18px;height:18px"><polyline points="16 2 16 8 22 8"/><line x1="22" y1="2" x2="16" y2="8"/><path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72c.127.96.361 1.903.7 2.81a2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45c.907.339 1.85.573 2.81.7A2 2 0 0 1 22 16.92z"/></svg>',
  'call-out': '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:18px;height:18px"><polyline points="22 8 22 2 16 2"/><line x1="16" y1="8" x2="22" y2="2"/><path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72c.127.96.361 1.903.7 2.81a2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45c.907.339 1.85.573 2.81.7A2 2 0 0 1 22 16.92z"/></svg>',
  record: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:18px;height:18px"><circle cx="12" cy="12" r="10"/><circle cx="12" cy="12" r="3"/></svg>',
  line: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:18px;height:18px"><path d="M18 20V10"/><path d="M12 20V4"/><path d="M6 20v-6"/></svg>',
  chart: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:18px;height:18px"><line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/></svg>',
  'user-chart': '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:18px;height:18px"><path d="M16 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2"/><circle cx="8.5" cy="7" r="4"/><polyline points="18 14 20 16 24 12"/></svg>',
  'queue-chart': '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:18px;height:18px"><rect x="3" y="12" width="4" height="9" rx="1"/><rect x="10" y="6" width="4" height="15" rx="1"/><rect x="17" y="9" width="4" height="12" rx="1"/></svg>',
  'line-chart': '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:18px;height:18px"><polyline points="3 17 9 11 13 15 21 7"/><polyline points="14 7 21 7 21 14"/></svg>',
  trend: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:18px;height:18px"><polyline points="23 6 13.5 15.5 8.5 10.5 1 18"/><polyline points="17 6 23 6 23 12"/></svg>',
  setting: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:18px;height:18px"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 010 2.83 2 2 0 01-2.83 0l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-4 0v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83 0 2 2 0 010-2.83l.06-.06a1.65 1.65 0 00.33-1.82 1.65 1.65 0 00-1.51-1H3a2 2 0 010-4h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 010-2.83 2 2 0 012.83 0l.06.06a1.65 1.65 0 001.82.33H9a1.65 1.65 0 001-1.51V3a2 2 0 014 0v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 0 2 2 0 010 2.83l-.06.06a1.65 1.65 0 00-.33 1.82V9a1.65 1.65 0 001.51 1H21a2 2 0 010 4h-.09a1.65 1.65 0 00-1.51 1z"/></svg>',
  log: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:18px;height:18px"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="8" y1="13" x2="16" y2="13"/><line x1="8" y1="17" x2="16" y2="17"/><line x1="8" y1="9" x2="10" y2="9"/></svg>',
  license: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:18px;height:18px"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/><polyline points="9 12 11 14 15 10"/></svg>',
  building: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:18px;height:18px"><rect x="4" y="2" width="16" height="20" rx="2"/><line x1="9" y1="22" x2="9" y2="2"/><line x1="15" y1="22" x2="15" y2="2"/><line x1="7" y1="6" x2="7.01" y2="6"/><line x1="12" y1="6" x2="12.01" y2="6"/><line x1="17" y1="6" x2="17.01" y2="6"/><line x1="7" y1="10" x2="7.01" y2="10"/><line x1="12" y1="10" x2="12.01" y2="10"/><line x1="17" y1="10" x2="17.01" y2="10"/><line x1="7" y1="14" x2="7.01" y2="14"/><line x1="12" y1="14" x2="12.01" y2="14"/><line x1="17" y1="14" x2="17.01" y2="14"/><line x1="7" y1="18" x2="7.01" y2="18"/><line x1="12" y1="18" x2="12.01" y2="18"/><line x1="17" y1="18" x2="17.01" y2="18"/></svg>',
  ticket: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:18px;height:18px"><path d="M3 9V7a2 2 0 012-2h14a2 2 0 012 2v2a3 3 0 000 6v2a2 2 0 01-2 2H5a2 2 0 01-2-2v-2a3 3 0 000-6z"/><line x1="13" y1="5" x2="13" y2="19" stroke-dasharray="2 2"/></svg>',
}
