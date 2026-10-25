<template>
  <div class="tabs-view" :class="{ 'tabs-view--joined': joined }">
    <n-tabs
      type="card"
      size="medium"
      :value="tabStore.activeTab"
      class="tabs-nav"
      @update:value="handleTabClick"
      @close="handleClose"
    >
      <n-tab
        v-for="tab in tabStore.tabs"
        :key="tab.path"
        :name="tab.path"
        :closable="!tab.affix"
      >
        {{ tab.title }}
      </n-tab>
    </n-tabs>
  </div>
</template>

<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'
import { useTabStore } from '@/store/modules/tab'

defineProps<{
  /** 页签与通栏搜索卡片拼接为一体 */
  joined?: boolean
}>()

const router = useRouter()
const route = useRoute()
const tabStore = useTabStore()

function handleTabClick(path: string) {
  if (path !== route.path) {
    router.push(path)
  }
}

function handleClose(path: string) {
  const currentPath = route.path
  tabStore.removeTab(path)

  // 如果关闭的是当前标签，跳转到新的 active 标签
  if (currentPath === path) {
    router.push(tabStore.activeTab)
  }
}
</script>

<style scoped>
/* 标签栏整体透明（见 layouts/default/index.vue 的 .tabs-bar），
   只有页签悬浮，页签右侧露出页面底色 */
.tabs-view {
  width: 100%;
}

.tabs-nav {
  height: 100%;
}

/* scroll-content 是页签（tab-wrapper）的直接 flex 容器：页签贴底、页签间留间隙 */
.tabs-view :deep(.n-tabs-nav-scroll-content) {
  align-items: flex-end;
  gap: 12px;
}

/* card 型页签覆盖：选择器沿用 naive 的 .n-tabs.n-tabs--card-type .n-tabs-tab 路径，
   前置 scoped 根后特异性 (0,5,0)，高于 naive 注入规则 (0,4,0)——
   naive 把 card 型选中页签背景写死为透明 */

/* 默认（无搜索区）：页签为独立白色 chip，四角圆角 */
.tabs-view :deep(.n-tabs.n-tabs--card-type .n-tabs-tab) {
  background-color: var(--card-bg);
  border: 1px solid var(--tab-border);
  border-radius: 6px;
}

/* 拼接态：页签上圆下直，坐在通栏搜索卡片上。
   margin-bottom:-1px 向下重叠 1px，消除页签与卡片之间 1px 的底色缝隙；
   z-index 让页签压在卡片之上，盖住卡片顶部外阴影形成的淡线 */
.tabs-view--joined :deep(.n-tabs.n-tabs--card-type .n-tabs-tab) {
  border-radius: 8px 8px 0 0;
  margin-bottom: -1px;
  z-index: 1;
}

/* 非选中页签为浅灰绿色块，与白色搜索卡片以色差分界；
   底边与块同色，块直接坐在白卡上不留灰色边框线 */
.tabs-view--joined :deep(.n-tabs.n-tabs--card-type .n-tabs-tab:not(.n-tabs-tab--active)) {
  background-color: var(--tab-muted-bg);
  border-color: var(--tab-muted-border);
  border-bottom-color: var(--tab-muted-bg);
}

/* 选中页签边框全部转白：白色从页签盒子边缘开始，
   与无边框搜索卡片左右对齐、无缝连通 */
.tabs-view--joined :deep(.n-tabs.n-tabs--card-type .n-tabs-tab--active) {
  background-color: var(--card-bg);
  border-color: var(--card-bg);
}

/* 关闭按钮：非激活页签默认透明且缩小、不响应点击；
   悬浮页签或激活页签时以缩放 + 淡入动画显现。
   按钮始终占位，显现与隐藏都不引起页签宽度变化 */
.tabs-view :deep(.n-tabs.n-tabs--card-type .n-tabs-tab__close) {
  opacity: 0;
  transform: scale(0.6);
  pointer-events: none;
  transition:
    opacity 0.2s cubic-bezier(0.4, 0, 0.2, 1),
    transform 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

.tabs-view :deep(.n-tabs.n-tabs--card-type .n-tabs-tab:hover .n-tabs-tab__close),
.tabs-view :deep(.n-tabs.n-tabs--card-type .n-tabs-tab--active .n-tabs-tab__close) {
  opacity: 1;
  transform: scale(1);
  pointer-events: auto;
}
</style>
