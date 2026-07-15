import { defineConfig } from 'vitepress'

const demoUrl = process.env.VITE_DEMO_URL || 'https://regulus-academy-web-production.up.railway.app'
const githubUrl = process.env.VITE_GITHUB_URL || 'https://github.com/liuwenji007/regulus-academy'
const designUrl = `${githubUrl}/blob/main/DESIGN.md`
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
    ['link', { rel: 'icon', href: '/logo.png', type: 'image/png' }],
  ],
  themeConfig: {
    logo: '/logo.png',
    siteTitle: 'Regulus Academy',
    nav: [
      { text: '快速上手', link: '/guide/quick-start' },
      { text: '功能', link: '/guide/features' },
      { text: '立即体验', link: demoUrl, target: '_blank' },
      { text: 'GitHub', link: githubUrl, target: '_blank' },
    ],
    sidebar: [
      {
        text: '快速开始',
        items: [
          { text: '快速上手', link: '/guide/quick-start' },
          { text: '教练流程', link: '/guide/coach-flow' },
          { text: '在线体验版', link: '/guide/cloud-demo' },
        ],
      },
      {
        text: '功能介绍',
        items: [
          { text: '功能一览', link: '/guide/features' },
          { text: '行动助手', link: '/guide/action-assistant' },
          { text: 'Coach Skill 下载', link: '/guide/agent-offline' },
          { text: '知识图谱', link: '/guide/knowledge-graph' },
          { text: 'AI 模型', link: '/guide/model-config' },
          { text: 'IM 频道', link: '/guide/im' },
          { text: '导出学习笔记', link: '/guide/learning-notes' },
          { text: '界面预览', link: '/guide/screenshots' },
        ],
      },
      {
        text: '教学理念',
        items: [
          { text: '教学模式', link: '/guide/teaching-model' },
          { text: '设计理念', link: designUrl, target: '_blank' },
        ],
      },
      {
        text: '开发与部署',
        items: [
          { text: '自托管部署', link: '/guide/self-host' },
          { text: '本地开发', link: '/guide/development' },
          { text: '环境变量', link: '/reference/env' },
        ],
      },
      {
        text: '贡献',
        items: [
          { text: '参与贡献', link: '/guide/contributing' },
          { text: '教学质量', link: '/guide/contributing-teaching' },
        ],
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
