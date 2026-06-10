import type { TreeNode, UserProgress } from './api'
import { unmetPrerequisiteTitles } from './tree-normalize'

export interface RenderNodeItemOpts {
  node: TreeNode
  layerKey: string
  layerLabel?: string
  progressMap: Map<string, UserProgress>
  focusSet: Set<string>
  titleMap: Map<string, string>
}

export function renderNodeItem(opts: RenderNodeItemOpts): string {
  const { node, layerKey, layerLabel, progressMap, focusSet, titleMap } = opts
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
  const layerTag = layerLabel
    ? `<span class="node-layer-tag">${escapeHtml(layerLabel)}</span>`
    : ''
  return `
    <li class="node-item${prereqClass}${isFocus ? ' node-item--focus' : ''}" data-node="${escapeHtmlAttr(node.key)}" data-layer="${escapeHtmlAttr(layerKey)}" tabindex="0" role="button">
      <span class="node-status ${statusClass}" aria-hidden="true"></span>
      <span class="node-title-wrap">
        <span class="node-title-row">
          <span class="node-title">${escapeHtml(node.title)}</span>
          ${layerTag}
        </span>
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
  bindClickableNodes(container, '.node-item', onNodeClick, signal)
}

/** 图谱目录模式：节点 chip 云（紧凑流式布局） */
export function renderOutlineNodeChip(opts: RenderNodeItemOpts): string {
  const { node, layerKey, layerLabel, progressMap, focusSet, titleMap } = opts
  const st = progressMap.get(node.key)
  const status = st?.status ?? 'pending'
  const isFocus = focusSet.has(node.key)
  const unmetPrereqs = unmetPrerequisiteTitles(node, progressMap, titleMap)
  const layerVariant = outlineLayerVariant(layerLabel ?? '', layerKey)
  const prereqTitle =
    unmetPrereqs.length > 0 ? `建议先完成：${unmetPrereqs.join('、')}` : ''
  const statusHint =
    status === 'completed' ? '复习' : status === 'in_progress' ? '继续' : layerLabel ?? ''
  const ariaLabel = [node.title, layerLabel, statusHint, prereqTitle].filter(Boolean).join(' · ')

  return `
    <button
      type="button"
      class="graph-outline-node-chip is-${status}${isFocus ? ' graph-outline-node-chip--focus' : ''}${unmetPrereqs.length > 0 ? ' graph-outline-node-chip--prereq' : ''} ${layerVariant}"
      data-node="${escapeHtmlAttr(node.key)}"
      data-layer="${escapeHtmlAttr(layerKey)}"
      title="${escapeHtmlAttr(prereqTitle || ariaLabel)}"
      aria-label="${escapeHtmlAttr(ariaLabel)}"
    >
      <span class="graph-outline-node-chip-dot" aria-hidden="true"></span>
      <span class="graph-outline-node-chip-label">${escapeHtml(node.title)}</span>
    </button>
  `
}

export function bindOutlineNodeChips(
  container: HTMLElement,
  onNodeClick: (nodeKey: string, layerKey: string) => void,
  signal?: AbortSignal
): void {
  bindClickableNodes(container, '.graph-outline-node-chip', onNodeClick, signal)
}

function outlineLayerVariant(layerLabel: string, layerKey: string): string {
  const text = `${layerLabel} ${layerKey}`.toLowerCase()
  if (text.includes('精通') || text.includes('advanced') || text.includes('master')) {
    return 'graph-outline-node-chip--layer-advanced'
  }
  if (text.includes('熟悉') || text.includes('intermediate') || text.includes('familiar')) {
    return 'graph-outline-node-chip--layer-familiar'
  }
  return 'graph-outline-node-chip--layer-intro'
}

function bindClickableNodes(
  container: HTMLElement,
  selector: string,
  onNodeClick: (nodeKey: string, layerKey: string) => void,
  signal?: AbortSignal
): void {
  const opts = signal ? { signal } : undefined
  container.querySelectorAll<HTMLElement>(selector).forEach((el) => {
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
