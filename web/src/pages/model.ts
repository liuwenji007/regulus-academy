import {
  getLLMConfig,
  saveLLMProfiles,
  pingLLMProfile,
  ApiError,
  type LLMConfigResponse,
  type LLMProfileInput,
  type LLMProfileView,
  type LLMPreset,
  type LLMSettingsPayload,
} from '../lib/api'
import { setBreadcrumb, updateSidebar, refreshLLMStatus, publishLLMConfig } from '../components/layout'
import { showTypeConfirm } from '../components/type-confirm'

function newProfileId(): string {
  return `m-${Date.now()}`
}

const PROVIDER_HINT: Record<string, string> = {
  deepseek: 'https://platform.deepseek.com/api_keys',
  openai: 'https://platform.openai.com/api-keys',
  openrouter: 'https://openrouter.ai/keys',
  ollama: '本地 Ollama，通常无需 Key',
  custom: 'OpenAI 兼容接口，须填写 Base URL',
}

type CardTestState = 'idle' | 'pending' | 'ok' | 'error'

const cardTestCache = new Map<string, { state: CardTestState; message: string }>()

export async function renderModelSettings(container: HTMLElement): Promise<void> {
  void updateSidebar({ active: 'settings' })
  setBreadcrumb([
    { label: '开始学习', href: '#/' },
    { label: '设置', href: '#/settings' },
    { label: 'AI 模型' },
  ])

  container.innerHTML = loadingHtml()

  try {
    const cfg = await getLLMConfig()
    publishLLMConfig(cfg)
    mountPage(container, cfg)
  } catch (e) {
    container.innerHTML = `
      <section class="page page-model">
        <div class="alert alert-error">${escapeHtml(e instanceof ApiError ? e.message : '加载失败')}</div>
      </section>
    `
  }
}

function loadingHtml(): string {
  return `
    <section class="page page-model">
      <div class="page-loading"><div class="spinner" aria-hidden="true"></div><p>加载模型配置…</p></div>
    </section>
  `
}

function mountPage(container: HTMLElement, cfg: LLMConfigResponse): void {
  container.innerHTML = renderPage(cfg)
  bindPage(container, cfg)
}

function renderPage(cfg: LLMConfigResponse): string {
  const presets = cfg.presets ?? []
  const profiles = cfg.profiles ?? []
  const activeId = cfg.activeProfileId ?? profiles[0]?.id ?? ''
  const statusClass = cfg.configured ? 'ok' : 'warn'
  const active = profiles.find((p) => p.id === activeId)
  const statusText = cfg.configured
    ? active?.name || cfg.provider || '已配置'
    : '未配置 API Key，AI 教练不可用'

  const globalKeyHint = cfg.settings?.apiKeySet
    ? '全局 Key 已配置（各模型可单独覆盖，留空则沿用全局）'
    : '请至少在下方某条模型或全局配置中填写 API Key'

  return `
    <section class="page page-model" data-active-profile-id="${escapeAttr(activeId)}">
      <header class="model-hero">
        <div class="model-hero-text">
          <h1 class="page-title">AI 模型</h1>
          <p class="page-sub">添加、命名并管理真实在用的模型；左下角切换「当前使用」的模型，每张卡片单独保存。</p>
        </div>
        <div class="model-hero-aside">
          <div class="model-hero-status model-hero-status--${statusClass}">
            <span class="model-status-dot" aria-hidden="true"></span>
            <span>${escapeHtml(statusText)}</span>
          </div>
        </div>
      </header>

      <div id="model-profiles-form" class="model-form">
        <div id="model-form-error"></div>
        <div id="model-form-toast"></div>

        <div class="model-form-toolbar-anchor" aria-hidden="true"></div>
        <div class="model-form-toolbar" role="toolbar" aria-label="模型配置操作">
          <p class="model-form-note">每张卡片点 <strong>保存</strong> 后写入 <code class="inline-code">data/llm-profiles.json</code> 与 <code class="inline-code">.env</code>（仅当该条为当前使用时更新运行时），<strong>立即生效</strong>。</p>
          <div class="model-form-toolbar-actions">
            <button type="button" class="btn btn-ghost" id="model-add-btn">添加模型</button>
          </div>
        </div>

        <p class="model-global-hint">${escapeHtml(globalKeyHint)}</p>

        <div id="model-profiles-list" class="model-profiles-list">
          ${profiles.map((p) => renderProfileCard(p, activeId, presets)).join('')}
        </div>
      </div>
    </section>
  `
}

