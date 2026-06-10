import { Network, type Data, type Options } from 'vis-network'
import { DataSet } from 'vis-data'
import 'vis-network/styles/vis-network.css'
import type { KnowledgeTree, UserProgress } from './api'
import {
  type GraphCanvasTheme,
  readGraphCanvasThemeFrom,
} from './graph-canvas-theme'
import {
  getGraphThemeTokens,
  hexWithAlpha,
  moduleColorAtRatio,
  type GraphLabelStyle,
  type GraphPalette,
} from './graph-theme-palette'
import {
  constellationSeparationLength,
  groupDomainsIntoConstellations,
  layoutDomainCentersByConstellation,
  type ConstellationGroup,
} from './graph-constellation'
import { lodFromScale, type GraphLodLevel } from './graph-lod'
import {
  domainCompletionRatio,
  moduleCompletionRatio,
  pathEdgeOpacity,
  PENDING_NODE_OPACITY,
} from './graph-progress-visual'
import { resolveGraphModules, nodeLayerKeyMap, nodeTitleMap, unmetPrerequisiteTitles } from './tree-normalize'

export type NodeProgressStatus = 'pending' | 'in_progress' | 'completed'

export interface KnowledgeGraphMount {
  destroy: () => void
  fit: () => void
  /** 将视图缩放到某一领域的全部节点 */
  focusDomain: (domainId: string) => void
  /** 当前缩放 LOD 层级 */
  getLodLevel: () => GraphLodLevel
}

const LABEL_SIZE = {
  root: 16,
  module: 14,
  topic: 13,
  topicPending: 12,
  topicFocus: 14,
} as const

let graphLabel: GraphLabelStyle = getGraphThemeTokens('paper').label
let graphPalette: GraphPalette = getGraphThemeTokens('paper').palette
let graphTheme: GraphCanvasTheme = 'paper'

function applyGraphTheme(theme: GraphCanvasTheme): void {
  const tokens = getGraphThemeTokens(theme)
  graphLabel = tokens.label
  graphPalette = tokens.palette
  graphTheme = theme
}

type GraphNode = {
  id: string
  label: string
  shape: string
  size: number
  mass?: number
  font: {
    size: number
    color: string
    face: string
    strokeWidth: number
    strokeColor: string
    vadjust?: number
    bold?: boolean
    align?: 'center'
  }
  color: {
    background: string
    border: string
    highlight: { background: string; border: string }
    hover?: { background: string; border: string }
  }
  borderWidth: number
  borderWidthSelected?: number
  chosen?: { node: boolean; label: boolean }
  nodeKey?: string
  layerKey?: string
  moduleKey?: string
  domainId?: string
  title?: string
  fullLabel?: string
  nodeRole?: 'domain' | 'module' | 'topic'
  hidden?: boolean
  fixed?: boolean | { x: boolean; y: boolean }
  x?: number
  y?: number
}

function normalizeStatus(status: string | undefined): NodeProgressStatus {
  if (status === 'completed' || status === 'in_progress') return status
  return 'pending'
}

type GraphModule = ReturnType<typeof resolveGraphModules>['modules'][number]

function computeDomainGraphProgress(
  progressMap: Map<string, UserProgress>,
  graphModules: GraphModule[],
  layerByNode: Map<string, string>,
  titles: Map<string, string>
): { domainComplete: boolean; moduleLit: Map<string, boolean> } {
  const moduleLit = new Map<string, boolean>()
  let total = 0
  let completed = 0

  for (const mod of graphModules) {
    let modTotal = 0
    let modDone = 0
    for (const nodeKey of mod.nodes) {
      if (!layerByNode.has(nodeKey) || !titles.has(nodeKey)) continue
      modTotal++
      total++
      if (normalizeStatus(progressMap.get(nodeKey)?.status) === 'completed') {
        modDone++
        completed++
      }
    }
    moduleLit.set(mod.key, modTotal > 0 && modDone === modTotal)
  }

  return {
    domainComplete: total > 0 && completed === total,
    moduleLit,
  }
}

/** vis-network 悬停时会读 color.hover；不设则会回退到库默认色，导致节点看起来和图例不一致 */
function steadyNodeColor(background: string, border: string): GraphNode['color'] {
  const steady = { background, border }
  return { background, border, highlight: steady, hover: steady }
}

function labelFont(size: number, bold = false) {
  const px = Math.max(graphLabel.minPx, size)
  return {
    size: px,
    color: graphLabel.text,
    face: graphLabel.face,
    strokeWidth: bold ? 3 : 2.5,
    strokeColor: graphLabel.stroke,
    vadjust: 22,
    bold,
    align: 'center' as const,
  }
}

function buildRootNode(opts: {
  id: string
  label: string
  size: number
  mass: number
  domainId: string
  title: string
  starlit?: boolean
}): GraphNode {
  const { id, label, size, mass, domainId, title, starlit = false } = opts
  const palette = starlit ? graphPalette.rootStarlit : graphPalette.root
  const fill = palette.fill
  const border = palette.border
  return {
    id,
    label,
    shape: 'dot',
    size,
    mass,
    font: labelFont(LABEL_SIZE.root, true),
    color: steadyNodeColor(fill, border),
    borderWidth: 2.5,
    borderWidthSelected: 2,
    chosen: { node: false, label: false },
    domainId,
    nodeRole: 'domain',
    title,
  }
}

