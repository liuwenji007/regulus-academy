import {
  listUsers,
  updateUserProfile,
  refineUserProfile,
  migrateUserProfile,
  ApiError,
  type UserProfile,
  type DomainProfileEntry,
} from '../lib/api'
import { iconSparkles } from '../lib/icons'
import { getActiveProfile, setActiveProfile } from '../lib/profile'
import { parseProfileSections, resolveEditableProfileFields } from '../lib/profile-edit-fields'
import { setBreadcrumb, updateSidebar } from '../components/layout'

function escapeHtml(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}

function syncLocalProfile(user: UserProfile): void {
  const active = getActiveProfile()
  if (active?.id === user.id) {
    setActiveProfile({
      id: user.id,
      displayName: user.displayName,
      profileSummary: user.profileSummary,
      profileBackground: user.profileBackground,
      profileGoal: user.profileGoal,
      profilePreference: user.profilePreference,
      domainProfiles: user.domainProfiles,
      onboardedAt: user.onboardedAt,
    })
  }
}

function formatProfileUpdatedAt(iso?: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const now = new Date()
  const diffMs = now.getTime() - d.getTime()
  const dayMs = 86400000
  if (diffMs < dayMs) return '今天更新'
  if (diffMs < dayMs * 2) return '昨天更新'
  if (diffMs < dayMs * 7) return `${Math.floor(diffMs / dayMs)} 天前`
  return d.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
}

