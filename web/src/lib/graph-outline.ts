import type { KnowledgeTree } from './api'
import type { MultiDomainGraphEntry } from './knowledge-graph'
import { groupDomainsIntoConstellations, type ConstellationGroup } from './graph-constellation'
import {
  nodeLayerKeyMap,
  nodeTitleMap,
  resolveGraphModules,
} from './tree-normalize'
import { bindOutlineNodeChips, renderOutlineNodeChip } from './node-list'

export interface GraphOutlineSummary {
  domainCount: number
  totalNodes: number
  completedNodes: number
  nextDomainId: string
  nextDomainName: string
  nextNodeKey: string
  nextNodeTitle: string
  nextLayerKey: string
}

export function computeGraphOutlineSummary(entries: MultiDomainGraphEntry[]): GraphOutlineSummary {
  let totalNodes = 0
  let completedNodes = 0

  for (const entry of entries) {
    for (const layer of entry.tree.layers) {
      for (const node of layer.nodes) {
        totalNodes++
        if (entry.progressMap.get(node.key)?.status === 'completed') {
          completedNodes++
        }
      }
    }
  }

  let nextDomainId = ''
  let nextDomainName = ''
  let nextNodeKey = ''
  let nextNodeTitle = ''
  let nextLayerKey = ''

  outer: for (const entry of entries) {
    for (const layer of entry.tree.layers) {
      for (const node of layer.nodes) {
        if (entry.progressMap.get(node.key)?.status !== 'completed') {
          nextDomainId = entry.domainId
          nextDomainName = entry.tree.domainName
          nextNodeKey = node.key
          nextNodeTitle = node.title
          nextLayerKey = layer.key
          break outer
        }
      }
    }
  }

  return {
    domainCount: entries.length,
    totalNodes,
    completedNodes,
    nextDomainId,
    nextDomainName,
    nextNodeKey,
    nextNodeTitle,
    nextLayerKey,
  }
}

function nodeLayerLabelMap(tree: KnowledgeTree): Map<string, string> {
  const map = new Map<string, string>()
  for (const layer of tree.layers) {
    const label = layer.label || layer.key
    for (const node of layer.nodes) {
      map.set(node.key, label)
    }
  }
  return map
}

function domainNodeCount(entry: MultiDomainGraphEntry): number {
  return entry.tree.layers.reduce((n, l) => n + l.nodes.length, 0)
}

/** 进行中 → 有未完成 → 已圆满 */
function domainSortRank(entry: MultiDomainGraphEntry): number {
  const hasInProgress = [...entry.progressMap.values()].some((p) => p.status === 'in_progress')
  if (hasInProgress) return 0
  const total = domainNodeCount(entry)
  const completed = [...entry.progressMap.values()].filter((p) => p.status === 'completed').length
  if (completed < total) return 1
  return 2
}

function sortEntries(entries: MultiDomainGraphEntry[]): MultiDomainGraphEntry[] {
  return [...entries].sort((a, b) => domainSortRank(a) - domainSortRank(b))
}

function sortConstellationGroups(groups: ConstellationGroup[], nextDomainId: string): ConstellationGroup[] {
  if (!nextDomainId) return groups
  return [...groups].sort((a, b) => {
    const aHas = a.domainIds.includes(nextDomainId) ? 0 : 1
    const bHas = b.domainIds.includes(nextDomainId) ? 0 : 1
    return aHas - bHas
  })
}

function constellationSectionId(key: string): string {
  return `graph-constellation-${key.replace(/[^a-z0-9_-]/gi, '-')}`
}

export function constellationSectionTitle(group: ConstellationGroup, totalGroups: number): string {
  if (totalGroups <= 1) return ''
  if (group.domainIds.length > 1) return `${group.label} · ${group.domainIds.length} 门`
  return group.label
}

function buildConstellationLayout(entries: MultiDomainGraphEntry[], nextDomainId: string) {
  const entryById = new Map(entries.map((e) => [e.domainId, e]))
  const inputs = entries.map((e) => ({
    domainId: e.domainId,
    name: e.tree.domainName,
    slug: e.slug,
    nodeCount: domainNodeCount(e),
  }))
  const groups = sortConstellationGroups(groupDomainsIntoConstellations(inputs), nextDomainId)
  return { entryById, groups }
}

