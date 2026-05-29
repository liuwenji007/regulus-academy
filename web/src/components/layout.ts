import { getLLMInfo, type LLMInfo } from '../lib/api'
import { renderSidebar, setSidebarLLMStatus, type NavKey, type SidebarContext } from './sidebar'
import { iconMenu, iconChevronRight } from '../lib/icons'

let shellRoot: HTMLElement | null = null
let contentEl: HTMLElement | null = null
let breadcrumbEl: HTMLElement | null = null
let sidebarBound = false

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

  updateSidebar({ active: 'home' })
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
    const link = target.closest<HTMLAnchorElement>('.sidebar-link:not(.is-disabled)')

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

export function updateSidebar(ctx: SidebarContext): void {
  if (!shellRoot) return
  const slot = shellRoot.querySelector('#sidebar-slot')
  if (!slot) return
  slot.innerHTML = renderSidebar(ctx)
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
