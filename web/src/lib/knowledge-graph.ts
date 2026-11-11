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
import { loadGraphLayout, saveGraphLayout } from './graph-layout-persist'
import { armSpans, fieldNameAnchors, galaxyOuter, packConstellations, starChart } from './constellation-packing'
import {
  drawConstellationSegments,
  drawConstellationStars,
  drawSkyBackdrop,
  buildGalaxyDust,
  drawFieldNames,
  drawGalaxy,
  drawInkBlooms,
  drawStarChart,
  type FieldNameLabel,
  type InkBloom,
  type ConstellationName,
  type DrawableSegment,
  type DrawableStar,
  type StarVisualState,
} from './constellation-render'
import { lodFromScale, lodThresholds, type GraphLodLevel } from './graph-lod'
import { preloadBrushes } from './paper-brush'
import { PENDING_NODE_OPACITY } from './graph-progress-visual'
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

function topicVisualState(status: NodeProgressStatus, unmet: number, recommended: boolean): StarVisualState {
  if (status === 'completed') return 'done'
  if (status === 'in_progress') return 'progress'
  if (unmet === 0) return recommended ? 'recommended' : 'available'
  return 'locked'
}

function sessionGet(key: string): string | null {
  try {
    return sessionStorage.getItem(key)
  } catch {
    return null
  }
}

function sessionSet(key: string, value: string): void {
  try {
    sessionStorage.setItem(key, value)
  } catch {
    /* private mode */
  }
}

