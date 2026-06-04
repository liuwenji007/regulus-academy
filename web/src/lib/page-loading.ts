/** 全页/区块 loading 占位 HTML（与 .page-loading 样式配套） */

export function pageLoadingHtml(title: string, hint?: string): string {
  const hintHtml = hint
    ? `<p class="page-loading-hint">${escapeHtml(hint)}</p>`
    : ''
  return `
    <div class="page-loading" role="status" aria-live="polite">
      <div class="spinner" aria-hidden="true"></div>
      <p>${escapeHtml(title)}</p>
      ${hintHtml}
    </div>
  `
}

function escapeHtml(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}
