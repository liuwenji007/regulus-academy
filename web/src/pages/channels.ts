import {
  getGatewayInfo,
  saveGatewayConfig,
  ApiError,
  type GatewayInfo,
  type GatewayPlatform,
  type GatewaySettingsPayload,
  type GatewaySettingsView,
} from '../lib/api'
import { getActiveProfile } from '../lib/profile'
import { setBreadcrumb, updateSidebar } from '../components/layout'

const PLATFORM_LABEL: Record<string, string> = {
  telegram: 'Telegram',
  dingtalk: '钉钉',
  feishu: '飞书',
  wecom: '企业微信',
}

const PLATFORM_ICON: Record<string, string> = {
  telegram: 'TG',
  dingtalk: '钉',
  feishu: '飞',
  wecom: '企',
}

export async function renderChannels(container: HTMLElement): Promise<void> {
  void updateSidebar({ active: 'channels' })
  setBreadcrumb([{ label: '开始学习', href: '#/' }, { label: 'IM 频道' }])

  container.innerHTML = loadingHtml()

  try {
    const info = await getGatewayInfo()
    const profile = getActiveProfile()
    mountPage(container, info, profile?.displayName ?? '未选择')
  } catch (e) {
    container.innerHTML = `
      <section class="page page-channels">
        <div class="alert alert-error">${e instanceof ApiError ? e.message : '加载失败'}</div>
      </section>
    `
  }
}

function loadingHtml(): string {
  return `
    <section class="page page-channels">
      <div class="page-loading"><div class="spinner" aria-hidden="true"></div><p>加载频道配置…</p></div>
    </section>
  `
}

function mountPage(container: HTMLElement, info: GatewayInfo, userName: string): void {
  container.innerHTML = renderPage(info, userName)
  bindPage(container)
}

function renderPage(info: GatewayInfo, userName: string): string {
  const s = info.settings
  const statusClass = info.enabled ? (info.activePlatforms > 0 ? 'ok' : 'warn') : 'off'
  const statusText = info.enabled
    ? info.activePlatforms > 0
      ? `${info.activePlatforms} 个平台已就绪`
      : '已启用，等待配置凭证'
    : 'Gateway 未启用'

  return `
    <section class="page page-channels">
      <header class="channel-hero">
        <div class="channel-hero-text">
          <p class="page-eyebrow">Channel</p>
          <h1 class="page-title">IM 频道</h1>
          <p class="page-sub">在常用 IM 中与同一教练对话，学习进度与 Web 同步。</p>
        </div>
        <div class="channel-hero-status channel-hero-status--${statusClass}">
          <span class="channel-status-dot" aria-hidden="true"></span>
          <span>${escapeHtml(statusText)}</span>
        </div>
      </header>

      <form id="gateway-form" class="channel-form" novalidate>
        <div id="channel-form-error"></div>
        <div id="channel-form-toast"></div>

        <section class="card channel-global-card">
          <div class="channel-global-row">
            <label class="channel-switch">
              <input type="checkbox" name="enabled" ${s.enabled ? 'checked' : ''} />
              <span class="channel-switch-ui" aria-hidden="true"></span>
              <span class="channel-switch-label">
                <strong>启用 IM Gateway</strong>
                <small>开启后连接 Telegram / 钉钉 / 飞书 / 企微</small>
              </span>
            </label>
          </div>
          <div class="channel-field">
            <label class="field-label" for="publicUrl">公网地址（Webhook 展示用）</label>
            <input class="input" id="publicUrl" name="publicUrl" type="url" placeholder="https://your.domain.com" value="${escapeAttr(s.publicUrl)}" />
            <p class="field-hint">本地开发可留空，将使用当前访问地址</p>
          </div>
        </section>

        <div class="channel-grid">
          ${info.platforms.map((p) => renderPlatformForm(p, s, info.publicBaseUrl)).join('')}
        </div>

        <div class="channel-form-actions card">
          <p class="channel-form-note">保存后写入 <code class="inline-code">.env</code>，<strong>需重启服务</strong>后 Gateway 才会加载新配置。</p>
          <button type="submit" class="btn btn-primary" id="channel-save-btn">保存配置</button>
        </div>
      </form>

      <div class="channel-panels">
        ${renderBindPanel(info, userName)}
        ${renderCmdPanel(info)}
      </div>
    </section>
  `
}