function silenceNode(node: GraphNode): GraphNode {
  return {
    ...node,
    label: '',
    font: { ...node.font, size: 0, strokeWidth: 0, color: 'rgba(0,0,0,0)' },
    color: paperInkNodeColor(),
    borderWidth: 0,
    borderWidthSelected: 0,
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
  createdAt?: string
  field?: string
  nodeTotal?: number
  completed?: number
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

  // 宣纸是盖天星图、星空是旋臂，两种布局各存各的拖拽偏移
  const chartMode = graphTheme === 'paper'
  const layoutVariant = chartMode ? 'chart' : ''
  const graphDomainIds = domains.map((d) => d.domainId)
  const savedOffsets: Record<string, { x: number; y: number }> = loadGraphLayout(graphDomainIds, layoutVariant) ?? {}

  const nodes = new DataSet<GraphNode>([])
  const moduleClusterIds = new Map<string, string[]>()
  const domainClusterIds = new Map<string, string[]>()
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

  const packed = packConstellations(
    domains.map((d) => ({
      domainId: d.domainId,
      slug: d.slug,
      name: d.tree.domainName?.trim() || '课程',
      createdAt: d.createdAt,
      field: d.field,
      nodeTotal: d.nodeTotal,
      completed: d.completed,
      started: [...d.progressMap.values()].some((p) => normalizeStatus(p.status) !== 'pending'),
      tree: d.tree,
    })),
    chartMode ? 'chart' : 'spiral'
  )
  const packedById = new Map(packed.map((item) => [item.domainId, item]))
  const fieldNames: FieldNameLabel[] = chartMode
    ? []
    : fieldNameAnchors(packed).map((item) => ({ name: item.field, x: item.x, y: item.y, slot: item.slot }))
  const masteryById = new Map(
    domains.map((d) => [d.domainId, d.nodeTotal ? Math.min(1, (d.completed ?? 0) / d.nodeTotal) : 0])
  )
  /** 课程根星相对星座中心的位移，拖动后据根星当前位置找回墨晕中心 */
  const rootDeltaById = new Map<string, { x: number; y: number }>()
  const clusterSlots = new Map(packed.map((item) => [item.domainId, item.slot]))
  const drawStars: DrawableStar[] = []
  const drawSegments: DrawableSegment[] = []
  const constellationNames: ConstellationName[] = []
  const prereqTargets = new Map<string, string[]>()
  const autoRootById = new Map<string, { x: number; y: number }>()
  const litStarIds = new Set<string>()
  const burstIds = new Set<string>()
  let burstUntil = 0

  for (let di = 0; di < domains.length; di++) {
    const entry = domains[di]!
    const { domainId, tree, progressMap, focusKeys } = entry
    const packed = packedById.get(domainId)
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
    const { domainComplete, moduleLit } = computeDomainGraphProgress(
      progressMap,
      graphModules,
      layerByNode,
      titles
    )
    const domainCluster: string[] = [rootId]
    const rootBaseSize = multiDomain ? 28 : 32

    const baseLayout = packed?.layout
    const offset = savedOffsets[rootId] ?? { x: 0, y: 0 }
    const origin = { x: (packed?.cx ?? 0) + offset.x, y: (packed?.cy ?? 0) + offset.y }
    autoRootById.set(domainId, { x: packed?.cx ?? 0, y: packed?.cy ?? 0 })
    const activeLayout = baseLayout
    const placeOf = new Map((activeLayout?.stars ?? []).map((star) => [star.id, { x: star.x + origin.x, y: star.y + origin.y }]))
    const angleOfStar = new Map((activeLayout?.stars ?? []).map((star) => [star.id, star.labelAngle]))
    const rootPos = placeOf.get(rootId) ?? origin
    rootDeltaById.set(domainId, { x: rootPos.x - origin.x, y: rootPos.y - origin.y })
    nodes.add(
      silenceNode({
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
        x: rootPos.x,
        y: rootPos.y,
        fixed: false,
      })
    )
    if (domainComplete) litStarIds.add(rootId)
    const rootStar: DrawableStar = {
      id: rootId,
      role: 'domain',
      label: domainTitle,
      labelAngle: angleOfStar.get(rootId) ?? -Math.PI / 2,
      state: domainComplete ? 'done' : 'root',
      domainId,
    }
    drawStars.push(rootStar)
    const layoutStars = (activeLayout?.stars ?? []).filter((star) => star.role !== 'topic')
    const minX = Math.min(0, ...layoutStars.map((star) => star.x))
    const maxX = Math.max(0, ...layoutStars.map((star) => star.x))
    const maxY = Math.max(0, ...layoutStars.map((star) => star.y))
    constellationNames.push({
      text: domainTitle,
      anchorId: rootId,
      complete: domainComplete,
      dx: (minX + maxX) / 2,
      dy: maxY,
      weight: layoutStars.length,
    })
    let recommendedKey: string | null = null
    for (const layer of tree.layers) {
      for (const node of layer.nodes) {
        if (!validKeys.has(node.key)) continue
        if (normalizeStatus(progressMap.get(node.key)?.status) !== 'pending') continue
        if (unmetPrerequisiteTitles(node, progressMap, titles).length > 0) continue
        recommendedKey = node.key
        break
      }
      if (recommendedKey) break
    }
    const completedNow: string[] = []
    for (const key of validKeys) {
      if (normalizeStatus(progressMap.get(key)?.status) === 'completed') completedNow.push(key)
    }
    rootStar.mastery = completedNow.length / Math.max(1, validKeys.size)
    const litKey = `regulus:graph-lit:${domainId}`
    const previousLit = sessionGet(litKey)
    if (previousLit) {
      try {
        const prev = new Set<string>(JSON.parse(previousLit) as string[])
        const allowBurst = !window.matchMedia('(prefers-reduced-motion: reduce)').matches
        for (const key of completedNow) {
          if (!prev.has(key) && allowBurst) {
            burstIds.add(`topic:${domainId}:${key}`)
            burstUntil = 18
          }
        }
      } catch {
        /* ignore broken session */
      }
    }
    sessionSet(litKey, JSON.stringify(completedNow))

    const topicMeta = new Map<string, { topicId: string; layerKey: string; moduleKey: string }>()

    for (const mod of graphModules) {
      const moduleId = `module:${domainId}:${mod.key}`
      const clusterIds = [moduleId]
      const modulePos = placeOf.get(moduleId) ?? rootPos
      const moduleComplete = moduleLit.get(mod.key) ?? false
      const validModuleKeys = mod.nodes.filter(
        (k) => layerByNode.has(k) && titles.has(k)
      )
      nodes.add(
        silenceNode({
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
            completionRatio: moduleComplete ? 1 : 0,
            topicCount: validModuleKeys.length,
          }),
          x: modulePos.x,
          y: modulePos.y,
          fixed: { x: true, y: true },
        })
      )
      if (moduleComplete) litStarIds.add(moduleId)
      drawStars.push({
        id: moduleId,
        role: 'module',
        label: mod.label,
        labelAngle: angleOfStar.get(moduleId) ?? Math.PI / 2,
        state: moduleComplete ? 'done' : 'module',
        domainId,
        mastery:
          validModuleKeys.filter((k) => normalizeStatus(progressMap.get(k)?.status) === 'completed').length /
          Math.max(1, validModuleKeys.length),
      })

      validModuleKeys.forEach((nodeKey) => {
        const layerKey = layerByNode.get(nodeKey)!
        const title = titles.get(nodeKey)!
        const topicId = `topic:${domainId}:${nodeKey}`
        const status = normalizeStatus(progressMap.get(nodeKey)?.status)
        const focused = focusKeys.has(nodeKey)
        const topicPos = placeOf.get(topicId) ?? modulePos
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
        nodes.add(silenceNode({ ...topicNode, x: topicPos.x, y: topicPos.y, fixed: { x: true, y: true } }))
        const state = topicVisualState(status, unmetPrereqs.length, nodeKey === recommendedKey)
        if (status === 'completed') litStarIds.add(topicId)
        drawStars.push({
          id: topicId,
          role: 'topic',
          label: title,
          labelAngle: angleOfStar.get(topicId) ?? Math.PI / 2,
          state,
          domainId,
          mastery: progressMap.get(nodeKey)?.mastery,
        })
        clusterIds.push(topicId)
        domainCluster.push(topicId)
        topicMeta.set(nodeKey, { topicId, layerKey, moduleKey: mod.key })
      })

      moduleClusterIds.set(moduleId, clusterIds)
      domainCluster.push(moduleId)
    }

    for (const layer of tree.layers) {
      for (const node of layer.nodes) {
        if (!node.requires?.length) continue
        const curr = topicMeta.get(node.key)?.topicId
        if (!curr) continue
        const ownModule = topicMeta.get(node.key)?.moduleKey
        const unmet = node.requires.filter((req) => {
          if (progressMap.get(req)?.status === 'completed') return false
          const reqMod = topicMeta.get(req)?.moduleKey
          return reqMod !== undefined && reqMod !== ownModule
        })
        if (unmet.length) prereqTargets.set(curr, unmet.map((req) => `topic:${domainId}:${req}`))
      }
    }

    const recommendedId = recommendedKey ? `topic:${domainId}:${recommendedKey}` : ''
    for (const segment of activeLayout?.segments ?? []) {
      const both = litStarIds.has(segment.from) && litStarIds.has(segment.to)
      const either = litStarIds.has(segment.from) || litStarIds.has(segment.to)
      drawSegments.push({
        from: segment.from,
        to: segment.to,
        kind: segment.kind,
        tone: segment.to === recommendedId ? 'flow' : both ? 'gold' : either ? 'mid' : 'dim',
      })
    }

    domainClusterIds.set(domainId, domainCluster)
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
      if (anchor?.moduleKey && nodes.get(`module:${parentDomainId}:${anchor.moduleKey}`)) {
        from = `module:${parentDomainId}:${anchor.moduleKey}`
      } else if (anchor?.nodeKey && nodes.get(`topic:${parentDomainId}:${anchor.nodeKey}`)) {
        from = `topic:${parentDomainId}:${anchor.nodeKey}`
      }
      if (!nodes.get(from)) continue
      drawSegments.push({ from, to, kind: 'parent', tone: 'parent' })
    }
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
      const hidden = level === 'galaxy' && role === 'topic'
      updates.push({ id: node.id, hidden })
    }
    nodes.update(updates)
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
    physics: { enabled: false },
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
  if (graphTheme === 'paper') void preloadBrushes().then(() => network.redraw())

  network.on('hoverNode', (params) => {
    hoveredNodeId = params.node as string
    network.redraw()
  })
  network.on('blurNode', () => {
    hoveredNodeId = null
    network.redraw()
  })

  const hoverPrereqSegments = (): DrawableSegment[] => {
    if (!hoveredNodeId) return []
    const targets = prereqTargets.get(hoveredNodeId)
    if (!targets?.length) return []
    return targets.map((from) => ({ from, to: hoveredNodeId as string, kind: 'prereq', tone: 'prereq' }))
  }

  const allStarIds = drawStars.map((star) => star.id)
  const chart = multiDomain && chartMode ? starChart(packed) : null
  const galaxyDust = multiDomain && !chartMode ? buildGalaxyDust(galaxyOuter(packed), armSpans(packed)) : null
  /** 全景时要装进视野的半径；单课程为 0，不按全景处理 */
  const galaxyExtent = chart ? chart.outer * 1.12 : (galaxyDust?.extent ?? 0)
  const inkBlooms = (): InkBloom[] => {
    const roots = network.getPositions(domainRootIds)
    return domains.flatMap((d) => {
      const root = roots[`domain:${d.domainId}`]
      const delta = rootDeltaById.get(d.domainId)
      const layout = packedById.get(d.domainId)?.layout
      if (!root || !delta || !layout) return []
      return [{ id: d.domainId, x: root.x - delta.x, y: root.y - delta.y, radius: layout.radius, ratio: masteryById.get(d.domainId) ?? 0 }]
    })
  }
  const galaxyFade = (scale: number): number => {
    const { galaxyMax } = lodThresholds(graphTheme)
    const full = galaxyMax * 0.35
    return Math.max(0, Math.min(1, (galaxyMax - scale) / (galaxyMax - full)))
  }
  network.on('beforeDrawing', (ctx) => {
    const canvas = ctx as CanvasRenderingContext2D
    const scale = network.getScale()
    const lod = lodFromScale(scale, multiDomain, graphTheme)
    if (lod !== currentLod) {
      // 动画缩放途中层级也要跟上；vis 节点的显隐不能在绘制回调里改，推迟到下一帧
      currentLod = lod
      if (lodRaf) cancelAnimationFrame(lodRaf)
      lodRaf = requestAnimationFrame(() => {
        lodRaf = 0
        applyLod(currentLod)
      })
    }
    if (graphTheme === 'sky') {
      drawSkyBackdrop(canvas, network.getViewPosition(), scale)
      if (galaxyDust) drawGalaxy(canvas, galaxyDust, scale, galaxyFade(scale))
    } else if (chart) {
      drawInkBlooms(canvas, inkBlooms(), galaxyFade(scale))
      drawStarChart(canvas, chart, scale, galaxyFade(scale))
    }
    drawConstellationSegments({
      reducedMotion,
      ctx: canvas,
      positions: network.getPositions(allStarIds),
      stars: drawStars,
      segments: drawSegments.concat(hoverPrereqSegments()),
      theme: graphTheme,
      lod: currentLod,
      scale: network.getScale(),
      phase: pulsePhase,
      burstIds: burstUntil > 0 ? burstIds : undefined,
      galaxyFade: multiDomain ? galaxyFade(scale) : 0,
    })
  })

  network.on('afterDrawing', (ctx) => {
    drawConstellationStars({
      ctx: ctx as CanvasRenderingContext2D,
      positions: network.getPositions(allStarIds),
      stars: drawStars,
      names: constellationNames,
      theme: graphTheme,
      lod: currentLod,
      scale: network.getScale(),
      phase: pulsePhase,
      hoveredId: hoveredNodeId,
      reducedMotion,
      galaxyFade: multiDomain ? galaxyFade(network.getScale()) : 0,
      clusterSlots: multiDomain ? clusterSlots : undefined,
      burstIds: burstUntil > 0 ? burstIds : undefined,
    })
    if (currentLod === 'galaxy' && fieldNames.length) {
      drawFieldNames(ctx as CanvasRenderingContext2D, fieldNames, network.getScale())
    }
  })

  const FRAME_MS = 34
  let lastTick = 0
  let animating = false
  const motionNeeded = () =>
    !reducedMotion &&
    (drawSegments.some((seg) => seg.tone === 'flow') ||
      drawStars.some((star) => star.state === 'progress') ||
      burstUntil > 0)

  const stopAnim = () => {
    animating = false
    if (rafId) cancelAnimationFrame(rafId)
    rafId = 0
  }

  const tick = (now: number) => {
    if (!animating) return
    if (document.hidden || !motionNeeded()) {
      stopAnim()
      return
    }
    if (now - lastTick < FRAME_MS) {
      rafId = requestAnimationFrame(tick)
      return
    }
    lastTick = now
    pulsePhase += 0.08
    if (burstUntil > 0) burstUntil -= 1
    network.redraw()
    rafId = requestAnimationFrame(tick)
  }

  const startAnim = () => {
    if (animating || !motionNeeded() || document.hidden) return
    animating = true
    lastTick = 0
    rafId = requestAnimationFrame(tick)
  }

  const onVisibility = () => {
    if (document.hidden) stopAnim()
    else startAnim()
  }
  document.addEventListener('visibilitychange', onVisibility)
  startAnim()

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


  let dragCluster: Array<{ id: string; dx: number; dy: number }> = []
  let dragRootId = ''
  let dragFrame = 0

  const placeCluster = (rootX: number, rootY: number) => {
    if (!dragCluster.length) return
    nodes.update(
      dragCluster.map((item) => ({
        id: item.id,
        x: rootX + item.dx,
        y: rootY + item.dy,
      }))
    )
  }

  network.on('dragStart', (params) => {
    const dragIds = ((params.nodes ?? []) as string[]).map(String)
    const rootId = dragIds.find((id) => id.startsWith('domain:')) ?? ''
    if (!rootId) return
    dragRootId = rootId
    const positions = network.getPositions()
    const root = positions[rootId]
    if (!root) return
    const domainId = rootId.slice('domain:'.length)
    dragCluster = (domainClusterIds.get(domainId) ?? [])
      .filter((id) => id !== rootId && positions[id])
      .map((id) => ({
        id,
        dx: positions[id]!.x - root.x,
        dy: positions[id]!.y - root.y,
      }))
  })

  network.on('dragging', (params) => {
    const rootId = String((params.nodes ?? [])[0] ?? '')
    if (rootId !== dragRootId || !dragCluster.length) return
    if (dragFrame) return
    dragFrame = requestAnimationFrame(() => {
      dragFrame = 0
      const root = network.getPositions([dragRootId])[dragRootId]
      if (root) placeCluster(root.x, root.y)
    })
  })

  network.on('dragEnd', (params) => {
    const dragIds = ((params.nodes ?? []) as string[]).map(String)
    const rootId = dragIds.find((id) => id.startsWith('domain:'))
    if (dragFrame) {
      cancelAnimationFrame(dragFrame)
      dragFrame = 0
    }
    if (!rootId) {
      dragCluster = []
      dragRootId = ''
      return
    }
    const root = network.getPositions([rootId])[rootId]
    if (root) placeCluster(root.x, root.y)
    const domainId = rootId.slice('domain:'.length)
    const auto = autoRootById.get(domainId) ?? { x: 0, y: 0 }
    if (root) savedOffsets[rootId] = { x: root.x - auto.x, y: root.y - auto.y }
    saveGraphLayout(graphDomainIds, savedOffsets, layoutVariant)
    dragCluster = []
    dragRootId = ''
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

  // 最远缩到整片银河（含外围尘埃）都在视野里，最近 MAX_SCALE；vis-network 没有内建缩放上下限
  const MAX_SCALE = 2.2
  const minScale = (): number => {
    if (!multiDomain) return 0.3
    const diameter = galaxyExtent * 2.5
    const fitScale = Math.min(container.clientWidth, container.clientHeight) / diameter
    return Math.min(0.3, Math.max(0.004, fitScale))
  }
  // vis 以指针为中心缩放：比例被拦住时它仍会平移视图，所以到边界后直接吞掉滚轮
  const onWheelCapture = (e: WheelEvent) => {
    const scale = network.getScale()
    const zoomingOut = e.deltaY > 0
    if ((zoomingOut && scale <= minScale() * 1.001) || (!zoomingOut && scale >= MAX_SCALE * 0.999)) {
      e.preventDefault()
      e.stopPropagation()
    }
  }
  container.addEventListener('wheel', onWheelCapture, { capture: true, passive: false })

  const viewCenter = (): { x: number; y: number } => {
    if (multiDomain) return { x: 0, y: 0 }
    const pts = Object.values(network.getPositions(allStarIds))
    if (!pts.length) return { x: 0, y: 0 }
    return {
      x: pts.reduce((sum, p) => sum + p.x, 0) / pts.length,
      y: pts.reduce((sum, p) => sum + p.y, 0) / pts.length,
    }
  }
  const panLimit = (): number => {
    if (multiDomain) return galaxyExtent * 1.2
    const c = viewCenter()
    const pts = Object.values(network.getPositions(allStarIds))
    return Math.max(600, ...pts.map((p) => Math.hypot(p.x - c.x, p.y - c.y) + 400))
  }
  /** 视图中心跑出银河外沿时拉回边缘 */
  const keepInBounds = () => {
    const c = viewCenter()
    const view = network.getViewPosition()
    const dx = view.x - c.x
    const dy = view.y - c.y
    const dist = Math.hypot(dx, dy)
    const limit = panLimit()
    if (dist <= limit) {
      lastGoodView = { position: view, scale: network.getScale() }
      return
    }
    const position = { x: c.x + (dx / dist) * limit, y: c.y + (dy / dist) * limit }
    lastGoodView = { position, scale: network.getScale() }
    network.moveTo({
      position,
      scale: network.getScale(),
      animation: reducedMotion ? false : { duration: 280, easingFunction: 'easeOutQuad' },
    })
  }
  network.on('dragEnd', (params) => {
    if (!(params.nodes ?? []).length) keepInBounds()
  })

  let lastGoodView = { position: network.getViewPosition(), scale: network.getScale() }
  let clamping = false
  let boundsTimer = 0
  network.on('zoom', () => {
    const scale = network.getScale()
    const floor = minScale()
    if (!clamping && (scale < floor || scale > MAX_SCALE)) {
      clamping = true
      network.moveTo({ position: lastGoodView.position, scale: Math.min(MAX_SCALE, Math.max(floor, scale)) })
      clamping = false
    } else if (!clamping) {
      lastGoodView = { position: network.getViewPosition(), scale }
    }
    window.clearTimeout(boundsTimer)
    boundsTimer = window.setTimeout(keepInBounds, 220)
    if (lodRaf) cancelAnimationFrame(lodRaf)
    lodRaf = requestAnimationFrame(() => {
      lodRaf = 0
      syncLodFromZoom()
    })
  })

  const showOverview = (animate: boolean) => {
    const animation = animate && !reducedMotion ? { duration: 360, easingFunction: 'easeInOutQuad' as const } : false
    if (multiDomain) {
      network.moveTo({ position: { x: 0, y: 0 }, scale: minScale(), animation })
    } else {
      network.fit({ animation })
    }
  }

  network.fit({ animation: false })
  lastGoodView = { position: network.getViewPosition(), scale: network.getScale() }
  applyLod(lodFromScale(network.getScale(), multiDomain, graphTheme))

  return {
    destroy: () => {
      container.removeEventListener('wheel', onWheelCapture, { capture: true })
      window.clearTimeout(boundsTimer)
      document.removeEventListener('visibilitychange', onVisibility)
      if (rafId) cancelAnimationFrame(rafId)
      if (lodRaf) cancelAnimationFrame(lodRaf)
      if (dragFrame) cancelAnimationFrame(dragFrame)
      clearPendingClick()
      network.destroy()
    },
    fit: () => showOverview(true),
    focusDomain,
    getLodLevel: () => currentLod,
  }
}
