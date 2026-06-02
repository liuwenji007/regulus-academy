import { deleteDomain, regenerateDomain, ApiError, type BuildDomainResult } from '../lib/api'
import { setAppBusy, clearAppBusyIf } from '../lib/app-busy'

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
  {
    title: string
    description: string
    confirmLabel: string
    busyTitle: string
    busyHint: string
  }
> = {
  delete: {
    title: '移除课程',
    description: '将永久删除该课程的知识树、学习进度和全部聊天记录，且无法恢复。',
    confirmLabel: '确认移除',
    busyTitle: '正在移除课程',
    busyHint: '请稍候…',
  },
  regenerate: {
    title: '重新生成课程',
    description: '将清除当前课程的全部进度与聊天记录，并根据同一主题重新生成知识树。',
    confirmLabel: '确认重新生成',
    busyTitle: '正在重新生成课程',
    busyHint: 'AI 正在根据同一主题规划知识树，通常需要 30 秒～2 分钟，请稍候',
  },
}

export function showDomainConfirm(options: DomainConfirmOptions): Promise<DomainConfirmResult> {
  const { domainId, domainName, action } = options
  const copy = COPY[action]

  return new Promise((resolve) => {
    const overlay = document.createElement('div')
    overlay.className = 'profile-overlay'
    overlay.innerHTML = `
      <div class="profile-modal card profile-delete-modal domain-action-modal" role="alertdialog" aria-labelledby="domain-action-title">
        <div class="domain-action-form">
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
        <div class="domain-action-loading" hidden>
          <div class="spinner domain-action-spinner" aria-hidden="true"></div>
          <p class="domain-action-loading-title">${escapeHtml(copy.busyTitle)}</p>
          <p class="domain-action-loading-course">课程：${escapeHtml(domainName)}</p>
          <p class="domain-action-loading-hint">${escapeHtml(copy.busyHint)}</p>
        </div>
      </div>
    `
    document.body.appendChild(overlay)

    const modalEl = overlay.querySelector<HTMLElement>('.domain-action-modal')!
    const loadingEl = overlay.querySelector<HTMLElement>('.domain-action-loading')!
    const errEl = overlay.querySelector<HTMLDivElement>('#domain-action-error')!
    const confirmInput = overlay.querySelector<HTMLInputElement>('#domain-action-confirm-name')!
    const confirmBtn = overlay.querySelector<HTMLButtonElement>('#domain-action-confirm-btn')!
    const cancelBtn = overlay.querySelector<HTMLButtonElement>('#domain-action-cancel-btn')!

    let busy = false

    const dismiss = (result: DomainConfirmResult) => {
      overlay.remove()
      resolve(result)
    }

    const syncConfirmBtn = () => {
      if (busy) return
      confirmBtn.disabled = confirmInput.value.trim() !== domainName
    }

    const setBusy = (next: boolean) => {
      busy = next
      modalEl.classList.toggle('is-busy', next)
      loadingEl.hidden = !next
      confirmInput.disabled = next
      cancelBtn.disabled = next
      confirmBtn.disabled = next
      if (next) {
        overlay.setAttribute('aria-busy', 'true')
      } else {
        overlay.removeAttribute('aria-busy')
        syncConfirmBtn()
      }
    }

    confirmInput.addEventListener('input', syncConfirmBtn)
    cancelBtn.addEventListener('click', () => {
      if (busy) return
      dismiss({ ok: false })
    })
    overlay.addEventListener('click', (e) => {
      if (busy) return
      if (e.target === overlay) dismiss({ ok: false })
    })

    confirmBtn.addEventListener('click', () => {
      void (async () => {
        setBusy(true)
        errEl.innerHTML = ''
        if (action === 'regenerate') setAppBusy(true, 'build')
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
          // busy 保持到 handleDomainRegenerate → 课程树 renderTree 完成后再清除
          dismiss({ ok: true, action: 'regenerate', result })
        } catch (e) {
          clearAppBusyIf('build')
          setBusy(false)
          errEl.innerHTML = `<div class="alert alert-error">${escapeHtml(e instanceof ApiError ? e.message : '操作失败，请重试')}</div>`
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
