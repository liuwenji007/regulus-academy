import { Network, type Data, type Options } from 'vis-network'
import { DataSet } from 'vis-data'
import 'vis-network/styles/vis-network.css'
import type { KnowledgeTree, UserProgress } from './api'

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
  }
  borderWidth: number
  nodeKey?: string
  layerKey?: string
  title?: string
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
  }
}

export function mountKnowledgeGraph(opts: {
  container: HTMLElement
  tree: KnowledgeTree
  progressMap: Map<string, UserProgress>
  focusKeys: Set<string>
  onTopicClick: (nodeKey: string, layerKey: string) => void
}): KnowledgeGraphMount {
  const { container, tree, progressMap, focusKeys, onTopicClick } = opts

  const nodes = new DataSet<GraphNode>([])
  const glowById = new Map<string, 'focus' | 'active' | 'done'>()
  const edges = new DataSet<{
    id: string
    from: string
    to: string
    dashes?: boolean | number[]
    color?: { color: string; highlight: string; opacity: number }
    width?: number
    smooth?: { enabled: boolean; type: string; roundness: number }
  }>([])

  const domainTitle = tree.domainName?.trim() || '课程'
  const rootLabel = domainTitle.length > 20 ? domainTitle.slice(0, 19) + '…' : domainTitle

  nodes.add({
    id: 'root',
    label: rootLabel,
    group: 'root',
    shape: 'dot',
    size: 32,
    mass: 4,
    font: labelFont(14, true),
    color: {
      background: PALETTE.root.fill,
      border: PALETTE.root.border,
      highlight: { background: PALETTE.root.fill, border: PALETTE.edge.highlight },
    },
    borderWidth: 2,
    title: domainTitle,
  })

  const orderedTopics: { topicId: string; layerKey: string }[] = []
  const layers = Array.isArray(tree.layers) ? tree.layers : []

  for (const layer of layers) {
    if (!layer?.nodes?.length) continue
    for (const node of layer.nodes) {
      const topicId = `topic:${node.key}`
      const status = normalizeStatus(progressMap.get(node.key)?.status)
      const focused = focusKeys.has(node.key)

      const topicNode = buildTopicNode({
        id: topicId,
        title: node.title,
        status,
        focused,
        nodeKey: node.key,
        layerKey: layer.key,
      })
      nodes.add(topicNode)
      if (focused) glowById.set(topicId, 'focus')
      else if (status === 'in_progress') glowById.set(topicId, 'active')
      else if (status === 'completed') glowById.set(topicId, 'done')

      edges.add({
        id: `e-belong-${node.key}`,
        from: 'root',
        to: topicId,
        color: { color: PALETTE.edge.belong, highlight: PALETTE.edge.highlight, opacity: 0.55 },
        width: 0.8,
        smooth: { enabled: true, type: 'continuous', roundness: 0.25 },
      })

      orderedTopics.push({ topicId, layerKey: layer.key })
    }
  }

  for (let i = 1; i < orderedTopics.length; i++) {
    const prev = orderedTopics[i - 1]!
    const curr = orderedTopics[i]!
    edges.add({
      id: `e-path-${i}`,
      from: prev.topicId,
      to: curr.topicId,
      dashes: [5, 8],
      color: { color: PALETTE.edge.path, highlight: PALETTE.edge.highlight, opacity: 0.45 },
      width: 1.2,
      smooth: { enabled: true, type: 'curvedCW', roundness: 0.15 },
    })
  }

  const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  let pulsePhase = 0
  let rafId = 0

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
    },
    physics: reducedMotion
      ? { enabled: false }
      : {
          enabled: true,
          solver: 'forceAtlas2Based',
          forceAtlas2Based: {
            gravitationalConstant: -55,
            centralGravity: 0.012,
            springLength: 130,
            springConstant: 0.045,
            damping: 0.55,
            avoidOverlap: 0.85,
          },
          stabilization: { iterations: 220, updateInterval: 20 },
        },
    nodes: {
      shape: 'dot',
      scaling: { min: 8, max: 36 },
    },
    edges: {
      selectionWidth: 0,
      smooth: { enabled: true, type: 'continuous', roundness: 0.2 },
    },
  }

  const graphData: Data = { nodes: nodes as unknown as Data['nodes'], edges }
  const network = new Network(container, graphData, options)

  const drawGlows = (ctx: CanvasRenderingContext2D) => {
    const positions = network.getPositions()
    const scale = network.getScale()
    const pulse = reducedMotion ? 1 : 0.85 + 0.15 * Math.sin(pulsePhase)

    for (const node of nodes.get()) {
      const tier = glowById.get(node.id)
      if (!tier) continue
      const pos = positions[node.id]
      if (!pos) continue

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
    if (params.nodes.length !== 1) return
    const id = params.nodes[0] as string
    if (!id.startsWith('topic:')) return
    const item = nodes.get(id)
    if (!item?.nodeKey || !item.layerKey) return
    onTopicClick(item.nodeKey, item.layerKey)
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
