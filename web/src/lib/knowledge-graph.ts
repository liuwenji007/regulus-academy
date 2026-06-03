import { Network, type Data, type Options } from 'vis-network'
import { DataSet } from 'vis-data'
import 'vis-network/styles/vis-network.css'
import type { KnowledgeTree, UserProgress } from './api'
import {
  type GraphCanvasTheme,
  readGraphCanvasThemeFrom,
} from './graph-canvas-theme'
import { getGraphThemeTokens, type GraphLabelStyle, type GraphPalette } from './graph-theme-palette'
import { resolveGraphModules, nodeLayerKeyMap, nodeTitleMap, unmetPrerequisiteTitles } from './tree-normalize'

export type NodeProgressStatus = 'pending' | 'in_progress' | 'completed'

export interface KnowledgeGraphMount {
  destroy: () => void
  fit: () => void
  /** 将视图缩放到某一领域的全部节点 */
  focusDomain: (domainId: string) => void
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

function applyGraphTheme(theme: GraphCanvasTheme): void {
  const tokens = getGraphThemeTokens(theme)
  graphLabel = tokens.label
  graphPalette = tokens.palette
}

type GraphNode = {
  id: string
  label: string
  group: string
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
  const steady = { background: fill, border }
  return {
    id,
    label,
    group: starlit ? 'rootStarlit' : 'root',
    shape: 'dot',
    size,
    mass,
    font: labelFont(LABEL_SIZE.root, true),
    color: {
      background: fill,
      border,
      highlight: steady,
      hover: steady,
    },
    borderWidth: 2.5,
    borderWidthSelected: 2,
    chosen: { node: false, label: false },
    domainId,
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
      group: 'focus',
      shape: 'dot',
      size: 19,
      font: labelFont(LABEL_SIZE.topicFocus, true),
      color: {
        background: graphPalette.focus.fill,
        border: graphPalette.focus.border,
        highlight: { background: '#d96a32', border: '#ffffff' },
      },
      borderWidth: 3,
      nodeKey,
      layerKey,
      title: tooltipTitle,
      chosen: { node: false, label: false },
    }
  }

  if (status === 'completed') {
    return {
      id,
      label: short,
      group: 'completed',
      shape: 'dot',
      size: 16,
      font: labelFont(LABEL_SIZE.topic, true),
      color: {
        background: graphPalette.done.fill,
        border: graphPalette.done.border,
        highlight: { background: '#fff0a8', border: '#c9a227' },
      },
      borderWidth: 2.5,
      nodeKey,
      layerKey,
      title: tooltipTitle,
      chosen: { node: false, label: false },
    }
  }

  if (status === 'in_progress') {
    return {
      id,
      label: short,
      group: 'in_progress',
      shape: 'dot',
      size: 15,
      font: labelFont(LABEL_SIZE.topic, true),
      color: {
        background: graphPalette.active.fill,
        border: graphPalette.active.border,
        highlight: { background: '#e8753a', border: '#ffffff' },
      },
      borderWidth: 3,
      nodeKey,
      layerKey,
      title: tooltipTitle,
      chosen: { node: false, label: false },
    }
  }

  const pendingBorder =
    unmetPrereqs.length > 0 ? 'rgba(120, 113, 108, 0.55)' : graphPalette.pending.border

  return {
    id,
    label: short,
    group: 'pending',
    shape: 'dot',
    size: 12,
    font: labelFont(LABEL_SIZE.topicPending),
    color: {
      background: graphPalette.pending.fill,
      border: pendingBorder,
      highlight: { background: '#fff8f2', border: '#c45c26' },
    },
    borderWidth: unmetPrereqs.length > 0 ? 2 : 1.5,
    nodeKey,
    layerKey,
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
}): GraphNode {
  const { id, label, domainId, moduleKey, title, multiDomain, lit = false } = opts
  const short = label.length > 14 ? label.slice(0, 13) + '…' : label
  const palette = lit ? graphPalette.moduleLit : graphPalette.module
  const hover = lit
    ? { background: '#fff0a8', border: '#c9a227' }
    : { background: graphPalette.moduleHover.fill, border: graphPalette.moduleHover.border }
  return {
    id,
    label: short,
    group: lit ? 'moduleLit' : 'module',
    shape: 'dot',
    size: multiDomain ? 20 : 22,
    mass: multiDomain ? 3.5 : 3,
    font: labelFont(LABEL_SIZE.module, true),
    color: {
      background: palette.fill,
      border: palette.border,
      highlight: hover,
      hover,
    },
    borderWidth: 2.5,
    borderWidthSelected: 2,
    chosen: { node: false, label: false },
    domainId,
    moduleKey,
    title,
  }
}

function domainLayoutCenters(count: number): Array<{ x: number; y: number }> {
  if (count <= 1) return [{ x: 0, y: 0 }]
  const radius = 300 + Math.max(0, count - 2) * 70
  return Array.from({ length: count }, (_, i) => {
    const angle = (2 * Math.PI * i) / count - Math.PI / 2
    return { x: radius * Math.cos(angle), y: radius * Math.sin(angle) }
  })
}

