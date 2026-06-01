import { buildDomain, getDomains, getPublicDomains, ApiError, type DomainSummary, type PublicDomainEntry } from '../lib/api'
import { setAppBusy } from '../lib/app-busy'
import { stashPrefetchTree } from '../lib/course-prefetch'
import { navigateHash } from '../lib/navigate'
import { iconTree, iconChevronRight, iconRefresh, iconTrash } from '../lib/icons'
import {
  setBreadcrumb,
  updateSidebar,
  invalidateSidebarCourses,
} from '../components/layout'
import { showDomainConfirm } from '../components/domain-confirm'
import { handleDomainDelete, handleDomainRegenerate } from '../lib/domain-actions'

const LAST_DOMAIN_KEY = 'regulus:lastDomainId'
const TREE_FOCUS_PREFIX = 'regulus:treeFocus:'

function saveTreeFocus(domainId: string, focusNodeKeys?: string[], focusLabel?: string): void {
  if (!focusNodeKeys?.length) {
    sessionStorage.removeItem(TREE_FOCUS_PREFIX + domainId)
    return
  }
  sessionStorage.setItem(
    TREE_FOCUS_PREFIX + domainId,
    JSON.stringify({ keys: focusNodeKeys, label: focusLabel ?? '' })
  )
}

function navigateToTree(domainId: string, result?: { focusNodeKeys?: string[]; focusLabel?: string; message?: string }, toastEl?: HTMLElement | null): void {
  saveTreeFocus(domainId, result?.focusNodeKeys, result?.focusLabel)
  if (result?.message && toastEl) {
    toastEl.innerHTML = `<div class="alert alert-success">${escapeHtml(result.message)}</div>`
  }
  localStorage.setItem(LAST_DOMAIN_KEY, domainId)
  invalidateSidebarCourses()
  navigateHash(`/tree/${domainId}`)
}

export function renderHome(container: HTMLElement): void {
  void updateSidebar({ active: 'home' })
  setBreadcrumb([{ label: '开始学习' }])

  container.innerHTML = `
    <section class="page page-home">
      <div class="page-hero">
        <p class="page-eyebrow">碎片化微训练</p>
        <h1 class="page-title">你想学什么？</h1>
        <p class="page-sub">用一句话说出你的目标，我会帮你规划学习路径。</p>
      </div>

      <div class="card card-elevated home-form-card">
        <div id="home-toast"></div>
        <div id="home-error"></div>
        <label class="field-label" for="domain-input">学习主题</label>
        <input class="input input-lg" id="domain-input" type="text" placeholder="例如：Rust、Go 并发、Agent 原理" autocomplete="off" />
        <button class="btn btn-primary btn-lg" id="start-btn">开始学习</button>
      </div>

      <div id="home-courses"></div>

      <div id="home-public"></div>
    </section>
  `

  const input = container.querySelector<HTMLInputElement>('#domain-input')!
  const btn = container.querySelector<HTMLButtonElement>('#start-btn')!
  const errEl = container.querySelector<HTMLDivElement>('#home-error')!
  const toastEl = container.querySelector<HTMLDivElement>('#home-toast')!
  const coursesEl = container.querySelector<HTMLDivElement>('#home-courses')!
  const publicEl = container.querySelector<HTMLDivElement>('#home-public')!

  void loadHomeCourses(coursesEl).then(() => {
    void loadPublicCatalog(publicEl, coursesEl.innerHTML !== '')
  })

  let submitting = false
  let composing = false
  let lastEnterSubmitAt = 0
  const ENTER_SUBMIT_COOLDOWN_MS = 600

  const submit = async (force = false) => {
    if (submitting) return
    const name = input.value.trim()
    if (!name) {
      errEl.innerHTML = '<div class="alert alert-error">请输入想学的领域</div>'
      return
    }
    submitting = true
    btn.disabled = true
    btn.textContent = '分析中…'
    errEl.innerHTML = ''
    toastEl.innerHTML = ''
    setAppBusy(true, 'build')
    let handoffToTree = false
    try {
      btn.textContent = '生成知识树…'
      const result = await buildDomain(name, { force })
      if (result.status === 'related' && result.existingDomain) {
        const goExisting = confirm(
          `${result.message ?? ''}\n\n点击「确定」继续现有课程，「取消」仍新建完整路径。`
        )
        if (goExisting) {
          handoffToTree = true
          localStorage.setItem(LAST_DOMAIN_KEY, result.existingDomain.id)
          invalidateSidebarCourses()
          navigateHash(`/tree/${result.existingDomain.id}`)
          return
        }
        submitting = false
        await submit(true)
        return
      }
      if (result.status !== 'ready' || !result.tree) {
        errEl.innerHTML = `<div class="alert alert-error">${result.message ?? '无法加载学习路径'}</div>`
        return
      }
      if (result.message && !result.focusNodeKeys?.length) {
        toastEl.innerHTML = `<div class="alert alert-success">${escapeHtml(result.message)}</div>`
      }
      handoffToTree = true
      stashPrefetchTree(result.tree)
      navigateToTree(result.tree.domainId, result, toastEl)
    } catch (e) {
      errEl.innerHTML = `<div class="alert alert-error">${e instanceof ApiError ? e.message : '网络错误，请稍后重试'}</div>`
    } finally {
      if (!handoffToTree) setAppBusy(false)
      submitting = false
      btn.disabled = false
      btn.textContent = '开始学习'
    }
  }

  btn.addEventListener('click', () => void submit())
  input.addEventListener('compositionstart', () => {
    composing = true
  })
  input.addEventListener('compositionend', () => {
    composing = false
  })
  input.addEventListener('keydown', (e) => {
    if (e.key !== 'Enter') return
    // 中文输入法选词时的回车不应触发提交
    if (e.isComposing || composing) return
    e.preventDefault()
    const now = Date.now()
    if (now - lastEnterSubmitAt < ENTER_SUBMIT_COOLDOWN_MS) return
    lastEnterSubmitAt = now
    void submit()
  })
  input.focus()
}