function renderTestStatus(profileId: string): string {
  const cached = cardTestCache.get(profileId)
  const state = cached?.state ?? 'idle'
  const message = cached?.message ?? ''
  const label =
    state === 'pending'
      ? '测试中…'
      : state === 'ok'
        ? message || '连接正常'
        : state === 'error'
          ? message || '连接失败'
          : '未测试'
  const title = state === 'ok' || state === 'error' ? message : ''
  return `<span class="model-profile-test-status model-profile-test-status--${state}" data-test-status="${state}" title="${escapeAttr(title)}">${escapeHtml(label)}</span>`
}

function renderInUseBadge(isActive: boolean): string {
  if (!isActive) return ''
  return `<span class="model-profile-in-use" title="左下角菜单可切换其他模型为当前使用">使用中</span>`
}

function renderProfileCard(p: LLMProfileView, activeId: string, presets: LLMPreset[]): string {
  const isActive = p.id === activeId
  const provider = p.provider || 'deepseek'
  const showBase = provider === 'custom'
  const showKey = provider !== 'ollama'
  const presetOpts = presets
    .map(
      (x) =>
        `<option value="${escapeAttr(x.id)}" ${x.id === provider ? 'selected' : ''}>${escapeHtml(x.name)}</option>`
    )
    .join('')

  return `
    <article class="card model-profile-card ${isActive ? 'is-active' : ''}" data-profile-id="${escapeAttr(p.id)}">
      <header class="model-profile-card-head">
        <div class="model-profile-card-head-left">
          ${renderInUseBadge(isActive)}
        </div>
        <div class="model-profile-card-actions" role="group" aria-label="模型操作">
          ${renderTestStatus(p.id)}
          <button type="button" class="btn btn-sm model-profile-btn model-profile-btn--test model-profile-ping" title="探测连接，不必先保存">测试</button>
          <button type="button" class="btn btn-sm model-profile-btn model-profile-btn--save model-profile-save" title="保存本条并写入配置">保存</button>
          <button type="button" class="btn btn-sm model-profile-btn model-profile-btn--delete model-profile-remove" data-remove-id="${escapeAttr(p.id)}" title="从列表移除并写入配置">删除</button>
        </div>
      </header>

      <div class="model-field">
        <label class="field-label">显示名称</label>
        <input class="input" name="name" type="text" value="${escapeAttr(p.name)}" placeholder="如：DeepSeek 主力" required />
      </div>

      <div class="model-field">
        <label class="field-label">接口类型</label>
        <select class="input" name="provider" data-provider-select>${presetOpts}</select>
        <p class="field-hint model-provider-hint">${escapeHtml(PROVIDER_HINT[provider] ?? '')}</p>
      </div>

      <div class="model-field model-profile-key" ${showKey ? '' : 'hidden'}>
        <label class="field-label">API Key（可选）</label>
        <input class="input" name="apiKey" type="password" autocomplete="new-password"
          placeholder="${p.apiKeySet ? '已配置 · 留空沿用全局或不改' : '留空则使用全局 Key'}" />
      </div>

      <div class="model-field model-profile-base" ${showBase ? '' : 'hidden'}>
        <label class="field-label">Base URL</label>
        <input class="input" name="baseUrl" type="url" value="${escapeAttr(p.baseUrl ?? '')}" placeholder="https://api.example.com" />
      </div>

      <div class="model-field">
        <label class="field-label">模型 ID</label>
        <input class="input" name="model" type="text" value="${escapeAttr(p.model)}" placeholder="${escapeAttr(defaultModelFor(provider, presets))}" required />
      </div>
      <p class="model-profile-card-toast" role="status" aria-live="polite" hidden></p>
    </article>
  `
}

function defaultModelFor(provider: string, presets: LLMPreset[]): string {
  return presets.find((x) => x.id === provider)?.defaultModel ?? 'deepseek-chat'
}

function getActiveProfileId(container: HTMLElement): string {
  const page = container.querySelector<HTMLElement>('.page-model')
  return page?.dataset.activeProfileId ?? ''
}

function setActiveProfileId(container: HTMLElement, activeId: string): void {
  const page = container.querySelector<HTMLElement>('.page-model')
  if (page) page.dataset.activeProfileId = activeId
}