function domainSeparationLength(domainCount: number): number {
  return 460 + Math.max(0, domainCount - 2) * 90
}

function moduleLayoutOffset(
  center: { x: number; y: number },
  moduleIndex: number,
  moduleCount: number,
  multiDomain: boolean
): { x: number; y: number } {
  if (moduleCount <= 1) {
    const dist = multiDomain ? 100 : 72
    return { x: center.x + dist, y: center.y }
  }
  const spread = multiDomain ? Math.PI * 0.75 : Math.PI * 0.62
  const angle = -spread / 2 + (spread * moduleIndex) / (moduleCount - 1)
  const dist = multiDomain ? 105 : 78
  return {
    x: center.x + dist * Math.cos(angle),
    y: center.y + dist * Math.sin(angle),
  }
}

/** 主题节点围绕模块扇形排布，单领域时更紧凑 */
function topicLayoutOffset(
  modPos: { x: number; y: number },
  topicIndex: number,
  topicCount: number,
  multiDomain: boolean
): { x: number; y: number } {
  const dist = multiDomain ? 58 : 46
  if (topicCount <= 1) {
    return { x: modPos.x + dist, y: modPos.y }
  }
  const spread = multiDomain ? Math.PI * 0.85 : Math.PI * 0.7
  const angle = -spread / 2 + (spread * topicIndex) / (topicCount - 1)
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
  const edges = new DataSet<{
    id: string
    from: string
    to: string
    length?: number
    dashes?: boolean | number[]
    color?: { color: string; highlight: string; opacity: number }
    width?: number
    smooth?: { enabled: boolean; type: string; roundness: number }
  }>([])

  const multiDomain = domains.length > 1
  const domainCenters = domainLayoutCenters(domains.length)
  const domainRootIds: string[] = []

  for (let di = 0; di < domains.length; di++) {
    const entry = domains[di]!
    const { domainId, tree, progressMap, focusKeys } = entry
    const center = domainCenters[di]!
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
    const { domainComplete, moduleLit } = computeDomainGraphProgress(
      progressMap,
      graphModules,
      layerByNode,
      titles
    )
    const domainCluster: string[] = [rootId]

    if (domainComplete) starlitRootIds.add(rootId)

    nodes.add({
      ...buildRootNode({
        id: rootId,
        label: rootLabel,
        size: multiDomain ? 28 : 32,
        mass: multiDomain ? 7 : 4,
        domainId,
        title: domainComplete
          ? `${domainTitle} · 本领域已全部学完`
          : `${domainTitle} · 点击查看课程列表`,
        starlit: domainComplete,
      }),
      x: center.x,
      y: center.y,
    })

    const topicMeta = new Map<string, { topicId: string; layerKey: string; moduleKey: string }>()

    for (let mi = 0; mi < graphModules.length; mi++) {
      const mod = graphModules[mi]!
      const moduleId = `module:${domainId}:${mod.key}`
      const clusterIds = [moduleId]
      const modPos = moduleLayoutOffset(center, mi, graphModules.length, multiDomain)

      const moduleComplete = moduleLit.get(mod.key) ?? false
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
        }),
        x: modPos.x,
        y: modPos.y,
      })

      if (moduleComplete) glowById.set(moduleId, 'done')

      edges.add({
        id: `e-dm-${domainId}-${mod.key}`,
        from: rootId,
        to: moduleId,
        length: multiDomain ? 175 : 88,
        color: { color: graphPalette.edge.domainModule, highlight: graphPalette.edge.highlight, opacity: 0.65 },
        width: 1.5,
        smooth: { enabled: true, type: 'continuous', roundness: 0.2 },
      })

      const validModuleKeys = mod.nodes.filter(
        (k) => layerByNode.has(k) && titles.has(k)
      )

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
          length: multiDomain ? 135 : 52,
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
        edges.add({
          id: `e-path-${domainId}-${mod.key}-${i}`,
          from: prev,
          to: curr,
          dashes: [5, 8],
          color: { color: graphPalette.edge.path, highlight: graphPalette.edge.highlight, opacity: 0.45 },
          width: 1.2,
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
          edges.add({
            id: `e-req-${domainId}-${req}-${node.key}`,
            from: prev,
            to: curr,
            length: multiDomain ? 118 : 68,
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
    const sepLen = domainSeparationLength(domains.length)
    for (let i = 0; i < domainRootIds.length; i++) {
      for (let j = i + 1; j < domainRootIds.length; j++) {
        edges.add({
          id: `e-domain-sep-${i}-${j}`,
          from: domainRootIds[i]!,
          to: domainRootIds[j]!,
          length: sepLen,
          color: { color: 'rgba(0,0,0,0)', highlight: 'rgba(0,0,0,0)', opacity: 0 },
          width: 0.01,
          smooth: { enabled: false, type: 'continuous', roundness: 0 },
        })
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

  const options: Options = {
    autoResize: true,
    interaction: {
      hover: true,
      tooltipDelay: 80,
      zoomView: true,
      dragView: true,
      dragNodes: true,
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
            gravitationalConstant: multiDomain ? -130 : -32,
            centralGravity: multiDomain ? 0.002 : 0.028,
            springLength: multiDomain ? 175 : 105,
            springConstant: multiDomain ? 0.032 : 0.055,
            damping: multiDomain ? 0.65 : 0.72,
            avoidOverlap: multiDomain ? 0.95 : 0.92,
          },
          stabilization: { iterations: multiDomain ? 380 : 260, updateInterval: 20 },
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

  const graphData: Data = {
    nodes: nodes.get().map((n) => ({ ...n, selectable: false })) as unknown as Data['nodes'],
    edges,
  }
  const network = new Network(container, graphData, options)

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

      if (node.group === 'root' && hoveredNodeId === node.id) {
        drawRootHover(ctx, node, pos, scale)
      }

      if (node.group === 'module' && hoveredNodeId === node.id) {
        drawModuleHover(ctx, node, pos, scale)
      }

      const baseR = (node.size ?? 12) * scale

      if (starlitRootIds.has(node.id)) {
        drawDomainStarlight(ctx, pos, baseR, pulsePhase)
        continue
      }

      const tier = glowById.get(node.id)
      if (!tier || tier === 'starlight') continue
      const mul = tier === 'focus' ? 2.8 * pulse : tier === 'active' ? 2.4 : 2.5
      const outerR = baseR * mul
      const inner =
        tier === 'focus' ? graphPalette.glow.focus : tier === 'active' ? graphPalette.glow.active : graphPalette.glow.done

      const midStop =
        tier === 'done' ? 'rgba(245, 220, 106, 0.14)' : 'rgba(196, 92, 38, 0.12)'
      const outerStop =
        tier === 'done' ? 'rgba(245, 220, 106, 0)' : 'rgba(196, 92, 38, 0)'

      const g = ctx.createRadialGradient(pos.x, pos.y, baseR * 0.2, pos.x, pos.y, outerR)
      g.addColorStop(0, inner)
      g.addColorStop(0.5, midStop)
      g.addColorStop(1, outerStop)

      ctx.save()
      ctx.beginPath()
      ctx.arc(pos.x, pos.y, outerR, 0, Math.PI * 2)
      ctx.fillStyle = g
      ctx.fill()

      if (tier === 'focus') {
        ctx.beginPath()
        ctx.arc(pos.x, pos.y, baseR + 4 * pulse, 0, Math.PI * 2)
        ctx.strokeStyle = `rgba(255, 255, 255, ${0.35 + 0.2 * Math.sin(pulsePhase)})`
        ctx.lineWidth = 2
        ctx.stroke()
      } else if (tier === 'active') {
        ctx.beginPath()
        ctx.arc(pos.x, pos.y, baseR + 5, 0, Math.PI * 2)
        ctx.strokeStyle = 'rgba(154, 63, 24, 0.75)'
        ctx.lineWidth = 2
        ctx.setLineDash([4, 4])
        ctx.stroke()
        ctx.setLineDash([])
      }
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
      pulsePhase += 0.06
      network.redraw()
      rafId = requestAnimationFrame(tick)
    }
    rafId = requestAnimationFrame(tick)
  }

  network.on('click', (params) => {
    network.unselectAll()
    if (params.nodes.length !== 1) return
    const id = params.nodes[0] as string
    const item = nodes.get(id)
    if (!item) return

    if (id.startsWith('domain:') && item.domainId && onDomainClick) {
      onDomainClick(item.domainId)
      return
    }

    if (id.startsWith('module:')) {
      const cluster = moduleClusterIds.get(id)
      if (cluster?.length) {
        network.fit({
          nodes: cluster,
          animation: { duration: 350, easingFunction: 'easeInOutQuad' },
        })
      }
      return
    }

    if (!id.startsWith('topic:')) return
    if (!item.nodeKey || !item.layerKey || !item.domainId) return
    onTopicClick(item.domainId, item.nodeKey, item.layerKey)
  })

  network.on('doubleClick', (params) => {
    if (params.nodes.length === 1) {
      network.focus(params.nodes[0], {
        scale: 1.35,
        animation: { duration: 300, easingFunction: 'easeInOutQuad' },
      })
    }
  })

  network.once('stabilizationIterationsDone', () => {
    if (!multiDomain && !reducedMotion) {
      network.setOptions({ physics: { enabled: false } })
    }
    network.fit({
      animation: reducedMotion ? false : { duration: 450, easingFunction: 'easeInOutQuad' },
    })
  })

  if (reducedMotion) {
    network.fit({ animation: false })
  }

  const focusDomain = (domainId: string) => {
    const cluster = domainClusterIds.get(domainId)
    if (!cluster?.length) return
    network.fit({
      nodes: cluster,
      animation: reducedMotion ? false : { duration: 400, easingFunction: 'easeInOutQuad' },
    })
  }

  return {
    destroy: () => {
      if (rafId) cancelAnimationFrame(rafId)
      network.destroy()
    },
    fit: () => network.fit({ animation: { duration: 300, easingFunction: 'easeInOutQuad' } }),
    focusDomain,
  }
}
