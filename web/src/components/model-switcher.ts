import { activateLLMProfile, ApiError, type LLMConfigResponse, type LLMProfileView } from '../lib/api'
import { iconChevronRight } from '../lib/icons'

let bound = false
let onLLMChanged: (() => void) | null = null

export function setOnLLMChanged(fn: () => void): void {
  onLLMChanged = fn
}

export function renderLLMSwitcher(info: LLMConfigResponse | LLMInfoLike): string {
  if (!info.configured) {
    return `<div class="sidebar-llm-switcher">
      <a href="#/settings/model" class="sidebar-llm-badge sidebar-llm-badge--warn">LLM 未配置 · 去设置</a>
    </div>`
  }

  const profiles = info.profiles ?? []
  const activeId = info.activeProfileId ?? profiles[0]?.id ?? ''
  const active = profiles.find((p) => p.id === activeId)
  const badgeLabel = active?.name || info.provider || '未选择模型'
  const badgeTitle = active ? `${active.name}（${active.model}）` : badgeLabel

  if (profiles.length === 0) {
    return `<div class="sidebar-llm-switcher">
      <a href="#/settings/model" class="sidebar-llm-badge sidebar-llm-badge--ok">${escapeHtml(badgeLabel)}</a>
    </div>`
  }

  const menuItems = profiles
    .map((p) => {
      const isActive = p.id === activeId ? ' is-active' : ''
      return `<button type="button" class="sidebar-llm-menu-item${isActive}" data-profile-id="${escapeAttr(p.id)}" title="${escapeAttr(`${p.name}（${p.model}）`)}">
        <span class="sidebar-llm-menu-name">${escapeHtml(p.name)}</span>
      </button>`
    })
    .join('')

  return `<div class="sidebar-llm-switcher">
    <button type="button" class="sidebar-llm-badge sidebar-llm-badge--ok sidebar-llm-trigger" id="sidebar-llm-btn" aria-expanded="false" aria-haspopup="listbox" title="${escapeAttr(badgeTitle)}">
      <span class="sidebar-llm-dot" aria-hidden="true"></span>
      <span class="sidebar-llm-text">${escapeHtml(badgeLabel)}</span>
      <span class="sidebar-llm-chevron" aria-hidden="true">${iconChevronRight()}</span>
    </button>
    <div class="sidebar-llm-menu" id="sidebar-llm-menu" role="listbox" hidden>
      <p class="sidebar-llm-menu-title">切换模型</p>
      ${menuItems}
      <a href="#/settings/model" class="sidebar-llm-menu-link">管理模型…</a>
    </div>
  </div>`
}

type LLMInfoLike = {
  provider: string
  model: string
  configured: boolean
  profiles?: LLMProfileView[]
  activeProfileId?: string
}

export function bindModelSwitcher(root: HTMLElement): void {
  if (bound) return
  bound = true

  root.addEventListener('click', (e) => {
    const target = e.target as HTMLElement
    const trigger = target.closest('#sidebar-llm-btn')
    const menuItem = target.closest<HTMLButtonElement>('.sidebar-llm-menu-item[data-profile-id]')
    const menu = root.querySelector<HTMLElement>('#sidebar-llm-menu')
    const btn = root.querySelector<HTMLButtonElement>('#sidebar-llm-btn')

    if (menuItem && menuItem.dataset.profileId) {
      e.preventDefault()
      e.stopPropagation()
      closeMenu(menu, btn)
      void switchProfile(menuItem.dataset.profileId)
      return
    }

    if (trigger) {
      e.preventDefault()
      const open = menu?.toggleAttribute('hidden') === false
      btn?.setAttribute('aria-expanded', open ? 'true' : 'false')
      return
    }

    if (!target.closest('.sidebar-llm-switcher')) {
      closeMenu(menu, btn)
    }
  })

  document.addEventListener('keydown', (e) => {
    if (e.key !== 'Escape') return
    const menu = root.querySelector<HTMLElement>('#sidebar-llm-menu')
    const btn = root.querySelector<HTMLButtonElement>('#sidebar-llm-btn')
    closeMenu(menu, btn)
  })
}

function closeMenu(menu: HTMLElement | null, btn: HTMLButtonElement | null): void {
  menu?.setAttribute('hidden', '')
  btn?.setAttribute('aria-expanded', 'false')
}

async function switchProfile(profileId: string): Promise<void> {
  try {
    await activateLLMProfile(profileId)
    onLLMChanged?.()
  } catch (err) {
    console.error(err instanceof ApiError ? err.message : '切换模型失败')
  }
}

export function setLastLLMConfig(_cfg: LLMConfigResponse): void {
  /* cache hook for settings page */
}

function escapeHtml(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}

function escapeAttr(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;')
}
