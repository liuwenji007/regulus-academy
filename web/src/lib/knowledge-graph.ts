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
import {
  loadGraphLayout,
  persistGraphLayoutFromNetwork,
  resolveNodePlacement,
} from './graph-layout-persist'
import { lodFromScale, lodThresholds, topicSizeForLod, topicLabelsVisible, type GraphLodLevel } from './graph-lod'
import {
  drawOrganicInkSpeckle,
  drawOrganicInkWash,
} from './ink-blot'
import { drawInkStamp, drawPaperStamp, preloadInkStamps, type InkStampRole, type PaperNodeVisual } from './ink-stamps'
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

/** 宣纸近景：节点由 canvas 手绘墨点呈现，vis 圆点透明 */
function paperInkNodeColor(): GraphNode['color'] {
  return steadyNodeColor('transparent', 'transparent')
}

function nodeColorForTheme(fill: string, border: string, inkOnCanvas = false): GraphNode['color'] {
  if (graphTheme === 'paper' && inkOnCanvas) return paperInkNodeColor()
  return steadyNodeColor(fill, border)
}

function paperNodeBorderWidth(inkOnCanvas: boolean, fallback: number): number {
  if (graphTheme === 'paper' && inkOnCanvas) return 0
  return fallback
}

function paperStampOnCanvas(): boolean {
  return graphTheme === 'paper'
}

/**
 * 墨迹图章半径（vis 网络坐标，与 node.size 同系）。
 * border 为屏幕像素，需除以 viewScale 换算到网络坐标，才能与未点亮 vis 圆点对齐。
 */
