import { iconHome, iconTree, iconMessage, iconSparkles } from '../lib/icons'

const LAST_DOMAIN_KEY = 'regulus:lastDomainId'

export type NavKey = 'home' | 'tree' | 'coach'

export interface SidebarContext {
  active: NavKey
  domainId?: string
  domainName?: string
  nodeTitle?: string
}

export function renderSidebar(ctx: SidebarContext): string {
  const lastDomainId = localStorage.getItem(LAST_DOMAIN_KEY) ?? ctx.domainId ?? ''
  const treeHref = lastDomainId ? `#/tree/${lastDomainId}` : ''
  const treeDisabled = !lastDomainId

  return `
    <aside class="sidebar" id="sidebar" aria-label="主导航">
      <div class="sidebar-brand">
        <div class="sidebar-logo" aria-hidden="true">${iconSparkles()}</div>
        <div class="sidebar-brand-text">
          <span class="sidebar-brand-name">Regulus Academy</span>
          <span class="sidebar-brand-tag">AI 学习教练</span>
        </div>
      </div>

      <nav class="sidebar-nav">
        <a href="#/" class="sidebar-link ${ctx.active === 'home' ? 'is-active' : ''}" data-nav="home">
          <span class="sidebar-link-icon">${iconHome()}</span>
          <span class="sidebar-link-label">开始学习</span>
        </a>
        <a href="${treeDisabled ? '#' : treeHref}" class="sidebar-link ${ctx.active === 'tree' ? 'is-active' : ''} ${treeDisabled ? 'is-disabled' : ''}" data-nav="tree" ${treeDisabled ? 'aria-disabled="true" tabindex="-1"' : ''}>
          <span class="sidebar-link-icon">${iconTree()}</span>
          <span class="sidebar-link-label">知识树</span>
        </a>
        ${ctx.active === 'coach' && ctx.nodeTitle ? `
          <div class="sidebar-link sidebar-link-static is-active" aria-current="page">
            <span class="sidebar-link-icon">${iconMessage()}</span>
            <span class="sidebar-link-label sidebar-link-truncate">${escapeHtml(ctx.nodeTitle)}</span>
          </div>
        ` : ''}
      </nav>

      ${ctx.domainName ? `
        <div class="sidebar-course">
          <span class="sidebar-course-label">当前课程</span>
          <span class="sidebar-course-name">${escapeHtml(ctx.domainName)}</span>
        </div>
      ` : ''}

      <div class="sidebar-footer">
        <div id="sidebar-llm" class="sidebar-llm"></div>
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
