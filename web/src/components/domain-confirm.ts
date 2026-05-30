import { deleteDomain, regenerateDomain, ApiError, type BuildDomainResult } from '../lib/api'

export type DomainConfirmAction = 'delete' | 'regenerate'

export type DomainConfirmResult =
  | { ok: false }
  | { ok: true; action: 'delete' }
  | { ok: true; action: 'regenerate'; result: BuildDomainResult }

export interface DomainConfirmOptions {
  domainId: string
  domainName: string
  action: DomainConfirmAction
}

const COPY: Record<
  DomainConfirmAction,
  { title: string; description: string; confirmLabel: string; busyLabel: string }
> = {
  delete: {
    title: '移除课程',
    description: '将永久删除该课程的知识树、学习进度和全部聊天记录，且无法恢复。',
    confirmLabel: '确认移除',
    busyLabel: '移除中…',
  },
  regenerate: {
    title: '重新生成课程',
    description: '将清除当前课程的全部进度与聊天记录，并根据同一主题重新生成知识树。',
    confirmLabel: '确认重新生成',
    busyLabel: '生成中…',
  },
}

export function showDomainConfirm(options: DomainConfirmOptions): Promise<DomainConfirmResult> {
  const { domainId, domainName, action } = options
  const copy = COPY[action]

  return new Promise((resolve) => {
    const overlay = document.createElement('div')
    overlay.className = 'profile-overlay'
    overlay.innerHTML = `
      <div class="profile-modal card profile-delete-modal" role="alertdialog" aria-labelledby="domain-action-title">
        <h3 id="domain-action-title" class="profile-modal-title">${escapeHtml(copy.title)}</h3>
        <p class="profile-modal-sub">
          ${escapeHtml(copy.description)}
          <br><br>
          课程：<strong>${escapeHtml(domainName)}</strong>
        </p>
        <div id="domain-action-error"></div>
        <label class="field-label" for="domain-action-confirm-name">
          请输入 <strong>${escapeHtml(domainName)}</strong> 以确认
        </label>
        <input class="input" id="domain-action-confirm-name" type="text" placeholder="输入课程名" autocomplete="off" />
        <div class="profile-delete-actions">
          <button type="button" class="btn btn-ghost" id="domain-action-cancel-btn">取消</button>
          <button type="button" class="btn ${action === 'delete' ? 'btn-danger' : 'btn-primary'}" id="domain-action-confirm-btn" disabled>${escapeHtml(copy.confirmLabel)}</button>
        </div>
      </div>
    `
    document.body.appendChild(overlay)

    const errEl = overlay.querySelector<HTMLDivElement>('#domain-action-error')!
    const confirmInput = overlay.querySelector<HTMLInputElement>('#domain-action-confirm-name')!
    const confirmBtn = overlay.querySelector<HTMLButtonElement>('#domain-action-confirm-btn')!
    const cancelBtn = overlay.querySelector<HTMLButtonElement>('#domain-action-cancel-btn')!

    const dismiss = (result: DomainConfirmResult) => {
      overlay.remove()
      resolve(result)
    }

    const syncConfirmBtn = () => {
      confirmBtn.disabled = confirmInput.value.trim() !== domainName
    }

    confirmInput.addEventListener('input', syncConfirmBtn)
    cancelBtn.addEventListener('click', () => dismiss({ ok: false }))
    overlay.addEventListener('click', (e) => {
      if (e.target === overlay) dismiss({ ok: false })
    })

    confirmBtn.addEventListener('click', () => {
      void (async () => {
        confirmBtn.disabled = true
        confirmBtn.textContent = copy.busyLabel
        errEl.innerHTML = ''
        try {
          const confirmName = confirmInput.value.trim()
          if (action === 'delete') {
            await deleteDomain(domainId, confirmName)
            dismiss({ ok: true, action: 'delete' })
            return
          }
          const result = await regenerateDomain(domainId, confirmName)
          if (result.status !== 'ready' || !result.tree) {
            throw new ApiError(result.message ?? '重新生成失败')
          }
          dismiss({ ok: true, action: 'regenerate', result })
        } catch (e) {
          errEl.innerHTML = `<div class="alert alert-error">${escapeHtml(e instanceof ApiError ? e.message : '操作失败，请重试')}</div>`
          confirmBtn.textContent = copy.confirmLabel
          syncConfirmBtn()
        }
      })()
    })

    confirmInput.focus()
  })
}

function escapeHtml(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}
