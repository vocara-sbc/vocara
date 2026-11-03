<template>
  <n-layout has-sider class="admin-layout">
    <!-- 侧边栏：全高白色面板，折叠后仅保留一级菜单图标 -->
    <n-layout-sider
      class="layout-sider"
      :class="{ 'sider-collapsed': themeStore.collapsed }"
      collapse-mode="width"
      :collapsed-width="64"
      :width="180"
      :collapsed="themeStore.collapsed"
      :native-scrollbar="false"
    >
      <!-- Logo -->
      <div class="logo-wrapper" @click="$router.push('/dashboard')">
        <div class="logo-icon">V</div>
        <!-- 文字宽度随折叠连续收束，与侧边栏宽度动画同步，图标位置无跳变 -->
        <div class="logo-text-mask" :class="{ 'is-hidden': themeStore.collapsed }">
          <span class="logo-text">VOCARA</span>
        </div>
      </div>

      <!-- 菜单 -->
      <div class="menu-wrapper">
        <n-menu
          :value="activeMenu"
          :options="menuOptions"
          :collapsed="themeStore.collapsed"
          :collapsed-width="40"
          :icon-size="18"
          :collapsed-icon-size="18"
          :root-indent="11"
          :indent="26"
          @update:value="handleMenuClick"
        />
      </div>
    </n-layout-sider>

    <!-- 主内容区：浅灰绿底，三段白色悬浮卡片 -->
    <n-layout class="main-layout">
      <!-- 授权状态横幅：占主区顶部正常位，下推内容，不遮挡电话条 -->
      <LicenseGuard />
      <!-- 顶部 Header 卡片：左折叠按钮+面包屑，中电话条绝对居中，右通知/账号 -->
      <n-layout-header class="layout-header">
        <div class="header-inner">
          <div class="header-left">
            <div
              class="header-icon-btn"
              :title="themeStore.collapsed ? '展开导航' : '收起导航'"
              @click="themeStore.toggleCollapsed"
            >
              <n-icon size="18">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M3 6h18M3 12h18M3 18h12" />
                </svg>
              </n-icon>
            </div>
            <n-breadcrumb>
              <n-breadcrumb-item v-for="item in breadcrumbs" :key="item.path">
                {{ item.title }}
              </n-breadcrumb-item>
            </n-breadcrumb>
          </div>

          <!-- 电话条（全局常驻：座席跨页面工作时持续等待来电，v-show 保留 DOM 与连接；
               绝对居中于 Header，左右区域伸缩不影响其位置与状态） -->
          <div v-show="showPhoneBar" class="header-center phonebar-wrapper">
            <PhoneBar />
          </div>

          <div class="header-right">
            <!-- 通知 -->
            <div class="header-icon-btn" title="通知">
              <n-badge :value="3" :max="99">
                <n-icon size="18">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
                  </svg>
                </n-icon>
              </n-badge>
            </div>

            <!-- 全屏 -->
            <div class="header-icon-btn" @click="toggleFullscreen" title="全屏">
              <n-icon size="18">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M8 3H5a2 2 0 00-2 2v3m18 0V5a2 2 0 00-2-2h-3m0 18h3a2 2 0 002-2v-3M3 16v3a2 2 0 002 2h3" />
                </svg>
              </n-icon>
            </div>

            <!-- 用户信息 -->
            <n-dropdown :options="userMenuOptions" @select="handleUserMenuSelect" trigger="click">
              <div class="user-info">
                <n-avatar round size="32">
                  {{ userStore.userInfo?.nickname?.charAt(0) || 'A' }}
                </n-avatar>
                <span class="username">{{ userStore.userInfo?.nickname || '管理员' }}</span>
                <n-icon size="14" style="margin-left: 4px">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="6 9 12 15 18 9" />
                  </svg>
                </n-icon>
              </div>
            </n-dropdown>
          </div>
        </div>
      </n-layout-header>

      <!-- 标签栏：透明底，白色 tab 页签悬浮于底色之上，页签右侧露出页面底色；
           页签左缘与下方卡片左缘对齐 -->
      <div class="tabs-bar">
        <tabs-view :joined="searchJoined" />
      </div>

      <!-- 内容区：背景透明，页面内卡片自然悬浮；
           拼接态顶部无间隙，默认态顶部保留卡片间隙 -->
      <n-layout-content class="layout-content" :class="{ 'layout-content--joined': searchJoined }">
        <router-view v-slot="{ Component, route }">
          <keep-alive :include="keepAliveNames">
            <component :is="Component" :key="route.fullPath" />
          </keep-alive>
        </router-view>
      </n-layout-content>
    </n-layout>

    <!-- 修改密码全局弹窗：Header 下拉与个人中心页共用 -->
    <ChangePasswordModal />
  </n-layout>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/store/modules/user'
