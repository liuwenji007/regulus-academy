import { defineConfig } from 'vite'
import { VitePWA } from 'vite-plugin-pwa'

export default defineConfig({
  plugins: [
    VitePWA({
      registerType: 'autoUpdate',
      workbox: {
        // 避免 SW 把 /api 请求回退成 index.html
        navigateFallbackDenylist: [/^\/api/, /^\/health/],
        runtimeCaching: [],
      },
      devOptions: {
        enabled: false,
      },
      manifest: {
        name: 'Regulus Academy',
        short_name: 'Regulus',
        description: '碎片化学习 AI 私教',
        theme_color: '#f7f3eb',
        background_color: '#f7f3eb',
        display: 'standalone',
        lang: 'zh-CN',
        start_url: '/',
        icons: [
          {
            src: '/icon.svg',
            sizes: 'any',
            type: 'image/svg+xml',
            purpose: 'any maskable',
          },
        ],
      },
    }),
  ],
  server: {
    port: Number(process.env.VITE_DEV_PORT) || 5173,
    proxy: {
      '/api': process.env.VITE_DEV_API_TARGET || 'http://localhost:8080',
      '/health': process.env.VITE_DEV_API_TARGET || 'http://localhost:8080',
    },
  },
})