function refreshActiveBadges(container: HTMLElement, activeId: string): void {
  const list = container.querySelector('#model-profiles-list')
  if (!list) return
  list.querySelectorAll<HTMLElement>('.model-profile-card').forEach((card) => {
    const id = card.dataset.profileId ?? ''
    const isActive = id === activeId
    card.classList.toggle('is-active', isActive)
    const left = card.querySelector('.model-profile-card-head-left')
    if (left) {
      left.innerHTML = renderInUseBadge(isActive)
    }
  })
}

function bindToolbarStick(container: HTMLElement): void {
  const anchor = container.querySelector('.model-form-toolbar-anchor')
  const toolbar = container.querySelector('.model-form-toolbar')
  const scrollRoot = container.closest('.main-content')
  if (!anchor || !toolbar || !scrollRoot) return

  const observer = new IntersectionObserver(
    ([entry]) => {
      toolbar.classList.toggle('is-stuck', !entry.isIntersecting)
    },
    { root: scrollRoot, threshold: 0 }
  )
  observer.observe(anchor)
}

function setCardTestUI(card: HTMLElement, state: CardTestState, message = ''): void {
  const id = card.dataset.profileId
  if (id) {
    if (state === 'idle') {
      cardTestCache.delete(id)
    } else {
      cardTestCache.set(id, { state, message })
    }
  }
  const el = card.querySelector<HTMLElement>('.model-profile-test-status')
  if (!el) return
  el.dataset.testStatus = state
  const label =
    state === 'pending'
      ? '测试中…'
      : state === 'ok'
        ? message || '连接正常'
        : state === 'error'
          ? message || '连接失败'
          : '未测试'
  el.textContent = label
  el.className = `model-profile-test-status model-profile-test-status--${state}`
  el.title = state === 'ok' || state === 'error' ? message : ''
}

function markCardTestStale(card: HTMLElement): void {
  const status = card.querySelector<HTMLElement>('.model-profile-test-status')
  if (status?.dataset.testStatus === 'idle') return
  setCardTestUI(card, 'idle')
}

function showCardToast(card: HTMLElement, kind: 'success' | 'error', message: string): void {
  const el = card.querySelector<HTMLElement>('.model-profile-card-toast')
  if (!el) return
  el.hidden = false
  el.className = `model-profile-card-toast model-profile-card-toast--${kind}`
  el.textContent = message
}

function clearCardToast(card: HTMLElement): void {
  const el = card.querySelector<HTMLElement>('.model-profile-card-toast')
  if (!el) return
  el.hidden = true
  el.textContent = ''
}

function collectCardPingPayload(card: HTMLElement): LLMSettingsPayload {
  const provider = String(card.querySelector<HTMLSelectElement>('[name="provider"]')?.value ?? 'deepseek')
  const model = String(card.querySelector<HTMLInputElement>('[name="model"]')?.value ?? '').trim()
  const baseUrl = String(card.querySelector<HTMLInputElement>('[name="baseUrl"]')?.value ?? '').trim()
  const apiKey = String(card.querySelector<HTMLInputElement>('[name="apiKey"]')?.value ?? '').trim()
  const payload: LLMSettingsPayload = { provider, model }
  if (baseUrl) payload.baseUrl = baseUrl
  if (apiKey) payload.apiKey = apiKey
  return payload
}

async function runCardPing(card: HTMLElement): Promise<void> {
  const btn = card.querySelector<HTMLButtonElement>('.model-profile-ping')
  clearCardToast(card)
  setCardTestUI(card, 'pending')
  if (btn) {
    btn.disabled = true
    btn.textContent = '测试中…'
  }
  try {
    const res = await pingLLMProfile(collectCardPingPayload(card))
    setCardTestUI(card, 'ok', res.message)
  } catch (e) {
    const msg = e instanceof ApiError ? e.message : '连接失败'
    setCardTestUI(card, 'error', msg)
  } finally {
    if (btn) {
      btn.disabled = false
      btn.textContent = '测试'
    }
  }
}

async function confirmRemoveProfile(displayName: string): Promise<boolean> {
  const phrase = displayName.trim() || '新模型'
  return showTypeConfirm({
    title: '删除模型',
    description: '将从列表中移除该条模型配置，并立即写入配置文件。',
    subjectLabel: '模型',
    subjectName: phrase,
    confirmPhrase: phrase,
    confirmLabel: '确认删除',
    inputPlaceholder: '输入显示名称',
  })
}