function renderPlatformForm(p: GatewayPlatform, s: GatewaySettingsView, baseUrl: string): string {
  const statusLabel = p.status === 'ready' ? '已就绪' : p.status === 'pending' ? '待配置' : '未启用'
  const icon = PLATFORM_ICON[p.id] ?? 'IM'

  let fields = ''
  switch (p.id) {
    case 'telegram':
      fields = `
        ${secretField('telegramBotToken', 'Bot Token', s.telegramBotTokenSet)}
        ${textField('telegramAllowedUsers', '允许的用户 ID（可选，逗号分隔）', s.telegramAllowedUsers)}
      `
      break
    case 'dingtalk':
      fields = `
        ${textField('dingtalkClientId', 'Client ID', s.dingtalkClientId)}
        ${secretField('dingtalkClientSecret', 'Client Secret', s.dingtalkClientSecretSet)}
      `
      break
    case 'feishu':
      fields = `
        ${textField('feishuAppId', 'App ID', s.feishuAppId)}
        ${secretField('feishuAppSecret', 'App Secret', s.feishuAppSecretSet)}
        <div class="channel-field">
          <label class="field-label" for="feishuMode">连接模式</label>
          <select class="input" id="feishuMode" name="feishuMode">
            <option value="websocket" ${s.feishuMode !== 'webhook' ? 'selected' : ''}>WebSocket（内网可用）</option>
            <option value="webhook" ${s.feishuMode === 'webhook' ? 'selected' : ''}>Webhook（需公网 HTTPS）</option>
          </select>
        </div>
      `
      break
    case 'wecom':
      fields = `
        ${textField('wecomCorpId', 'Corp ID', s.wecomCorpId)}
        ${textField('wecomAgentId', 'Agent ID', s.wecomAgentId)}
        ${secretField('wecomSecret', 'Secret', s.wecomSecretSet)}
        ${secretField('wecomToken', 'Token', s.wecomTokenSet)}
        ${secretField('wecomEncodingAesKey', 'EncodingAESKey', s.wecomEncodingAesKeySet)}
        ${textField('wecomAllowedUsers', '允许的用户（可选）', s.wecomAllowedUsers)}
      `
      break
  }

  const platformEnabled = platformEnabledValue(p.id, s)
  const webhook =
    p.webhookUrl || (p.id === 'wecom' ? `${baseUrl}/webhook/wecom` : p.id === 'feishu' && s.feishuMode === 'webhook' ? `${baseUrl}/webhook/feishu` : '')

  return `
    <article class="card channel-platform channel-platform--${p.id} channel-platform--${p.status}">
      <div class="channel-platform-top">
        <div class="channel-platform-brand">
          <span class="channel-platform-avatar channel-platform-avatar--${p.id}">${icon}</span>
          <div>
            <h3 class="channel-platform-name">${escapeHtml(p.name)}</h3>
            <p class="channel-platform-connection">${escapeHtml(p.connection ?? '')}</p>
          </div>
        </div>
        <div class="channel-platform-meta">
          <span class="channel-platform-badge channel-platform-badge--${p.status}">${statusLabel}</span>
          <label class="channel-switch channel-switch--compact">
            <input type="checkbox" name="${p.id}Enabled" ${platformEnabled ? 'checked' : ''} />
            <span class="channel-switch-ui" aria-hidden="true"></span>
          </label>
        </div>
      </div>
      ${p.setupHint ? `<p class="channel-platform-hint">${escapeHtml(p.setupHint)}</p>` : ''}
      <div class="channel-platform-fields">${fields}</div>
      ${
        webhook && (p.id === 'wecom' || p.id === 'feishu')
          ? `
        <div class="channel-webhook">
          <span class="channel-webhook-label">回调 URL</span>
          <code class="channel-webhook-url">${escapeHtml(webhook)}</code>
          <button type="button" class="btn btn-ghost btn-sm channel-copy-btn" data-copy="${escapeAttr(webhook)}">复制</button>
        </div>
      `
          : ''
      }
    </article>
  `
}

function platformEnabledValue(id: string, s: GatewaySettingsView): boolean {
  switch (id) {
    case 'telegram':
      return s.telegramEnabled
    case 'dingtalk':
      return s.dingtalkEnabled
    case 'feishu':
      return s.feishuEnabled
    case 'wecom':
      return s.wecomEnabled
    default:
      return false
  }
}

function textField(name: string, label: string, value: string): string {
  return `
    <div class="channel-field">
      <label class="field-label" for="${name}">${escapeHtml(label)}</label>
      <input class="input" id="${name}" name="${name}" type="text" value="${escapeAttr(value)}" autocomplete="off" />
    </div>
  `
}

function secretField(name: string, label: string, isSet: boolean): string {
  const hint = isSet ? '已配置 · 留空则不修改' : '尚未配置'
  return `
    <div class="channel-field">
      <label class="field-label" for="${name}">${escapeHtml(label)}</label>
      <input class="input" id="${name}" name="${name}" type="password" placeholder="${escapeAttr(hint)}" autocomplete="new-password" />
    </div>
  `
}

function renderBindPanel(info: GatewayInfo, userName: string): string {
  return `
    <section class="card channel-panel">
      <h2 class="channel-panel-title">绑定当前角色</h2>
      <p class="channel-panel-sub">在 IM 中发送以下消息，绑定到「<strong>${escapeHtml(userName)}</strong>」：</p>
      <div class="channel-bind-cmd">
        <code>绑定 ${escapeHtml(userName)}</code>
        <button type="button" class="btn btn-ghost btn-sm channel-copy-btn" data-copy="绑定 ${escapeAttr(userName)}">复制</button>
      </div>
      ${
        info.bindings.length > 0
          ? `
        <ul class="channel-bindings-list">
          ${info.bindings
            .map(
              (b) => `
            <li class="channel-binding-item">
              <span class="channel-binding-platform">${escapeHtml(PLATFORM_LABEL[b.platform] ?? b.platform)}</span>
              <span class="channel-binding-id">${escapeHtml(b.platformUserId)}</span>
            </li>
          `
            )
            .join('')}
        </ul>
      `
          : `<p class="channel-panel-hint">当前角色尚未绑定 IM 账号</p>`
      }
    </section>
  `
}