function renderDomainProfilesHtml(entries: DomainProfileEntry[] | undefined): string {
  if (!entries?.length) {
    return `
      <div class="profile-domain-empty">
        <p class="profile-domain-empty-title">暂无按课摘要</p>
        <p class="profile-domain-empty-hint">完成节点并点亮后，系统会把本课掌握情况自动写在这里，供该课 Coach 讲解时参考。</p>
      </div>
    `
  }
  return `
    <ul class="profile-domain-list" role="list">
      ${entries
        .map((d) => {
          const name = escapeHtml(d.domainName || '未命名课程')
          const updated = formatProfileUpdatedAt(d.updatedAt)
          const treeHref = d.domainId ? `#/tree/${encodeURIComponent(d.domainId)}` : ''
          const title = treeHref
            ? `<a href="${treeHref}" class="profile-domain-link">${name}</a>`
            : `<span class="profile-domain-link">${name}</span>`
          return `
        <li class="profile-domain-card">
          <div class="profile-domain-card-head">
            <p class="profile-domain-name">${title}</p>
            ${updated ? `<time class="profile-domain-updated" datetime="${escapeHtml(d.updatedAt ?? '')}">${updated}</time>` : ''}
          </div>
          <p class="profile-domain-summary-label">掌握摘要</p>
          <p class="profile-domain-summary">${escapeHtml(d.summary)}</p>
        </li>
      `
        })
        .join('')}
    </ul>
  `
}

function renderStructuredProfileHtml(user: UserProfile): string {
  const bg = (user.profileBackground ?? '').trim()
  const goal = (user.profileGoal ?? '').trim()
  const pref = (user.profilePreference ?? '').trim()
  if (bg || goal || pref) {
    const parts: string[] = []
    if (bg) parts.push(`<div class="profile-section"><p class="profile-section-label">背景</p><p class="profile-section-text">${escapeHtml(bg)}</p></div>`)
    if (pref) parts.push(`<div class="profile-section"><p class="profile-section-label">偏好</p><p class="profile-section-text">${escapeHtml(pref)}</p></div>`)
    if (goal) parts.push(`<div class="profile-section"><p class="profile-section-label">目标</p><p class="profile-section-text">${escapeHtml(goal)}</p></div>`)
    return `<div class="profile-sections">${parts.join('')}</div>`
  }
  return renderProfileViewHtml(user.profileSummary ?? '')
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

  let { background: savedBackground, goal: savedGoal, preference: savedPreference } =
    resolveEditableProfileFields(user)
  let savedDomains = user.domainProfiles ?? []
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
          <h2 class="channel-panel-title profile-view-title">全局画像</h2>
          <div class="profile-view-actions">
            <button type="button" class="btn btn-ghost btn-sm" id="profile-migrate-btn">整理画像</button>
            <button type="button" class="profile-edit-link" id="profile-edit-btn">手动编辑</button>
          </div>
        </div>
        <div class="profile-view-body" id="profile-view-body">
          ${renderStructuredProfileHtml(user)}
        </div>
      </section>

      <section class="profile-block profile-block--view profile-block--domains" id="profile-domain-wrap">
        <div class="profile-view-head profile-view-head--stacked">
          <div>
            <h2 class="channel-panel-title profile-view-title">按课进展</h2>
            <p class="profile-block-sub profile-domain-hint">各课掌握摘要独立沉淀，点亮节点后自动更新</p>
          </div>
          ${savedDomains.length > 0 ? `<span class="profile-domain-count" id="profile-domain-count">${savedDomains.length} 门课</span>` : '<span class="profile-domain-count" id="profile-domain-count" hidden></span>'}
        </div>
        <div class="profile-view-body profile-view-body--domains" id="profile-domain-body">
          ${renderDomainProfilesHtml(savedDomains)}
        </div>
      </section>

      <section class="profile-block profile-block--edit" id="profile-edit-panel">
        <div class="profile-view-head">
          <h2 class="channel-panel-title profile-view-title">编辑全局画像</h2>
        </div>
        <label class="field-label" for="profile-background-edit">背景</label>
        <textarea class="input profile-editor-input" id="profile-background-edit" rows="3" placeholder="职业、技术栈…">${escapeHtml(savedBackground)}</textarea>
        <label class="field-label" for="profile-goal-edit">目标</label>
        <textarea class="input profile-editor-input" id="profile-goal-edit" rows="2" placeholder="跨课学习目标…">${escapeHtml(savedGoal)}</textarea>
        <label class="field-label" for="profile-preference-edit">讲解偏好（可选）</label>
        <textarea class="input profile-editor-input" id="profile-preference-edit" rows="2" placeholder="偏实战、先结构后细节…">${escapeHtml(savedPreference)}</textarea>
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
  const domainBody = page.querySelector<HTMLElement>('#profile-domain-body')!
  const domainCountEl = page.querySelector<HTMLElement>('#profile-domain-count')!
  const bgEl = page.querySelector<HTMLTextAreaElement>('#profile-background-edit')!
  const goalEl = page.querySelector<HTMLTextAreaElement>('#profile-goal-edit')!
  const prefEl = page.querySelector<HTMLTextAreaElement>('#profile-preference-edit')!
  const supplementEl = page.querySelector<HTMLTextAreaElement>('#profile-supplement')!

  const updateView = (u: UserProfile) => {
    const resolved = resolveEditableProfileFields(u)
    savedBackground = resolved.background
    savedGoal = resolved.goal
    savedPreference = resolved.preference
    savedDomains = u.domainProfiles ?? []
    viewBody.innerHTML = renderStructuredProfileHtml(u)
    domainBody.innerHTML = renderDomainProfilesHtml(savedDomains)
    if (domainCountEl) {
      if (savedDomains.length > 0) {
        domainCountEl.hidden = false
        domainCountEl.textContent = `${savedDomains.length} 门课`
      } else {
        domainCountEl.hidden = true
        domainCountEl.textContent = ''
      }
    }
  }

  const setEditing = (editing: boolean) => {
    sheet.classList.toggle('profile-card--editing', editing)
    if (editing) {
      if (bgEl) bgEl.value = savedBackground
      if (goalEl) goalEl.value = savedGoal
      if (prefEl) prefEl.value = savedPreference
      bgEl?.focus()
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
    setEditing(false)
  })

  page.querySelector<HTMLButtonElement>('#profile-migrate-btn')?.addEventListener('click', () => {
    void (async () => {
      showMsg('')
      const btn = page.querySelector<HTMLButtonElement>('#profile-migrate-btn')
      if (btn) {
        btn.disabled = true
        btn.textContent = '整理中…'
      }
      try {
        const updated = await migrateUserProfile()
        syncLocalProfile(updated)
        updateView(updated)
        showMsg('<div class="alert alert-success">画像已按课程整理</div>')
      } catch (e) {
        showMsg(`<div class="alert alert-error">${escapeHtml(e instanceof ApiError ? e.message : '整理失败')}</div>`)
      } finally {
        if (btn) {
          btn.disabled = false
          btn.textContent = '整理画像'
        }
      }
    })()
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
        const updated = await updateUserProfile({
          profileBackground: bgEl?.value.trim() ?? '',
          profileGoal: goalEl?.value.trim() ?? '',
          profilePreference: prefEl?.value.trim() ?? '',
        })
        syncLocalProfile(updated)
        updateView(updated)
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
        updateView(updated)
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
