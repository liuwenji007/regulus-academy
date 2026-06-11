import { defineConfig } from 'vitepress'

const demoUrl = process.env.VITE_DEMO_URL || 'https://regulus-academy-web-production.up.railway.app'
const githubUrl = process.env.VITE_GITHUB_URL || 'https://github.com/liuwenji007/regulus-academy'

export default defineConfig({
  title: 'Regulus Academy',
  description: '碎片化学习 AI 私教 — 使用文档',
  ignoreDeadLinks: [/localhost/],
  themeConfig: {
    nav: [
      { text: '立即体验', link: demoUrl, target: '_blank' },
      { text: 'GitHub', link: githubUrl, target: '_blank' },
    ],
    sidebar: [
      {
        text: '指南',
        items: [
          { text: '介绍', link: '/' },
          { text: '快速上手', link: '/guide/quick-start' },
          { text: '在线体验版', link: '/guide/cloud-demo' },
          { text: '自托管部署', link: '/guide/self-host' },
        ],
      },
      {
        text: '参考',
        items: [{ text: '环境变量', link: '/reference/env' }],
      },
    ],
    socialLinks: [{ icon: 'github', link: githubUrl }],
  },
})