function renderClusterSection(
  group: ConstellationGroup,
  groupEntries: MultiDomainGraphEntry[],
  filterDomainId: string
): string {
  const cardsHtml = groupEntries.map((e) => renderDomainCard(e, filterDomainId, group.key)).join('')
  if (!cardsHtml) return ''

  const sectionId = constellationSectionId(group.key)
  const title = constellationSectionTitle(group, 2)
  const relatedNames = groupEntries.map((e) => e.tree.domainName).join(' · ')

  return `
    <section class="graph-outline-constellation" data-constellation-key="${escapeHtmlAttr(group.key)}">
      <header class="graph-outline-constellation-head">
        <h2 class="graph-outline-constellation-title" id="${escapeHtmlAttr(sectionId)}">${escapeHtml(title)}</h2>
        <p class="graph-outline-constellation-related">${escapeHtml(relatedNames)}</p>
      </header>
      <div class="graph-outline-cards">${cardsHtml}</div>
    </section>
  `
}

function renderDomainCard(
  entry: MultiDomainGraphEntry,
  filterDomainId: string,
  constellationKey?: string
): string {
  if (filterDomainId && entry.domainId !== filterDomainId) return ''

  const { modules } = resolveGraphModules(entry.tree)
  const layerMap = nodeLayerKeyMap(entry.tree)
  const layerLabelMap = nodeLayerLabelMap(entry.tree)
  const titleMap = nodeTitleMap(entry.tree)
  const total = domainNodeCount(entry)
  const completed = [...entry.progressMap.values()].filter((p) => p.status === 'completed').length
  const pct = total > 0 ? Math.round((completed / total) * 100) : 0

  const modulesHtml = modules
    .map((mod) => {
      const modCompleted = mod.nodes.filter((k) => entry.progressMap.get(k)?.status === 'completed').length
      const modPct = mod.nodes.length > 0 ? Math.round((modCompleted / mod.nodes.length) * 100) : 0
      const nodesHtml = mod.nodes
        .map((nodeKey) => {
          const layer = layerMap.get(nodeKey)
          if (!layer) return ''
          const node = entry.tree.layers.flatMap((l) => l.nodes).find((n) => n.key === nodeKey)
          if (!node) return ''
          return renderOutlineNodeChip({
            node,
            layerKey: layer,
            layerLabel: layerLabelMap.get(nodeKey),
            progressMap: entry.progressMap,
            focusSet: entry.focusKeys,
            titleMap,
          })
        })
        .filter(Boolean)
        .join('')

      return `
        <section class="graph-outline-module">
          <header class="graph-outline-module-head">
            <h3 class="graph-outline-module-label">${escapeHtml(mod.label)}</h3>
            <div class="graph-outline-module-head-meta">
              <span class="graph-outline-module-meta">${modCompleted} / ${mod.nodes.length}</span>
              <div class="graph-outline-module-progress" aria-hidden="true">
                <i class="graph-outline-module-progress-fill" style="width:${modPct}%"></i>
              </div>
            </div>
          </header>
          <div class="graph-outline-node-chips" role="list">${nodesHtml}</div>
        </section>
      `
    })
    .join('')

  const constellationAttr = constellationKey
    ? ` data-constellation-key="${escapeHtmlAttr(constellationKey)}"`
    : ''

  return `
    <article class="card graph-domain-card graph-domain-card--outline" data-domain-id="${escapeHtmlAttr(entry.domainId)}"${constellationAttr}>
      <div class="graph-domain-card-progress-line" style="width:${pct}%" aria-hidden="true"></div>
      <header class="graph-domain-card-header">
        <h2 class="graph-domain-card-title">
          ${escapeHtml(entry.tree.domainName)}
          <a href="#/tree/${escapeHtmlAttr(entry.domainId)}" class="graph-domain-card-link">详情</a>
        </h2>
      </header>
      <div class="graph-outline-modules">${modulesHtml}</div>
    </article>
  `
}

