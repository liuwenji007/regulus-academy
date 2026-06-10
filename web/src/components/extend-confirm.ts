export type ExtendConfirmResult = { ok: false } | { ok: true; goal: string }

export interface ExtendConfirmOptions {
  domainName: string
  completed: number
  total: number
  minRatio: number
}

export function showExtendConfirm(options: ExtendConfirmOptions): Promise<ExtendConfirmResult> {
  const { domainName, completed, total, minRatio } = options
  const minPct = Math.round(minRatio * 100)

  return new Promise((resolve) => {
    const overlay = document.createElement('div')
    overlay.className = 'profile-overlay'
    overlay.innerHTML = `
      <div class="profile-modal card profile-delete-modal domain-action-modal" role="alertdialog" aria-labelledby="extend-confirm-title">
        <div class="domain-action-form">
          <h3 id="extend-confirm-title" class="profile-modal-title">解锁进阶路径</h3>
          <p class="profile-modal-sub">
            你已完成 <strong>${completed}/${total}</strong> 个节点（≥${minPct}%）。
            系统将根据课程规模追加约 2～8 个节点（窄主题最多 5 个），可含精通纵深或熟悉层生产实战；学完后再次达到完成度可继续扩展。原有进度会保留。
            <br><br>
            课程：<strong>${escapeHtml(domainName)}</strong>
          </p>
          <label class="field-label" for="extend-goal-input">进阶学习目标（可选）</label>
          <input
            class="input"
            id="extend-goal-input"
            type="text"
            placeholder="例如：深入并发模式、准备面试…"
            autocomplete="off"
            maxlength="200"
          />
          <div class="profile-delete-actions">
            <button type="button" class="btn btn-ghost" id="extend-cancel-btn">取消</button>
            <button type="button" class="btn btn-primary" id="extend-confirm-btn">确认解锁</button>
          </div>
        </div>
      </div>
    `
    document.body.appendChild(overlay)

    const goalInput = overlay.querySelector<HTMLInputElement>('#extend-goal-input')!
    const cancelBtn = overlay.querySelector<HTMLButtonElement>('#extend-cancel-btn')!
    const confirmBtn = overlay.querySelector<HTMLButtonElement>('#extend-confirm-btn')!

    const close = (result: ExtendConfirmResult) => {
      overlay.remove()
      resolve(result)
    }

    cancelBtn.addEventListener('click', () => close({ ok: false }))
    overlay.addEventListener('click', (e) => {
      if (e.target === overlay) close({ ok: false })
    })
    confirmBtn.addEventListener('click', () => {
      close({ ok: true, goal: goalInput.value.trim() })
    })
    goalInput.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') {
        e.preventDefault()
        close({ ok: true, goal: goalInput.value.trim() })
      }
      if (e.key === 'Escape') close({ ok: false })
    })
    goalInput.focus()
  })
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}
