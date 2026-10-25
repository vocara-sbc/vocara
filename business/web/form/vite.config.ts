import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

// 库模式构建：自包含 IIFE（Vue/Naive UI 全部内联），全局对象 VocaraForm；
// 产物直出 admin 的 public/form（与 web/phonebar 的 build→同步同一模式）
export default defineConfig({
  plugins: [vue()],
  // IIFE 直出浏览器即为最终产物，没有下游打包器承接环境变量替换；
  // 显式钉住 NODE_ENV，内联依赖（Vue/Naive UI）中的开发态分支才会被编译移除
  define: {
    'process.env.NODE_ENV': JSON.stringify('production'),
  },
  build: {
    outDir: resolve(__dirname, '../admin/public/form'),
    emptyOutDir: true,
    lib: {
      entry: resolve(__dirname, 'src/main.ts'),
      name: 'VocaraForm',
      formats: ['iife'],
      fileName: () => 'vocara-form.js',
      cssFileName: 'vocara-form',
    },
    rollupOptions: {
      output: {
        // 库模式 CSS 产物与 JS 同名：默认的 style.css 固定输出为 vocara-form.css
        assetFileNames: (assetInfo) =>
          assetInfo.name === 'style.css' ? 'vocara-form.css' : '[name][extname]',
      },
    },
  },
})