import { useCallStore } from '@/store/modules/call'
import { useThemeStore } from '@/store/modules/theme'
import { useUiStore } from '@/store/modules/ui'
import { getFilteredMenus, getMenuTrail } from './menu'
import TabsView from './components/TabsView.vue'
import PhoneBar from '@/components/PhoneBar/index.vue'
import ChangePasswordModal from '@/components/ChangePasswordModal/index.vue'
import LicenseGuard from './components/LicenseGuard.vue'
import { useFullscreen } from '@vueuse/core'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const callStore = useCallStore()
const themeStore = useThemeStore()
const uiStore = useUiStore()
const { isFullscreen, toggle: toggleFullscreen } = useFullscreen()

// 电话条配置：仅当前登录账号绑定了分机时显示（未绑定分机的管理员/超级管理员不显示）
const showPhoneBar = computed(() => !!callStore.myDn)

// 搜索卡片与标签栏拼接为一体（页面第一张卡片为搜索区）
const searchJoined = computed(() => !!route.meta.searchJoined)

// 根据用户权限过滤后的菜单
const menuOptions = computed(() => getFilteredMenus())

// 当前激活的菜单
const activeMenu = computed(() => {
  const path = route.path
  // 取二级路径作为菜单激活项
  const parts = path.split('/').filter(Boolean)
  if (parts.length >= 2) {
    return `/${parts[0]}/${parts[1]}`
  }
  return path
})

// 面包屑
const breadcrumbs = computed(() => {
  // 优先用菜单分组结构（含"系统管理"等不在路由中的分组），回退到 matched 标题
  const trail = getMenuTrail(route.path)
  if (trail && trail.length) {
    return trail.map((title) => ({ path: '', title }))
  }
  return route.matched
    .filter((r) => r.meta && r.meta.title)
    .map((r) => ({ path: r.path, title: r.meta?.title as string }))
})

// keep-alive 名称列表
const keepAliveNames = computed(() => {
  const names: string[] = []
  route.matched.forEach((r) => {
    if (r.meta?.keepAlive && r.name) {
      names.push(r.name as string)
    }
  })
  return names
})

// 用户下拉菜单
const userMenuOptions = [
  {
    label: '个人中心',
    key: 'profile',
    icon: () => h('span', '👤'),
  },
  {
    label: '修改密码',
    key: 'password',
    icon: () => h('span', '🔑'),
  },
  {
    type: 'divider' as const,
    key: 'divider',
  },
  {
    label: '退出登录',
    key: 'logout',
    icon: () => h('span', '🚪'),
  },
]

function handleMenuClick(key: string) {
  if (key !== route.path) {
    router.push(key)
  }
}

function handleUserMenuSelect(key: string) {
  switch (key) {
    case 'profile':
      router.push('/profile')
      break
    case 'password':
      uiStore.openPasswordModal()
      break
    case 'logout':
      callStore.destroyAdapter()
      userStore.logout()
      router.push('/login')
      break
  }
}

onMounted(() => {
  themeStore.restoreSettings()
  // 拉取当前账号分机绑定情况，确定电话条是否显示
  callStore.loadMyDn()
})

import { h } from 'vue'
</script>

<style scoped lang="css">
/* ===== 悬浮卡片风格 token =====
   页面底色浅灰绿，卡片白底圆角 + 轻阴影，卡片间隙露出底色形成悬浮感 */
.admin-layout {
  --page-bg: #ecf3ef;
  --card-bg: #ffffff;
  --card-radius: 10px;
  --card-shadow: 0 1px 4px rgba(15, 40, 25, 0.06);
  --card-gap: 12px;
  --side-gap: 16px;
  /* 页签颜色：默认 chip 边框、拼接态非选中块底色与边框 */
  --tab-border: #e2e8e4;
  --tab-muted-bg: #f4f7f5;
  --tab-muted-border: #e7ede9;

  height: 100vh;
  overflow: hidden;
}

/* ===== 侧边栏：全高白色面板 ===== */
.layout-sider {
  background: var(--card-bg);
  box-shadow: 1px 0 6px rgba(15, 40, 25, 0.05);
}

.logo-wrapper {
  display: flex;
  align-items: center;
  /* 图标与文字作为整体在 logo 区水平居中；折叠时内容只剩图标，仍居中于侧边栏 */
  justify-content: center;
  height: 60px;
  padding: 0 16px;
  cursor: pointer;
  overflow: hidden;
}

