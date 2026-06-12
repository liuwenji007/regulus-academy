import { defineConfig } from 'vitepress'

const demoUrl = process.env.VITE_DEMO_URL || 'https://regulus-academy-web-production.up.railway.app'
const githubUrl = process.env.VITE_GITHUB_URL || 'https://github.com/liuwenji007/regulus-academy'
const docsUrl = 'https://regulus-academy-docs.vercel.app'

export default defineConfig({
  title: 'Regulus Academy',
  description: '碎片化学习 AI 私教 — 使用文档',
  ignoreDeadLinks: [/localhost/],
  head: [
    ['meta', { name: 'theme-color', content: '#c45c26' }],
    [
      'meta',
      {
        property: 'og:description',
        content: '讲解 → 练习 → 反馈 → 点亮节点。在线体验或 Docker 自托管。',
      },
    ],
    ['link', { rel: 'icon', href: '/banner.png', type: 'image/png' }],
  ],
  themeConfig: {
    logo: '/banner.png',
    siteTitle: 'Regulus Academy',
    nav: [
      { text: '立即体验', link: demoUrl, target: '_blank' },
      { text: 'GitHub', link: githubUrl, target: '_blank' },
    ],
    sidebar: [
      {
        text: '指南',
        items: [
          { text: '介绍', link: '/' },
          { text: '教学模式', link: '/guide/teaching-model' },
          { text: '教练流程', link: '/guide/coach-flow' },
          { text: '贡献 · 教学质量', link: '/guide/contributing-teaching' },
          { text: '快速上手', link: '/guide/quick-start' },
          { text: '功能一览', link: '/guide/features' },
          { text: '界面预览', link: '/guide/screenshots' },
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
    footer: {
      message: `在线 Demo · <a href="${demoUrl}" target="_blank">立即体验</a>`,
      copyright: `Regulus Academy · <a href="${docsUrl}">文档站</a>`,
    },
    outline: [2, 3],
  },
})
