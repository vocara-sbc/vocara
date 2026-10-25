import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import basicSsl from '@vitejs/plugin-basic-ssl'
import { resolve } from 'path'
import viteCompression from 'vite-plugin-compression'

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd())

  return {
    base: './',
    plugins: [
      vue(),
      // https 启动（自签证书，开发环境用；浏览器首次访问需确认安全例外）
      // 目的：WebRTC getUserMedia 要求安全上下文，IP 访问时 http 会被浏览器禁用麦克风
      basicSsl(),
      // gzip 压缩
      viteCompression({
        verbose: true,
        disable: mode !== 'production',
        threshold: 10240,
        algorithm: 'gzip',
        ext: '.gz',
      }),
    ],
    resolve: {
      alias: {
        '@': resolve(__dirname, 'src'),
      },
    },
    css: {
      preprocessorOptions: {
        less: {
          javascriptEnabled: true,
          modifyVars: {
            // 主题色（青色系）
            'primary-color': '#00b894',
          },
        },
      },
    },
    server: {
      host: '0.0.0.0',
      port: 5173,
      open: false,
      proxy: {
        // 代理目标是后端源站，不要用 VITE_API_BASE_URL（那是前端请求前缀 /api）
        '/api': {
          target: env.VITE_PROXY_TARGET || 'http://localhost:8080',
          changeOrigin: true,
        },
        '/ws': {
          target: env.VITE_WS_PROXY_TARGET || 'ws://localhost:8084',
          changeOrigin: true,
          ws: true,
        },
        // 呼叫控制服务（callctrl 8083），前端走 /callapi/**
        '/callapi': {
          target: env.VITE_CALL_PROXY_TARGET || 'http://localhost:8083',
          changeOrigin: true,
          rewrite: (path) => path.replace(/^\/callapi/, '/api/v1/call'),
        },
        // 座席中心（agentcenter 8081），前端走 /agentapi/**
        '/agentapi': {
          target: env.VITE_AGENT_PROXY_TARGET || 'http://localhost:8081',
          changeOrigin: true,
          rewrite: (path) => path.replace(/^\/agentapi/, '/api/v1/agent'),
        },
        // SIP over WebSocket（座席侧 FS 5066），前端走 /sip-ws/**
        // https 页面不允许 ws:// 明文连接（mixed content），经 vite 代理后浏览器侧为 wss 同源
        '/sip-ws': {
          target: env.VITE_SIP_PROXY_TARGET || 'ws://127.0.0.1:5066',
          changeOrigin: true,
          ws: true,
          // 注意：/sip-ws 恰好命中时 replace 结果是空串，必须回退为 /，
          // 否则请求行路径为空，FS 直接 400
          rewrite: (path) => path.replace(/^\/sip-ws/, '') || '/',
          configure: (proxy) => {
            // FreeSWITCH 的 WS 握手解析对头名大小写敏感（不符合 RFC 头名大小写不敏感规范）：
            // node http-proxy 会把请求头名全部小写化（req.headers 语义），
            // FS 收到小写头名的握手一律 400，导致浏览器经代理永远连不上 sip-ws。
            // 在 proxyReqWs 阶段把握手相关头恢复为 RFC 惯例驼峰即可正常 101。
            proxy.on('proxyReqWs', (proxyReq) => {
              const cases: Array<[string, string]> = [
                ['connection', 'Connection'],
                ['upgrade', 'Upgrade'],
                ['sec-websocket-version', 'Sec-WebSocket-Version'],
                ['sec-websocket-key', 'Sec-WebSocket-Key'],
                ['sec-websocket-protocol', 'Sec-WebSocket-Protocol'],
                ['sec-websocket-extensions', 'Sec-WebSocket-Extensions'],
                ['origin', 'Origin'],
              ]
              for (const [lower, camel] of cases) {
                const val = proxyReq.getHeader(lower)
                if (val !== undefined) {
                  proxyReq.removeHeader(lower)
                  proxyReq.setHeader(camel, val as string)
                }
              }
            })
          },
        },
      },
    },
    build: {
      outDir: 'dist',
      sourcemap: false,
      chunkSizeWarningLimit: 2000,
      rollupOptions: {
        output: {
          // 静态资源分类打包
          chunkFileNames: 'static/js/[name]-[hash].js',
          entryFileNames: 'static/js/[name]-[hash].js',
          assetFileNames: 'static/[ext]/[name]-[hash].[ext]',
          manualChunks: {
            // 拆分大体积包
            vue: ['vue', 'vue-router', 'pinia'],
            'naive-ui': ['naive-ui'],
            echarts: ['echarts', 'vue-echarts'],
          },
        },
      },
    },
  }
})