function renderCmdPanel(info: GatewayInfo): string {
  return `
    <section class="card channel-panel">
      <h2 class="channel-panel-title">IM 命令</h2>
      <div class="channel-cmd-list">
        ${info.commands
          .map(
            (c) => `
          <div class="channel-cmd-row">
            <code>${escapeHtml(c.command)}</code>
            <span>${escapeHtml(c.description)}</span>
          </div>
        `
          )
          .join('')}
      </div>
    </section>
  `
}

function bindPage(container: HTMLElement): void {
  const form = container.querySelector<HTMLFormElement>('#gateway-form')
  form?.addEventListener('submit', (e) => {
    e.preventDefault()
    void submitForm(container, form)
  })

  container.querySelectorAll<HTMLButtonElement>('.channel-copy-btn').forEach((btn) => {
    btn.addEventListener('click', () => void copyText(btn))
  })
}

async function submitForm(container: HTMLElement, form: HTMLFormElement): Promise<void> {
  const errEl = container.querySelector<HTMLDivElement>('#channel-form-error')!
  const toastEl = container.querySelector<HTMLDivElement>('#channel-form-toast')!
  const btn = container.querySelector<HTMLButtonElement>('#channel-save-btn')!
  errEl.innerHTML = ''
  toastEl.innerHTML = ''
  btn.disabled = true
  btn.textContent = '保存中…'

  const fd = new FormData(form)
  const payload: GatewaySettingsPayload = {
    enabled: fd.get('enabled') === 'on',
    publicUrl: String(fd.get('publicUrl') ?? '').trim(),
    telegramEnabled: fd.get('telegramEnabled') === 'on',
    telegramAllowedUsers: String(fd.get('telegramAllowedUsers') ?? '').trim(),
    dingtalkEnabled: fd.get('dingtalkEnabled') === 'on',
    dingtalkClientId: String(fd.get('dingtalkClientId') ?? '').trim(),
    feishuEnabled: fd.get('feishuEnabled') === 'on',
    feishuMode: String(fd.get('feishuMode') ?? 'websocket'),
    feishuAppId: String(fd.get('feishuAppId') ?? '').trim(),
    wecomEnabled: fd.get('wecomEnabled') === 'on',
    wecomCorpId: String(fd.get('wecomCorpId') ?? '').trim(),
    wecomAgentId: String(fd.get('wecomAgentId') ?? '').trim(),
    wecomAllowedUsers: String(fd.get('wecomAllowedUsers') ?? '').trim(),
  }

  const telegramBotToken = String(fd.get('telegramBotToken') ?? '').trim()
  if (telegramBotToken) payload.telegramBotToken = telegramBotToken
  const dingtalkClientSecret = String(fd.get('dingtalkClientSecret') ?? '').trim()
  if (dingtalkClientSecret) payload.dingtalkClientSecret = dingtalkClientSecret
  const feishuAppSecret = String(fd.get('feishuAppSecret') ?? '').trim()
  if (feishuAppSecret) payload.feishuAppSecret = feishuAppSecret
  const wecomSecret = String(fd.get('wecomSecret') ?? '').trim()
  if (wecomSecret) payload.wecomSecret = wecomSecret
  const wecomToken = String(fd.get('wecomToken') ?? '').trim()
  if (wecomToken) payload.wecomToken = wecomToken
  const wecomEncodingAesKey = String(fd.get('wecomEncodingAesKey') ?? '').trim()
  if (wecomEncodingAesKey) payload.wecomEncodingAesKey = wecomEncodingAesKey

  try {
    const info = await saveGatewayConfig(payload)
    toastEl.innerHTML = '<div class="alert alert-success">配置已保存，请重启服务使 Gateway 生效</div>'
    const profile = getActiveProfile()
    mountPage(container, info, profile?.displayName ?? '未选择')
  } catch (e) {
    errEl.innerHTML = `<div class="alert alert-error">${e instanceof ApiError ? e.message : '保存失败'}</div>`
  } finally {
    const saveBtn = container.querySelector<HTMLButtonElement>('#channel-save-btn')
    if (saveBtn) {
      saveBtn.disabled = false
      saveBtn.textContent = '保存配置'
    }
  }
}

async function copyText(btn: HTMLButtonElement): Promise<void> {
  const text = btn.dataset.copy ?? ''
  try {
    await navigator.clipboard.writeText(text)
    const prev = btn.textContent
    btn.textContent = '已复制'
    setTimeout(() => {
      btn.textContent = prev
    }, 1200)
  } catch {
    btn.textContent = '失败'
  }
}

function escapeHtml(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}

function escapeAttr(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;')
}
