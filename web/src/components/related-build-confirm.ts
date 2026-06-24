import type { DomainSummary } from '../lib/api'

export type RelatedBuildChoice = 'merge' | 'separate' | 'cancel'

export interface RelatedBuildConfirmOptions {
  message?: string
  relation?: string
  existingDomain: DomainSummary
  newCourseName: string
}

export function showRelatedBuildConfirm(
  options: RelatedBuildConfirmOptions
): Promise<RelatedBuildChoice> {
  const { message, relation, existingDomain, newCourseName } = options
  const existingName = existingDomain.name?.trim() || '现有课程'
  const rootName = newCourseName.trim() || '根课程'
  const progress =
    existingDomain.nodeTotal > 0
      ? `${existingDomain.completed ?? 0} / ${existingDomain.nodeTotal} 节点已学`
      : ''

  const isSameSlug = relation === 'existing_same_slug'
  const eyebrow = isSameSlug ? '已有课程' : '课程关联'
  const title = isSameSlug ? '发现同名课程' : '发现相关课程'
  const defaultMessage = isSameSlug
    ? `你已有一门「${existingName}」课程。是否根据你的学习画像重新规划路径？`
    : `你已在学习「${existingName}」，它与「${rootName}」属于同一主题。请选择如何处理。`

  const mergeTitle = isSameSlug
    ? `按画像更新「${existingName}」`
    : `合并到「${rootName}」`
  const mergeDesc = isSameSlug
    ? '根据你的背景与目标裁剪学习路径，保留已学进度'
    : '生成完整学习路径，迁移已学进度，并移除独立子课'

  const separateTitle = isSameSlug
    ? `继续现有「${existingName}」`
    : `单独创建「${rootName}」`
  const separateDesc = isSameSlug
    ? '不修改当前课程，直接进入已有学习路径'
    : `保留「${existingName}」，另建一门独立根课`

  return new Promise((resolve) => {
    const overlay = document.createElement('div')
    overlay.className = 'profile-overlay'
    overlay.innerHTML = `
      <div class="profile-modal card related-build-modal" role="alertdialog" aria-labelledby="related-build-title" aria-describedby="related-build-desc">
        <div class="related-build-header">
          <span class="related-build-eyebrow">${escapeHtml(eyebrow)}</span>
          <h3 id="related-build-title" class="profile-modal-title">${escapeHtml(title)}</h3>
          <p id="related-build-desc" class="profile-modal-sub related-build-message">
            ${escapeHtml(message ?? defaultMessage)}
          </p>
        </div>

        <div class="related-build-existing card">
          <span class="related-build-existing-label">当前在学</span>
          <strong class="related-build-existing-name">${escapeHtml(existingName)}</strong>
          ${progress ? `<span class="related-build-existing-meta">${escapeHtml(progress)}</span>` : ''}
        </div>

        <div class="related-build-options" role="group" aria-label="建课方式">
          <button type="button" class="related-build-option related-build-option--merge" data-choice="merge">
            <span class="related-build-option-badge">推荐</span>
            <span class="related-build-option-title">${escapeHtml(mergeTitle)}</span>
            <span class="related-build-option-desc">${escapeHtml(mergeDesc)}</span>
          </button>
          <button type="button" class="related-build-option related-build-option--separate" data-choice="separate">
            <span class="related-build-option-title">${escapeHtml(separateTitle)}</span>
            <span class="related-build-option-desc">${escapeHtml(separateDesc)}</span>
          </button>
        </div>

        <div class="related-build-footer">
          <button type="button" class="btn btn-ghost btn-sm" id="related-build-cancel">取消建课</button>
        </div>
      </div>
    `
    document.body.appendChild(overlay)

    const close = (choice: RelatedBuildChoice) => {
      overlay.remove()
      document.removeEventListener('keydown', onKeydown)
      resolve(choice)
    }

    const onKeydown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') close('cancel')
    }
    document.addEventListener('keydown', onKeydown)

    overlay.addEventListener('click', (e) => {
      if (e.target === overlay) close('cancel')
    })

    overlay.querySelectorAll<HTMLButtonElement>('[data-choice]').forEach((btn) => {
      btn.addEventListener('click', () => {
        const choice = btn.dataset.choice as RelatedBuildChoice
        close(choice)
      })
    })

    overlay.querySelector<HTMLButtonElement>('#related-build-cancel')!.addEventListener('click', () => {
      close('cancel')
    })

    overlay.querySelector<HTMLButtonElement>('[data-choice="merge"]')?.focus()
  })
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}
