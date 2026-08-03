import type { CourseDerivation, TreeNode, UserProgress } from './api'
import { unmetPrerequisiteTitles } from './tree-normalize'

export interface NodeAssetMaps {
  notes: Map<string, string>
  mistakes: Map<string, string[]>
}

export interface RenderNodeItemOpts {
  node: TreeNode
  layerKey: string
  layerLabel?: string
  progressMap: Map<string, UserProgress>
  focusSet: Set<string>
  titleMap: Map<string, string>
  /** 有笔记或踩坑时显示行内展开按钮 */
  assets?: NodeAssetMaps
}

export function renderNodeItem(opts: RenderNodeItemOpts): string {
  const { node, layerKey, layerLabel, progressMap, focusSet, titleMap, assets } = opts
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
  const note = assets?.notes.get(node.key)?.trim() ?? ''
  const concepts = assets?.mistakes.get(node.key) ?? []
  const hasAssets = note.length > 0 || concepts.length > 0
  const assetBtn = hasAssets
    ? `<button type="button" class="node-asset-toggle" data-node-asset="${escapeHtmlAttr(node.key)}" aria-expanded="false" title="查看学习笔记与踩坑">笔记</button>`
    : ''
  const assetPanel = hasAssets
    ? `<div class="node-asset-panel" data-node-asset-panel="${escapeHtmlAttr(node.key)}" hidden></div>`
    : ''
  return `
    <li class="node-item-wrap${prereqClass}${isFocus ? ' node-item-wrap--focus' : ''}">
      <div class="node-item${prereqClass}${isFocus ? ' node-item--focus' : ''}" data-node="${escapeHtmlAttr(node.key)}" data-layer="${escapeHtmlAttr(layerKey)}" tabindex="0" role="button">
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
        ${assetBtn}
      </div>
      ${assetPanel}
    </li>
  `
}

export function renderDerivationJump(d: CourseDerivation): string {
  return `
    <li class="node-derivation" role="presentation">
      <a class="node-derivation-link" href="#/tree/${escapeHtmlAttr(d.childDomainId)}">
        <span class="node-derivation-icon" aria-hidden="true">↗</span>
        <span class="node-derivation-label">${escapeHtml(d.label)}</span>
        <span class="node-derivation-hint">专题衍生课程</span>
      </a>
    </li>
  `
}

export function renderLayerNodeList(
  layerKey: string,
  nodes: TreeNode[],
  opts: Omit<RenderNodeItemOpts, 'node' | 'layerKey'>,
  derivationsByAfterKey: Map<string, CourseDerivation[]>
): string {
  return nodes
    .map((node) => {
      const item = renderNodeItem({ ...opts, node, layerKey })
      const derivs = derivationsByAfterKey.get(node.key)
      if (!derivs?.length) return item
      return item + derivs.map((d) => renderDerivationJump(d)).join('')
    })
    .join('')
}

export function bindNodeList(
  container: HTMLElement,
  onNodeClick: (nodeKey: string, layerKey: string) => void,
  signal?: AbortSignal
): void {
  bindClickableNodes(container, '.node-item', onNodeClick, signal)
}

/** 绑定笔记/踩坑行内展开；点击按钮不触发开练 */
export function bindNodeAssetPanels(
  container: HTMLElement,
  assets: NodeAssetMaps,
  renderPanel: (nodeKey: string, note: string, concepts: string[]) => string,
  signal?: AbortSignal
): void {
  const opts = signal ? { signal } : undefined
  container.querySelectorAll<HTMLButtonElement>('.node-asset-toggle').forEach((btn) => {
    btn.addEventListener(
      'click',
      (e) => {
        e.preventDefault()
        e.stopPropagation()
        const nodeKey = btn.dataset.nodeAsset
        if (!nodeKey) return
        const panel = container.querySelector<HTMLElement>(
          `[data-node-asset-panel="${CSS.escape(nodeKey)}"]`
        )
        if (!panel) return
        const open = panel.hasAttribute('hidden')
        // 先收起其他面板
        container.querySelectorAll<HTMLElement>('.node-asset-panel').forEach((p) => {
          p.setAttribute('hidden', '')
        })
        container.querySelectorAll<HTMLButtonElement>('.node-asset-toggle').forEach((b) => {
          b.setAttribute('aria-expanded', 'false')
        })
        if (!open) return
        const note = assets.notes.get(nodeKey)?.trim() ?? ''
        const concepts = assets.mistakes.get(nodeKey) ?? []
        panel.innerHTML = renderPanel(nodeKey, note, concepts)
        panel.removeAttribute('hidden')
        btn.setAttribute('aria-expanded', 'true')
      },
      opts
    )
  })
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
