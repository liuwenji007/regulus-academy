import { getLLMInfo, getDomains, type LLMInfo, type DomainSummary } from '../lib/api'
import { renderSidebar, setSidebarLLMStatus, type NavKey, type SidebarContext } from './sidebar'
import { iconMenu, iconChevronRight } from '../lib/icons'

let shellRoot: HTMLElement | null = null
let contentEl: HTMLElement | null = null
let breadcrumbEl: HTMLElement | null = null
let sidebarBound = false
let lastSidebarCtx: SidebarContext = { active: 'home' }
let cachedCourses: DomainSummary[] | null = null
let sidebarUpdateSeq = 0
let coursesFetchPromise: Promise<DomainSummary[]> | null = null

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
          <button type="button" class="sidebar-toggle" id="sidebar-toggle" aria-label="打开菜单" aria-expanded="false" aria-controls="sidebar">
            ${iconMenu()}
          </button>
          <nav class="breadcrumb" id="breadcrumb" aria-label="面包屑"></nav>
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

/** 新建课程后刷新侧边栏课程列表 */
export function invalidateSidebarCourses(): void {
  cachedCourses = null
  coursesFetchPromise = null
}

function mergeCurrentDomain(courses: DomainSummary[], ctx: SidebarContext): DomainSummary[] {
  if (!ctx.domainId || !ctx.domainName) return courses

  const idx = courses.findIndex((c) => c.id === ctx.domainId)
  if (idx >= 0) {
    if (ctx.domainNodeTotal === undefined) return courses
    const next = [...courses]
    next[idx] = {
      ...next[idx],
      name: ctx.domainName,
      nodeTotal: ctx.domainNodeTotal,
      completed: ctx.domainCompleted ?? next[idx].completed,
    }
    return next
  }

  return [
    {
      id: ctx.domainId,
      name: ctx.domainName,
      createdAt: new Date().toISOString(),
      nodeTotal: ctx.domainNodeTotal ?? 0,
      completed: ctx.domainCompleted ?? 0,
    },
    ...courses,
  ]
}

async function loadSidebarCourses(force: boolean): Promise<{ courses: DomainSummary[]; error: boolean }> {
  if (!force && cachedCourses !== null && cachedCourses.length > 0) {
    return { courses: cachedCourses, error: false }
  }

  if (!coursesFetchPromise || force) {
    coursesFetchPromise = getDomains()
      .then((list) => {
        cachedCourses = list
        return list
      })
      .catch(() => {
        if (cachedCourses === null) cachedCourses = []
        throw new Error('load courses failed')
      })
  }

  try {
    const courses = await coursesFetchPromise
    return { courses, error: false }
  } catch {
    return {
      courses: cachedCourses ?? [],
      error: (cachedCourses ?? []).length === 0,
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
  })

  // 保留 LLM 状态（sidebar 重绘后需写回）
  void refreshLLMStatus()
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
  try {
    const info = await getLLMInfo()
    setSidebarLLMStatus(shellRoot, renderLLMBadge(info))
  } catch {
    setSidebarLLMStatus(
      shellRoot,
      '<div class="sidebar-llm-badge sidebar-llm-badge--error">后端未连接</div>'
    )
  }
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
  return 'home'
}

function escapeHtml(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}