async function loadPublicCatalog(el: HTMLElement, hasCourses: boolean): Promise<void> {
  try {
    const domains = await getPublicDomains()
    if (domains.length === 0) {
      el.innerHTML = ''
      return
    }
    const featured = domains.slice(0, hasCourses ? 1 : 2)
    const title = hasCourses ? '想加一门新课？' : '或者试试这些主题'
    const desc = hasCourses
      ? '社区已有标准路径，点选即可开始。'
      : '不确定学什么时，可以从社区维护的路径起步。'
    el.innerHTML = `
      <section class="home-public-section home-public-section--compact">
        <div class="section-head">
          <h2 class="section-title section-title--soft">${escapeHtml(title)}</h2>
          <p class="section-desc">${escapeHtml(desc)}</p>
        </div>
        <div class="public-grid">${featured.map(renderPublicCard).join('')}</div>
      </section>
    `
    el.querySelectorAll<HTMLButtonElement>('[data-public-start]').forEach((btn) => {
      btn.addEventListener('click', () => {
        void startPublicDomain(btn, el.closest('.page-home')?.querySelector<HTMLInputElement>('#domain-input'))
      })
    })
  } catch {
    el.innerHTML = ''
  }
}

async function startPublicDomain(
  btn: HTMLButtonElement,
  input?: HTMLInputElement | null
): Promise<void> {
  const name = btn.dataset.publicName?.trim()
  if (!name) return
  if (input) input.value = name
  const errEl = btn.closest('.page-home')?.querySelector<HTMLDivElement>('#home-error')
  const toastEl = btn.closest('.page-home')?.querySelector<HTMLDivElement>('#home-toast')
  btn.disabled = true
  const prev = btn.textContent
  btn.textContent = '加载中…'
  if (errEl) errEl.innerHTML = ''
  if (toastEl) toastEl.innerHTML = ''
  setAppBusy(true, 'build')
  let handoffToTree = false
  try {
    const result = await buildDomain(name)
    if (result.status !== 'ready' || !result.tree) {
      if (errEl) {
        errEl.innerHTML = `<div class="alert alert-error">${result.message ?? '无法加载学习路径'}</div>`
      }
      return
    }
    if (result.personalized && toastEl) {
      toastEl.innerHTML = '<div class="alert alert-success">已根据你的背景裁剪学习路径</div>'
    }
    handoffToTree = true
    stashPrefetchTree(result.tree)
    navigateToTree(result.tree.domainId, result, toastEl)
  } catch (e) {
    if (errEl) {
      errEl.innerHTML = `<div class="alert alert-error">${e instanceof ApiError ? e.message : '网络错误，请稍后重试'}</div>`
    }
  } finally {
    if (!handoffToTree) setAppBusy(false)
    btn.disabled = false
    btn.textContent = prev ?? '开始学习'
  }
}

