import { getLLMInfo, getDomains, type LLMInfo, type DomainSummary } from '../lib/api'
import { isAppBusy } from '../lib/app-busy'
import { getActiveProfile } from '../lib/profile'
import { renderSidebar, setSidebarLLMStatus, type NavKey, type SidebarContext } from './sidebar'
import { iconMenu, iconChevronRight, iconSettings } from '../lib/icons'

let shellRoot: HTMLElement | null = null
let contentEl: HTMLElement | null = null
let breadcrumbEl: HTMLElement | null = null
let sidebarBound = false
let lastSidebarCtx: SidebarContext = { active: 'home' }
let cachedCourses: DomainSummary[] | null = null
let sidebarUpdateSeq = 0
let coursesFetchGen = 0
let coursesFetchPromise: Promise<DomainSummary[]> | null = null
let lastLLMBadgeHtml: string | null = null
let llmRefreshSeq = 0

export function getContentEl(): HTMLElement {
  if (!contentEl) throw new Error('App shell not mounted')
  return contentEl
}

export function mountAppShell(app: HTMLElement): HTMLElement {
  if (shellRoot) return contentEl!

  app.innerHTML = `
    <div class="app-shell" id="app-shell">
      <div id="sidebar-slot"></div>
      <div class="main-panel">
        <header class="main-header">
          <div class="main-header-start">
            <button type="button" class="sidebar-toggle" id="sidebar-toggle" aria-label="打开菜单" aria-expanded="false" aria-controls="sidebar">
              ${iconMenu()}
            </button>
            <nav class="breadcrumb" id="breadcrumb" aria-label="面包屑"></nav>
          </div>
          <div class="main-header-actions">
            <a href="#/settings" class="header-settings-btn" id="header-settings-btn" aria-label="设置" title="设置">
              ${iconSettings()}
            </a>
          </div>
        </header>
        <main class="main-content" id="main-content" tabindex="-1">
          <div class="main-content__inner" id="page-content"></div>
        </main>
      </div>
    </div>
  `

  shellRoot = app.querySelector('#app-shell')
  contentEl = app.querySelector('#page-content')!
  breadcrumbEl = app.querySelector('#breadcrumb')!

  void updateSidebar({ active: 'home' })
  void refreshLLMStatus()
  bindSidebarOnce(app.querySelector('#app-shell')!)
  return contentEl
}

function bindSidebarOnce(root: HTMLElement): void {
  if (sidebarBound) return
  sidebarBound = true

  root.addEventListener('click', (e) => {
    const target = e.target as HTMLElement
    const toggle = target.closest('#sidebar-toggle')
    const backdrop = target.closest('#sidebar-backdrop')
    const link = target.closest<HTMLAnchorElement>(
      '.sidebar-link:not(.is-disabled), .sidebar-tree-item'
    )

    const sidebar = root.querySelector<HTMLElement>('#sidebar')
    const toggleBtn = root.querySelector<HTMLButtonElement>('#sidebar-toggle')
    const backdropEl = root.querySelector<HTMLDivElement>('#sidebar-backdrop')

    if (toggle) {
      const open = sidebar?.classList.toggle('is-open')
      if (open) {
        backdropEl?.removeAttribute('hidden')
        toggleBtn?.setAttribute('aria-expanded', 'true')
      } else {
        backdropEl?.setAttribute('hidden', '')
        toggleBtn?.setAttribute('aria-expanded', 'false')
      }
    }

    if (backdrop) {
      sidebar?.classList.remove('is-open')
      backdropEl?.setAttribute('hidden', '')
      toggleBtn?.setAttribute('aria-expanded', 'false')
    }

    if (link && window.matchMedia('(max-width: 768px)').matches) {
      sidebar?.classList.remove('is-open')
      backdropEl?.setAttribute('hidden', '')
      toggleBtn?.setAttribute('aria-expanded', 'false')
    }
  })
}

/** 新建 / 删除课程后刷新侧边栏课程列表 */
export function invalidateSidebarCourses(): void {
  cachedCourses = null
  coursesFetchPromise = null
  coursesFetchGen++
}

/** 切换学习角色后：丢弃旧用户课程缓存与「当前课」上下文，避免快捷列表串号 */
export function resetSidebarAfterProfileChange(): void {
  invalidateSidebarCourses()
  sidebarUpdateSeq++
  lastSidebarCtx = { active: 'home' }
}

const PLACEHOLDER_DOMAIN_NAMES = new Set(['当前课程', '课程'])

function isPlaceholderDomainName(name?: string): boolean {
  const t = name?.trim()
  return !t || PLACEHOLDER_DOMAIN_NAMES.has(t)
}

/** 用页面上下文刷新当前课进度；避免占位名 / 0 节点覆盖列表 API 已有数据 */
function mergeCurrentDomain(courses: DomainSummary[], ctx: SidebarContext): DomainSummary[] {
  if (!ctx.domainId) return courses

  const nameOk = !isPlaceholderDomainName(ctx.domainName)
  const totalsOk = ctx.domainNodeTotal !== undefined && ctx.domainNodeTotal > 0

  const idx = courses.findIndex((c) => c.id === ctx.domainId)
  if (idx >= 0) {
    if (!nameOk && !totalsOk && ctx.domainCompleted === undefined) return courses
    const cur = courses[idx]
    const next = [...courses]
    next[idx] = {
      ...cur,
      ...(nameOk ? { name: ctx.domainName!.trim() } : {}),
      ...(totalsOk
        ? {
            nodeTotal: ctx.domainNodeTotal!,
            completed: ctx.domainCompleted ?? cur.completed,
          }
        : ctx.domainCompleted !== undefined && cur.nodeTotal > 0
          ? { completed: ctx.domainCompleted }
          : {}),
    }
    return next
  }

  if (!nameOk && !totalsOk) return courses

  return [
    {
      id: ctx.domainId,
      name: nameOk ? ctx.domainName!.trim() : '我的课程',
      createdAt: new Date().toISOString(),
      nodeTotal: totalsOk ? ctx.domainNodeTotal! : 0,
      completed: ctx.domainCompleted ?? 0,
    },
    ...courses,
  ]
}