function buildTopicNode(opts: {
  id: string
  title: string
  status: NodeProgressStatus
  focused: boolean
  nodeKey: string
  layerKey: string
  unmetPrereqs?: string[]
}): GraphNode {
  const { id, title, status, focused, nodeKey, layerKey, unmetPrereqs = [] } = opts
  const short = title.length > 20 ? title.slice(0, 19) + '…' : title
  const tooltipTitle =
    unmetPrereqs.length > 0 ? `${title} · 建议先学：${unmetPrereqs.join('、')}` : title

  if (focused) {
    return {
      id,
      label: short,
      shape: 'dot',
      size: 19,
      font: labelFont(LABEL_SIZE.topicFocus, true),
      color: steadyNodeColor(graphPalette.focus.fill, graphPalette.focus.border),
      borderWidth: 3,
      nodeKey,
      layerKey,
      nodeRole: 'topic',
      title: tooltipTitle,
      chosen: { node: false, label: false },
    }
  }

  if (status === 'completed') {
    return {
      id,
      label: short,
      shape: 'dot',
      size: 16,
      font: labelFont(LABEL_SIZE.topic, true),
      color: steadyNodeColor(graphPalette.done.fill, graphPalette.done.border),
      borderWidth: 2.5,
      nodeKey,
      layerKey,
      nodeRole: 'topic',
      title: tooltipTitle,
      chosen: { node: false, label: false },
    }
  }

  if (status === 'in_progress') {
    return {
      id,
      label: short,
      shape: 'dot',
      size: 15,
      font: labelFont(LABEL_SIZE.topic, true),
      color: steadyNodeColor(graphPalette.active.fill, graphPalette.active.border),
      borderWidth: 3,
      nodeKey,
      layerKey,
      nodeRole: 'topic',
      title: tooltipTitle,
      chosen: { node: false, label: false },
    }
  }

  const pendingFill = hexWithAlpha(graphPalette.pending.fill, PENDING_NODE_OPACITY)
  const pendingBorderRaw =
    unmetPrereqs.length > 0 ? hexWithAlpha(graphPalette.pending.border, 0.55) : graphPalette.pending.border
  const pendingBorder = hexWithAlpha(
    pendingBorderRaw.startsWith('rgba') ? graphPalette.pending.border : pendingBorderRaw,
    PENDING_NODE_OPACITY
  )

  return {
    id,
    label: short,
    shape: 'dot',
    size: 12,
    font: labelFont(LABEL_SIZE.topicPending),
    color: steadyNodeColor(pendingFill, pendingBorder),
    borderWidth: unmetPrereqs.length > 0 ? 2 : 1.5,
    nodeKey,
    layerKey,
    nodeRole: 'topic',
    title: tooltipTitle,
    chosen: { node: false, label: false },
  }
}

function buildModuleNode(opts: {
  id: string
  label: string
  domainId: string
  moduleKey: string
  title: string
  multiDomain: boolean
  lit?: boolean
  completionRatio?: number
  topicCount?: number
}): GraphNode {
  const { id, label, domainId, moduleKey, title, multiDomain, lit = false, completionRatio = 0, topicCount = 0 } = opts
  const short = label.length > 14 ? label.slice(0, 13) + '…' : label
  const palette = lit
    ? graphPalette.moduleLit
    : moduleColorAtRatio(graphPalette.module, graphPalette.moduleLit, completionRatio)
  const hubMass = (multiDomain ? 3.5 : 3) + Math.min(topicCount, 12) * 0.12
  return {
    id,
    label: short,
    fullLabel: label,
    nodeRole: 'module',
    shape: 'dot',
    size: multiDomain ? 20 : 22,
    mass: hubMass,
    font: labelFont(LABEL_SIZE.module, true),
    color: steadyNodeColor(palette.fill, palette.border),
    borderWidth: 2.5,
    borderWidthSelected: 2,
    chosen: { node: false, label: false },
    domainId,
    moduleKey,
    title,
  }
}

function moduleDisplayLabel(full: string, ratio: number, lod: GraphLodLevel): string {
  if (lod === 'node' && ratio >= 0.5) return full
  if (full.length > 14) return full.slice(0, 13) + '…'
  return full
}

function moduleLayoutOffset(
  center: { x: number; y: number },
  moduleIndex: number,
  moduleCount: number,
  multiDomain: boolean
): { x: number; y: number } {
  const dist = multiDomain ? 220 : 200
  if (moduleCount <= 1) {
    return { x: center.x + dist, y: center.y }
  }
  const angle = (2 * Math.PI * moduleIndex) / moduleCount - Math.PI / 2
  return {
    x: center.x + dist * Math.cos(angle),
    y: center.y + dist * Math.sin(angle),
  }
}

/** 主题节点围绕模块全圆排布 */
function topicLayoutOffset(
  modPos: { x: number; y: number },
  topicIndex: number,
  topicCount: number,
  multiDomain: boolean
): { x: number; y: number } {
  const dist = multiDomain ? 140 : 120
  if (topicCount <= 1) {
    return { x: modPos.x + dist, y: modPos.y }
  }
  const angle = (2 * Math.PI * topicIndex) / topicCount - Math.PI / 2
  return {
    x: modPos.x + dist * Math.cos(angle),
    y: modPos.y + dist * Math.sin(angle),
  }
}

export function mountKnowledgeGraph(opts: {
  container: HTMLElement
  tree: KnowledgeTree
  progressMap: Map<string, UserProgress>
  focusKeys: Set<string>
  onTopicClick: (nodeKey: string, layerKey: string) => void
}): KnowledgeGraphMount {
  const domainId = opts.tree.domainId
  return mountMultiDomainKnowledgeGraph({
    container: opts.container,
    domains: [
      {
        domainId,
        tree: opts.tree,
        progressMap: opts.progressMap,
        focusKeys: opts.focusKeys,
      },
    ],
    onTopicClick: (_domainId, nodeKey, layerKey) => opts.onTopicClick(nodeKey, layerKey),
  })
}

