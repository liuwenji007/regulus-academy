import { buildDomain, getPublicDomains, ApiError, type PublicDomainEntry } from '../lib/api'
import { clearAppBusyIfAfter, setAppBusy } from '../lib/app-busy'
import { refreshLLMStatusAfterBusy } from '../components/layout'
import { setHomeBuildLoading } from '../lib/home-build-loading'
import { stashPrefetchTree } from '../lib/course-prefetch'
import { navigateHash } from '../lib/navigate'
import { setBreadcrumb, updateSidebar, invalidateSidebarCourses } from '../components/layout'

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
      <p class="home-courses-link"><a href="#/courses">查看我的课程</a> · <a href="#/graph">知识图谱</a></p>
    </div>

    <div id="home-public"></div>
    </section>
  `

  const input = container.querySelector<HTMLInputElement>('#domain-input')!
  const btn = container.querySelector<HTMLButtonElement>('#start-btn')!
  const errEl = container.querySelector<HTMLDivElement>('#home-error')!
  const toastEl = container.querySelector<HTMLDivElement>('#home-toast')!
  const publicEl = container.querySelector<HTMLDivElement>('#home-public')!

  void loadPublicCatalog(publicEl, container)

  let submitting = false
  let composing = false
  let lastEnterSubmitAt = 0
  const ENTER_SUBMIT_COOLDOWN_MS = 600

  const submit = async (force = false): Promise<void> => {
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
    setHomeBuildLoading(container, true, '正在分析学习目标…')
    let handoffToTree = false
    try {
      btn.textContent = '生成知识树…'
      setHomeBuildLoading(container, true, '正在生成知识树…')
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
        return // 内层 submit 负责 busy / overlay 收尾
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
      if (!handoffToTree) {
        await setHomeBuildLoading(container, false)
        clearAppBusyIfAfter('build', refreshLLMStatusAfterBusy)
      }
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

async function loadPublicCatalog(el: HTMLElement, pageContainer: HTMLElement): Promise<void> {
  try {
    const domains = await getPublicDomains()
    if (domains.length === 0) {
      el.innerHTML = ''
      return
    }
    el.innerHTML = `
      <section class="home-public-section home-public-section--compact">
        <div class="section-head">
          <h2 class="section-title section-title--soft">或者试试这些主题</h2>
          <p class="section-desc">不确定学什么时，可以从社区维护的路径起步。</p>
        </div>
        <div class="public-grid">${domains.slice(0, 2).map(renderPublicCard).join('')}</div>
      </section>
    `
    el.querySelectorAll<HTMLButtonElement>('[data-public-start]').forEach((btn) => {
      btn.addEventListener('click', () => {
        void startPublicDomain(
          btn,
          pageContainer.querySelector<HTMLInputElement>('#domain-input'),
          pageContainer
        )
      })
    })
  } catch {
    el.innerHTML = ''
  }
}

async function startPublicDomain(
  btn: HTMLButtonElement,
  input?: HTMLInputElement | null,
  container?: HTMLElement
): Promise<void> {
  const name = btn.dataset.publicName?.trim()
  if (!name) return
  if (input) input.value = name
  const errEl = btn.closest('.page-home')?.querySelector<HTMLDivElement>('#home-error')
  const toastEl = btn.closest('.page-home')?.querySelector<HTMLDivElement>('#home-toast')
  const page = container ?? btn.closest<HTMLElement>('.page-home')?.parentElement ?? undefined
  btn.disabled = true
  const prev = btn.textContent
  btn.textContent = '加载中…'
  if (errEl) errEl.innerHTML = ''
  if (toastEl) toastEl.innerHTML = ''
  setAppBusy(true, 'build')
  if (page) setHomeBuildLoading(page, true, '正在加载课程…')
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
    if (!handoffToTree) {
      if (page) await setHomeBuildLoading(page, false)
      clearAppBusyIfAfter('build', refreshLLMStatusAfterBusy)
    }
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

function escapeHtml(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}
