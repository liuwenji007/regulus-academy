import {
  listUsers,
  updateUserProfile,
  refineUserProfile,
  ApiError,
  type UserProfile,
} from '../lib/api'
import { iconSparkles } from '../lib/icons'
import { getActiveProfile, setActiveProfile } from '../lib/profile'
import { setBreadcrumb, updateSidebar } from '../components/layout'

const MAX_PROFILE_CHARS = 500

function escapeHtml(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}

function charCount(text: string): number {
  return [...text].length
}

function syncLocalProfile(user: UserProfile): void {
  const active = getActiveProfile()
  if (active?.id === user.id) {
    setActiveProfile({
      id: user.id,
      displayName: user.displayName,
      profileSummary: user.profileSummary,
      onboardedAt: user.onboardedAt,
    })
  }
}

function bindCharCounter(textarea: HTMLTextAreaElement, counterEl: HTMLElement): void {
  const sync = () => {
    const n = charCount(textarea.value)
    counterEl.textContent = `${n} / ${MAX_PROFILE_CHARS}`
    counterEl.classList.toggle('profile-char-count--warn', n > MAX_PROFILE_CHARS * 0.9)
  }
  textarea.addEventListener('input', sync)
  sync()
}

type ProfileSection = { label: string; body: string }

function parseProfileSections(summary: string): ProfileSection[] {
  const text = summary.trim()
  if (!text) return []
  const re = /【([^】]+)】/g
  const hits: { index: number; label: string; end: number }[] = []
  let m: RegExpExecArray | null
  while ((m = re.exec(text)) !== null) {
    hits.push({ index: m.index, label: m[1].trim(), end: m.index + m[0].length })
  }
  if (hits.length === 0) return [{ label: '', body: text }]
  const sections: ProfileSection[] = []
  for (let i = 0; i < hits.length; i++) {
    const start = hits[i].end
    const end = i + 1 < hits.length ? hits[i + 1].index : text.length
    const body = text.slice(start, end).trim()
    if (body) sections.push({ label: hits[i].label, body })
  }
  return sections.length > 0 ? sections : [{ label: '', body: text }]
}

function renderProfileViewHtml(summary: string): string {
  const sections = parseProfileSections(summary)
  if (sections.length === 0) {
    return `<p class="profile-view-empty">还没有画像，在上方用一句话告诉 AI 你的背景与目标即可。</p>`
  }
  if (sections.length === 1 && !sections[0].label) {
    return `<p class="profile-view-text">${escapeHtml(sections[0].body)}</p>`
  }
  return `
    <div class="profile-sections">
      ${sections
        .map(
          (s) => `
        <div class="profile-section">
          ${s.label ? `<p class="profile-section-label">${escapeHtml(s.label)}</p>` : ''}
          <p class="profile-section-text">${escapeHtml(s.body)}</p>
        </div>
      `,
        )
        .join('')}
    </div>
  `
}

