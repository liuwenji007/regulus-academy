import { buildDomain, getLLMInfo, ApiError, type LLMInfo } from '../lib/api'

const LAST_DOMAIN_KEY = 'regulus:lastDomainId'

export function renderHome(container: HTMLElement): void {
  container.innerHTML = `
    <div class="page">
      <h1 class="page-title">你想学什么？</h1>
      <p class="page-sub">用一句话说出你的目标，我会理解你想学什么，并为你生成专属知识树。</p>
      <div id="home-llm"></div>
      <div id="home-resume"></div>
      <div id="home-toast"></div>
      <div id="home-error"></div>
      <input class="input" id="domain-input" type="text" placeholder="例如：Rust、Go 并发、Agent 原理" autocomplete="off" />
      <button class="btn" id="start-btn">开始学习</button>
    </div>
  `

  const input = container.querySelector<HTMLInputElement>('#domain-input')!
  const btn = container.querySelector<HTMLButtonElement>('#start-btn')!
  const errEl = container.querySelector<HTMLDivElement>('#home-error')!
  const llmEl = container.querySelector<HTMLDivElement>('#home-llm')!
  const resumeEl = container.querySelector<HTMLDivElement>('#home-resume')!
  const toastEl = container.querySelector<HTMLDivElement>('#home-toast')!

  void getLLMInfo()
    .then((info) => renderLLMStatus(llmEl, info))
    .catch(() => {
      llmEl.innerHTML =
        '<div class="error-banner">无法连接后端，请先运行 <code>go run ./cmd/server</code></div>'
    })

  const savedDomainId = localStorage.getItem(LAST_DOMAIN_KEY)
  if (savedDomainId) {
    resumeEl.innerHTML = `
      <button class="btn btn-ghost" id="resume-btn">继续上次学习</button>
    `
    resumeEl.querySelector<HTMLButtonElement>('#resume-btn')!.addEventListener('click', () => {
      location.hash = `#/tree/${savedDomainId}`
      window.dispatchEvent(new HashChangeEvent('hashchange'))
    })
  }

  const submit = async () => {
    const name = input.value.trim()
    if (!name) {
      errEl.innerHTML = '<div class="error-banner">请输入想学的领域</div>'
      return
    }
    btn.disabled = true
    btn.textContent = '分析中…'
    errEl.innerHTML = ''
    toastEl.innerHTML = ''
    try {
      btn.textContent = '生成知识树…'
      const result = await buildDomain(name)
      if (result.status !== 'ready' || !result.tree) {
        errEl.innerHTML = `<div class="error-banner">${result.message ?? '无法加载学习路径'}</div>`
        return
      }
      if (result.generated) {
        toastEl.innerHTML =
          '<div class="toast-success">已根据你的目标生成学习路径，开始学习吧</div>'
      }
      localStorage.setItem(LAST_DOMAIN_KEY, result.tree.domainId)
      location.hash = `#/tree/${result.tree.domainId}`
      window.dispatchEvent(new HashChangeEvent('hashchange'))
    } catch (e) {
      errEl.innerHTML = `<div class="error-banner">${e instanceof ApiError ? e.message : '网络错误，请稍后重试'}</div>`
    } finally {
      btn.disabled = false
      btn.textContent = '开始学习'
    }
  }

  btn.addEventListener('click', submit)
  input.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') submit()
  })
  input.focus()
}

function renderLLMStatus(el: HTMLElement, info: LLMInfo): void {
  if (!info.configured) {
    el.innerHTML = `
      <div class="error-banner">
        未配置 LLM：复制 <code>.env.example</code> 为 <code>.env</code>，填入 <code>LLM_API_KEY</code> 后重启后端（意图分析与建树需要模型）。
      </div>
    `
    return
  }
  el.innerHTML = `
    <div class="toast-success">模型已连接：${escapeHtml(info.provider)} / ${escapeHtml(info.model)}</div>
  `
}

function escapeHtml(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}