function paperInkDrawRadius(
  nodeSize: number,
  viewScale: number,
  role?: GraphNode['nodeRole']
): number {
  const borderPx = role === 'topic' ? 2 : 2.5
  const borderWorld = borderPx / Math.max(viewScale, 0.001)
  return nodeSize + borderWorld
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
  const stampOnCanvas = paperStampOnCanvas()
  return {
    id,
    label,
    shape: 'dot',
    size,
    mass,
    font: labelFont(LABEL_SIZE.root, true),
    color: nodeColorForTheme(fill, border, stampOnCanvas),
    borderWidth: paperNodeBorderWidth(stampOnCanvas, 2.5),
    borderWidthSelected: paperNodeBorderWidth(stampOnCanvas, 2),
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
    unmetPrereqs.length > 0
      ? `${title} · 建议先学：${unmetPrereqs.join('、')} · 双击开始学习`
      : `${title} · 单击定位 · 双击开始学习`

  const paperInkLit = paperStampOnCanvas()

  if (focused) {
    return {
      id,
      label: short,
      shape: 'dot',
      size: 19,
      font: labelFont(LABEL_SIZE.topicFocus, true),
      color: nodeColorForTheme(graphPalette.focus.fill, graphPalette.focus.border, paperInkLit),
      borderWidth: paperNodeBorderWidth(paperInkLit, 3),
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
      color: nodeColorForTheme(graphPalette.done.fill, graphPalette.done.border, paperInkLit),
      borderWidth: paperNodeBorderWidth(paperInkLit, 2.5),
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
      color: nodeColorForTheme(graphPalette.active.fill, graphPalette.active.border, paperInkLit),
      borderWidth: paperNodeBorderWidth(paperInkLit, 3),
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
    color: nodeColorForTheme(pendingFill, pendingBorder, paperStampOnCanvas()),
    borderWidth: paperNodeBorderWidth(paperStampOnCanvas(), unmetPrereqs.length > 0 ? 2 : 1.5),
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
  const paperInkLit = paperStampOnCanvas()
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
    color: nodeColorForTheme(palette.fill, palette.border, paperInkLit),
    borderWidth: paperNodeBorderWidth(paperInkLit, 2.5),
    borderWidthSelected: paperNodeBorderWidth(paperInkLit, 2),
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
  parentSlug?: string
  /** 父课知识树中的锚点：优先连到模块，其次连到主题节点 */
  parentAnchor?: { nodeKey?: string; moduleKey?: string }
  tree: KnowledgeTree
  progressMap: Map<string, UserProgress>
  focusKeys: Set<string>
}

function topicRoot(slug: string): string {
  const s = slug.toLowerCase().trim()
  if (!s) return ''
  if (s === 'go' || s === 'golang' || s === 'go-language' || s.startsWith('go-')) return 'go'
  return s
}

function findParentDomainId(domains: MultiDomainGraphEntry[], childParentSlug: string): string | undefined {
  const want = childParentSlug.toLowerCase().trim()
  if (!want) return undefined
  const wantRoot = topicRoot(want)
  let exact: string | undefined
  let family: string | undefined
  for (const d of domains) {
    const s = d.slug?.toLowerCase().trim()
    if (!s) continue
    if (s === want) {
      exact = d.domainId
      break
    }
    if (!family && (s === wantRoot || topicRoot(s) === wantRoot)) {
      family = d.domainId
    }
  }
  return exact ?? family
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

  const graphDomainIds = domains.map((d) => d.domainId)
  const savedPositions = loadGraphLayout(graphDomainIds)
  const usePersistedLayout = savedPositions !== null

  const nodes = new DataSet<GraphNode>([])
  const glowById = new Map<string, 'focus' | 'active' | 'done' | 'starlight'>()
  const starlitRootIds = new Set<string>()
  const moduleClusterIds = new Map<string, string[]>()
  const domainClusterIds = new Map<string, string[]>()
  const moduleRatioById = new Map<string, number>()
  const domainRatioById = new Map<string, number>()
  const domainBaseSizeById = new Map<string, number>()
  const topicBaseSizeById = new Map<string, number>()
  const topicLabelFontById = new Map<string, GraphNode['font']>()
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
    arrows?: { to?: { enabled: boolean; scaleFactor?: number } }
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

  // 有本地布局时跳过随机抖动，避免覆盖用户拖拽结果
  if (multiDomain && !usePersistedLayout) {
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

    const rootPlacement = resolveNodePlacement(rootId, center, savedPositions)
    const domainAnchor = { x: rootPlacement.x, y: rootPlacement.y }
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
      x: rootPlacement.x,
      y: rootPlacement.y,
      fixed: rootPlacement.fixed ?? { x: true, y: true },
    })

    const topicMeta = new Map<string, { topicId: string; layerKey: string; moduleKey: string }>()

    for (let mi = 0; mi < graphModules.length; mi++) {
      const mod = graphModules[mi]!
      const moduleId = `module:${domainId}:${mod.key}`
      const clusterIds = [moduleId]
      const modDefault = moduleLayoutOffset(domainAnchor, mi, graphModules.length, multiDomain)
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
        x: modDefault.x,
        y: modDefault.y,
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
        const topicDefault = topicLayoutOffset(modDefault, ti, validModuleKeys.length, multiDomain)
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
        topicBaseSizeById.set(topicId, topicNode.size)
        topicLabelFontById.set(topicId, topicNode.font)
        nodes.add({ ...topicNode, x: topicDefault.x, y: topicDefault.y })
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
          color: {
            color: graphPalette.edge.belong,
            highlight: graphPalette.edge.highlight,
            opacity: graphTheme === 'paper' ? 0.26 : 0.45,
          },
          width: graphTheme === 'paper' ? 0.6 : 0.75,
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
        // 宣纸水墨：节点间路径线省略，顺序由布局与节点状态表达
        if (graphTheme === 'paper') continue
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
          // 宣纸水墨：同模块前置省略（与路径线重复）；跨模块保留
          if (graphTheme === 'paper' && !crossModule) continue
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

  if (multiDomain) {
    for (const child of domains) {
      const parentSlug = child.parentSlug?.toLowerCase().trim()
      if (!parentSlug) continue
      const parentDomainId = findParentDomainId(domains, parentSlug)
      if (!parentDomainId || parentDomainId === child.domainId) continue
      const to = `domain:${child.domainId}`
      if (!domainRootIds.includes(to)) continue

      let from = `domain:${parentDomainId}`
      const anchor = child.parentAnchor
      if (anchor?.moduleKey) {
        const moduleId = `module:${parentDomainId}:${anchor.moduleKey}`
        if (nodes.get(moduleId)) from = moduleId
      } else if (anchor?.nodeKey) {
        const topicId = `topic:${parentDomainId}:${anchor.nodeKey}`
        if (nodes.get(topicId)) from = topicId
      }
      if (!nodes.get(from)) continue

      edges.add({
        id: `e-parent-child-${parentDomainId}-${child.domainId}`,
        from,
        to,
        length: from.startsWith('domain:') ? 200 : 280,
        color: {
          color: graphPalette.edge.domainParentChild,
          highlight: graphPalette.edge.highlight,
          opacity: 0.92,
        },
        width: 2.2,
        arrows: { to: { enabled: true, scaleFactor: 0.6 } },
        smooth: { enabled: true, type: 'curvedCW', roundness: 0.25 },
      })
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
        hidden = graphTheme === 'paper' ? false : role === 'topic'
      }
      const patch: Partial<GraphNode> & { id: string } = { id: node.id, hidden }
      if (role === 'topic' && !hidden && graphTheme === 'paper') {
        const baseSize = topicBaseSizeById.get(node.id) ?? node.size ?? 12
        const baseFont = topicLabelFontById.get(node.id)
        patch.size = topicSizeForLod(baseSize, level, graphTheme)
        if (baseFont) {
          patch.font = topicLabelsVisible(level, graphTheme)
            ? baseFont
            : { ...baseFont, size: 0, strokeWidth: 0, color: 'rgba(0,0,0,0)' }
        }
      }
      if (role === 'domain' && !hidden) {
        const ratio = domainRatioById.get(node.domainId ?? '') ?? 0
        const base = domainBaseSizeById.get(node.id) ?? node.size ?? 28
        const progressScale = 0.75 + 0.55 * ratio
        const starlit = starlitRootIds.has(node.id)
        const palette = starlit ? graphPalette.rootStarlit : graphPalette.root
        const stampOnCanvas = graphTheme === 'paper'
        if (level === 'galaxy') {
          patch.size = Math.round(base * progressScale * 1.5)
          patch.font = { ...labelFont(LABEL_SIZE.root, true), size: 0, strokeWidth: 0, color: 'rgba(0,0,0,0)' }
          if (graphTheme === 'sky') {
            patch.color = steadyNodeColor('rgba(255, 255, 255, 0.9)', 'rgba(170, 195, 255, 0.25)')
          }
        } else {
          patch.size = Math.round(base * progressScale)
          patch.font = labelFont(LABEL_SIZE.root, true)
          patch.color = nodeColorForTheme(palette.fill, palette.border, stampOnCanvas)
          patch.borderWidth = paperNodeBorderWidth(stampOnCanvas, 2.5)
        }
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
    const next = lodFromScale(scale, multiDomain, graphTheme)
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
  if (graphTheme === 'paper') void preloadInkStamps().then(() => network.redraw())

  const hashId = (s: string): number => {
    let h = 0
    for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) | 0
    return h >>> 0
  }

  /** 屏幕像素半径 → vis-network 世界坐标（afterDrawing 画布使用世界坐标） */
  const screenRadiusToWorld = (screenPx: number, viewScale: number): number =>
    screenPx / Math.max(viewScale, 0.001)

  // 宣纸主题：领域墨团周围散落的墨点
  const drawInkSpeckles = (
    ctx: CanvasRenderingContext2D,
    pos: { x: number; y: number },
    baseR: number,
    id: string,
    viewScale: number
  ) => {
    const h = hashId(id)
    const count = 3 + (h % 3)
    for (let i = 0; i < count; i++) {
      const angle = (((h >> (i * 4)) & 0xff) / 255) * Math.PI * 2
      const dist = baseR * (1.45 + (((h >> (i * 3)) & 0x3f) / 63) * 0.95)
      const r = Math.max(baseR * (0.06 + (((h >> (i * 5)) & 0x1f) / 31) * 0.09), 0.8)
      const alpha = 0.12 + (((h >> (i * 2)) & 0xf) / 15) * 0.14
      drawOrganicInkSpeckle(
        ctx,
        pos.x + Math.cos(angle) * dist,
        pos.y + Math.sin(angle) * dist,
        r,
        h + i * 17,
        alpha,
        viewScale
      )
    }
  }

  /** 宣纸：节点图章状态（学完 / 未学 / 进行中） */
  const resolvePaperNodeVisual = (nodeId: string, role: InkStampRole): PaperNodeVisual => {
    const tier = glowById.get(nodeId)
    if (role === 'domain') {
      return starlitRootIds.has(nodeId) ? 'lit' : 'empty'
    }
    if (role === 'module') {
      if (tier === 'done') return 'lit'
      if (tier === 'active') return 'progress'
      return 'empty'
    }
    if (tier === 'done') return 'lit'
    if (tier === 'focus' || tier === 'active') return 'progress'
    return 'empty'
  }

  const isPaperDomainLit = (nodeId: string): boolean => resolvePaperNodeVisual(nodeId, 'domain') === 'lit'

  /** 宣纸：近景节点图章（学完 dot / 未学 empty / 进行中 pre） */
  const drawPaperNodeStamp = (
    ctx: CanvasRenderingContext2D,
    node: GraphNode,
    pos: { x: number; y: number },
    viewScale: number
  ) => {
    const role = node.nodeRole
    if (role !== 'domain' && role !== 'module' && role !== 'topic') return
    const visual = resolvePaperNodeVisual(node.id, role)
    const defaultSize = role === 'domain' ? 28 : role === 'module' ? 22 : 12
    const nodeSize = node.size ?? defaultSize
    drawPaperStamp(
      ctx,
      pos.x,
      pos.y,
      paperInkDrawRadius(nodeSize, viewScale, role),
      hashId(node.id),
      role,
      visual,
      1,
      viewScale
    )
  }

  /** 宣纸 · 远景圆满：静态墨晕 + 墨点 */
  const drawStarlitInkGalaxyWash = (
    ctx: CanvasRenderingContext2D,
    pos: { x: number; y: number },
    viewScale: number,
    alpha: number,
    seed: number
  ) => {
    if (alpha <= 0) return
    ctx.save()
    ctx.globalAlpha = alpha
    const spread = screenRadiusToWorld(46, viewScale)
    const dotR = screenRadiusToWorld(6.5, viewScale)
    drawOrganicInkWash(ctx, pos.x, pos.y, spread, seed, 1, 0.95, viewScale)
    drawInkStamp(ctx, pos.x, pos.y, dotR, seed, 'domain', 1, viewScale)
    ctx.restore()
  }

  /** 宣纸 · 近景圆满根节点：静态墨晕 + 墨点 */
  const drawStarlitInkRootWash = (
    ctx: CanvasRenderingContext2D,
    pos: { x: number; y: number },
    modelR: number,
    alpha: number,
    seed: number,
    viewScale: number
  ) => {
    if (alpha <= 0) return
    ctx.save()
    ctx.globalAlpha = alpha
    const spread = modelR * 3.2
    drawOrganicInkWash(ctx, pos.x, pos.y, spread, seed, 1, 0.9, viewScale)
    drawInkStamp(ctx, pos.x, pos.y, paperInkDrawRadius(modelR, viewScale, 'domain'), seed, 'domain', 1, viewScale)
    ctx.restore()
  }

  /** 宣纸 · 圆满课程簇水墨晕染（绘制在节点与边之下，静态） */
  const drawStarlitInkDomainCluster = (
    ctx: CanvasRenderingContext2D,
    rootPos: { x: number; y: number },
    haloR: number,
    alpha: number,
    seed: number,
    viewScale: number
  ) => {
    if (alpha <= 0) return
    ctx.save()
    ctx.globalAlpha = alpha
    drawOrganicInkWash(ctx, rootPos.x, rootPos.y, haloR, seed, 1, 0.95, viewScale)
    ctx.restore()
  }

  /** 星座主题色（与目录视图 data-constellation-key 对齐） */
  const constellationTint = (key: string): { core: string; mid: string; outer: string } => {
    switch (key) {
      case 'python':
        return { core: 'rgba(100, 165, 230, 0.2)', mid: 'rgba(55, 118, 171, 0.09)', outer: 'rgba(35, 85, 150, 0)' }
      case 'go':
        return { core: 'rgba(90, 210, 245, 0.18)', mid: 'rgba(0, 173, 216, 0.08)', outer: 'rgba(0, 120, 175, 0)' }
      case 'rust':
        return { core: 'rgba(240, 195, 165, 0.17)', mid: 'rgba(222, 165, 132, 0.08)', outer: 'rgba(175, 115, 85, 0)' }
      case 'agent':
        return { core: 'rgba(190, 150, 255, 0.16)', mid: 'rgba(150, 110, 230, 0.07)', outer: 'rgba(100, 70, 190, 0)' }
      case 'math':
        return { core: 'rgba(170, 200, 255, 0.15)', mid: 'rgba(120, 160, 230, 0.07)', outer: 'rgba(80, 120, 200, 0)' }
      default:
        return { core: 'rgba(155, 185, 245, 0.16)', mid: 'rgba(110, 150, 225, 0.07)', outer: 'rgba(70, 110, 195, 0)' }
    }
  }

  const drawSoftSkyStarGlow = (
    ctx: CanvasRenderingContext2D,
    pos: { x: number; y: number },
    coreR: number,
    phase: number,
    intensity = 1
  ) => {
    const pulse = reducedMotion ? 1 : 0.86 + 0.14 * Math.sin(phase)
    const outerR = Math.max(coreR * (9 + pulse * 1.5), 52) * intensity
    const outer = ctx.createRadialGradient(pos.x, pos.y, 0, pos.x, pos.y, outerR)
    outer.addColorStop(0, 'rgba(250, 253, 255, 0.55)')
    outer.addColorStop(0.1, 'rgba(225, 238, 255, 0.38)')
    outer.addColorStop(0.28, 'rgba(185, 210, 250, 0.16)')
    outer.addColorStop(0.55, 'rgba(140, 175, 235, 0.06)')
    outer.addColorStop(1, 'rgba(90, 130, 210, 0)')
    ctx.beginPath()
    ctx.arc(pos.x, pos.y, outerR, 0, Math.PI * 2)
    ctx.fillStyle = outer
    ctx.fill()

    const midR = Math.max(coreR * (4.2 * pulse), 24) * intensity
    const mid = ctx.createRadialGradient(pos.x, pos.y, 0, pos.x, pos.y, midR)
    mid.addColorStop(0, 'rgba(240, 248, 255, 0.65)')
    mid.addColorStop(0.35, 'rgba(200, 220, 255, 0.22)')
    mid.addColorStop(1, 'rgba(160, 190, 245, 0)')
    ctx.beginPath()
    ctx.arc(pos.x, pos.y, midR, 0, Math.PI * 2)
    ctx.fillStyle = mid
    ctx.fill()
  }

  /** 星座簇氛围光：远景星云 + 中景星座雾 */
  const drawConstellationAtmosphere = (
    ctx: CanvasRenderingContext2D,
    positions: Record<string, { x: number; y: number }>,
    scale: number
  ) => {
    if (!multiDomain) return
    if (currentLod !== 'galaxy' && currentLod !== 'constellation') return

    const isGalaxy = currentLod === 'galaxy'
    const includeModules = currentLod === 'constellation'

    for (const group of constellationGroups) {
      const pts: { x: number; y: number }[] = []
      for (const domainId of group.domainIds) {
        const rootPos = positions[`domain:${domainId}`]
        if (rootPos) pts.push(rootPos)
      }
      if (includeModules) {
        for (const node of nodes.get()) {
          if (node.hidden || node.nodeRole !== 'module') continue
          if (!group.domainIds.includes(node.domainId ?? '')) continue
          const p = positions[node.id]
          if (p) pts.push(p)
        }
      }
      if (pts.length === 0) continue

      const cx = pts.reduce((s, p) => s + p.x, 0) / pts.length
      const cy = pts.reduce((s, p) => s + p.y, 0) / pts.length
      let spread = 0
      for (const p of pts) {
        spread = Math.max(spread, Math.hypot(p.x - cx, p.y - cy))
      }
      spread = Math.max(spread, includeModules ? 110 : 130)
      const phase = pulsePhase + (hashId(group.key) % 200) / 100
      const pulse = reducedMotion ? 1 : 0.92 + 0.08 * Math.sin(phase)
      const outerR = Math.max((spread + (isGalaxy ? 120 : 85)) * scale * pulse, isGalaxy ? 68 : 52)

      if (graphTheme === 'sky') {
        const tint = constellationTint(group.key)
        const wash = ctx.createRadialGradient(cx, cy, 0, cx, cy, outerR)
        wash.addColorStop(0, tint.core)
        wash.addColorStop(0.35, tint.mid)
        wash.addColorStop(0.72, 'rgba(70, 110, 200, 0.025)')
        wash.addColorStop(1, tint.outer)
        ctx.beginPath()
        ctx.arc(cx, cy, outerR, 0, Math.PI * 2)
        ctx.fillStyle = wash
        ctx.fill()

        // 第二层更淡的外围雾，让星座边界更柔和
        const mistR = outerR * (isGalaxy ? 1.35 : 1.22)
        const mist = ctx.createRadialGradient(cx, cy, outerR * 0.35, cx, cy, mistR)
        mist.addColorStop(0, 'rgba(120, 155, 230, 0.04)')
        mist.addColorStop(0.5, 'rgba(90, 130, 210, 0.025)')
        mist.addColorStop(1, 'rgba(60, 95, 180, 0)')
        ctx.beginPath()
        ctx.arc(cx, cy, mistR, 0, Math.PI * 2)
        ctx.fillStyle = mist
        ctx.fill()
      } else {
        const inkR = outerR * 0.95
        const ink = ctx.createRadialGradient(cx, cy, 0, cx, cy, inkR)
        ink.addColorStop(0, 'rgba(58, 54, 51, 0.14)')
        ink.addColorStop(0.45, 'rgba(58, 54, 51, 0.06)')
        ink.addColorStop(1, 'rgba(58, 54, 51, 0)')
        ctx.beginPath()
        ctx.arc(cx, cy, inkR, 0, Math.PI * 2)
        ctx.fillStyle = ink
        ctx.fill()
      }
    }
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

  /** 圆满课程簇半径（世界坐标）：覆盖全部子节点，不随 viewScale 缩小 */
  const computeStarlitClusterRadius = (
    rootPos: { x: number; y: number },
    clusterIds: string[],
    positions: Record<string, { x: number; y: number }>
  ): number => {
    let spread = 0
    for (const id of clusterIds) {
      const node = nodes.get(id)
      if (!node) continue
      const pos = positions[id]
      if (!pos) continue
      const nodeR = (node.size ?? 12) + 24
      spread = Math.max(spread, Math.hypot(pos.x - rootPos.x, pos.y - rootPos.y) + nodeR)
    }
    return Math.max(spread, 160)
  }

  /** 圆满态：整门课包裹光晕（绘制在节点与边之下） */
  const drawStarlitDomainCluster = (
    ctx: CanvasRenderingContext2D,
    rootPos: { x: number; y: number },
    haloR: number,
    phase: number,
    alpha = 1
  ) => {
    if (alpha <= 0) return
    ctx.save()
    ctx.globalAlpha = alpha
    const pulse = reducedMotion ? 1 : 0.94 + 0.06 * Math.sin(phase)
    const R = haloR * pulse

    const wash = ctx.createRadialGradient(rootPos.x, rootPos.y, R * 0.06, rootPos.x, rootPos.y, R)
    if (graphTheme === 'sky') {
      wash.addColorStop(0, 'rgba(255, 248, 210, 0.1)')
      wash.addColorStop(0.18, 'rgba(245, 220, 106, 0.2)')
      wash.addColorStop(0.45, 'rgba(245, 220, 106, 0.09)')
      wash.addColorStop(0.72, 'rgba(245, 220, 106, 0.03)')
      wash.addColorStop(1, 'rgba(245, 220, 106, 0)')
    } else {
      wash.addColorStop(0, 'rgba(42, 38, 34, 0.08)')
      wash.addColorStop(0.22, 'rgba(58, 54, 51, 0.14)')
      wash.addColorStop(0.5, 'rgba(72, 68, 62, 0.06)')
      wash.addColorStop(0.78, 'rgba(58, 54, 51, 0.02)')
      wash.addColorStop(1, 'rgba(58, 54, 51, 0)')
    }
    ctx.beginPath()
    ctx.arc(rootPos.x, rootPos.y, R, 0, Math.PI * 2)
    ctx.fillStyle = wash
    ctx.fill()

    const mistR = R * 1.1
    const mist = ctx.createRadialGradient(rootPos.x, rootPos.y, R * 0.55, rootPos.x, rootPos.y, mistR)
    mist.addColorStop(0, 'rgba(58, 54, 51, 0)')
    mist.addColorStop(0.45, 'rgba(58, 54, 51, 0.02)')
    mist.addColorStop(1, 'rgba(58, 54, 51, 0)')
    ctx.beginPath()
    ctx.arc(rootPos.x, rootPos.y, mistR, 0, Math.PI * 2)
    ctx.fillStyle = mist
    ctx.fill()
    ctx.restore()
  }

  /** 远景点光晕权重：1=全景点光，0=近景簇光，随 scale 连续过渡 */
  const starlitFarMix = (viewScale: number): number => {
    const farEnd = 0.012
    const farStart = 0.034
    if (viewScale <= farEnd) return 1
    if (viewScale >= farStart) return 0
    return (farStart - viewScale) / (farStart - farEnd)
  }

  /** 远景点光晕：屏幕像素稳定，略亮于未圆满恒星 */
  const drawStarlitGalaxyGlow = (
    ctx: CanvasRenderingContext2D,
    pos: { x: number; y: number },
    phase: number,
    viewScale: number,
    alpha = 1
  ) => {
    if (alpha <= 0) return
    ctx.save()
    ctx.globalAlpha = alpha
    const pulse = reducedMotion ? 1 : 0.86 + 0.14 * Math.sin(phase)
    const outerScreen = 40 + (reducedMotion ? 0 : 2 * Math.sin(phase))
    const midScreen = 18 * pulse
    const coreScreen = 10
    const outerR = screenRadiusToWorld(outerScreen, viewScale)
    const midR = screenRadiusToWorld(midScreen, viewScale)
    const coreR = screenRadiusToWorld(coreScreen, viewScale)

    const outer = ctx.createRadialGradient(pos.x, pos.y, 0, pos.x, pos.y, outerR)
    outer.addColorStop(0, 'rgba(255, 248, 210, 0.42)')
    outer.addColorStop(0.12, 'rgba(245, 220, 106, 0.32)')
    outer.addColorStop(0.35, 'rgba(245, 220, 106, 0.14)')
    outer.addColorStop(0.62, 'rgba(245, 220, 106, 0.05)')
    outer.addColorStop(1, 'rgba(245, 220, 106, 0)')
    ctx.beginPath()
    ctx.arc(pos.x, pos.y, outerR, 0, Math.PI * 2)
    ctx.fillStyle = outer
    ctx.fill()

    const mid = ctx.createRadialGradient(pos.x, pos.y, 0, pos.x, pos.y, midR)
    mid.addColorStop(0, 'rgba(255, 246, 215, 0.68)')
    mid.addColorStop(0.38, 'rgba(245, 220, 106, 0.26)')
    mid.addColorStop(1, 'rgba(245, 220, 106, 0)')
    ctx.beginPath()
    ctx.arc(pos.x, pos.y, midR, 0, Math.PI * 2)
    ctx.fillStyle = mid
    ctx.fill()

    const core = ctx.createRadialGradient(pos.x, pos.y, 0, pos.x, pos.y, coreR)
    core.addColorStop(0, 'rgba(255, 252, 240, 0.88)')
    core.addColorStop(0.55, 'rgba(245, 220, 106, 0.45)')
    core.addColorStop(1, 'rgba(245, 220, 106, 0)')
    ctx.beginPath()
    ctx.arc(pos.x, pos.y, coreR, 0, Math.PI * 2)
    ctx.fillStyle = core
    ctx.fill()
    ctx.restore()
  }

  /** 圆满领域根节点核心亮斑（世界坐标，随缩放自然等比） */
  const drawStarlitRootCore = (
    ctx: CanvasRenderingContext2D,
    pos: { x: number; y: number },
    modelR: number,
    phase: number,
    alpha = 1
  ) => {
    if (alpha <= 0) return
    ctx.save()
    ctx.globalAlpha = alpha
    const pulse = reducedMotion ? 1 : 0.88 + 0.12 * Math.sin(phase)
    const coreR = modelR * (2.1 * pulse)
    const core = ctx.createRadialGradient(pos.x, pos.y, 0, pos.x, pos.y, coreR)
    core.addColorStop(0, graphPalette.glow.starlight)
    core.addColorStop(0.5, 'rgba(245, 220, 106, 0.3)')
    core.addColorStop(1, 'rgba(245, 220, 106, 0)')
    ctx.beginPath()
    ctx.arc(pos.x, pos.y, coreR, 0, Math.PI * 2)
    ctx.fillStyle = core
    ctx.fill()

    ctx.beginPath()
    ctx.arc(pos.x, pos.y, modelR + 1.5 * pulse, 0, Math.PI * 2)
    ctx.strokeStyle = reducedMotion
      ? 'rgba(201, 162, 39, 0.5)'
      : `rgba(201, 162, 39, ${0.35 + 0.2 * Math.sin(phase * 1.2)})`
    ctx.lineWidth = 1.75
    ctx.stroke()
    ctx.restore()
  }

  const drawStarlitClusterGlows = (ctx: CanvasRenderingContext2D) => {
    if (!starlitRootIds.size) return
    const positions = network.getPositions()
    const viewScale = network.getScale()
    const clusterMix = multiDomain ? 1 - starlitFarMix(viewScale) : 1
    if (clusterMix <= 0) return

    for (const rootId of starlitRootIds) {
      const domainId = rootId.slice('domain:'.length)
      const cluster = domainClusterIds.get(domainId)
      const rootPos = positions[rootId]
      if (!cluster?.length || !rootPos) continue
      const haloR = computeStarlitClusterRadius(rootPos, cluster, positions)
      if (graphTheme === 'paper') {
        drawStarlitInkDomainCluster(ctx, rootPos, haloR, clusterMix, hashId(rootId), viewScale)
      } else {
        drawStarlitDomainCluster(ctx, rootPos, haloR, pulsePhase, clusterMix)
      }
    }
  }

  const drawGlows = (ctx: CanvasRenderingContext2D) => {
    const positions = network.getPositions()
    const scale = network.getScale()
    const pulse = reducedMotion ? 1 : 0.85 + 0.15 * Math.sin(pulsePhase)

    drawConstellationAtmosphere(ctx, positions, scale)

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
      const MIN_GALAXY_DOMAIN_R = 10
      const baseR = (currentLod === 'galaxy' && node.nodeRole === 'domain')
        ? Math.max(rawBaseR, MIN_GALAXY_DOMAIN_R)
        : rawBaseR

      if (starlitRootIds.has(node.id)) {
        const farMix = multiDomain ? starlitFarMix(scale) : 0
        const modelR = node.size ?? domainBaseSizeById.get(node.id) ?? 28
        const phase = pulsePhase + (hashId(node.id) % 628) / 100
        const inkSeed = hashId(node.id)
        if (graphTheme === 'paper') {
          if (farMix > 0) {
            drawStarlitInkGalaxyWash(ctx, pos, scale, farMix, inkSeed)
          }
          if (farMix < 1) {
            drawStarlitInkRootWash(ctx, pos, modelR, 1 - farMix, inkSeed, scale)
          }
        } else {
          if (farMix > 0) {
            drawStarlitGalaxyGlow(ctx, pos, phase, scale, farMix)
          }
          if (farMix < 1) {
            drawStarlitRootCore(ctx, pos, modelR, phase, 1 - farMix)
          }
        }
        continue
      }

      // 主题氛围装饰：宣纸 = 领域墨团旁的墨点；星空 = 领域恒星背景光晕
      if (graphTheme === 'paper' && node.nodeRole === 'domain' && isPaperDomainLit(node.id)) {
        drawInkSpeckles(ctx, pos, baseR, node.id, scale)
      } else if (graphTheme === 'sky' && node.nodeRole === 'domain' && currentLod === 'node') {
        drawSoftSkyStarGlow(
          ctx,
          pos,
          baseR,
          pulsePhase + (hashId(node.id) % 628) / 100,
          0.55
        )
      }

      // galaxy / constellation LOD：domain 星点光晕
      if (
        (currentLod === 'galaxy' || currentLod === 'constellation') &&
        node.nodeRole === 'domain'
      ) {
        if (graphTheme === 'paper') {
          const visual = resolvePaperNodeVisual(node.id, 'domain')
          if (visual === 'lit') {
            const spread = baseR * (currentLod === 'galaxy' ? 5.8 : 4.2)
            drawOrganicInkWash(ctx, pos.x, pos.y, spread, hashId(node.id), 1, 0.85, scale)
          }
          drawPaperStamp(
            ctx,
            pos.x,
            pos.y,
            paperInkDrawRadius(node.size ?? 28, scale, 'domain'),
            hashId(node.id),
            'domain',
            visual,
            1,
            scale
          )
        } else {
          const intensity = currentLod === 'galaxy' ? 1 : 0.78
          drawSoftSkyStarGlow(
            ctx,
            pos,
            baseR,
            pulsePhase + (hashId(node.id) % 628) / 100,
            intensity
          )
        }
        continue
      }

      // constellation LOD：模块节点淡雾，强化「星团」感
      if (currentLod === 'constellation' && node.nodeRole === 'module' && graphTheme === 'sky') {
        const mistR = baseR * (3.6 + 0.5 * Math.sin(pulsePhase + hashId(node.id) % 50))
        const mist = ctx.createRadialGradient(pos.x, pos.y, 0, pos.x, pos.y, mistR)
        mist.addColorStop(0, 'rgba(200, 220, 255, 0.22)')
        mist.addColorStop(0.45, 'rgba(160, 190, 245, 0.08)')
        mist.addColorStop(1, 'rgba(120, 160, 230, 0)')
        ctx.beginPath()
        ctx.arc(pos.x, pos.y, mistR, 0, Math.PI * 2)
        ctx.fillStyle = mist
        ctx.fill()
        continue
      }

      // 宣纸：canvas 图章（vis 圆点已透明；圆满根节点在 starlit 分支单独绘制）
      if (graphTheme === 'paper' && !starlitRootIds.has(node.id)) {
        if (currentLod === 'node') {
          drawPaperNodeStamp(ctx, node, pos, scale)
        } else if (
          currentLod === 'constellation' &&
          (node.nodeRole === 'module' || node.nodeRole === 'topic')
        ) {
          drawPaperNodeStamp(ctx, node, pos, scale)
        }
      }

      const tier = glowById.get(node.id)
      if (!tier || tier === 'starlight') continue
      const mul = tier === 'focus' ? 2.8 * pulse : tier === 'active' ? 2.4 * pulse : 2.5 * pulse
      const outerR = baseR * mul

      if (graphTheme === 'paper') continue

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

  network.on('beforeDrawing', (ctx) => {
    drawStarlitClusterGlows(ctx as CanvasRenderingContext2D)
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
    const rootId = `domain:${domainId}`
    const animDuration = reducedMotion ? 0 : 400
    const animation = reducedMotion
      ? false
      : { duration: animDuration, easingFunction: 'easeInOutQuad' as const }
    const scale = network.getScale()
    // 远景 fit 整簇往往几乎不放大；直接 focus 领域根并拉到节点层可读尺度
    if (currentLod === 'galaxy' || scale < 0.2) {
      const { constellationMax } = lodThresholds(graphTheme)
      network.focus(rootId, {
        scale: Math.max(0.35, constellationMax * 2.5),
        animation,
      })
    } else {
      network.fit({ nodes: cluster, animation })
    }
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

  const persistDomainLayoutSnapshot = () => {
    const domainNodeIds = nodes.getIds().map(String).filter((id) => id.startsWith('domain:'))
    if (!domainNodeIds.length) return
    persistGraphLayoutFromNetwork(graphDomainIds, domainNodeIds, (nodeIds) =>
      network.getPositions(nodeIds)
    )
  }

  network.on('dragEnd', (params) => {
    const dragIds = (params.nodes ?? []) as string[]
    if (!dragIds.length) return
    const positions = network.getPositions(dragIds)
    const draggedDomain = dragIds.some((id) => String(id).startsWith('domain:'))
    nodes.update(
      dragIds.map((id) => ({
        id,
        fixed: { x: true, y: true },
        x: positions[id]?.x,
        y: positions[id]?.y,
      }))
    )
    if (draggedDomain) {
      persistDomainLayoutSnapshot()
    }
    freezeAfterSettle()
  })

  // selectable:false 时 click/doubleClick 的 params.nodes 来自 selection（常为空），
  // 只有拖过节点才会被旁路选中。命中需用 pointer 坐标查节点。
  // 远景全景层会把领域画成至少 ~10px 的光点，但 vis 命中盒按 size*scale，极远时不足 1px，
  // 需按屏幕像素扩大命中，否则「看得见点不中」。
  const hitNodeId = (params: { pointer?: { DOM?: { x: number; y: number } } }): string | null => {
    const dom = params.pointer?.DOM
    if (!dom) return null
    const direct = network.getNodeAt(dom)
    if (direct != null) return String(direct)

    const scale = network.getScale()
    const positions = network.getPositions()
    const galaxyPad = currentLod === 'galaxy' ? 16 : 0
    const farPad = scale < 0.25 ? 10 : 4
    let bestId: string | null = null
    let bestDist = Infinity

    for (const node of nodes.get()) {
      if (node.hidden) continue
      const pos = positions[node.id]
      if (!pos) continue
      const screen = network.canvasToDOM(pos)
      const dist = Math.hypot(screen.x - dom.x, screen.y - dom.y)
      const modelR = (node.size ?? 12) * scale
      const visualMin =
        currentLod === 'galaxy' && node.nodeRole === 'domain' ? 14 : 0
      const radius = Math.max(modelR, visualMin) + galaxyPad + farPad
      if (dist <= radius && dist < bestDist) {
        bestDist = dist
        bestId = String(node.id)
      }
    }
    return bestId
  }

  // 领域/主题：单击定位、双击进入。不用 vis 的 doubleClick 导航——Hammer 偶发把单击认成
  // doubletap，会误跳转。改为在两次 click 间隔内自行判定。
  const DBLCLICK_MS = 320
  type PendingGraphClick =
    | { kind: 'domain'; domainId: string; timer: number }
    | { kind: 'topic'; domainId: string; nodeKey: string; layerKey: string; timer: number }
  let pendingClick: PendingGraphClick | null = null

  const clearPendingClick = () => {
    if (!pendingClick) return
    window.clearTimeout(pendingClick.timer)
    pendingClick = null
  }

  const armPendingClick = (pending: PendingGraphClick, run: () => void) => {
    clearPendingClick()
    const timer = window.setTimeout(() => {
      pendingClick = null
      run()
    }, DBLCLICK_MS)
    pendingClick = { ...pending, timer }
  }

  network.on('click', (params) => {
    network.unselectAll()
    const id = hitNodeId(params)
    if (!id) {
      clearPendingClick()
      return
    }
    const item = nodes.get(id)
    if (!item) {
      clearPendingClick()
      return
    }

    if (id.startsWith('domain:') && item.domainId) {
      const domainId = item.domainId
      if (pendingClick?.kind === 'domain' && pendingClick.domainId === domainId) {
        clearPendingClick()
        onDomainClick?.(domainId)
        return
      }
      armPendingClick({ kind: 'domain', domainId, timer: 0 }, () => focusDomain(domainId))
      return
    }

    if (id.startsWith('module:')) {
      clearPendingClick()
      const cluster = moduleClusterIds.get(id)
      if (cluster?.length) {
        network.fit({
          nodes: cluster,
          animation: reducedMotion ? false : { duration: 350, easingFunction: 'easeInOutQuad' },
        })
      }
      return
    }

    if (!id.startsWith('topic:')) {
      clearPendingClick()
      return
    }
    if (!item.nodeKey || !item.layerKey || !item.domainId) {
      clearPendingClick()
      return
    }

    const topicDomainId = item.domainId
    const nodeKey = item.nodeKey
    const layerKey = item.layerKey
    if (
      pendingClick?.kind === 'topic' &&
      pendingClick.domainId === topicDomainId &&
      pendingClick.nodeKey === nodeKey
    ) {
      clearPendingClick()
      onTopicClick(topicDomainId, nodeKey, layerKey)
      return
    }
    armPendingClick({ kind: 'topic', domainId: topicDomainId, nodeKey, layerKey, timer: 0 }, () => {
      network.focus(id, {
        scale: 1.35,
        animation: reducedMotion ? false : { duration: 300, easingFunction: 'easeInOutQuad' },
      })
    })
  })

  network.on('doubleClick', (params) => {
    const id = hitNodeId(params)
    // 领域/主题进课已在第二次 click 处理；忽略 vis doubleClick，避免误跳转
    if (id?.startsWith('domain:') || id?.startsWith('topic:')) return
    if (!id) return
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
      clearPendingClick()
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