function collectProfiles(container: HTMLElement, activeId: string): { activeId: string; profiles: LLMProfileInput[] } {
  const list = container.querySelector('#model-profiles-list')!
  const profiles: LLMProfileInput[] = []

  list.querySelectorAll<HTMLElement>('.model-profile-card').forEach((card) => {
    const id = card.dataset.profileId ?? newProfileId()
    const name = String(card.querySelector<HTMLInputElement>('[name="name"]')?.value ?? '').trim()
    const provider = String(card.querySelector<HTMLSelectElement>('[name="provider"]')?.value ?? 'deepseek')
    const model = String(card.querySelector<HTMLInputElement>('[name="model"]')?.value ?? '').trim()
    const baseUrl = String(card.querySelector<HTMLInputElement>('[name="baseUrl"]')?.value ?? '').trim()
    const apiKey = String(card.querySelector<HTMLInputElement>('[name="apiKey"]')?.value ?? '').trim()
    const entry: LLMProfileInput = { id, name, provider, model }
    if (baseUrl) entry.baseUrl = baseUrl
    if (apiKey) entry.apiKey = apiKey
    profiles.push(entry)
  })

  const resolvedActive =
    activeId && profiles.some((p) => p.id === activeId) ? activeId : profiles[0]?.id ?? ''

  return { activeId: resolvedActive, profiles }
}

async function persistProfiles(
  container: HTMLElement,
  opts?: { focusCard?: HTMLElement; successMessage?: string }
): Promise<LLMConfigResponse | null> {
  const errEl = container.querySelector<HTMLDivElement>('#model-form-error')!
  const toastEl = container.querySelector<HTMLDivElement>('#model-form-toast')!
  errEl.innerHTML = ''
  toastEl.innerHTML = ''

  const activeId = getActiveProfileId(container)
  const payload = collectProfiles(container, activeId)

  if (payload.profiles.length === 0) {
    errEl.innerHTML = '<div class="alert alert-error">至少保留一条模型配置</div>'
    return null
  }

  for (const p of payload.profiles) {
    if (!p.name) {
      errEl.innerHTML = '<div class="alert alert-error">请为每条模型填写显示名称</div>'
      return null
    }
  }

  try {
    const saved = await saveLLMProfiles(payload)
    publishLLMConfig(saved)
    const newActiveId = saved.activeProfileId ?? payload.activeId
    setActiveProfileId(container, newActiveId)
    refreshActiveBadges(container, newActiveId)
    void refreshLLMStatus(true)
    const msg = opts?.successMessage ?? '已保存'
    if (opts?.focusCard) {
      showCardToast(opts.focusCard, 'success', msg)
    } else {
      toastEl.innerHTML = `<div class="alert alert-success">${escapeHtml(msg)}</div>`
    }
    return saved
  } catch (e) {
    const message = e instanceof ApiError ? e.message : '保存失败'
    if (opts?.focusCard) {
      showCardToast(opts.focusCard, 'error', message)
    } else {
      errEl.innerHTML = `<div class="alert alert-error">${escapeHtml(message)}</div>`
    }
    return null
  }
}

async function saveCard(container: HTMLElement, card: HTMLElement): Promise<void> {
  const btn = card.querySelector<HTMLButtonElement>('.model-profile-save')
  clearCardToast(card)
  if (btn) {
    btn.disabled = true
    btn.textContent = '保存中…'
  }
  const name = String(card.querySelector<HTMLInputElement>('[name="name"]')?.value ?? '').trim()
  if (!name) {
    showCardToast(card, 'error', '请填写显示名称')
    if (btn) {
      btn.disabled = false
      btn.textContent = '保存'
    }
    return
  }

  const activeId = getActiveProfileId(container)
  const isActive = card.dataset.profileId === activeId
  const saved = await persistProfiles(container, {
    focusCard: card,
    successMessage: isActive ? '已保存，当前模型已生效' : '已保存',
  })

  if (btn) {
    btn.disabled = false
    btn.textContent = '保存'
  }

  if (saved) {
    const id = card.dataset.profileId
    const updated = saved.profiles?.find((p) => p.id === id)
    if (updated) {
      card.dataset.profileId = updated.id
    }
  }
}

