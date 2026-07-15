import { getLLMConfig, getLearningShortcuts, type LearningShortcuts, type LLMConfigResponse } from '../lib/api'
import { isAppBusy, onAppBusyChange } from '../lib/app-busy'
import { getActiveProfile } from '../lib/profile'
import {
  bindModelSwitcher,
  renderLLMSwitcher,
  setOnLLMChanged,
} from './model-switcher'
import { mountBuildNotification } from './build-notification'
import { resumePendingDomainBuildJob } from '../lib/domain-build-job'
import { renderSidebar, setSidebarLLMStatus, type NavKey, type SidebarContext } from './sidebar'
import { iconMenu, iconChevronRight, iconSettings } from '../lib/icons'

let shellRoot: HTMLElement | null = null
let contentEl: HTMLElement | null = null
let breadcrumbEl: HTMLElement | null = null
let sidebarBound = false
let lastSidebarCtx: SidebarContext = { active: 'home' }
let cachedShortcuts: LearningShortcuts | null = null
let sidebarUpdateSeq = 0
let shortcutsFetchGen = 0
let shortcutsFetchPromise: Promise<LearningShortcuts> | null = null
let lastLLMBadgeHtml: string | null = null
let llmRefreshSeq = 0
let llmConfigFetchedAt = 0

/** 侧边栏重绘时复用缓存，避免每次 updateSidebar 都打 /api/llm/config */
const LLM_CONFIG_MIN_INTERVAL_MS = 8000

export function publishLLMConfig(cfg: LLMConfigResponse): void {
  lastLLMBadgeHtml = renderLLMSwitcher(cfg)
  llmConfigFetchedAt = Date.now()
  if (shellRoot && lastLLMBadgeHtml) {
    setSidebarLLMStatus(shellRoot, lastLLMBadgeHtml)
  }
}

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

  setOnLLMChanged(() => {
    void refreshLLMStatus(true)
  })
  onAppBusyChange(() => applySidebarLLMBadge())
  mountBuildNotification(app)
  void resumePendingDomainBuildJob({ onReleased: refreshLLMStatusAfterBusy })
  void updateSidebar({ active: 'home' })
  bindSidebarOnce(app.querySelector('#app-shell')!)
  bindModelSwitcher(app.querySelector('#app-shell')!)
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

/** 新建 / 删除课程 / 学完后刷新侧栏上一节与今日推荐 */
export function invalidateSidebarCourses(): void {
  cachedShortcuts = null
  shortcutsFetchPromise = null
  shortcutsFetchGen++
}

/** 切换学习角色后：丢弃旧用户缓存与「当前课」上下文，避免快捷入口串号 */
export function resetSidebarAfterProfileChange(): void {
  invalidateSidebarCourses()
  sidebarUpdateSeq++
  lastSidebarCtx = { active: 'home' }
}

async function loadSidebarShortcuts(force: boolean): Promise<{ shortcuts: LearningShortcuts | null; error: boolean }> {
  const gen = shortcutsFetchGen
  if (!force && cachedShortcuts !== null) {
    return { shortcuts: cachedShortcuts, error: false }
  }

  if (!shortcutsFetchPromise || force) {
    const fetchGen = shortcutsFetchGen
    shortcutsFetchPromise = getLearningShortcuts().then((data) => {
      if (fetchGen === shortcutsFetchGen) {
        cachedShortcuts = data
      }
      return data
    })
  }

  try {
    const shortcuts = await shortcutsFetchPromise
    if (gen !== shortcutsFetchGen) {
      return loadSidebarShortcuts(true)
    }
    return { shortcuts, error: false }
  } catch {
    if (gen !== shortcutsFetchGen) {
      return loadSidebarShortcuts(true)
    }
    const fallback = cachedShortcuts
    return {
      shortcuts: fallback,
      error: fallback === null && !isAppBusy(),
    }
  } finally {
    shortcutsFetchPromise = null
  }
}

export async function updateSidebar(ctx: Partial<SidebarContext>): Promise<void> {
  if (!shellRoot) return
  lastSidebarCtx = { ...lastSidebarCtx, ...ctx }
  const seq = ++sidebarUpdateSeq

  let shortcuts: LearningShortcuts | null | undefined = lastSidebarCtx.shortcuts
  let shortcutsError = Boolean(lastSidebarCtx.shortcutsError)

  if (ctx.shortcuts !== undefined) {
    cachedShortcuts = ctx.shortcuts
    shortcuts = ctx.shortcuts
    shortcutsError = Boolean(ctx.shortcutsError)
  } else {
    // 上一节 / 今日推荐强时效：每次侧栏更新都拉一遍（接口很轻）
    const loaded = await loadSidebarShortcuts(true)
    if (seq !== sidebarUpdateSeq) return
    shortcuts = loaded.shortcuts
    shortcutsError = loaded.error
  }

  if (seq !== sidebarUpdateSeq) return

  const slot = shellRoot.querySelector('#sidebar-slot')
  if (!slot) return
  slot.innerHTML = renderSidebar({
    ...lastSidebarCtx,
    shortcuts,
    shortcutsError,
    userName: getActiveProfile()?.displayName,
  })

  applySidebarLLMBadge()
  syncHeaderNav(lastSidebarCtx.active)
}

function applySidebarLLMBadge(): void {
  if (!shellRoot) return
  if (isAppBusy()) {
    setSidebarLLMStatus(
      shellRoot,
      '<div class="sidebar-llm-badge sidebar-llm-badge--loading"><span class="sidebar-llm-dot" aria-hidden="true"></span><span class="sidebar-llm-text">课程准备中…</span></div>'
    )
    return
  }
  if (lastLLMBadgeHtml) {
    setSidebarLLMStatus(shellRoot, lastLLMBadgeHtml)
    return
  }
  void refreshLLMStatus()
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

export async function refreshLLMStatus(force = false): Promise<void> {
  if (!shellRoot) return
  const seq = ++llmRefreshSeq

  if (isAppBusy()) {
    setSidebarLLMStatus(
      shellRoot,
      '<div class="sidebar-llm-badge sidebar-llm-badge--loading"><span class="sidebar-llm-dot" aria-hidden="true"></span><span class="sidebar-llm-text">课程准备中…</span></div>'
    )
    return
  }

  if (
    !force &&
    lastLLMBadgeHtml &&
    Date.now() - llmConfigFetchedAt < LLM_CONFIG_MIN_INTERVAL_MS
  ) {
    setSidebarLLMStatus(shellRoot, lastLLMBadgeHtml)
    return
  }

  try {
    const info = await getLLMConfig()
    if (seq !== llmRefreshSeq) return
    publishLLMConfig(info)
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
  void refreshLLMStatus(true)
}

export function navFromHash(hash: string): NavKey {
  if (hash.match(/^\/coach\//)) return 'coach'
  if (hash.match(/^\/assistant/)) return 'assistant'
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
