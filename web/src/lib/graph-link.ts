export function graphNavLink(opts?: { label?: string; ariaLabel?: string }): string {
  const label = opts?.label ?? '知识图谱'
  const ariaLabel = opts?.ariaLabel ?? '查看知识图谱'
  return `
    <a class="tree-graph-link" href="#/graph" aria-label="${escapeAttr(ariaLabel)}">
      <svg class="tree-graph-link-icon" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
        <circle cx="3.5" cy="8" r="2" stroke="currentColor" stroke-width="1.25"/>
        <circle cx="12.5" cy="3.5" r="2" stroke="currentColor" stroke-width="1.25"/>
        <circle cx="12.5" cy="12.5" r="2" stroke="currentColor" stroke-width="1.25"/>
        <path d="M5.4 7.2l5-2.8M5.4 8.8l5 2.2" stroke="currentColor" stroke-width="1.25" stroke-linecap="round"/>
      </svg>
      ${escapeHtml(label)}
    </a>
  `
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

function escapeAttr(s: string): string {
  return escapeHtml(s)
}
