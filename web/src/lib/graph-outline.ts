import type { MultiDomainGraphEntry } from './knowledge-graph'
import {
  nodeLayerKeyMap,
  nodeTitleMap,
  resolveGraphModules,
  type ResolvedGraphModule,
} from './tree-normalize'
import { groupDomainsIntoConstellations } from './graph-constellation'
import { bindNodeList, renderNodeItem } from './node-list'

export interface GraphOutlineSummary {
  domainCount: number
  totalNodes: number
  completedNodes: number
  nextDomainName: string
  nextNodeTitle: string
}

export function computeGraphOutlineSummary(entries: MultiDomainGraphEntry[]): GraphOutlineSummary {
  let totalNodes = 0
  let completedNodes = 0
  let nextDomainName = ''
  let nextNodeTitle = ''

  outer: for (const entry of entries) {
    for (const layer of entry.tree.layers) {
      for (const node of layer.nodes) {
        totalNodes++
        const st = entry.progressMap.get(node.key)
        if (st?.status === 'completed') {
          completedNodes++
        } else if (!nextNodeTitle) {
          nextDomainName = entry.tree.domainName
          nextNodeTitle = node.title
          break outer
        }
      }
    }
  }

  return {
    domainCount: entries.length,
    totalNodes,
    completedNodes,
    nextDomainName,
    nextNodeTitle,
  }
}

function moduleShouldExpand(
  mod: ResolvedGraphModule,
  progressMap: Map<string, { status?: string }>,
  focusKeys: Set<string>
): boolean {
  for (const nodeKey of mod.nodes) {
    if (focusKeys.has(nodeKey)) return true
    if (progressMap.get(nodeKey)?.status === 'in_progress') return true
  }
  return false
}

function pickDefaultExpandedModule(
  modules: ResolvedGraphModule[],
  progressMap: Map<string, { status?: string }>,
  focusKeys: Set<string>
): string | null {
  for (const mod of modules) {
    if (moduleShouldExpand(mod, progressMap, focusKeys)) return mod.key
  }
  for (const mod of modules) {
    const hasIncomplete = mod.nodes.some((k) => progressMap.get(k)?.status !== 'completed')
    if (hasIncomplete) return mod.key
  }
  return modules[0]?.key ?? null
}

function relatedDomainNames(
  entry: MultiDomainGraphEntry,
  allEntries: MultiDomainGraphEntry[]
): string[] {
  const inputs = allEntries.map((e) => ({
    domainId: e.domainId,
    name: e.tree.domainName,
    slug: e.slug,
    nodeCount: e.tree.layers.reduce((n, l) => n + l.nodes.length, 0),
  }))
  const groups = groupDomainsIntoConstellations(inputs)
  const group = groups.find((g) => g.domainIds.includes(entry.domainId))
  if (!group || group.domainIds.length < 2) return []
  return allEntries
    .filter((e) => group.domainIds.includes(e.domainId) && e.domainId !== entry.domainId)
    .map((e) => e.tree.domainName)
}