function renderConstellationSections(
  entries: MultiDomainGraphEntry[],
  filterDomainId: string,
  nextDomainId: string
): string {
  const { entryById, groups } = buildConstellationLayout(entries, nextDomainId)
  if (!groups.length) return '<p class="graph-outline-empty">没有匹配的领域</p>'

  const groupEntriesFor = (group: ConstellationGroup) =>
    sortEntries(
      group.domainIds.map((id) => entryById.get(id)).filter((e): e is MultiDomainGraphEntry => Boolean(e))
    )

  if (groups.length === 1) {
    const group = groups[0]!
    const groupEntries = groupEntriesFor(group)
    if (group.domainIds.length > 1) {
      return renderClusterSection(group, groupEntries, filterDomainId)
    }
    const cardsHtml = groupEntries.map((e) => renderDomainCard(e, filterDomainId)).join('')
    return cardsHtml
      ? `<div class="graph-outline-cards">${cardsHtml}</div>`
      : '<p class="graph-outline-empty">没有匹配的领域</p>'
  }

  const parts: string[] = []
  let pendingSingles: MultiDomainGraphEntry[] = []

  const flushSingles = () => {
    if (!pendingSingles.length) return
    const cardsHtml = pendingSingles.map((e) => renderDomainCard(e, filterDomainId)).join('')
    if (cardsHtml) {
      parts.push(`<div class="graph-outline-cards graph-outline-cards--standalone">${cardsHtml}</div>`)
    }
    pendingSingles = []
  }

  for (const group of groups) {
    const groupEntries = groupEntriesFor(group)
    if (group.domainIds.length > 1) {
      flushSingles()
      const section = renderClusterSection(group, groupEntries, filterDomainId)
      if (section) parts.push(section)
    } else {
      pendingSingles.push(...groupEntries)
    }
  }
  flushSingles()

  return parts.join('') || '<p class="graph-outline-empty">没有匹配的领域</p>'
}

export function renderGraphOutlineHtml(
  entries: MultiDomainGraphEntry[],
  filterDomainId: string
): string {
  const visible = filterDomainId ? entries.filter((e) => e.domainId === filterDomainId) : entries
  const summarySource = visible.length ? visible : entries
  const summary = computeGraphOutlineSummary(summarySource)

  const nextHint =
    summary.nextNodeTitle && summary.nextDomainName
      ? `<span class="graph-outline-next">
          <span class="graph-outline-next-label">推荐下一步</span>
          <button
            type="button"
            class="graph-outline-next-btn"
            data-domain-id="${escapeHtmlAttr(summary.nextDomainId)}"
            data-node-key="${escapeHtmlAttr(summary.nextNodeKey)}"
            data-layer-key="${escapeHtmlAttr(summary.nextLayerKey)}"
          >${escapeHtml(summary.nextDomainName)} · ${escapeHtml(summary.nextNodeTitle)}</button>
        </span>`
      : summary.completedNodes === summary.totalNodes && summary.totalNodes > 0
        ? '<span class="graph-outline-next graph-outline-next--done">全部节点已学完</span>'
        : ''

  const sectionsHtml = renderConstellationSections(entries, filterDomainId, summary.nextDomainId)

  return `
    <div class="graph-outline-summary card">
      <span>${summary.domainCount} 门课</span>
      <span class="graph-outline-summary-sep">·</span>
      <span>已完成 ${summary.completedNodes} / ${summary.totalNodes} 节点</span>
      ${nextHint}
    </div>
    <div class="graph-outline-constellations">${sectionsHtml}</div>
  `
}

export function bindGraphOutline(
  container: HTMLElement,
  onNodeClick: (domainId: string, nodeKey: string, layerKey: string) => void,
  signal?: AbortSignal
): void {
  const opts = signal ? { signal } : undefined
  container.querySelectorAll<HTMLElement>('.graph-domain-card').forEach((card) => {
    const domainId = card.dataset.domainId!
    bindOutlineNodeChips(
      card,
      (nodeKey, layerKey) => onNodeClick(domainId, nodeKey, layerKey),
      signal
    )
  })

  container.querySelectorAll<HTMLButtonElement>('.graph-outline-next-btn').forEach((btn) => {
    const open = () => {
      const domainId = btn.dataset.domainId
      const nodeKey = btn.dataset.nodeKey
      const layerKey = btn.dataset.layerKey
      if (!domainId || !nodeKey || !layerKey) return
      onNodeClick(domainId, nodeKey, layerKey)
    }
    btn.addEventListener('click', open, opts)
    btn.addEventListener(
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

  const firstFocus = container.querySelector<HTMLElement>('.graph-outline-node-chip--focus')
  if (firstFocus) {
    firstFocus.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
  }
}

function escapeHtml(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}

function escapeHtmlAttr(s: string): string {
  return escapeHtml(s).replace(/"/g, '&quot;')
}