function bindCard(card: HTMLElement, presets: LLMPreset[]): void {
  const syncCard = (): void => {
    const provider = card.querySelector<HTMLSelectElement>('[name="provider"]')?.value ?? 'deepseek'
    const hint = card.querySelector('.model-provider-hint')
    if (hint) hint.textContent = PROVIDER_HINT[provider] ?? ''
    card.querySelector('.model-profile-base')?.toggleAttribute('hidden', provider !== 'custom')
    card.querySelector('.model-profile-key')?.toggleAttribute('hidden', provider === 'ollama')
    const modelInput = card.querySelector<HTMLInputElement>('[name="model"]')
    if (modelInput && !modelInput.value) {
      modelInput.placeholder = defaultModelFor(provider, presets)
    }
  }

  card.querySelector('[name="provider"]')?.addEventListener('change', () => {
    const provider = card.querySelector<HTMLSelectElement>('[name="provider"]')?.value ?? 'deepseek'
    const preset = presets.find((x) => x.id === provider)
    if (provider === 'custom') {
      const base = card.querySelector<HTMLInputElement>('[name="baseUrl"]')
      const model = card.querySelector<HTMLInputElement>('[name="model"]')
      if (base) base.value = ''
      if (model) model.value = ''
    } else if (preset) {
      const base = card.querySelector<HTMLInputElement>('[name="baseUrl"]')
      const model = card.querySelector<HTMLInputElement>('[name="model"]')
      if (base) base.value = preset.baseUrl ?? ''
      if (model) model.value = preset.defaultModel ?? ''
    }
    markCardTestStale(card)
    syncCard()
  })
  card.querySelectorAll('input, select').forEach((el) => {
    el.addEventListener('input', () => markCardTestStale(card))
    el.addEventListener('change', () => markCardTestStale(card))
  })
  syncCard()
}

function bindPage(container: HTMLElement, cfg: LLMConfigResponse): void {
  const list = container.querySelector<HTMLDivElement>('#model-profiles-list')!
  const presets = cfg.presets ?? []

  bindToolbarStick(container)

  list.querySelectorAll<HTMLElement>('.model-profile-card').forEach((card) => {
    bindCard(card, presets)
  })

  container.querySelector('#model-add-btn')?.addEventListener('click', () => {
    const id = newProfileId()
    const draft: LLMProfileView = {
      id,
      name: '新模型',
      provider: 'deepseek',
      model: 'deepseek-chat',
    }
    const activeId = getActiveProfileId(container)
    list.insertAdjacentHTML('beforeend', renderProfileCard(draft, activeId, presets))
    const card = list.querySelector<HTMLElement>(`[data-profile-id="${id}"]`)!
    bindCard(card, presets)
    card.querySelector<HTMLInputElement>('[name="name"]')?.focus()
  })

  list.addEventListener('click', (e) => {
    const pingBtn = (e.target as HTMLElement).closest<HTMLButtonElement>('.model-profile-ping')
    if (pingBtn) {
      e.preventDefault()
      const card = pingBtn.closest<HTMLElement>('.model-profile-card')
      if (card) void runCardPing(card)
      return
    }

    const saveBtn = (e.target as HTMLElement).closest<HTMLButtonElement>('.model-profile-save')
    if (saveBtn) {
      e.preventDefault()
      const card = saveBtn.closest<HTMLElement>('.model-profile-card')
      if (card) void saveCard(container, card)
      return
    }

    const btn = (e.target as HTMLElement).closest<HTMLButtonElement>('.model-profile-remove')
    if (!btn?.dataset.removeId) return
    e.preventDefault()
    void (async () => {
      const cards = list.querySelectorAll('.model-profile-card')
      if (cards.length <= 1) {
        alert('至少保留一条模型配置')
        return
      }
      const card = list.querySelector<HTMLElement>(`[data-profile-id="${btn.dataset.removeId}"]`)
      const displayName = card?.querySelector<HTMLInputElement>('[name="name"]')?.value ?? ''
      if (!(await confirmRemoveProfile(displayName))) return

      const removedId = btn.dataset.removeId!
      cardTestCache.delete(removedId)

      let activeId = getActiveProfileId(container)
      if (removedId === activeId) {
        const next = list.querySelector<HTMLElement>(`.model-profile-card:not([data-profile-id="${removedId}"])`)
        activeId = next?.dataset.profileId ?? ''
        setActiveProfileId(container, activeId)
      }

      list.querySelector(`[data-profile-id="${removedId}"]`)?.remove()
      await persistProfiles(container, { successMessage: '已删除并保存' })
      refreshActiveBadges(container, getActiveProfileId(container))
    })()
  })
}

function escapeHtml(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}

function escapeAttr(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;')
}
