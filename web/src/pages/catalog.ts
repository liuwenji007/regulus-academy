import { getPublicDomains, ApiError } from '../lib/api'
import { setBreadcrumb, updateSidebar } from '../components/layout'
import { fetchCloudInfo, isCloudDeployment } from '../lib/cloud'
import { renderPublicCard, bindPublicDomainStarts } from '../lib/public-catalog'
import { setPageBuildLoading, syncHomeBuildOverlay } from '../lib/home-build-loading'
import {
  getDomainBuildJob,
  isDomainBuildRunning,
  onDomainBuildJobChange,
} from '../lib/domain-build-job'

let catalogBuildUnsub: (() => void) | null = null

export async function renderCatalog(container: HTMLElement): Promise<void> {
  catalogBuildUnsub?.()
  catalogBuildUnsub = onDomainBuildJobChange(() => {
    const job = getDomainBuildJob()
    if (!job || !isDomainBuildRunning()) {
      void setPageBuildLoading(container, false)
      return
    }
    void setPageBuildLoading(container, true, job.message)
  })

  void updateSidebar({ active: 'home' })
  setBreadcrumb([
    { label: '开始学习', href: '#/' },
    { label: '内置课程' },
  ])

  const cloudInfo = await fetchCloudInfo()
  const cloudHint = isCloudDeployment(cloudInfo)
    ? '<p class="page-sub page-catalog-hint">内置课程建课不占每日建课额度。</p>'
    : ''

  container.innerHTML = `
    <section class="page page-catalog">
      <header class="page-header">
        <div class="page-header-main">
          <h1 class="page-title">内置课程</h1>
          <div class="page-tree-meta">
            <p class="page-sub page-tree-hint">社区维护的标准学习路径，选定后即可按你的背景裁剪并开始学习。</p>
            ${cloudHint}
          </div>
        </div>
      </header>
      <div id="catalog-toast"></div>
      <div id="catalog-error"></div>
      <div id="catalog-content" class="catalog-content">
        <div class="page-loading"><div class="spinner" aria-hidden="true"></div><p>加载课程…</p></div>
      </div>
    </section>
  `

  const contentEl = container.querySelector<HTMLDivElement>('#catalog-content')!
  const errEl = container.querySelector<HTMLDivElement>('#catalog-error')!
  const toastEl = container.querySelector<HTMLDivElement>('#catalog-toast')!

  syncHomeBuildOverlay(container)
  await loadCatalog(contentEl, container, errEl, toastEl)
}

async function loadCatalog(
  el: HTMLElement,
  pageContainer: HTMLElement,
  errEl: HTMLElement,
  toastEl: HTMLElement
): Promise<void> {
  try {
    const domains = await getPublicDomains()
    if (domains.length === 0) {
      el.innerHTML = `
        <div class="card courses-empty">
          <p>暂无内置课程</p>
          <a href="#/" class="btn btn-primary btn-sm">返回首页</a>
        </div>
      `
      return
    }
    el.innerHTML = `<div class="public-grid">${domains.map(renderPublicCard).join('')}</div>`
    bindPublicDomainStarts(el, { errEl, toastEl, pageContainer })
  } catch (e) {
    el.innerHTML = ''
    errEl.innerHTML = `<div class="alert alert-error">${escapeHtml(e instanceof ApiError ? e.message : '加载失败')}</div>`
  }
}

function escapeHtml(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}
