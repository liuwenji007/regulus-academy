import { defineConfig } from 'vitepress'

const demoUrl = process.env.VITE_DEMO_URL || 'https://demo.awoshuile.cn'
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
        content: '有边界的知识地图 · 会追着你练完的教练 · 越学越懂你。在线体验或 Docker 自托管。',
      },
    ],
    ['link', { rel: 'icon', href: '/logo.png', type: 'image/png' }],
  ],
  themeConfig: {
    logo: '/logo.png',
    siteTitle: 'Regulus Academy',
    nav: [
      { text: '快速上手', link: '/guide/quick-start' },
      { text: '为什么是 Regulus', link: '/guide/why-regulus' },
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
          { text: '行动助手', link: '/guide/action-assistant' },
          { text: '学习画像', link: '/guide/learning-profile' },
          { text: '课程体检', link: '/guide/course-audit' },
          { text: '知识图谱', link: '/guide/knowledge-graph' },
          { text: '界面预览', link: '/guide/screenshots' },
        ],
      },
      {
        text: '部署与配置',
        items: [
          { text: '自托管部署', link: '/guide/self-host' },
          { text: '本地开发', link: '/guide/development' },
          { text: '环境变量', link: '/reference/env' },
          { text: 'AI 模型', link: '/guide/model-config' },
          { text: 'IM 频道', link: '/guide/im' },
          { text: 'Coach Skill 下载', link: '/guide/agent-offline' },
          { text: '导出学习笔记', link: '/guide/learning-notes' },
        ],
      },
      {
        text: '了解与参与',
        items: [
          { text: '为什么是 Regulus', link: '/guide/why-regulus' },
          { text: '教学模式', link: '/guide/teaching-model' },
          { text: '参与贡献', link: '/guide/contributing' },
          { text: '教学质量', link: '/guide/contributing-teaching' },
          { text: '功能一览', link: '/guide/features' },
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
