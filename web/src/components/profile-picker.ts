import { listUsers, createUser, deleteUser, type UserProfile, ApiError } from '../lib/api'
import { getActiveProfile, setActiveProfile, clearActiveProfile } from '../lib/profile'

export interface ProfilePickerOptions {
  /** 首次进入必须选择；切换角色时可取消 */
  required?: boolean
  title?: string
  subtitle?: string
}

export function showProfilePicker(options: ProfilePickerOptions = {}): Promise<UserProfile | null> {
  const required = options.required ?? false
  const title = options.title ?? (required ? '选择学习角色' : '切换学习角色')
  const subtitle =
    options.subtitle ??
    (required
      ? '输入你的名字，或选择已有角色。每人有独立的知识库与学习进度，无需密码。'
      : '切换后将看到该角色专属的课程与学习记录。')

  return new Promise((resolve) => {
    const overlay = document.createElement('div')
    overlay.className = 'profile-overlay'
    overlay.innerHTML = `
      <div class="profile-modal card" role="dialog" aria-modal="true" aria-labelledby="profile-title">
        <div class="profile-modal-header">
          <h2 id="profile-title" class="profile-modal-title">${escapeHtml(title)}</h2>
          <p class="profile-modal-sub">${escapeHtml(subtitle)}</p>
          ${!required ? `<button type="button" class="profile-close" aria-label="关闭">×</button>` : ''}
        </div>
        <div id="profile-error"></div>
        <form class="profile-form" id="profile-form">
          <label class="field-label" for="profile-name">你的名字</label>
          <input class="input input-lg" id="profile-name" type="text" placeholder="例如：小明" maxlength="32" autocomplete="name" />
          <button type="submit" class="btn btn-primary btn-lg" id="profile-create-btn">开始使用</button>
        </form>
        <div class="profile-list-wrap">
          <h3 class="profile-list-title">已有角色</h3>
          <div id="profile-list" class="profile-list">
            <p class="profile-list-empty">加载中…</p>
          </div>
        </div>
      </div>
    `

    const close = (result: UserProfile | null) => {
      overlay.remove()
      resolve(result)
    }

    const errEl = overlay.querySelector<HTMLDivElement>('#profile-error')!
    const listEl = overlay.querySelector<HTMLDivElement>('#profile-list')!
    const form = overlay.querySelector<HTMLFormElement>('#profile-form')!
    const nameInput = overlay.querySelector<HTMLInputElement>('#profile-name')!
    const createBtn = overlay.querySelector<HTMLButtonElement>('#profile-create-btn')!

    let usersCache: UserProfile[] = []

    overlay.querySelector('.profile-close')?.addEventListener('click', () => close(null))

    if (!required) {
      overlay.addEventListener('click', (e) => {
        if (e.target === overlay) close(null)
      })
    }

    const selectProfile = (profile: UserProfile) => {
      setActiveProfile(profile)
      close(profile)
    }

    const showDeleteConfirm = (user: UserProfile) => {
      const confirmOverlay = document.createElement('div')
      confirmOverlay.className = 'profile-overlay profile-overlay-nested'
      confirmOverlay.innerHTML = `
        <div class="profile-modal card profile-delete-modal" role="alertdialog" aria-labelledby="delete-title">
          <h3 id="delete-title" class="profile-modal-title">移除角色</h3>
          <p class="profile-modal-sub">
            将永久删除「${escapeHtml(user.displayName)}」的知识库、学习进度和聊天记录，且无法恢复。
          </p>
          <div id="delete-error"></div>
          <label class="field-label" for="delete-confirm-name">
            请输入 <strong>${escapeHtml(user.displayName)}</strong> 以确认
          </label>
          <input class="input" id="delete-confirm-name" type="text" placeholder="输入角色名" autocomplete="off" />
          <div class="profile-delete-actions">
            <button type="button" class="btn btn-ghost" id="delete-cancel-btn">取消</button>
            <button type="button" class="btn btn-danger" id="delete-confirm-btn" disabled>确认移除</button>
          </div>
        </div>
      `
      document.body.appendChild(confirmOverlay)

      const deleteErrEl = confirmOverlay.querySelector<HTMLDivElement>('#delete-error')!
      const confirmInput = confirmOverlay.querySelector<HTMLInputElement>('#delete-confirm-name')!
      const confirmBtn = confirmOverlay.querySelector<HTMLButtonElement>('#delete-confirm-btn')!
      const cancelBtn = confirmOverlay.querySelector<HTMLButtonElement>('#delete-cancel-btn')!

      const dismiss = () => confirmOverlay.remove()

      const syncConfirmBtn = () => {
        confirmBtn.disabled = confirmInput.value.trim() !== user.displayName
      }

      confirmInput.addEventListener('input', syncConfirmBtn)
      cancelBtn.addEventListener('click', dismiss)
      confirmOverlay.addEventListener('click', (e) => {
        if (e.target === confirmOverlay) dismiss()
      })

      confirmBtn.addEventListener('click', () => {
        void (async () => {
          confirmBtn.disabled = true
          confirmBtn.textContent = '移除中…'
          deleteErrEl.innerHTML = ''
          try {
            await deleteUser(user.id, confirmInput.value.trim())
            dismiss()
            const wasActive = getActiveProfile()?.id === user.id
            if (wasActive) {
              clearActiveProfile()
              close(null)
              window.location.reload()
              return
            }
            usersCache = usersCache.filter((u) => u.id !== user.id)
            renderList(usersCache)
          } catch (err) {
            deleteErrEl.innerHTML = `<div class="alert alert-error">${err instanceof ApiError ? err.message : '移除失败'}</div>`
            confirmBtn.disabled = false
            confirmBtn.textContent = '确认移除'
            syncConfirmBtn()
          }
        })()
      })

      confirmInput.focus()
    }

    const renderList = (users: UserProfile[]) => {
      usersCache = users
      const active = getActiveProfile()
      if (users.length === 0) {
        listEl.innerHTML = '<p class="profile-list-empty">还没有角色，输入名字创建一个吧</p>'
        return
      }
      listEl.innerHTML = users
        .map(
          (u) => `
          <div class="profile-item-row ${active?.id === u.id ? 'is-active' : ''}">
            <button type="button" class="profile-item" data-id="${escapeHtml(u.id)}">
              <span class="profile-item-avatar" aria-hidden="true">${escapeHtml(u.displayName.slice(0, 1))}</span>
              <span class="profile-item-name">${escapeHtml(u.displayName)}</span>
              ${active?.id === u.id ? '<span class="profile-item-tag">当前</span>' : ''}
            </button>
            <button type="button" class="profile-delete-btn" data-id="${escapeHtml(u.id)}" aria-label="移除 ${escapeHtml(u.displayName)}" title="移除角色">×</button>
          </div>
        `
        )
        .join('')

      listEl.querySelectorAll<HTMLButtonElement>('.profile-item').forEach((btn) => {
        btn.addEventListener('click', () => {
          const user = usersCache.find((u) => u.id === btn.dataset.id)
          if (user) selectProfile(user)
        })
      })

      listEl.querySelectorAll<HTMLButtonElement>('.profile-delete-btn').forEach((btn) => {
        btn.addEventListener('click', (e) => {
          e.stopPropagation()
          const user = usersCache.find((u) => u.id === btn.dataset.id)
          if (user) showDeleteConfirm(user)
        })
      })
    }

    void listUsers()
      .then(renderList)
      .catch(() => {
        listEl.innerHTML = '<p class="profile-list-empty">无法加载角色列表</p>'
      })

    let composing = false
    nameInput.addEventListener('compositionstart', () => {
      composing = true
    })
    nameInput.addEventListener('compositionend', () => {
      composing = false
    })

    form.addEventListener('submit', (e) => {
      e.preventDefault()
      void (async () => {
        const name = nameInput.value.trim()
        if (!name) {
          errEl.innerHTML = '<div class="alert alert-error">请输入你的名字</div>'
          return
        }
        createBtn.disabled = true
        createBtn.textContent = '创建中…'
        errEl.innerHTML = ''
        try {
          const user = await createUser(name)
          selectProfile(user)
        } catch (err) {
          errEl.innerHTML = `<div class="alert alert-error">${err instanceof ApiError ? err.message : '创建失败'}</div>`
        } finally {
          createBtn.disabled = false
          createBtn.textContent = '开始使用'
        }
      })()
    })

    nameInput.addEventListener('keydown', (e) => {
      if (e.key === 'Enter' && !e.isComposing && !composing) {
        e.preventDefault()
        form.requestSubmit()
      }
    })

    document.body.appendChild(overlay)
    nameInput.focus()
  })
}

export async function ensureProfile(): Promise<UserProfile> {
  const existing = getActiveProfile()
  if (existing) return existing
  const picked = await showProfilePicker({ required: true })
  if (!picked) {
    return ensureProfile()
  }
  return picked
}

function escapeHtml(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}