export async function renderProfileSettings(container: HTMLElement): Promise<void> {
  void updateSidebar({ active: 'settings' })
  setBreadcrumb([
    { label: '开始学习', href: '#/' },
    { label: '设置', href: '#/settings' },
    { label: '学习画像' },
  ])

  container.innerHTML = `
    <section class="page page-profile-settings">
      <div class="page-loading"><div class="spinner" aria-hidden="true"></div><p>加载学习画像…</p></div>
    </section>
  `

  const page = container.querySelector<HTMLElement>('.page-profile-settings')
  if (!page) return

  const active = getActiveProfile()
  if (!active?.id) {
    page.innerHTML = `
      <header class="page-header">
        <h1 class="page-title">学习画像</h1>
      </header>
      <div class="alert alert-error">请先选择学习角色</div>
    `
    return
  }

  let user: UserProfile = active
  try {
    const list = await listUsers()
    const fresh = list.find((u) => u.id === active.id)
    if (fresh) user = fresh
  } catch {
    /* 使用本地缓存 */
  }

  let savedSummary = (user.profileSummary ?? '').trim()
  const onboarded = Boolean(user.onboardedAt)
  const statusClass = onboarded ? 'profile-meta-badge--ok' : 'profile-meta-badge--warn'
  const statusText = onboarded ? '已完成冷启动' : '待完成冷启动'

  page.innerHTML = `
    <header class="page-header">
      <h1 class="page-title">学习画像</h1>
      <p class="page-sub">用一句话补充近况即可更新；Coach 与课程规划会据此调整。</p>
    </header>

    <div class="card profile-card" id="profile-sheet">
      <div class="profile-card-meta">
        <span class="profile-role-chip">${escapeHtml(user.displayName)}</span>
        <span class="profile-meta-badge ${statusClass}">${escapeHtml(statusText)}</span>
      </div>
      <p class="profile-card-note">重新生成会按当前画像裁剪已掌握节点，不等于保留旧课结构。</p>

      <section class="profile-block" aria-labelledby="profile-merge-label">
        <div class="profile-block-head">
          <span class="settings-row-icon" aria-hidden="true">${iconSparkles()}</span>
          <div>
            <h2 id="profile-merge-label" class="channel-panel-title">一句话更新画像</h2>
            <p class="channel-panel-sub profile-block-sub">描述最近学了什么、希望怎么讲，AI 会自动合并进下方画像</p>
          </div>
        </div>
        <label class="field-label visually-hidden" for="profile-supplement">补充内容</label>
        <textarea
          class="input profile-merge-input"
          id="profile-supplement"
          rows="2"
          placeholder="例如：项目里用上了 channel，希望讲解偏实战"
        ></textarea>
        <div class="profile-block-actions">
          <button type="button" class="btn btn-primary" id="profile-refine-btn">
            <span class="profile-merge-btn-label">AI 合并</span>
          </button>
        </div>
      </section>

      <section class="profile-block profile-block--view" id="profile-view-wrap">
        <div class="profile-view-head">
          <h2 class="channel-panel-title profile-view-title">当前画像</h2>
          <button type="button" class="profile-edit-link" id="profile-edit-btn">手动编辑</button>
        </div>
        <div class="profile-view-body" id="profile-view-body">
          ${renderProfileViewHtml(savedSummary)}
        </div>
      </section>

      <section class="profile-block profile-block--edit" id="profile-edit-panel">
        <div class="profile-view-head">
          <h2 class="channel-panel-title profile-view-title">编辑画像</h2>
          <span class="profile-char-count" id="profile-char-count" aria-live="polite">0 / ${MAX_PROFILE_CHARS}</span>
        </div>
        <label class="field-label visually-hidden" for="profile-summary-edit">编辑画像</label>
        <textarea
          class="input profile-editor-input"
          id="profile-summary-edit"
          maxlength="${MAX_PROFILE_CHARS}"
          rows="8"
          placeholder="你的职业背景、已掌握技能、学习目标…"
        >${escapeHtml(savedSummary)}</textarea>
        <div class="profile-block-actions">
          <button type="button" class="btn btn-ghost btn-sm" id="profile-cancel-btn">取消</button>
          <button type="button" class="btn btn-primary btn-sm" id="profile-save-btn">保存画像</button>
        </div>
      </section>
    </div>

    <div id="profile-settings-msg" class="profile-settings-msg" role="status"></div>
  `

  const sheet = page.querySelector<HTMLElement>('#profile-sheet')!
  const msgEl = page.querySelector<HTMLDivElement>('#profile-settings-msg')!
  const viewBody = page.querySelector<HTMLElement>('#profile-view-body')!
  const summaryEl = page.querySelector<HTMLTextAreaElement>('#profile-summary-edit')!
  const supplementEl = page.querySelector<HTMLTextAreaElement>('#profile-supplement')!
  const counterEl = page.querySelector<HTMLElement>('#profile-char-count')!

  if (summaryEl && counterEl) bindCharCounter(summaryEl, counterEl)

  const updateView = (text: string) => {
    savedSummary = text.trim()
    viewBody.innerHTML = renderProfileViewHtml(savedSummary)
  }

  const setEditing = (editing: boolean) => {
    sheet.classList.toggle('profile-card--editing', editing)
    if (editing && summaryEl) {
      summaryEl.value = savedSummary
      summaryEl.focus()
      summaryEl.dispatchEvent(new Event('input', { bubbles: true }))
    }
  }

  const showMsg = (html: string) => {
    msgEl.innerHTML = html
  }

  const setBtnBusy = (btn: HTMLButtonElement | null, busy: boolean, busyLabel: string, idleLabel: string) => {
    if (!btn) return
    btn.disabled = busy
    const label = btn.querySelector<HTMLElement>('.profile-merge-btn-label')
    if (label) label.textContent = busy ? busyLabel : idleLabel
    else btn.textContent = busy ? busyLabel : idleLabel
  }

  page.querySelector<HTMLButtonElement>('#profile-edit-btn')?.addEventListener('click', () => {
    showMsg('')
    setEditing(true)
  })

  page.querySelector<HTMLButtonElement>('#profile-cancel-btn')?.addEventListener('click', () => {
    if (summaryEl) summaryEl.value = savedSummary
    setEditing(false)
  })

  page.querySelector<HTMLButtonElement>('#profile-save-btn')?.addEventListener('click', () => {
    void (async () => {
      showMsg('')
      const btn = page.querySelector<HTMLButtonElement>('#profile-save-btn')
      if (btn) {
        btn.disabled = true
        btn.textContent = '保存中…'
      }
      try {
        const updated = await updateUserProfile(summaryEl.value.trim())
        syncLocalProfile(updated)
        updateView(updated.profileSummary ?? '')
        setEditing(false)
        showMsg('<div class="alert alert-success">画像已保存</div>')
      } catch (e) {
        showMsg(`<div class="alert alert-error">${escapeHtml(e instanceof ApiError ? e.message : '保存失败')}</div>`)
      } finally {
        if (btn) {
          btn.disabled = false
          btn.textContent = '保存画像'
        }
      }
    })()
  })

  page.querySelector<HTMLButtonElement>('#profile-refine-btn')?.addEventListener('click', () => {
    void (async () => {
      showMsg('')
      const supplement = supplementEl.value.trim()
      if (!supplement) {
        showMsg('<div class="alert alert-error">请先填写要合并的内容</div>')
        supplementEl.focus()
        return
      }
      const btn = page.querySelector<HTMLButtonElement>('#profile-refine-btn')
      setBtnBusy(btn, true, '合并中…', 'AI 合并')
      try {
        const updated = await refineUserProfile(supplement)
        syncLocalProfile(updated)
        updateView(updated.profileSummary ?? '')
        if (summaryEl) {
          summaryEl.value = savedSummary
          summaryEl.dispatchEvent(new Event('input', { bubbles: true }))
        }
        supplementEl.value = ''
        setEditing(false)
        showMsg('<div class="alert alert-success">已合并进画像</div>')
      } catch (e) {
        showMsg(`<div class="alert alert-error">${escapeHtml(e instanceof ApiError ? e.message : '合并失败')}</div>`)
      } finally {
        setBtnBusy(btn, false, '合并中…', 'AI 合并')
      }
    })()
  })
}
