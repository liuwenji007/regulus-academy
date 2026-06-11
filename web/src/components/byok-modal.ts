import { saveUserLLMKey } from '../lib/cloud'

export function showByokModal(): Promise<boolean> {
  return new Promise((resolve) => {
    const overlay = document.createElement('div')
    overlay.className = 'profile-overlay'
    overlay.innerHTML = `
      <div class="profile-modal card" role="dialog" aria-modal="true">
        <h2 class="profile-modal-title">填写你的 API Key 继续</h2>
        <p class="profile-modal-sub">今日免费额度已用尽。填入你自己的 LLM Key 后可继续使用，Key 仅保存在服务端。</p>
        <div id="byok-error"></div>
        <label class="field-label" for="byok-key">API Key</label>
        <input class="input" id="byok-key" type="password" autocomplete="off" placeholder="sk-..." />
        <label class="field-label" for="byok-provider">提供商</label>
        <select class="input" id="byok-provider">
          <option value="deepseek">DeepSeek</option>
          <option value="openai">OpenAI</option>
          <option value="openrouter">OpenRouter</option>
          <option value="custom">自定义</option>
        </select>
        <div class="profile-delete-actions" style="margin-top:1rem">
          <button type="button" class="btn btn-ghost" id="byok-cancel">稍后再说</button>
          <button type="button" class="btn btn-primary" id="byok-save">保存并继续</button>
        </div>
      </div>
    `
    const errEl = overlay.querySelector<HTMLDivElement>('#byok-error')!
    const close = (ok: boolean) => {
      overlay.remove()
      resolve(ok)
    }
    overlay.querySelector('#byok-cancel')?.addEventListener('click', () => close(false))
    overlay.querySelector('#byok-save')?.addEventListener('click', async () => {
      const key = overlay.querySelector<HTMLInputElement>('#byok-key')!.value.trim()
      const provider = overlay.querySelector<HTMLSelectElement>('#byok-provider')!.value
      if (!key) {
        errEl.innerHTML = '<div class="alert alert-error">请输入 API Key</div>'
        return
      }
      try {
        await saveUserLLMKey({ provider, apiKey: key })
        close(true)
      } catch (e) {
        errEl.innerHTML = `<div class="alert alert-error">${escapeHtml(e instanceof Error ? e.message : '保存失败')}</div>`
      }
    })
    document.body.appendChild(overlay)
  })
}

function escapeHtml(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}
