import { iconHome, iconSparkles, iconTree, iconGraph, iconCourses } from '../lib/icons'
import type { DomainSummary } from '../lib/api'

export type NavKey = 'home' | 'graph' | 'courses' | 'tree' | 'coach' | 'settings'

export interface SidebarContext {
  active: NavKey
  domainId?: string
  domainName?: string
  domainNodeTotal?: number
  domainCompleted?: number
  nodeTitle?: string
  userName?: string
  courses?: DomainSummary[]
  coursesError?: boolean
}

export function renderSidebar(ctx: SidebarContext): string {
  const courses = ctx.courses ?? []
  const activeDomainId = ctx.domainId ?? ''

  const coursesNavActive = ctx.active === 'courses' || ctx.active === 'tree' || ctx.active === 'coach'

  let coursesHtml: string
  if (ctx.coursesError) {
    coursesHtml = `<p class="sidebar-courses-empty">无法加载课程列表<br><span class="sidebar-courses-hint">请硬刷新页面（Cmd+Shift+R）清除旧缓存</span></p>`
  } else if (courses.length > 0) {
    coursesHtml = courses
      .map((c) => {
        const isActive =
          (ctx.active === 'tree' || ctx.active === 'coach') && c.id === activeDomainId
        const pct = c.nodeTotal > 0 ? Math.round((c.completed / c.nodeTotal) * 100) : 0
        return `
          <a href="#/tree/${c.id}" class="sidebar-tree-item ${isActive ? 'is-active' : ''}" data-nav="tree">
            <span class="sidebar-tree-item-icon">${iconTree()}</span>
            <span class="sidebar-tree-item-body">
              <span class="sidebar-tree-item-name">${escapeHtml(c.name)}</span>
              <span class="sidebar-tree-item-meta">${c.completed}/${c.nodeTotal} 节点 · ${pct}%</span>
            </span>
          </a>
        `
      })
      .join('')
  } else {
    coursesHtml = `<p class="sidebar-courses-empty">暂无课程<br><span class="sidebar-courses-hint">在「开始学习」输入主题即可生成</span></p>`
  }

  return `
    <aside class="sidebar" id="sidebar" aria-label="主导航">
      <div class="sidebar-brand">
        <div class="sidebar-logo" aria-hidden="true">${iconSparkles()}</div>
        <div class="sidebar-brand-text">
          <span class="sidebar-brand-name">Regulus Academy</span>
          <span class="sidebar-brand-tag">AI 学习教练</span>
        </div>
      </div>

      <div class="sidebar-body">
        <nav class="sidebar-nav" aria-label="主导航">
          <a href="#/" class="sidebar-link ${ctx.active === 'home' ? 'is-active' : ''}" data-nav="home">
            <span class="sidebar-link-icon">${iconHome()}</span>
            <span class="sidebar-link-label">开始学习</span>
          </a>
          <a href="#/graph" class="sidebar-link ${ctx.active === 'graph' ? 'is-active' : ''}" data-nav="graph">
            <span class="sidebar-link-icon">${iconGraph()}</span>
            <span class="sidebar-link-label">知识图谱</span>
          </a>
          <a href="#/courses" class="sidebar-link ${coursesNavActive ? 'is-active' : ''}" data-nav="courses">
            <span class="sidebar-link-icon">${iconCourses()}</span>
            <span class="sidebar-link-label">我的课程</span>
          </a>
        </nav>

        ${ctx.active === 'coach' && ctx.nodeTitle ? `
          <div class="sidebar-current-lesson" aria-label="当前学习节点">
            <span class="sidebar-current-lesson-label">正在学习</span>
            <p class="sidebar-current-lesson-title" title="${escapeHtml(ctx.nodeTitle)}">${escapeHtml(ctx.nodeTitle)}</p>
            ${
              ctx.domainId && ctx.domainName
                ? `<a href="#/tree/${ctx.domainId}" class="sidebar-current-lesson-course">${escapeHtml(ctx.domainName)}</a>`
                : ''
            }
          </div>
        ` : ''}

        <section class="sidebar-trees" aria-label="课程快捷入口">
          <h2 class="sidebar-section-title">
            <span class="sidebar-section-icon">${iconTree()}</span>
            课程快捷
          </h2>
          <div class="sidebar-trees-list">${coursesHtml}</div>
        </section>
      </div>

      <div class="sidebar-footer">
        <div id="sidebar-llm" class="sidebar-llm"></div>
        <div class="sidebar-profile">
          <button type="button" class="sidebar-profile-btn" id="switch-profile-btn" title="切换学习角色">
            <span class="sidebar-profile-avatar" aria-hidden="true">${escapeHtml((ctx.userName ?? '?').slice(0, 1))}</span>
            <span class="sidebar-profile-body">
              <span class="sidebar-profile-name">${escapeHtml(ctx.userName ?? '未选择')}</span>
              <span class="sidebar-profile-action">切换角色</span>
            </span>
          </button>
        </div>
      </div>
    </aside>
    <div class="sidebar-backdrop" id="sidebar-backdrop" hidden></div>
  `
}

export function setSidebarLLMStatus(root: HTMLElement, html: string): void {
  const el = root.querySelector('#sidebar-llm')
  if (el) el.innerHTML = html
}

function escapeHtml(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}
