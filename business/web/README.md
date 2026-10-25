# web — 前端项目集合

本目录承载业务区的前端工程：管理后台应用与动态表单 SDK。两者都基于 Vue 3 + TypeScript + Vite。

## 项目列表

### [`admin/`](./admin) — 管理后台

管理后台单页应用，技术栈 Vue 3 + TypeScript + Vite + Naive UI + Pinia + Vue Router，开发端口 `5173`。

- 布局：左侧全高白色菜单 + 右侧浅灰绿底三段白色悬浮卡片；主题色绿色 `#18a058`。
- 电话条在 Header 绝对居中、全局常驻，菜单切换不影响通话状态。
- 电话条本身不属本目录源码，其混淆构建产物由核心私有仓库 `phonebar/` 提供并同步到 `admin/public/phonebar/`。

常用命令：

```bash
cd business/web/admin
npm run dev          # 本地开发
npm run type-check   # 类型检查（vue-tsc）
npm run build        # 类型检查 + 生产构建
```

### [`form/`](./form) — 动态表单 SDK

动态表单的设计器 / 渲染器 / 实例管理，作为可嵌入的库构建。技术栈 Vue 3 + TypeScript + Vite + Naive UI + vue-draggable-plus。

- Vite 库模式，IIFE 格式，全局名 `VocaraForm`。
- 构建产物 `vocara-form.js` / `vocara-form.css` 直接输出到 `admin/public/form/`，供管理后台加载。

常用命令：

```bash
cd business/web/form
npm run build        # 类型检查 + 库构建，输出到 ../admin/public/form
npm run test:e2e     # 冒烟检查
```

## 构建顺序提示

表单 SDK 与电话条的产物都落在 `admin/public/` 下。全新环境建议先构建 `form`（及在需要时在核心仓库经 `make build-phonebar` 构建电话条），再构建 `admin`，使管理后台打包到最新的子产物。