export interface MultiDomainGraphEntry {
  domainId: string
  slug?: string
  tree: KnowledgeTree
  progressMap: Map<string, UserProgress>
  focusKeys: Set<string>
}

export function mountMultiDomainKnowledgeGraph(opts: {
  container: HTMLElement
  domains: MultiDomainGraphEntry[]
  theme?: GraphCanvasTheme
  onTopicClick: (domainId: string, nodeKey: string, layerKey: string) => void
  onDomainClick?: (domainId: string) => void
}): KnowledgeGraphMount {
  const { container, domains, onTopicClick, onDomainClick } = opts
  applyGraphTheme(opts.theme ?? readGraphCanvasThemeFrom(container))

  const nodes = new DataSet<GraphNode>([])
  const glowById = new Map<string, 'focus' | 'active' | 'done' | 'starlight'>()
  const starlitRootIds = new Set<string>()
  const moduleClusterIds = new Map<string, string[]>()
  const domainClusterIds = new Map<string, string[]>()
  const moduleRatioById = new Map<string, number>()
  const domainRatioById = new Map<string, number>()
  const domainBaseSizeById = new Map<string, number>()
  const edges = new DataSet<{
    id: string
    from: string
    to: string
    length?: number
    dashes?: boolean | number[]
    color?: { color: string; highlight: string; opacity: number }
    width?: number
    hidden?: boolean
    smooth?: { enabled: boolean; type: string; roundness: number }
  }>([])

  const multiDomain = domains.length > 1
  const domainRootIds: string[] = []
  const domainIdToGroupKey = new Map<string, string>()

  const countDomainGraphNodes = (tree: KnowledgeTree): number => {
    const layerByNode = nodeLayerKeyMap(tree)
    const titles = nodeTitleMap(tree)
    const { modules } = resolveGraphModules(tree)
    let count = 1 + modules.length
    for (const layer of tree.layers) {
      for (const node of layer.nodes) {
        if (layerByNode.has(node.key) && titles.has(node.key)) count++
      }
    }
    return count
  }

  const constellationGroups: ConstellationGroup[] = multiDomain
    ? groupDomainsIntoConstellations(
        domains.map((d) => ({
          domainId: d.domainId,
          name: d.tree.domainName?.trim() || '课程',
          slug: d.slug,
          nodeCount: countDomainGraphNodes(d.tree),
        }))
      )
    : []

  for (const group of constellationGroups) {
    for (const did of group.domainIds) domainIdToGroupKey.set(did, group.key)
  }

  const domainCenterById: Map<string, { x: number; y: number }> = multiDomain
    ? layoutDomainCentersByConstellation(constellationGroups)
    : new Map()

  // 对预计算坐标施加随机抖动，让星座位置更自然
  if (multiDomain) {
    const jitterSeed = Date.now()
    let s = jitterSeed
    const rand = () => { s = (s * 1664525 + 1013904223) & 0xffffffff; return (s >>> 0) / 0xffffffff }
    for (const [id, pos] of domainCenterById) {
      const r = 180 + rand() * 320
      const a = rand() * Math.PI * 2
      domainCenterById.set(id, { x: pos.x + r * Math.cos(a), y: pos.y + r * Math.sin(a) })
    }
  }

  const groupByKey = new Map(constellationGroups.map((g) => [g.key, g]))

  for (let di = 0; di < domains.length; di++) {
    const entry = domains[di]!
    const { domainId, tree, progressMap, focusKeys } = entry
    const center = domainCenterById.get(domainId) ?? { x: 0, y: 0 }
    const domainTitle = tree.domainName?.trim() || '课程'
    const rootId = `domain:${domainId}`
    domainRootIds.push(rootId)
    const rootLabel =
      multiDomain && domainTitle.length > 18
        ? domainTitle.slice(0, 17) + '…'
        : domainTitle.length > 24
          ? domainTitle.slice(0, 23) + '…'
          : domainTitle
    const layerByNode = nodeLayerKeyMap(tree)
    const titles = nodeTitleMap(tree)
    const nodesByKey = new Map<
      string,
      { key: string; title: string; requires?: string[] }
    >()
    for (const layer of tree.layers) {
      for (const node of layer.nodes) {
        nodesByKey.set(node.key, node)
      }
    }
    const { modules: graphModules } = resolveGraphModules(tree)
    const validKeys = new Set<string>()
    for (const layer of tree.layers) {
      for (const node of layer.nodes) {
        if (layerByNode.has(node.key) && titles.has(node.key)) validKeys.add(node.key)
      }
    }
    const domainRatio = domainCompletionRatio(graphModules, progressMap, validKeys)
    domainRatioById.set(domainId, domainRatio)
    const { domainComplete, moduleLit } = computeDomainGraphProgress(
      progressMap,
      graphModules,
      layerByNode,
      titles
    )
    const domainCluster: string[] = [rootId]
    const rootBaseSize = multiDomain ? 28 : 32

    if (domainComplete) starlitRootIds.add(rootId)
    domainBaseSizeById.set(rootId, rootBaseSize)

    nodes.add({
      ...buildRootNode({
        id: rootId,
        label: rootLabel,
        size: rootBaseSize,
        mass: multiDomain ? 7 : 1,
        domainId,
        title: domainComplete
          ? `${domainTitle} · 本领域已全部学完`
          : `${domainTitle} · 单击定位 · 双击进入课程`,
        starlit: domainComplete,
      }),
      x: center.x,
      y: center.y,
      fixed: { x: true, y: true },
    })

    const topicMeta = new Map<string, { topicId: string; layerKey: string; moduleKey: string }>()

    for (let mi = 0; mi < graphModules.length; mi++) {
      const mod = graphModules[mi]!
      const moduleId = `module:${domainId}:${mod.key}`
      const clusterIds = [moduleId]
      const modPos = moduleLayoutOffset(center, mi, graphModules.length, multiDomain)

      const moduleComplete = moduleLit.get(mod.key) ?? false
      const validModuleKeys = mod.nodes.filter(
        (k) => layerByNode.has(k) && titles.has(k)
      )
      const modRatio = moduleCompletionRatio(mod, progressMap, validKeys)
      moduleRatioById.set(moduleId, modRatio)

      nodes.add({
        ...buildModuleNode({
          id: moduleId,
          label: mod.label,
          domainId,
          moduleKey: mod.key,
          title: moduleComplete
            ? `${mod.label} · 子领域已学完`
            : mod.goal
              ? `${mod.label} · ${mod.goal}`
              : mod.label,
          multiDomain,
          lit: moduleComplete,
          completionRatio: modRatio,
          topicCount: validModuleKeys.length,
        }),
        x: modPos.x,
        y: modPos.y,
      })

      if (moduleComplete) glowById.set(moduleId, 'done')
      else if (modRatio >= 0.5) glowById.set(moduleId, 'active')

      edges.add({
        id: `e-dm-${domainId}-${mod.key}`,
        from: rootId,
        to: moduleId,
        length: multiDomain ? 220 : 200,
        color: { color: graphPalette.edge.domainModule, highlight: graphPalette.edge.highlight, opacity: 0.65 },
        width: 1.5,
        smooth: { enabled: true, type: 'continuous', roundness: 0.2 },
      })

      validModuleKeys.forEach((nodeKey, ti) => {
        const layerKey = layerByNode.get(nodeKey)!
        const title = titles.get(nodeKey)!
        const topicId = `topic:${domainId}:${nodeKey}`
        const status = normalizeStatus(progressMap.get(nodeKey)?.status)
        const focused = focusKeys.has(nodeKey)
        const topicPos = topicLayoutOffset(modPos, ti, validModuleKeys.length, multiDomain)
        const treeNode = nodesByKey.get(nodeKey)
        const unmetPrereqs = treeNode
          ? unmetPrerequisiteTitles(treeNode, progressMap, titles)
          : []

        const topicNode = buildTopicNode({
          id: topicId,
          title,
          status,
          focused,
          nodeKey,
          layerKey,
          unmetPrereqs,
        })
        topicNode.domainId = domainId
        nodes.add({ ...topicNode, x: topicPos.x, y: topicPos.y })
        clusterIds.push(topicId)
        domainCluster.push(topicId)
        topicMeta.set(nodeKey, { topicId, layerKey, moduleKey: mod.key })

        if (focused) glowById.set(topicId, 'focus')
        else if (status === 'in_progress') glowById.set(topicId, 'active')
        else if (status === 'completed') glowById.set(topicId, 'done')

        edges.add({
          id: `e-mt-${domainId}-${mod.key}-${nodeKey}`,
          from: moduleId,
          to: topicId,
          length: multiDomain ? 140 : 120,
          color: { color: graphPalette.edge.belong, highlight: graphPalette.edge.highlight, opacity: 0.45 },
          width: 0.75,
          smooth: { enabled: true, type: 'continuous', roundness: 0.22 },
        })
      })

      moduleClusterIds.set(moduleId, clusterIds)
      domainCluster.push(moduleId)

      // 模块内推荐路径：按 layers 全局顺序，仅连接同模块相邻节点
      const orderedInModule: string[] = []
      for (const layer of tree.layers) {
        for (const node of layer.nodes) {
          if (nodeKeyInModule(node.key, mod.nodes)) {
            orderedInModule.push(node.key)
          }
        }
      }
      for (let i = 1; i < orderedInModule.length; i++) {
        const prev = topicMeta.get(orderedInModule[i - 1]!)?.topicId
        const curr = topicMeta.get(orderedInModule[i]!)?.topicId
        if (!prev || !curr) continue
        const pathOpacity = pathEdgeOpacity(modRatio)
        edges.add({
          id: `e-path-${domainId}-${mod.key}-${i}`,
          from: prev,
          to: curr,
          dashes: modRatio >= 0.85 ? false : [5, 8],
          color: { color: graphPalette.edge.path, highlight: graphPalette.edge.highlight, opacity: pathOpacity },
          width: modRatio >= 0.5 ? 1.4 : 1.0,
          smooth: { enabled: true, type: 'curvedCW', roundness: 0.15 },
        })
      }
    }

    for (const layer of tree.layers) {
      for (const node of layer.nodes) {
        if (!node.requires?.length) continue
        const curr = topicMeta.get(node.key)?.topicId
        if (!curr) continue
        for (const req of node.requires) {
          const prev = topicMeta.get(req)?.topicId
          if (!prev || prev === curr) continue
          const crossModule = topicMeta.get(req)?.moduleKey !== topicMeta.get(node.key)?.moduleKey
          edges.add({
            id: `e-req-${domainId}-${req}-${node.key}`,
            from: prev,
            to: curr,
            length: crossModule ? (multiDomain ? 160 : 130) : multiDomain ? 72 : 58,
            color: {
              color: graphPalette.edge.prerequisite,
              highlight: graphPalette.edge.highlight,
              opacity: 0.72,
            },
            width: 1.6,
            smooth: { enabled: true, type: 'curvedCCW', roundness: 0.22 },
          })
        }
      }
    }

    domainClusterIds.set(domainId, domainCluster)
  }

  if (multiDomain) {
    for (let i = 0; i < domains.length; i++) {
      for (let j = i + 1; j < domains.length; j++) {
        const idA = domains[i]!.domainId
        const idB = domains[j]!.domainId
        const keyA = domainIdToGroupKey.get(idA) ?? idA
        const keyB = domainIdToGroupKey.get(idB) ?? idB
        const groupA = groupByKey.get(keyA) ?? {
          key: keyA,
          label: keyA,
          domainIds: [idA],
          nodeCount: 1,
        }
        const groupB = groupByKey.get(keyB) ?? {
          key: keyB,
          label: keyB,
          domainIds: [idB],
          nodeCount: 1,
        }
        const sameGroup = groupA.key === groupB.key
        if (!sameGroup) {
          // 不同类领域不连线，仅用透明边维持排斥距离
          edges.add({
            id: `e-domain-sep-${i}-${j}`,
            from: domainRootIds[i]!,
            to: domainRootIds[j]!,
            length: constellationSeparationLength(groupA, groupB),
            color: { color: 'rgba(0,0,0,0)', highlight: 'rgba(0,0,0,0)', opacity: 0 },
            width: 0.01,
            smooth: { enabled: false, type: 'continuous', roundness: 0 },
          })
        } else {
          edges.add({
            id: `e-domain-sep-${i}-${j}`,
            from: domainRootIds[i]!,
            to: domainRootIds[j]!,
            length: constellationSeparationLength(groupA, groupB),
            color: { color: graphPalette.edge.domainRelated, highlight: graphPalette.edge.highlight, opacity: 1 },
            width: 1.2,
            smooth: { enabled: true, type: 'continuous', roundness: 0.15 },
          })
        }
      }
    }
  }

  function nodeKeyInModule(key: string, moduleNodes: string[]): boolean {
    return moduleNodes.includes(key)
  }

  const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  let pulsePhase = 0
  let rafId = 0
  let hoveredNodeId: string | null = null
  let currentLod: GraphLodLevel = 'node'
  let lodRaf = 0

  const applyLod = (level: GraphLodLevel) => {
    currentLod = level
    const updates: Array<Partial<GraphNode> & { id: string }> = []
    for (const node of nodes.get()) {
      const role = node.nodeRole
      let hidden = false
      if (level === 'galaxy') {
        hidden = role !== 'domain'
      } else if (level === 'constellation') {
        hidden = role === 'topic'
      }
      const patch: Partial<GraphNode> & { id: string } = { id: node.id, hidden }
      if (role === 'domain' && !hidden) {
        const ratio = domainRatioById.get(node.domainId ?? '') ?? 0
        const base = domainBaseSizeById.get(node.id) ?? node.size ?? 28
        const progressScale = 0.75 + 0.55 * ratio
        // galaxy LOD 下缩小视图，放大 domain 节点化作光晗
        const lodScale = level === 'galaxy' ? 5.5 : 1
        patch.size = Math.round(base * progressScale * lodScale)
      }
      if (role === 'module' && !hidden) {
        const ratio = moduleRatioById.get(node.id) ?? 0
        const full = node.fullLabel ?? node.label
        patch.label = moduleDisplayLabel(full, ratio, level)
      }
      updates.push(patch)
    }
    nodes.update(updates)
    const edgeUpdates: Array<{ id: string; hidden: boolean }> = []
    for (const edge of edges.get()) {
      const from = nodes.get(edge.from)
      const to = nodes.get(edge.to)
      edgeUpdates.push({ id: edge.id, hidden: !!(from?.hidden || to?.hidden) })
    }
    edges.update(edgeUpdates)
    network.redraw()
  }

  const syncLodFromZoom = () => {
    const scale = network.getScale()
    const next = lodFromScale(scale, multiDomain)
    if (next !== currentLod) applyLod(next)
  }

  const options: Options = {
    autoResize: true,
    interaction: {
      hover: true,
      tooltipDelay: 80,
      zoomView: true,
      dragView: true,
      dragNodes: true,
      selectable: false,
      navigationButtons: false,
      keyboard: { enabled: false },
      selectConnectedEdges: false,
      multiselect: false,
    },
    physics: reducedMotion
      ? { enabled: false }
      : {
          enabled: true,
          solver: 'forceAtlas2Based',
          forceAtlas2Based: {
            gravitationalConstant: multiDomain ? -8 : -20,
            centralGravity: 0,
            springLength: multiDomain ? 220 : 200,
            springConstant: multiDomain ? 0.15 : 0.08,
            damping: multiDomain ? 0.7 : 0.75,
            avoidOverlap: multiDomain ? 0.3 : 0.8,
          },
          stabilization: { iterations: multiDomain ? 380 : 400, updateInterval: 20 },
        },
    nodes: {
      shape: 'dot',
      scaling: {
        min: 10,
        max: 40,
        label: {
          enabled: false,
        },
      },
      font: {
        size: LABEL_SIZE.topic,
        color: graphLabel.text,
        face: graphLabel.face,
        strokeWidth: 2.5,
        strokeColor: graphLabel.stroke,
        vadjust: 22,
        align: 'center',
      },
      chosen: { node: false, label: false },
    },
    edges: {
      selectionWidth: 0,
      smooth: { enabled: true, type: 'continuous', roundness: 0.2 },
    },
  }

  // 直接传 DataSet（而非数组拷贝），后续 nodes.update（LOD 隐藏、拖拽钉住等）才能实时生效
  // 节点外观高亮由各节点的 chosen: { node: false, label: false } 禁用，无需额外配置
  const graphData: Data = {
    nodes: nodes as unknown as Data['nodes'],
    edges,
  }
  const network = new Network(container, graphData, options)

  const hashId = (s: string): number => {
    let h = 0
    for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) | 0
    return h >>> 0
  }

  // 宣纸主题：领域墨团周围散落的墨点（按节点 id 确定性分布，不随帧抖动）
  const drawInkSpeckles = (
    ctx: CanvasRenderingContext2D,
    pos: { x: number; y: number },
    baseR: number,
    id: string
  ) => {
    const h = hashId(id)
    const count = 3 + (h % 3)
    for (let i = 0; i < count; i++) {
      const angle = (((h >> (i * 4)) & 0xff) / 255) * Math.PI * 2
      const dist = baseR * (1.45 + (((h >> (i * 3)) & 0x3f) / 63) * 0.95)
      const r = Math.max(baseR * (0.06 + (((h >> (i * 5)) & 0x1f) / 31) * 0.09), 0.8)
      const alpha = 0.12 + (((h >> (i * 2)) & 0xf) / 15) * 0.14
      ctx.beginPath()
      ctx.arc(pos.x + Math.cos(angle) * dist, pos.y + Math.sin(angle) * dist, r, 0, Math.PI * 2)
      ctx.fillStyle = `rgba(41, 37, 33, ${alpha.toFixed(3)})`
      ctx.fill()
    }
  }

  // 星空主题：四向星芒 + 微闪烁（从节点边缘向外延伸，不遮挡节点本体）
  const drawStarSpikes = (
    ctx: CanvasRenderingContext2D,
    pos: { x: number; y: number },
    baseR: number,
    phase: number,
    color: string,
    lenMul = 1.8
  ) => {
    const twinkle = reducedMotion ? 0.9 : 0.7 + 0.3 * Math.sin(phase)
    const len = baseR * lenMul * twinkle
    ctx.save()
    ctx.strokeStyle = color
    ctx.lineWidth = 1
    for (let k = 0; k < 4; k++) {
      const a = (Math.PI / 2) * k
      ctx.beginPath()
      ctx.moveTo(pos.x + Math.cos(a) * baseR * 0.65, pos.y + Math.sin(a) * baseR * 0.65)
      ctx.lineTo(pos.x + Math.cos(a) * len, pos.y + Math.sin(a) * len)
      ctx.stroke()
    }
    ctx.restore()
  }

  const drawModuleHover = (ctx: CanvasRenderingContext2D, node: GraphNode, pos: { x: number; y: number }, scale: number) => {
    const baseR = (node.size ?? 12) * scale
    const pulse = reducedMotion ? 1 : 0.92 + 0.08 * Math.sin(pulsePhase)
    ctx.beginPath()
    ctx.arc(pos.x, pos.y, baseR + 3 * pulse, 0, Math.PI * 2)
    ctx.strokeStyle = graphPalette.hover.moduleStroke
    ctx.lineWidth = 2
    ctx.stroke()
  }

  const drawRootHover = (ctx: CanvasRenderingContext2D, node: GraphNode, pos: { x: number; y: number }, scale: number) => {
    const baseR = (node.size ?? 12) * scale
    const pulse = reducedMotion ? 1 : 0.92 + 0.08 * Math.sin(pulsePhase)

    ctx.beginPath()
    ctx.arc(pos.x, pos.y, baseR + 3 * pulse, 0, Math.PI * 2)
    ctx.strokeStyle = graphPalette.hover.rootStroke
    ctx.lineWidth = 2
    ctx.stroke()
  }

  const drawDomainStarlight = (
    ctx: CanvasRenderingContext2D,
    pos: { x: number; y: number },
    baseR: number,
    phase: number
  ) => {
    const pulse = reducedMotion ? 1 : 0.72 + 0.28 * Math.sin(phase)

    const haloR = baseR * (3.4 * pulse)
    const halo = ctx.createRadialGradient(pos.x, pos.y, baseR * 0.25, pos.x, pos.y, haloR)
    halo.addColorStop(0, graphPalette.glow.starlight)
    halo.addColorStop(0.35, 'rgba(245, 220, 106, 0.28)')
    halo.addColorStop(0.7, 'rgba(245, 220, 106, 0.08)')
    halo.addColorStop(1, 'rgba(245, 220, 106, 0)')
    ctx.beginPath()
    ctx.arc(pos.x, pos.y, haloR, 0, Math.PI * 2)
    ctx.fillStyle = halo
    ctx.fill()

    const rayCount = 8
    for (let i = 0; i < rayCount; i++) {
      const angle = (Math.PI * 2 * i) / rayCount + phase * 0.12
      const len = baseR * (2.6 + (reducedMotion ? 0 : 0.4 * Math.sin(phase + i * 1.1)))
      const alpha = reducedMotion ? 0.22 : 0.1 + 0.2 * (0.5 + 0.5 * Math.sin(phase * 1.4 + i))
      ctx.beginPath()
      ctx.moveTo(pos.x + Math.cos(angle) * baseR * 0.5, pos.y + Math.sin(angle) * baseR * 0.5)
      ctx.lineTo(pos.x + Math.cos(angle) * len, pos.y + Math.sin(angle) * len)
      ctx.strokeStyle = `rgba(255, 236, 170, ${alpha})`
      ctx.lineWidth = 1.25
      ctx.stroke()
    }

    const sparkleCount = 7
    for (let s = 0; s < sparkleCount; s++) {
      const a = phase * 0.85 + (Math.PI * 2 * s) / sparkleCount
      const dist = baseR * (1.9 + (reducedMotion ? 0 : 0.3 * Math.sin(phase * 2 + s)))
      const sx = pos.x + Math.cos(a) * dist
      const sy = pos.y + Math.sin(a) * dist
      const r = reducedMotion ? 1.6 : 1 + 1.4 * (0.5 + 0.5 * Math.sin(phase * 2.8 + s * 1.6))
      const alpha = reducedMotion ? 0.8 : 0.3 + 0.7 * (0.5 + 0.5 * Math.sin(phase * 3.2 + s))
      ctx.beginPath()
      ctx.arc(sx, sy, r, 0, Math.PI * 2)
      ctx.fillStyle = `rgba(255, 252, 245, ${alpha})`
      ctx.fill()
    }

    ctx.beginPath()
    ctx.arc(pos.x, pos.y, baseR + 2 * pulse, 0, Math.PI * 2)
    ctx.strokeStyle = reducedMotion
      ? 'rgba(201, 162, 39, 0.65)'
      : `rgba(201, 162, 39, ${0.45 + 0.35 * Math.sin(phase * 1.2)})`
    ctx.lineWidth = 2
    ctx.stroke()
  }

  const drawGlows = (ctx: CanvasRenderingContext2D) => {
    const positions = network.getPositions()
    const scale = network.getScale()
    const pulse = reducedMotion ? 1 : 0.85 + 0.15 * Math.sin(pulsePhase)

    for (const node of nodes.get()) {
      const pos = positions[node.id]
      if (!pos) continue
      if (node.hidden) continue

      if (node.nodeRole === 'domain' && hoveredNodeId === node.id) {
        drawRootHover(ctx, node, pos, scale)
      }

      if (node.nodeRole === 'module' && hoveredNodeId === node.id) {
        drawModuleHover(ctx, node, pos, scale)
      }

      const rawBaseR = (node.size ?? 12) * scale
      // galaxy LOD 下保证 domain 节点最小屏幕半径 14px，缩再远也看得见
      const MIN_GALAXY_DOMAIN_R = 14
      const baseR = (currentLod === 'galaxy' && node.nodeRole === 'domain')
        ? Math.max(rawBaseR, MIN_GALAXY_DOMAIN_R)
        : rawBaseR

      if (starlitRootIds.has(node.id)) {
        drawDomainStarlight(ctx, pos, baseR, pulsePhase)
        continue
      }

      // 主题氛围装饰：宣纸 = 领域墨团旁的墨点；星空 = 领域恒星与点亮节点的星芒
      if (graphTheme === 'paper' && node.nodeRole === 'domain') {
        drawInkSpeckles(ctx, pos, baseR, node.id)
      } else if (graphTheme === 'sky') {
        if (node.nodeRole === 'domain') {
          drawStarSpikes(
            ctx, pos, baseR,
            pulsePhase + (hashId(node.id) % 628) / 100,
            'rgba(235, 242, 255, 0.5)', 1.7
          )
        } else if (node.nodeRole === 'topic' && glowById.get(node.id) === 'done') {
          drawStarSpikes(
            ctx, pos, baseR,
            pulsePhase * 1.3 + (hashId(node.id) % 628) / 100,
            hexWithAlpha(graphPalette.done.fill, 0.55), 2.0
          )
        }
      }

      // galaxy LOD：所有 domain 节点画柔和光晕（星空为星光、宣纸为墨晕）
      if (currentLod === 'galaxy' && node.nodeRole === 'domain') {
        const haloR = baseR * (3.2 + 0.8 * Math.sin(pulsePhase))
        const halo = ctx.createRadialGradient(pos.x, pos.y, baseR * 0.3, pos.x, pos.y, haloR)
        if (graphTheme === 'paper') {
          halo.addColorStop(0, 'rgba(58, 54, 51, 0.3)')
          halo.addColorStop(0.45, 'rgba(58, 54, 51, 0.1)')
          halo.addColorStop(1, 'rgba(58, 54, 51, 0)')
        } else {
          halo.addColorStop(0, 'rgba(200, 215, 255, 0.55)')
          halo.addColorStop(0.45, 'rgba(180, 200, 245, 0.18)')
          halo.addColorStop(1, 'rgba(160, 185, 235, 0)')
        }
        ctx.beginPath()
        ctx.arc(pos.x, pos.y, haloR, 0, Math.PI * 2)
        ctx.fillStyle = halo
        ctx.fill()
        continue
      }

      const tier = glowById.get(node.id)
      if (!tier || tier === 'starlight') continue
      const mul = tier === 'focus' ? 2.8 * pulse : tier === 'active' ? 2.4 * pulse : 2.5 * pulse
      const outerR = baseR * mul
      const inner =
        tier === 'focus' ? graphPalette.glow.focus : tier === 'active' ? graphPalette.glow.active : graphPalette.glow.done

      const midStop =
        tier === 'done'
          ? hexWithAlpha(graphPalette.done.fill, 0.14)
          : hexWithAlpha(graphPalette.active.fill, 0.12)
      const outerStop =
        tier === 'done'
          ? hexWithAlpha(graphPalette.done.fill, 0)
          : hexWithAlpha(graphPalette.active.fill, 0)

      const g = ctx.createRadialGradient(pos.x, pos.y, baseR * 0.2, pos.x, pos.y, outerR)
      g.addColorStop(0, inner)
      g.addColorStop(0.5, midStop)
      g.addColorStop(1, outerStop)

      ctx.save()
      ctx.beginPath()
      ctx.arc(pos.x, pos.y, outerR, 0, Math.PI * 2)
      ctx.fillStyle = g
      ctx.fill()
      ctx.restore()
    }
  }

  network.on('hoverNode', (params) => {
    hoveredNodeId = params.node as string
    network.redraw()
  })
  network.on('blurNode', () => {
    hoveredNodeId = null
    network.redraw()
  })

  network.on('afterDrawing', (ctx) => {
    drawGlows(ctx as CanvasRenderingContext2D)
  })

  if (!reducedMotion) {
    const tick = () => {
      pulsePhase += 0.012
      network.redraw()
      rafId = requestAnimationFrame(tick)
    }
    rafId = requestAnimationFrame(tick)
  }

  const focusDomain = (domainId: string) => {
    const cluster = domainClusterIds.get(domainId)
    if (!cluster?.length) return
    const animDuration = reducedMotion ? 0 : 400
    network.fit({
      nodes: cluster,
      animation: reducedMotion ? false : { duration: animDuration, easingFunction: 'easeInOutQuad' },
    })
    setTimeout(() => applyLod('node'), reducedMotion ? 0 : animDuration + 20)
  }


  if (!reducedMotion) {
    network.once('stabilizationIterationsDone', () => {
      network.setOptions({ physics: { enabled: false } })
    })
  }

  // ── Obsidian 式节点拖拽 ──
  // 拖动时临时恢复物理引擎，让相邻节点被弹簧牵动；释放后钉在用户摆放的位置并冻结，
  // 领域根节点保持 fixed 锚定（拖动时临时解锁），整体星座布局不会被打散
  let dragSettleTimer = 0
  let dragPhysicsOn = false

  const enableDragPhysics = () => {
    if (reducedMotion || dragPhysicsOn) return
    dragPhysicsOn = true
    network.setOptions({ physics: { enabled: true, stabilization: false } })
  }

  const freezeAfterSettle = () => {
    if (!dragPhysicsOn) return
    window.clearTimeout(dragSettleTimer)
    dragSettleTimer = window.setTimeout(() => {
      network.setOptions({ physics: { enabled: false } })
      dragPhysicsOn = false
    }, 650)
  }

  network.on('dragStart', (params) => {
    const dragIds = (params.nodes ?? []) as string[]
    if (!dragIds.length) return
    window.clearTimeout(dragSettleTimer)
    // fixed 节点（领域根、已钉住的节点）需先解锁才能被拖动
    nodes.update(dragIds.map((id) => ({ id, fixed: false })))
    enableDragPhysics()
  })

  network.on('dragEnd', (params) => {
    const dragIds = (params.nodes ?? []) as string[]
    if (!dragIds.length) return
    // 钉在放下的位置，防止物理回弹把用户的摆放冲掉
    const positions = network.getPositions(dragIds)
    nodes.update(
      dragIds.map((id) => ({
        id,
        fixed: { x: true, y: true },
        x: positions[id]?.x,
        y: positions[id]?.y,
      }))
    )
    freezeAfterSettle()
  })

  network.on('click', (params) => {
    network.unselectAll()
    if (params.nodes.length !== 1) return
    const id = params.nodes[0] as string
    const item = nodes.get(id)
    if (!item) return

    if (id.startsWith('domain:') && item.domainId) {
      focusDomain(item.domainId)
      return
    }

    if (id.startsWith('module:')) {
      const cluster = moduleClusterIds.get(id)
      if (cluster?.length) {
        network.fit({
          nodes: cluster,
          animation: reducedMotion ? false : { duration: 350, easingFunction: 'easeInOutQuad' },
        })
      }
      return
    }

    if (!id.startsWith('topic:')) return
    if (!item.nodeKey || !item.layerKey || !item.domainId) return
    onTopicClick(item.domainId, item.nodeKey, item.layerKey)
  })

  network.on('doubleClick', (params) => {
    if (params.nodes.length !== 1) return
    const id = params.nodes[0] as string
    const item = nodes.get(id)
    if (id.startsWith('domain:') && item?.domainId && onDomainClick) {
      onDomainClick(item.domainId)
      return
    }
    network.focus(id, {
      scale: 1.35,
      animation: { duration: 300, easingFunction: 'easeInOutQuad' },
    })
  })

  network.on('zoom', () => {
    if (lodRaf) cancelAnimationFrame(lodRaf)
    lodRaf = requestAnimationFrame(() => {
      lodRaf = 0
      syncLodFromZoom()
    })
  })

  setTimeout(() => {
    network.fit({
      animation: reducedMotion ? false : { duration: 400, easingFunction: 'easeInOutQuad' },
    })
    setTimeout(() => syncLodFromZoom(), reducedMotion ? 0 : 420)
  }, 0)

  return {
    destroy: () => {
      if (rafId) cancelAnimationFrame(rafId)
      if (lodRaf) cancelAnimationFrame(lodRaf)
      window.clearTimeout(dragSettleTimer)
      network.destroy()
    },
    fit: () => {
      network.fit({ animation: reducedMotion ? false : { duration: 300, easingFunction: 'easeInOutQuad' } })
      setTimeout(() => syncLodFromZoom(), reducedMotion ? 0 : 320)
    },
    focusDomain,
    getLodLevel: () => currentLod,
  }
}
