/** 与角色/课程删除一致的「输入名称以确认」二次确认弹层 */

export interface TypeConfirmOptions {
  title: string
  /** 说明文案（纯文本，会转义） */
  description: string
  /** 强调展示的名称（如课程名、模型显示名） */
  subjectLabel?: string
  subjectName: string
  /** 用户须完整输入此字符串才能确认 */
  confirmPhrase: string
  confirmLabel?: string
  inputPlaceholder?: string
  /** 叠在其它 profile-overlay 之上 */
  nested?: boolean
}

export function showTypeConfirm(options: TypeConfirmOptions): Promise<boolean> {
  const {
    title,
    description,
    subjectLabel = '名称',
    subjectName,
    confirmPhrase,
    confirmLabel = '确认',
    inputPlaceholder = '输入以确认',
    nested = false,
  } = options

  return new Promise((resolve) => {
    const overlay = document.createElement('div')
    overlay.className = nested ? 'profile-overlay profile-overlay-nested' : 'profile-overlay'
    overlay.innerHTML = `
      <div class="profile-modal card profile-delete-modal" role="alertdialog" aria-labelledby="type-confirm-title">
        <h3 id="type-confirm-title" class="profile-modal-title">${escapeHtml(title)}</h3>
        <p class="profile-modal-sub">
          ${escapeHtml(description)}
          <br><br>
          ${escapeHtml(subjectLabel)}：<strong>${escapeHtml(subjectName)}</strong>
        </p>
        <label class="field-label" for="type-confirm-input">
          请输入 <strong>${escapeHtml(confirmPhrase)}</strong> 以确认
        </label>
        <input class="input" id="type-confirm-input" type="text" placeholder="${escapeAttr(inputPlaceholder)}" autocomplete="off" />
        <div class="profile-delete-actions">
          <button type="button" class="btn btn-ghost" id="type-confirm-cancel-btn">取消</button>
          <button type="button" class="btn btn-danger" id="type-confirm-ok-btn" disabled>${escapeHtml(confirmLabel)}</button>
        </div>
      </div>
    `
    document.body.appendChild(overlay)

    const input = overlay.querySelector<HTMLInputElement>('#type-confirm-input')!
    const okBtn = overlay.querySelector<HTMLButtonElement>('#type-confirm-ok-btn')!
    const cancelBtn = overlay.querySelector<HTMLButtonElement>('#type-confirm-cancel-btn')!

    const dismiss = (ok: boolean) => {
      overlay.remove()
      resolve(ok)
    }

    const syncOkBtn = () => {
      okBtn.disabled = input.value.trim() !== confirmPhrase
    }

    input.addEventListener('input', syncOkBtn)
    cancelBtn.addEventListener('click', () => dismiss(false))
    overlay.addEventListener('click', (e) => {
      if (e.target === overlay) dismiss(false)
    })
    okBtn.addEventListener('click', () => dismiss(true))

    input.focus()
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
