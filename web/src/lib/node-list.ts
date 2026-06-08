import type { TreeNode, UserProgress } from './api'
import { unmetPrerequisiteTitles } from './tree-normalize'

export interface RenderNodeItemOpts {
  node: TreeNode
  layerKey: string
  progressMap: Map<string, UserProgress>
  focusSet: Set<string>
  titleMap: Map<string, string>
}

export function renderNodeItem(opts: RenderNodeItemOpts): string {
  const { node, layerKey, progressMap, focusSet, titleMap } = opts
  const st = progressMap.get(node.key)
  const statusClass = st?.status ?? 'pending'
  const resumeTag =
    statusClass === 'completed'
      ? '<span class="node-resume-tag node-resume-tag--review">复习</span>'
      : statusClass === 'in_progress'
        ? '<span class="node-resume-tag">继续</span>'
        : ''
  const isFocus = focusSet.has(node.key)
  const focusTag = isFocus ? '<span class="node-focus-tag">当前聚焦</span>' : ''
  const unmetPrereqs = unmetPrerequisiteTitles(node, progressMap, titleMap)
  const prereqTag =
    unmetPrereqs.length > 0
      ? `<span class="node-prereq-tag" title="建议先完成：${escapeHtml(unmetPrereqs.join('、'))}">建议先学 ${escapeHtml(unmetPrereqs.join('、'))}</span>`
      : ''
  const prereqClass = unmetPrereqs.length > 0 ? ' node-item--prereq' : ''
  return `
    <li class="node-item${prereqClass}${isFocus ? ' node-item--focus' : ''}" data-node="${escapeHtmlAttr(node.key)}" data-layer="${escapeHtmlAttr(layerKey)}" tabindex="0" role="button">
      <span class="node-status ${statusClass}" aria-hidden="true"></span>
      <span class="node-title-wrap">
        <span class="node-title">${escapeHtml(node.title)}</span>
        ${prereqTag}
      </span>
      ${focusTag}
      ${resumeTag}
    </li>
  `
}

export function bindNodeList(
  container: HTMLElement,
  onNodeClick: (nodeKey: string, layerKey: string) => void,
  signal?: AbortSignal
): void {
  const opts = signal ? { signal } : undefined
  container.querySelectorAll<HTMLElement>('.node-item').forEach((el) => {
    const nodeKey = el.dataset.node!
    const layer = el.dataset.layer!
    const open = () => onNodeClick(nodeKey, layer)
    el.addEventListener('click', open, opts)
    el.addEventListener(
      'keydown',
      (e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          open()
        }
      },
      opts
    )
  })
}

function escapeHtml(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}

function escapeHtmlAttr(s: string): string {
  return escapeHtml(s).replace(/"/g, '&quot;')
}