function renderDomainCard(
  entry: MultiDomainGraphEntry,
  allEntries: MultiDomainGraphEntry[],
  filterDomainId: string
): string {
  if (filterDomainId && entry.domainId !== filterDomainId) return ''

  const { modules } = resolveGraphModules(entry.tree)
  const layerMap = nodeLayerKeyMap(entry.tree)
  const titleMap = nodeTitleMap(entry.tree)
  const total = entry.tree.layers.reduce((n, l) => n + l.nodes.length, 0)
  const completed = [...entry.progressMap.values()].filter((p) => p.status === 'completed').length
  const pct = total > 0 ? Math.round((completed / total) * 100) : 0
  const expandedKey = pickDefaultExpandedModule(modules, entry.progressMap, entry.focusKeys)
  const related = relatedDomainNames(entry, allEntries)

  const modulesHtml = modules
    .map((mod) => {
      const modCompleted = mod.nodes.filter((k) => entry.progressMap.get(k)?.status === 'completed').length
      const isOpen = mod.key === expandedKey
      const nodesHtml = mod.nodes
        .map((nodeKey) => {
          const layer = layerMap.get(nodeKey)
          if (!layer) return ''
          const node = entry.tree.layers.flatMap((l) => l.nodes).find((n) => n.key === nodeKey)
          if (!node) return ''
          return renderNodeItem({
            node,
            layerKey: layer,
            progressMap: entry.progressMap,
            focusSet: entry.focusKeys,
            titleMap,
          })
        })
        .filter(Boolean)
        .join('')

      return `
        <details class="graph-outline-module"${isOpen ? ' open' : ''}>
          <summary class="graph-outline-module-summary">
            <span class="graph-outline-module-label">${escapeHtml(mod.label)}</span>
            <span class="graph-outline-module-meta">${modCompleted} / ${mod.nodes.length}</span>
          </summary>
          <ul class="node-list graph-outline-node-list">${nodesHtml}</ul>
        </details>
      `
    })
    .join('')

  const relatedHtml =
    related.length > 0
      ? `<div class="graph-outline-related">相关：${related.map((n) => `<span class="graph-outline-related-tag">${escapeHtml(n)}</span>`).join('')}</div>`
      : ''

  return `
    <article class="card graph-domain-card" data-domain-id="${escapeHtmlAttr(entry.domainId)}">
      <header class="graph-domain-card-header">
        <div class="graph-domain-card-main">
          <h2 class="graph-domain-card-title">${escapeHtml(entry.tree.domainName)}</h2>
          ${relatedHtml}
        </div>
        <div class="graph-domain-card-aside">
          <span class="graph-domain-card-progress" aria-label="完成 ${pct}%">${pct}%</span>
          <a href="#/tree/${escapeHtmlAttr(entry.domainId)}" class="graph-domain-card-link">课程详情</a>
        </div>
      </header>
      <div class="graph-outline-modules">${modulesHtml}</div>
    </article>
  `
}

export function renderGraphOutlineHtml(
  entries: MultiDomainGraphEntry[],
  filterDomainId: string
): string {
  const visible = filterDomainId ? entries.filter((e) => e.domainId === filterDomainId) : entries
  const summary = computeGraphOutlineSummary(visible.length ? visible : entries)
  const cardsHtml = entries.map((e) => renderDomainCard(e, entries, filterDomainId)).join('')

  const nextHint =
    summary.nextNodeTitle && summary.nextDomainName
      ? `<span class="graph-outline-next"><span class="graph-outline-next-label">推荐下一步</span><span class="graph-outline-next-value">${escapeHtml(summary.nextDomainName)} · ${escapeHtml(summary.nextNodeTitle)}</span></span>`
      : summary.completedNodes === summary.totalNodes && summary.totalNodes > 0
        ? '<span class="graph-outline-next graph-outline-next--done">全部节点已学完</span>'
        : ''

  return `
    <div class="graph-outline-summary card">
      <span>${summary.domainCount} 门课</span>
      <span class="graph-outline-summary-sep">·</span>
      <span>已完成 ${summary.completedNodes} / ${summary.totalNodes} 节点</span>
      ${nextHint}
    </div>
    <div class="graph-outline-cards">${cardsHtml || '<p class="graph-outline-empty">没有匹配的领域</p>'}</div>
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
    bindNodeList(
      card,
      (nodeKey, layerKey) => onNodeClick(domainId, nodeKey, layerKey),
      signal
    )
  })

  const firstFocus = container.querySelector<HTMLElement>('.node-item--focus')
  if (firstFocus) {
    firstFocus.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
  }

  container.querySelectorAll<HTMLDetailsElement>('.graph-outline-module').forEach((details) => {
    details.addEventListener(
      'toggle',
      () => {
        if (!details.open) return
        container.querySelectorAll<HTMLDetailsElement>('.graph-outline-module').forEach((other) => {
          if (other !== details) other.open = false
        })
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