function renderPublicCard(d: PublicDomainEntry): string {
  return `
    <article class="public-card card">
      <div class="public-card-head">
        <h3 class="public-card-title">${escapeHtml(d.name)}</h3>
        <span class="badge badge-muted">v${d.version}</span>
      </div>
      <p class="public-card-desc">${escapeHtml(d.description || '社区维护的标准学习路径')}</p>
      <p class="public-card-meta">${d.nodeCount} 个节点 · 标准三层路径</p>
      <button type="button" class="btn btn-secondary btn-sm public-card-btn" data-public-start data-public-name="${escapeHtml(d.name)}">开始学习</button>
    </article>
  `
}

async function loadHomeCourses(el: HTMLElement): Promise<void> {
  try {
    const courses = await getDomains()
    if (courses.length === 0) {
      el.innerHTML = ''
      return
    }
    el.innerHTML = `
      <section class="home-courses-section">
        <h2 class="section-title">我的课程</h2>
        <div class="course-grid">${courses.map(renderCourseCard).join('')}</div>
      </section>
    `
    el.querySelectorAll<HTMLElement>('.course-card').forEach((card) => {
      const id = card.dataset.domainId
      const course = courses.find((c) => c.id === id)
      if (!id || !course) return

      card.querySelector<HTMLAnchorElement>('.course-card-link')?.addEventListener('click', () => {
        localStorage.setItem(LAST_DOMAIN_KEY, id)
      })

      card.querySelector<HTMLButtonElement>('[data-action="regenerate"]')?.addEventListener('click', (e) => {
        e.preventDefault()
        e.stopPropagation()
        void (async () => {
          const outcome = await showDomainConfirm({
            domainId: id,
            domainName: course.name,
            action: 'regenerate',
          })
          if (!outcome.ok) return
          if (outcome.action === 'regenerate') {
            await handleDomainRegenerate(id, outcome.result.tree!.domainId, outcome.result)
            void loadHomeCourses(el)
          }
        })()
      })

      card.querySelector<HTMLButtonElement>('[data-action="delete"]')?.addEventListener('click', (e) => {
        e.preventDefault()
        e.stopPropagation()
        void (async () => {
          const outcome = await showDomainConfirm({
            domainId: id,
            domainName: course.name,
            action: 'delete',
          })
          if (!outcome.ok) return
          if (outcome.action === 'delete') {
            await handleDomainDelete(id)
            void loadHomeCourses(el)
          }
        })()
      })
    })
  } catch {
    el.innerHTML = ''
  }
}

function renderCourseCard(c: DomainSummary): string {
  const pct = c.nodeTotal > 0 ? Math.round((c.completed / c.nodeTotal) * 100) : 0
  return `
    <article class="course-card card" data-domain-id="${c.id}">
      <div class="course-card-tools">
        <button type="button" class="course-card-tool" data-action="regenerate" title="重新生成" aria-label="重新生成">${iconRefresh()}</button>
        <button type="button" class="course-card-tool course-card-tool--danger" data-action="delete" title="移除课程" aria-label="移除课程">${iconTrash()}</button>
      </div>
      <a href="#/tree/${c.id}" class="course-card-link">
        <div class="course-card-head">
          <span class="course-card-icon" aria-hidden="true">${iconTree()}</span>
          <h3 class="course-card-title">${escapeHtml(c.name)}</h3>
        </div>
        <div class="course-card-progress">
          <div class="course-card-progress-head">
            <p class="course-card-meta">${c.completed} / ${c.nodeTotal} 节点已完成</p>
            <span class="course-card-pct">${pct}%</span>
          </div>
          <div class="progress-bar" role="progressbar" aria-valuenow="${pct}" aria-valuemin="0" aria-valuemax="100">
            <div class="progress-fill" style="width:${pct}%"></div>
          </div>
        </div>
        <span class="course-card-enter">进入课程 ${iconChevronRight()}</span>
      </a>
    </article>
  `
}

function escapeHtml(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}