async function loadSidebarCourses(force: boolean): Promise<{ courses: DomainSummary[]; error: boolean }> {
  const gen = coursesFetchGen
  if (!force && cachedCourses !== null && cachedCourses.length > 0) {
    return { courses: cachedCourses, error: false }
  }

  if (!coursesFetchPromise || force) {
    const fetchGen = coursesFetchGen
    coursesFetchPromise = getDomains().then((list) => {
      if (fetchGen === coursesFetchGen) {
        cachedCourses = list
      }
      return list
    })
  }

  try {
    let courses = await coursesFetchPromise
    if (gen !== coursesFetchGen) {
      return loadSidebarCourses(true)
    }
    return { courses, error: false }
  } catch {
    if (gen !== coursesFetchGen) {
      return loadSidebarCourses(true)
    }
    const fallback = cachedCourses ?? []
    return {
      courses: fallback,
      error: fallback.length === 0 && !isAppBusy() && !lastSidebarCtx.domainId,
    }
  } finally {
    coursesFetchPromise = null
  }
}

export async function updateSidebar(ctx: Partial<SidebarContext>): Promise<void> {
  if (!shellRoot) return
  lastSidebarCtx = { ...lastSidebarCtx, ...ctx }
  const seq = ++sidebarUpdateSeq

  let courses: DomainSummary[]
  let coursesError = false

  if (ctx.courses !== undefined) {
    cachedCourses = ctx.courses
    courses = ctx.courses
  } else {
    const force = cachedCourses === null
    const loaded = await loadSidebarCourses(force)
    if (seq !== sidebarUpdateSeq) return
    courses = loaded.courses
    coursesError = loaded.error
  }

  courses = mergeCurrentDomain(courses, lastSidebarCtx)
  if (seq !== sidebarUpdateSeq) return

  const slot = shellRoot.querySelector('#sidebar-slot')
  if (!slot) return
  slot.innerHTML = renderSidebar({
    ...lastSidebarCtx,
    courses,
    coursesError,
    userName: getActiveProfile()?.displayName,
  })

  // 保留 LLM 状态（sidebar 重绘后需写回）
  void refreshLLMStatus()
  syncHeaderNav(lastSidebarCtx.active)
}

function syncHeaderNav(active: NavKey): void {
  const btn = shellRoot?.querySelector<HTMLAnchorElement>('#header-settings-btn')
  if (!btn) return
  btn.classList.toggle('is-active', active === 'settings')
}

export function setBreadcrumb(items: { label: string; href?: string }[]): void {
  if (!breadcrumbEl) return
  if (items.length === 0) {
    breadcrumbEl.innerHTML = ''
    return
  }
  breadcrumbEl.innerHTML = items
    .map((item, i) => {
      const isLast = i === items.length - 1
      const sep = i > 0 ? `<span class="breadcrumb-sep">${iconChevronRight()}</span>` : ''
      if (isLast || !item.href) {
        return `${sep}<span class="breadcrumb-item is-current" aria-current="page">${escapeHtml(item.label)}</span>`
      }
      return `${sep}<a href="${item.href}" class="breadcrumb-item">${escapeHtml(item.label)}</a>`
    })
    .join('')
}

export async function refreshLLMStatus(): Promise<void> {
  if (!shellRoot) return
  const seq = ++llmRefreshSeq

  if (isAppBusy()) {
    setSidebarLLMStatus(
      shellRoot,
      '<div class="sidebar-llm-badge sidebar-llm-badge--loading"><span class="sidebar-llm-dot" aria-hidden="true"></span><span class="sidebar-llm-text">课程准备中…</span></div>'
    )
    return
  }

  try {
    const info = await getLLMInfo()
    if (seq !== llmRefreshSeq) return
    lastLLMBadgeHtml = renderLLMBadge(info)
    setSidebarLLMStatus(shellRoot, lastLLMBadgeHtml)
  } catch {
    if (seq !== llmRefreshSeq) return
    if (lastLLMBadgeHtml) {
      setSidebarLLMStatus(shellRoot, lastLLMBadgeHtml)
      return
    }
    setSidebarLLMStatus(
      shellRoot,
      '<div class="sidebar-llm-badge sidebar-llm-badge--error">后端未连接</div>'
    )
  }
}

/** 长耗时建课结束后刷新侧边栏 LLM 状态（避免一直显示「准备中」） */
export function refreshLLMStatusAfterBusy(): void {
  void refreshLLMStatus()
}

function renderLLMBadge(info: LLMInfo): string {
  if (!info.configured) {
    return '<div class="sidebar-llm-badge sidebar-llm-badge--warn">LLM 未配置</div>'
  }
  return `<div class="sidebar-llm-badge sidebar-llm-badge--ok">
    <span class="sidebar-llm-dot" aria-hidden="true"></span>
    <span class="sidebar-llm-text">${escapeHtml(info.provider)} · ${escapeHtml(info.model)}</span>
  </div>`
}

export function navFromHash(hash: string): NavKey {
  if (hash.match(/^\/coach\//)) return 'coach'
  if (hash.match(/^\/tree\//)) return 'tree'
  if (hash === '/graph') return 'graph'
  if (hash === '/courses') return 'courses'
  if (hash === '/settings' || hash.startsWith('/settings/')) return 'settings'
  return 'home'
}

function escapeHtml(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}
