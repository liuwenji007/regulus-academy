/** 知识银河课程（领域根节点）坐标本地持久化（按用户 + 领域组合） */

import { getActiveUserId } from './profile'

const STORAGE_KEY = 'regulus:graph-layouts'
const SCHEMA_VERSION = 1

export type GraphNodePosition = { x: number; y: number }

type LayoutEntry = {
  v: typeof SCHEMA_VERSION
  updatedAt: number
  domainIds: string[]
  positions: Record<string, GraphNodePosition>
}

type LayoutStore = Record<string, LayoutEntry>

export function graphLayoutScopeKey(domainIds: string[], userId = getActiveUserId()): string {
  const sorted = [...domainIds].filter(Boolean).sort()
  return `${userId || 'anon'}:${sorted.join(',')}`
}

function readStore(): LayoutStore {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return {}
    const parsed = JSON.parse(raw) as LayoutStore
    return parsed && typeof parsed === 'object' ? parsed : {}
  } catch {
    return {}
  }
}

function writeStore(store: LayoutStore): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(store))
  } catch {
    /* quota / private mode */
  }
}

export function domainRootNodeId(domainId: string): string {
  return `domain:${domainId}`
}

function isDomainRootNodeId(nodeId: string): boolean {
  return nodeId.startsWith('domain:')
}

/** 读取已保存的课程根节点坐标；无记录或结构无效时返回 null */
export function loadGraphLayout(domainIds: string[]): Record<string, GraphNodePosition> | null {
  if (!domainIds.length) return null
  const key = graphLayoutScopeKey(domainIds)
  const entry = readStore()[key]
  if (!entry || entry.v !== SCHEMA_VERSION || !entry.positions) return null
  const positions: Record<string, GraphNodePosition> = {}
  for (const [id, pos] of Object.entries(entry.positions)) {
    if (!isDomainRootNodeId(id)) continue
    if (pos && Number.isFinite(pos.x) && Number.isFinite(pos.y)) {
      positions[id] = { x: pos.x, y: pos.y }
    }
  }
  return Object.keys(positions).length > 0 ? positions : null
}

export function saveGraphLayout(
  domainIds: string[],
  positions: Record<string, GraphNodePosition>
): void {
  if (!domainIds.length) return
  const key = graphLayoutScopeKey(domainIds)
  const store = readStore()
  store[key] = {
    v: SCHEMA_VERSION,
    updatedAt: Date.now(),
    domainIds: [...domainIds].filter(Boolean).sort(),
    positions,
  }
  writeStore(store)
}

export function clearGraphLayout(domainIds: string[]): void {
  if (!domainIds.length) return
  const key = graphLayoutScopeKey(domainIds)
  const store = readStore()
  if (!store[key]) return
  delete store[key]
  writeStore(store)
}

/** 从 vis-network 写入课程根节点坐标；仅持久化 domain:* 节点 */
export function persistGraphLayoutFromNetwork(
  domainIds: string[],
  nodeIds: string[],
  getPositions: (ids: string[]) => Record<string, GraphNodePosition>
): void {
  const domainNodeIds = nodeIds.filter(isDomainRootNodeId)
  if (!domainNodeIds.length) return
  const positions = getPositions(domainNodeIds)
  const out: Record<string, GraphNodePosition> = {}
  for (const id of domainNodeIds) {
    const p = positions[id]
    if (p && Number.isFinite(p.x) && Number.isFinite(p.y)) {
      out[id] = { x: p.x, y: p.y }
    }
  }
  if (Object.keys(out).length > 0) {
    saveGraphLayout(domainIds, out)
  }
}

export function resolveNodePlacement(
  nodeId: string,
  defaultPos: GraphNodePosition,
  saved: Record<string, GraphNodePosition> | null
): { x: number; y: number; fixed?: { x: boolean; y: boolean } } {
  const hit = saved?.[nodeId]
  if (hit) {
    return { x: hit.x, y: hit.y, fixed: { x: true, y: true } }
  }
  return { x: defaultPos.x, y: defaultPos.y }
}
