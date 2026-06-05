import { submitOnboarding, ApiError, type UserProfile } from '../lib/api'

export interface OnboardingResult {
  user: UserProfile
  skipped: boolean
}

export function needsOnboarding(user: UserProfile): boolean {
  return !user.onboardedAt
}

export function showOnboardingCard(userId: string): Promise<OnboardingResult | null> {
  return new Promise((resolve) => {
    const overlay = document.createElement('div')
    overlay.className = 'profile-overlay'
    overlay.innerHTML = `
      <div class="profile-modal card" role="dialog" aria-modal="true" aria-labelledby="onboarding-title">
        <div class="profile-modal-header">
          <h2 id="onboarding-title" class="profile-modal-title">了解一下你</h2>
          <p class="profile-modal-sub">填写后我会按你的背景规划学习路径；也可稍后再说。</p>
        </div>
        <div id="onboarding-error"></div>
        <form class="profile-form" id="onboarding-form">
          <label class="field-label" for="onboarding-role">身份 / 角色</label>
          <input class="input" id="onboarding-role" type="text" placeholder="例如：产品经理、后端开发、在校学生" maxlength="64" />
          <label class="field-label" for="onboarding-background">已有基础</label>
          <textarea class="input" id="onboarding-background" rows="2" maxlength="300" placeholder="例如：会 Python，没写过 Go"></textarea>
          <label class="field-label" for="onboarding-goal">学习目标（可选）</label>
          <input class="input" id="onboarding-goal" type="text" placeholder="例如：能独立写并发服务" maxlength="200" />
          <div class="profile-delete-actions" style="margin-top: 1rem;">
            <button type="button" class="btn btn-ghost" id="onboarding-skip-btn">稍后再说</button>
            <button type="submit" class="btn btn-primary" id="onboarding-submit-btn">保存并继续</button>
          </div>
        </form>
      </div>
    `

    const dismiss = (result: OnboardingResult | null) => {
      overlay.remove()
      resolve(result)
    }

    const errEl = overlay.querySelector<HTMLDivElement>('#onboarding-error')!
    const form = overlay.querySelector<HTMLFormElement>('#onboarding-form')!
    const roleInput = overlay.querySelector<HTMLInputElement>('#onboarding-role')!
    const bgInput = overlay.querySelector<HTMLTextAreaElement>('#onboarding-background')!
    const goalInput = overlay.querySelector<HTMLInputElement>('#onboarding-goal')!
    const submitBtn = overlay.querySelector<HTMLButtonElement>('#onboarding-submit-btn')!
    const skipBtn = overlay.querySelector<HTMLButtonElement>('#onboarding-skip-btn')!

    const setBusy = (busy: boolean) => {
      submitBtn.disabled = busy
      skipBtn.disabled = busy
      roleInput.disabled = busy
      bgInput.disabled = busy
      goalInput.disabled = busy
    }

    skipBtn.addEventListener('click', () => {
      void (async () => {
        setBusy(true)
        errEl.innerHTML = ''
        try {
          const user = await submitOnboarding(userId, { role: '', background: '', skip: true })
          dismiss({ user, skipped: true })
        } catch (e) {
          errEl.innerHTML = `<div class="alert alert-error">${escapeHtml(e instanceof ApiError ? e.message : '操作失败')}</div>`
          setBusy(false)
        }
      })()
    })

    form.addEventListener('submit', (e) => {
      e.preventDefault()
      void (async () => {
        const role = roleInput.value.trim()
        const background = bgInput.value.trim()
        if (!role || !background) {
          errEl.innerHTML = '<div class="alert alert-error">请填写身份与已有基础</div>'
          return
        }
        setBusy(true)
        errEl.innerHTML = ''
        try {
          const user = await submitOnboarding(userId, {
            role,
            background,
            goal: goalInput.value.trim() || undefined,
          })
          dismiss({ user, skipped: false })
        } catch (e) {
          errEl.innerHTML = `<div class="alert alert-error">${escapeHtml(e instanceof ApiError ? e.message : '保存失败')}</div>`
          setBusy(false)
        }
      })()
    })

    document.body.appendChild(overlay)
    roleInput.focus()
  })
}

function escapeHtml(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}
