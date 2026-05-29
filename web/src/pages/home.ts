import { buildDomain, ApiError } from '../lib/api'

export function renderHome(container: HTMLElement): void {
  container.innerHTML = `
    <div class="page">
      <h1 class="page-title">你想学什么？</h1>
      <p class="page-sub">输入一个领域，AI 教练会为你生成三层知识树。每次 15 分钟，完成一个节点。</p>
      <div id="home-error"></div>
      <input class="input" id="domain-input" type="text" placeholder="例如：Go 并发、Agent 原理" autocomplete="off" />
      <button class="btn" id="start-btn">开始学习</button>
    </div>
  `

  const input = container.querySelector<HTMLInputElement>('#domain-input')!
  const btn = container.querySelector<HTMLButtonElement>('#start-btn')!
  const errEl = container.querySelector<HTMLDivElement>('#home-error')!

  const submit = async () => {
    const name = input.value.trim()
    if (!name) {
      errEl.innerHTML = '<div class="error-banner">请输入想学的领域</div>'
      return
    }
    btn.disabled = true
    errEl.innerHTML = ''
    try {
      const tree = await buildDomain(name)
      location.hash = `#/tree/${tree.domainId}`
      window.dispatchEvent(new HashChangeEvent('hashchange'))
    } catch (e) {
      errEl.innerHTML = `<div class="error-banner">${e instanceof ApiError ? e.message : '网络错误，请稍后重试'}</div>`
    } finally {
      btn.disabled = false
    }
  }

  btn.addEventListener('click', submit)
  input.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') submit()
  })
  input.focus()
}
