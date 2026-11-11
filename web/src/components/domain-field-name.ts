/** 自定义领域名称输入弹层（替代原生 prompt） */

export function showDomainFieldNameDialog(initial = ''): Promise<string | null> {
  return new Promise((resolve) => {
    const overlay = document.createElement('div')
    overlay.className = 'profile-overlay profile-overlay-nested'
    overlay.innerHTML = `
      <div class="profile-modal card profile-delete-modal" role="dialog" aria-modal="true" aria-labelledby="domain-field-name-title">
        <h3 id="domain-field-name-title" class="profile-modal-title">自定义领域</h3>
        <p class="profile-modal-sub">用于归类课程，1～12 个字。</p>
        <label class="field-label" for="domain-field-name-input">领域名称</label>
        <input
          class="input"
          id="domain-field-name-input"
          type="text"
          maxlength="12"
          placeholder="例如：产品设计"
          autocomplete="off"
          value="${escapeAttr(initial.trim())}"
        />
        <p class="field-hint" id="domain-field-name-hint">还可输入 <span id="domain-field-name-left">12</span> 字</p>
        <div id="domain-field-name-error" class="alert alert-error" hidden></div>
        <div class="profile-delete-actions">
          <button type="button" class="btn btn-ghost" id="domain-field-name-cancel">取消</button>
          <button type="button" class="btn btn-primary" id="domain-field-name-ok" disabled>确定</button>
        </div>
      </div>
    `
    document.body.appendChild(overlay)

    const input = overlay.querySelector<HTMLInputElement>('#domain-field-name-input')!
    const okBtn = overlay.querySelector<HTMLButtonElement>('#domain-field-name-ok')!
    const cancelBtn = overlay.querySelector<HTMLButtonElement>('#domain-field-name-cancel')!
    const leftEl = overlay.querySelector<HTMLElement>('#domain-field-name-left')!
    const errEl = overlay.querySelector<HTMLElement>('#domain-field-name-error')!

    const dismiss = (value: string | null) => {
      document.removeEventListener('keydown', onKey)
      overlay.remove()
      resolve(value)
    }

    const sync = () => {
      const raw = input.value.trim()
      const len = [...raw].length
      leftEl.textContent = String(Math.max(0, 12 - len))
      errEl.hidden = true
      errEl.textContent = ''
      okBtn.disabled = len < 1 || len > 12
    }

    const submit = () => {
      const raw = input.value.trim()
      const len = [...raw].length
      if (len < 1 || len > 12) {
        errEl.hidden = false
        errEl.textContent = '领域名称需要 1～12 个字'
        okBtn.disabled = true
        return
      }
      dismiss(raw)
    }

    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault()
        dismiss(null)
      } else if (e.key === 'Enter') {
        e.preventDefault()
        submit()
      }
    }

    input.addEventListener('input', sync)
    cancelBtn.addEventListener('click', () => dismiss(null))
    okBtn.addEventListener('click', submit)
    overlay.addEventListener('click', (e) => {
      if (e.target === overlay) dismiss(null)
    })
    document.addEventListener('keydown', onKey)

    sync()
    input.focus()
    input.select()
  })
}

function escapeAttr(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;')
}
