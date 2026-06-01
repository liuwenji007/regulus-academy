import { Network, type Data, type Options } from 'vis-network'
import { DataSet } from 'vis-data'
import 'vis-network/styles/vis-network.css'
import type { KnowledgeTree, UserProgress } from './api'
import { resolveGraphModules, nodeLayerKeyMap, nodeTitleMap } from './tree-normalize'

export type NodeProgressStatus = 'pending' | 'in_progress' | 'completed'

export interface KnowledgeGraphMount {
  destroy: () => void
  fit: () => void
}

/** 标签始终在圆点下方、落在浅色画布上，因此统一深色字 + 浅色描边 */
const LABEL = {
  text: '#1c1917',
  stroke: 'rgba(255, 252, 247, 0.96)',
}

/** 图谱专用色板 */
const PALETTE = {
  canvas: '#ebe6dc',
  root: { fill: '#292524', border: '#1c1917' },
  module: { fill: '#57534e', border: '#44403c' },
  pending: { fill: '#ffffff', border: '#c9c4bc' },
  /** 进行中：陶土橙，与已学会黄色拉开差距 */
  active: { fill: '#c45c26', border: '#9a3f18' },
  /** 已学会：柔和金黄 + 光晕 */
  done: { fill: '#f5dc6a', border: '#c9a227' },
  focus: { fill: '#c45c26', border: '#ffffff' },
  glow: {
    focus: 'rgba(196, 92, 38, 0.5)',
    active: 'rgba(196, 92, 38, 0.42)',
    done: 'rgba(245, 220, 106, 0.55)',
  },
  edge: {
    belong: 'rgba(28, 25, 23, 0.1)',
    path: 'rgba(196, 92, 38, 0.28)',
    highlight: '#c45c26',
  },
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

function labelFont(size = 12, bold = false) {
  return {
    size,
    color: LABEL.text,
    face: 'Inter, "PingFang SC", "Microsoft YaHei", sans-serif',
    strokeWidth: bold ? 5 : 4,
    strokeColor: LABEL.stroke,
    vadjust: 28,
    bold,
  }
}

function buildRootNode(opts: {
  id: string
  label: string
  size: number
  mass: number
  domainId: string
  title: string
}): GraphNode {
  const { id, label, size, mass, domainId, title } = opts
  const fill = PALETTE.root.fill
  const border = '#57534e'
  // 与默认态一致，避免 vis-network 选中/悬停时叠粗橙色描边
  const steady = { background: fill, border }
  return {
    id,
    label,
    group: 'root',
    shape: 'dot',
    size,
    mass,
    font: labelFont(14, true),
    color: {
      background: fill,
      border,
      highlight: steady,
      hover: steady,
    },
    borderWidth: 2,
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
}): GraphNode {
  const { id, title, status, focused, nodeKey, layerKey } = opts
  const short = title.length > 18 ? title.slice(0, 17) + '…' : title

  if (focused) {
    return {
      id,
      label: short,
      group: 'focus',
      shape: 'dot',
      size: 19,
      font: labelFont(13, true),
      color: {
        background: PALETTE.focus.fill,
        border: PALETTE.focus.border,
        highlight: { background: '#d96a32', border: '#ffffff' },
      },
      borderWidth: 3,
      nodeKey,
      layerKey,
      title,
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
      font: labelFont(12, true),
      color: {
        background: PALETTE.done.fill,
        border: PALETTE.done.border,
        highlight: { background: '#fff0a8', border: '#c9a227' },
      },
      borderWidth: 2.5,
      nodeKey,
      layerKey,
      title,
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
      font: labelFont(12, true),
      color: {
        background: PALETTE.active.fill,
        border: PALETTE.active.border,
        highlight: { background: '#e8753a', border: '#ffffff' },
      },
      borderWidth: 3,
      nodeKey,
      layerKey,
      title,
      chosen: { node: false, label: false },
    }
  }

  return {
    id,
    label: short,
    group: 'pending',
    shape: 'dot',
    size: 11,
    font: labelFont(11),
    color: {
      background: PALETTE.pending.fill,
      border: PALETTE.pending.border,
      highlight: { background: '#fff8f2', border: '#c45c26' },
    },
    borderWidth: 1.5,
    nodeKey,
    layerKey,
    title,
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
}): GraphNode {
  const { id, label, domainId, moduleKey, title, multiDomain } = opts
  const short = label.length > 14 ? label.slice(0, 13) + '…' : label
  const steady = { background: PALETTE.module.fill, border: PALETTE.module.border }
  return {
    id,
    label: short,
    group: 'module',
    shape: 'dot',
    size: multiDomain ? 20 : 22,
    mass: multiDomain ? 3.5 : 3,
    font: labelFont(12, true),
    color: {
      background: PALETTE.module.fill,
      border: PALETTE.module.border,
      highlight: steady,
      hover: steady,
    },
    borderWidth: 2,
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
    const dist = multiDomain ? 100 : 85
    return { x: center.x + dist, y: center.y }
  }
  const spread = Math.PI * 0.75
  const angle = -spread / 2 + (spread * moduleIndex) / (moduleCount - 1)
  const dist = multiDomain ? 105 : 90
  return {
    x: center.x + dist * Math.cos(angle),
    y: center.y + dist * Math.sin(angle),
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
  onTopicClick: (domainId: string, nodeKey: string, layerKey: string) => void
  onDomainClick?: (domainId: string) => void
}): KnowledgeGraphMount {
  const { container, domains, onTopicClick, onDomainClick } = opts

  const nodes = new DataSet<GraphNode>([])
  const glowById = new Map<string, 'focus' | 'active' | 'done'>()
  const moduleClusterIds = new Map<string, string[]>()
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
    const rootLabel = domainTitle.length > 16 ? domainTitle.slice(0, 15) + '…' : domainTitle
    const layerByNode = nodeLayerKeyMap(tree)
    const titles = nodeTitleMap(tree)
    const { modules: graphModules } = resolveGraphModules(tree)

    nodes.add({
      ...buildRootNode({
        id: rootId,
        label: rootLabel,
        size: multiDomain ? 28 : 32,
        mass: multiDomain ? 7 : 4,
        domainId,
        title: `${domainTitle} · 点击查看课程列表`,
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

      nodes.add({
        ...buildModuleNode({
          id: moduleId,
          label: mod.label,
          domainId,
          moduleKey: mod.key,
          title: mod.goal ? `${mod.label} · ${mod.goal}` : mod.label,
          multiDomain,
        }),
        x: modPos.x,
        y: modPos.y,
      })

      edges.add({
        id: `e-dm-${domainId}-${mod.key}`,
        from: rootId,
        to: moduleId,
        color: { color: PALETTE.edge.belong, highlight: PALETTE.edge.highlight, opacity: 0.5 },
        width: 1,
        smooth: { enabled: true, type: 'continuous', roundness: 0.2 },
      })

      for (const nodeKey of mod.nodes) {
        const layerKey = layerByNode.get(nodeKey)
        const title = titles.get(nodeKey)
        if (!layerKey || !title) continue

        const topicId = `topic:${domainId}:${nodeKey}`
        const status = normalizeStatus(progressMap.get(nodeKey)?.status)
        const focused = focusKeys.has(nodeKey)

        const topicNode = buildTopicNode({
          id: topicId,
          title,
          status,
          focused,
          nodeKey,
          layerKey,
        })
        topicNode.domainId = domainId
        nodes.add(topicNode)
        clusterIds.push(topicId)
        topicMeta.set(nodeKey, { topicId, layerKey, moduleKey: mod.key })

        if (focused) glowById.set(topicId, 'focus')
        else if (status === 'in_progress') glowById.set(topicId, 'active')
        else if (status === 'completed') glowById.set(topicId, 'done')

        edges.add({
          id: `e-mt-${domainId}-${mod.key}-${nodeKey}`,
          from: moduleId,
          to: topicId,
          color: { color: PALETTE.edge.belong, highlight: PALETTE.edge.highlight, opacity: 0.45 },
          width: 0.75,
          smooth: { enabled: true, type: 'continuous', roundness: 0.22 },
        })
      }

      moduleClusterIds.set(moduleId, clusterIds)

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
          color: { color: PALETTE.edge.path, highlight: PALETTE.edge.highlight, opacity: 0.45 },
          width: 1.2,
          smooth: { enabled: true, type: 'curvedCW', roundness: 0.15 },
        })
      }
    }
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
            gravitationalConstant: multiDomain ? -130 : -55,
            centralGravity: multiDomain ? 0.002 : 0.012,
            springLength: multiDomain ? 175 : 135,
            springConstant: multiDomain ? 0.032 : 0.04,
            damping: multiDomain ? 0.65 : 0.6,
            avoidOverlap: multiDomain ? 0.95 : 0.88,
          },
          stabilization: { iterations: multiDomain ? 380 : 220, updateInterval: 20 },
        },
    nodes: {
      shape: 'dot',
      scaling: { min: 8, max: 36 },
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
    ctx.strokeStyle = 'rgba(255, 252, 247, 0.55)'
    ctx.lineWidth = 1.5
    ctx.stroke()
  }

  const drawRootHover = (ctx: CanvasRenderingContext2D, node: GraphNode, pos: { x: number; y: number }, scale: number) => {
    const baseR = (node.size ?? 12) * scale
    const pulse = reducedMotion ? 1 : 0.92 + 0.08 * Math.sin(pulsePhase)

    const haloR = baseR * (2.2 * pulse)
    const halo = ctx.createRadialGradient(pos.x, pos.y, baseR * 0.6, pos.x, pos.y, haloR)
    halo.addColorStop(0, 'rgba(255, 252, 247, 0.22)')
    halo.addColorStop(0.55, 'rgba(255, 252, 247, 0.08)')
    halo.addColorStop(1, 'rgba(255, 252, 247, 0)')
    ctx.beginPath()
    ctx.arc(pos.x, pos.y, haloR, 0, Math.PI * 2)
    ctx.fillStyle = halo
    ctx.fill()

    ctx.beginPath()
    ctx.arc(pos.x, pos.y, baseR + 2.5 * pulse, 0, Math.PI * 2)
    ctx.strokeStyle = 'rgba(255, 252, 247, 0.72)'
    ctx.lineWidth = 1.75
    ctx.stroke()

    ctx.beginPath()
    ctx.arc(pos.x, pos.y, baseR + 5.5 * pulse, 0, Math.PI * 2)
    ctx.strokeStyle = 'rgba(87, 83, 78, 0.28)'
    ctx.lineWidth = 1
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

      const tier = glowById.get(node.id)
      if (!tier) continue

      const baseR = (node.size ?? 12) * scale
      const mul = tier === 'focus' ? 2.8 * pulse : tier === 'active' ? 2.4 : 2.5
      const outerR = baseR * mul
      const inner =
        tier === 'focus' ? PALETTE.glow.focus : tier === 'active' ? PALETTE.glow.active : PALETTE.glow.done

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
    network.fit({
      animation: reducedMotion ? false : { duration: 450, easingFunction: 'easeInOutQuad' },
    })
  })

  if (reducedMotion) {
    network.fit({ animation: false })
  }

  return {
    destroy: () => {
      if (rafId) cancelAnimationFrame(rafId)
      network.destroy()
    },
    fit: () => network.fit({ animation: { duration: 300, easingFunction: 'easeInOutQuad' } }),
  }
}
