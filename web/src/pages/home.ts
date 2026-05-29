import { buildDomain, getDomains, ApiError, type DomainSummary } from '../lib/api'
import {
  setBreadcrumb,
  updateSidebar,
  invalidateSidebarCourses,
} from '../components/layout'

const LAST_DOMAIN_KEY = 'regulus:lastDomainId'

export function renderHome(container: HTMLElement): void {
  void updateSidebar({ active: 'home' })
  setBreadcrumb([{ label: '开始学习' }])

  container.innerHTML = `
    <section class="page page-home">
      <div class="page-hero">
        <p class="page-eyebrow">碎片化微训练</p>
        <h1 class="page-title">你想学什么？</h1>
        <p class="page-sub">用一句话说出你的目标。我会理解你的意图，并为你生成专属的三层知识树。</p>
      </div>

      <div class="card card-elevated home-form-card">
        <div id="home-toast"></div>
        <div id="home-error"></div>
        <label class="field-label" for="domain-input">学习主题</label>
        <input class="input input-lg" id="domain-input" type="text" placeholder="例如：Rust、Go 并发、Agent 原理" autocomplete="off" />
        <button class="btn btn-primary btn-lg" id="start-btn">开始学习</button>
      </div>

      <div id="home-courses"></div>

      <div class="home-features">
        <div class="feature-card">
          <span class="feature-num">01</span>
          <h3 class="feature-title">意图理解</h3>
          <p class="feature-desc">根据你的第一句话，识别真正想学的主题。</p>
        </div>
        <div class="feature-card">
          <span class="feature-num">02</span>
          <h3 class="feature-title">知识树生成</h3>
          <p class="feature-desc">自动规划入门、熟悉、精通三层路径。</p>
        </div>
        <div class="feature-card">
          <span class="feature-num">03</span>
          <h3 class="feature-title">节点微训练</h3>
          <p class="feature-desc">讲解、练习、反馈，每个节点约 15 分钟闭环。</p>
        </div>
      </div>
    </section>
  `

  const input = container.querySelector<HTMLInputElement>('#domain-input')!
  const btn = container.querySelector<HTMLButtonElement>('#start-btn')!
  const errEl = container.querySelector<HTMLDivElement>('#home-error')!
  const toastEl = container.querySelector<HTMLDivElement>('#home-toast')!
  const coursesEl = container.querySelector<HTMLDivElement>('#home-courses')!

  void loadHomeCourses(coursesEl)

  let submitting = false
  let composing = false
  let lastEnterSubmitAt = 0
  const ENTER_SUBMIT_COOLDOWN_MS = 600

  const submit = async () => {
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
    try {
      btn.textContent = '生成知识树…'
      const result = await buildDomain(name)
      if (result.status !== 'ready' || !result.tree) {
        errEl.innerHTML = `<div class="alert alert-error">${result.message ?? '无法加载学习路径'}</div>`
        return
      }
      if (result.generated) {
        toastEl.innerHTML =
          '<div class="alert alert-success">已根据你的目标生成学习路径</div>'
      }
      localStorage.setItem(LAST_DOMAIN_KEY, result.tree.domainId)
      invalidateSidebarCourses()
      location.hash = `#/tree/${result.tree.domainId}`
      window.dispatchEvent(new HashChangeEvent('hashchange'))
    } catch (e) {
      errEl.innerHTML = `<div class="alert alert-error">${e instanceof ApiError ? e.message : '网络错误，请稍后重试'}</div>`
    } finally {
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
    el.querySelectorAll<HTMLAnchorElement>('.course-card').forEach((card) => {
      card.addEventListener('click', () => {
        const id = card.dataset.domainId
        if (id) localStorage.setItem(LAST_DOMAIN_KEY, id)
      })
    })
  } catch {
    el.innerHTML = ''
  }
}

function renderCourseCard(c: DomainSummary): string {
  const pct = c.nodeTotal > 0 ? Math.round((c.completed / c.nodeTotal) * 100) : 0
  return `
    <a href="#/tree/${c.id}" class="course-card card" data-domain-id="${c.id}">
      <h3 class="course-card-title">${escapeHtml(c.name)}</h3>
      <p class="course-card-meta">已完成 ${c.completed} / ${c.nodeTotal} 节点</p>
      <div class="progress-bar" aria-hidden="true"><div class="progress-fill" style="width:${pct}%"></div></div>
    </a>
  `
}

function escapeHtml(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}
