import { iconHome, iconSparkles, iconTree, iconGraph, iconCourses, iconAssistant } from '../lib/icons'
import type { LearningShortcuts, LastLessonShortcut, ShortcutRecommendation } from '../lib/api'

export type NavKey = 'home' | 'graph' | 'courses' | 'tree' | 'coach' | 'assistant' | 'settings'

export interface SidebarContext {
  active: NavKey
  domainId?: string
  domainName?: string
  domainNodeTotal?: number
  domainCompleted?: number
  nodeTitle?: string
  userName?: string
  shortcuts?: LearningShortcuts | null
  shortcutsError?: boolean
  /** @deprecated 侧栏不再展示全量课程列表；保留字段以免旧调用报错 */
  courses?: unknown
  coursesError?: boolean
}

export function renderSidebar(ctx: SidebarContext): string {
  const shortcuts = ctx.shortcuts
  const last = shortcuts?.lastLesson ?? null
  const recs = shortcuts?.recommendations ?? []
  const hasCourses = shortcuts?.hasCourses ?? false
  const activeDomainId = ctx.domainId ?? ''

  const coursesNavActive = ctx.active === 'courses' || ctx.active === 'tree' || ctx.active === 'coach'

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
          <a href="#/assistant" class="sidebar-link ${ctx.active === 'assistant' ? 'is-active' : ''}" data-nav="assistant">
            <span class="sidebar-link-icon">${iconAssistant()}</span>
            <span class="sidebar-link-label">行动助手</span>
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

        <div class="sidebar-shortcuts" aria-label="学习快捷入口">
          ${renderLastLessonSection(last, ctx, activeDomainId, shortcuts === undefined && !ctx.shortcutsError)}
          ${renderTodaySection(recs, hasCourses, ctx.shortcutsError, shortcuts === undefined && !ctx.shortcutsError)}
        </div>
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

function renderLastLessonSection(
  last: LastLessonShortcut | null,
  ctx: SidebarContext,
  activeDomainId: string,
  loading: boolean
): string {
  let body: string
  if (ctx.shortcutsError) {
    body = `<p class="sidebar-courses-empty">无法加载<br><span class="sidebar-courses-hint">请硬刷新页面后再试</span></p>`
  } else if (loading) {
    body = `<p class="sidebar-courses-empty sidebar-courses-empty--soft">加载中…</p>`
  } else if (!last) {
    body = `<p class="sidebar-courses-empty">还没有上一节课<br><a href="#/" class="sidebar-empty-link">去开始学习</a></p>`
  } else {
    const isCurrent =
      (ctx.active === 'coach' || ctx.active === 'tree') && last.domainId === activeDomainId
    const metaParts: string[] = []
    if (last.nodeTitle) metaParts.push(last.nodeTitle)
    if (isCurrent || last.canResume) metaParts.push(last.canResume ? '进行中' : '已完成')
    const href = last.canResume && last.sessionId ? `#/coach/${last.sessionId}` : `#/tree/${last.domainId}`
    const cta = last.canResume ? '继续练习' : '打开课程'
    body = `
      <a href="${href}" class="sidebar-tree-item ${isCurrent ? 'is-active' : ''}" data-nav="tree">
        <span class="sidebar-tree-item-icon">${iconTree()}</span>
        <span class="sidebar-tree-item-body">
          <span class="sidebar-tree-item-name">${escapeHtml(last.domainName || '课程')}</span>
          <span class="sidebar-tree-item-meta">${escapeHtml(metaParts.join(' · '))}</span>
          <span class="sidebar-tree-item-cta">${cta}</span>
        </span>
      </a>`
  }

  return `
    <section class="sidebar-shortcut-section" aria-label="上一节学的课">
      <h2 class="sidebar-section-title">
        <span class="sidebar-section-icon">${iconTree()}</span>
        上一节学的课
      </h2>
      <div class="sidebar-trees-list">${body}</div>
    </section>`
}

function renderTodaySection(
  recs: ShortcutRecommendation[],
  hasCourses: boolean,
  error: boolean | undefined,
  loading: boolean
): string {
  let body: string
  if (error) {
    body = `<p class="sidebar-courses-empty">无法加载推荐</p>`
  } else if (loading) {
    body = `<p class="sidebar-courses-empty sidebar-courses-empty--soft">加载中…</p>`
  } else if (recs.length === 0) {
    if (!hasCourses) {
      body = `<p class="sidebar-courses-empty">暂无课程<br><a href="#/" class="sidebar-empty-link">去开始学习</a></p>`
    } else {
      body = `<p class="sidebar-courses-empty">还没有今日建议<br><a href="#/assistant" class="sidebar-empty-link">去行动助手整理今日</a></p>`
    }
  } else {
    body = recs
      .map((r) => {
        const href =
          r.canResume && r.sessionId ? `#/coach/${r.sessionId}` : `#/tree/${r.domainId}`
        const pct = r.nodeTotal > 0 ? Math.round((r.completed / r.nodeTotal) * 100) : 0
        const metaParts: string[] = []
        if (r.source === 'planning') {
          if (r.title) metaParts.push(r.title)
          else if (r.nodeTitle) metaParts.push(r.nodeTitle)
          if (r.minutes && r.minutes > 0) metaParts.push(`约 ${r.minutes} 分钟`)
        } else {
          metaParts.push(`${r.completed}/${r.nodeTotal} 节点 · ${pct}%`)
        }
        const badge =
          r.source === 'planning' ? `<span class="sidebar-rec-badge">今日</span>` : ''
        return `
          <a href="${href}" class="sidebar-tree-item" data-nav="tree">
            <span class="sidebar-tree-item-icon">${iconTree()}</span>
            <span class="sidebar-tree-item-body">
              <span class="sidebar-tree-item-name">${badge}${escapeHtml(r.domainName)}</span>
              <span class="sidebar-tree-item-meta">${escapeHtml(metaParts.join(' · '))}</span>
            </span>
          </a>`
      })
      .join('')
  }

  return `
    <section class="sidebar-shortcut-section" aria-label="今日推荐">
      <h2 class="sidebar-section-title">
        <span class="sidebar-section-icon">${iconSparkles()}</span>
        今日推荐
      </h2>
      <div class="sidebar-trees-list">${body}</div>
    </section>`
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