.logo-icon {
  width: 32px;
  height: 32px;
  background: linear-gradient(135deg, #18a058, #36ad6a);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-weight: 700;
  font-size: 18px;
  flex-shrink: 0;
}

/* 文字容器：max-width/边距/透明度随折叠连续过渡，时长曲线与侧边栏宽度动画一致 */
.logo-text-mask {
  max-width: 100px;
  margin-left: 12px;
  opacity: 1;
  overflow: hidden;
  white-space: nowrap;
  transition:
    max-width 0.3s cubic-bezier(0.4, 0, 0.2, 1),
    margin-left 0.3s cubic-bezier(0.4, 0, 0.2, 1),
    opacity 0.2s ease;
}

.logo-text-mask.is-hidden {
  max-width: 0;
  margin-left: 0;
  opacity: 0;
}

.logo-text {
  font-size: 18px;
  font-weight: 700;
  color: #18a058;
  letter-spacing: 2px;
  white-space: nowrap;
}

.menu-wrapper {
  padding: 8px 0;
  height: calc(100vh - 60px);
  overflow-y: auto;
}

/* 菜单项悬浮 chip：左右内缩、圆角 */
.menu-wrapper :deep(.n-menu-item-content) {
  margin: 0 12px 4px;
  border-radius: 8px;
}

/* 含子菜单的项：展开箭头紧跟文字，不推到最右（默认 grid 中间列 1fr 把箭头顶到右侧）。
   把三列都改为按内容宽度，箭头与文字间距 8px，和图标→文字的间距一致 */
.menu-wrapper :deep(.n-menu-item-content) {
  grid-template-columns: auto auto 1fr;
}
.menu-wrapper :deep(.n-menu-item-content__icon) {
  margin-right: 8px;
}
.menu-wrapper :deep(.n-menu-item-content__arrow) {
  grid-column: auto;
  margin-left: 8px;
}
/* 无箭头的叶子项：文字仍占第二列，第三列空，不影响 */
.menu-wrapper :deep(.n-menu-item-content-header) {
  grid-column: auto;
}

/* 选中态：绿色渐变 chip + 白字白图标 */
.menu-wrapper :deep(.n-menu-item-content--selected) {
  background: linear-gradient(90deg, #36ad6a 0%, #18a058 100%);
}

.menu-wrapper :deep(.n-menu-item-content--selected .n-menu-item-content-header),
.menu-wrapper :deep(.n-menu-item-content--selected .n-menu-item-content-header a),
.menu-wrapper :deep(.n-menu-item-content--selected .n-icon),
.menu-wrapper :deep(.n-menu-item-content--selected .n-menu-item-content__icon),
.menu-wrapper :deep(.n-menu-item-content--selected .menu-svg-icon),
.menu-wrapper :deep(.n-menu-item-content--selected .menu-svg-icon svg) {
  color: #ffffff !important;
}

/* 折叠态：chip 与展开态等宽边距（左右各 12px、宽 40px），
   图标中心与展开态图标中心位于同一垂直线（x=32） */
.sider-collapsed .menu-wrapper :deep(.n-menu-item-content) {
  margin: 0 12px 4px;
}

/* ===== 右侧主区域：浅灰绿底 ===== */
.main-layout {
  background: var(--page-bg);
}

/* 主区域滚动容器改为纵向 flex，内容区才能伸展填满 Header/标签页之外的剩余高度 */
.main-layout :deep(> .n-layout-scroll-container) {
  display: flex;
  flex-direction: column;
}

/* ===== Header 悬浮卡片 ===== */
.layout-header {
  height: auto;
  flex-shrink: 0;
  padding: var(--card-gap) var(--side-gap) 0;
  background: transparent;
}

.header-inner {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 60px;
  padding: 0 16px;
  background: var(--card-bg);
  border-radius: var(--card-radius);
  box-shadow: var(--card-shadow);
}

.header-left {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.header-right {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  min-width: 0;
}

/* 电话条绝对居中：左右两侧内容变化不影响其位置，
   组件本身挂在布局层且 v-show 常驻，菜单切换不影响通话状态 */
.header-center {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  display: flex;
  align-items: center;
  max-width: 60%;
  /* 高于标签栏 .n-tabs-nav-scroll-wrapper 的 z-index:1：呼叫状态卡伸出 Header
     压在标签栏几何区域时，点击不被标签栏的透明滚动容器拦截 */
  z-index: 10;
}

.header-icon-btn {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  cursor: pointer;
  color: #6b7280;
  transition: all 0.2s;
  flex-shrink: 0;
}

.header-icon-btn:hover {
  background: #f0fdf4;
  color: #18a058;
}

.user-info {
  display: flex;
  align-items: center;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 8px;
  transition: background 0.2s;
}

.user-info:hover {
  background: #f3f4f6;
}

.username {
  margin-left: 10px;
  font-size: 14px;
  color: #1f2937;
  white-space: nowrap;
}

/* ===== 标签栏：透明底，仅白色 tab 页签悬浮，页签之外的区域露出页面底色。
   无左右内边距：页签左缘与下方卡片左缘对齐 ===== */
.tabs-bar {
  margin: var(--card-gap) var(--side-gap) 0;
  height: 44px;
  flex-shrink: 0;
  padding: 0;
  display: flex;
  align-items: flex-end;
  background: transparent;
}

/* ===== 内容区：透明底，页面卡片悬浮；纵向伸展填满剩余高度。
   默认顶部保留一张卡片的间隙；拼接态（搜索卡片接标签栏）顶部无间隙 ===== */
.layout-content {
  flex: 1;
  min-height: 0;
  padding: var(--card-gap) var(--side-gap) var(--side-gap);
  overflow-y: auto;
  background: transparent;
}

.layout-content--joined {
  padding-top: 0;
}

/* 拼接态搜索卡片：仅左上角直角（页签连接处），其余三角随卡片圆角 */
.layout-content--joined :deep(.search-card.n-card) {
  border-radius: 0 var(--card-radius) var(--card-radius) var(--card-radius);
}

</style>
