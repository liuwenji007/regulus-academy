import type { Archetype, ConstellationLayout, StarPlacement } from './constellation-layout'
import { hashString } from './constellation-layout'

export interface ConstellationTemplate {
  id: string
  name: string
  points: Array<{ x: number; y: number }>
}

const TEMPLATES: Record<Archetype, ConstellationTemplate[]> = {
  chain: [
    {
      id: 'scorpio',
      name: '天蝎',
      points: [
        { x: 0, y: 0 },
        { x: 1.1, y: 0.35 },
        { x: 2.1, y: 0.15 },
        { x: 3.2, y: 0.55 },
        { x: 4.1, y: 0.2 },
        { x: 5.2, y: 0.7 },
        { x: 6.1, y: 1.35 },
        { x: 6.5, y: 2.3 },
      ],
    },
  ],
  twin: [
    {
      id: 'gemini',
      name: '双子',
      points: [
        { x: 0, y: 0 },
        { x: 0.2, y: 1.2 },
        { x: 0.55, y: 2.4 },
        { x: 1.5, y: 0.15 },
        { x: 1.7, y: 1.35 },
        { x: 2.05, y: 2.5 },
      ],
    },
  ],
  sickle: [
    {
      id: 'leo',
      name: '狮子',
      points: [
        { x: 0, y: 0 },
        { x: 0.9, y: 0.85 },
        { x: 1.7, y: 1.15 },
        { x: 2.35, y: 0.55 },
        { x: 2.1, y: -0.35 },
        { x: 3.3, y: -0.2 },
        { x: 4.6, y: 0.25 },
      ],
    },
  ],
  cross: [
    {
      id: 'cygnus',
      name: '天鹅',
      points: [
        { x: 0, y: 1.6 },
        { x: 0, y: 0.4 },
        { x: 0, y: -0.8 },
        { x: 0, y: -2 },
        { x: -1.5, y: 0.15 },
        { x: 1.55, y: 0.2 },
      ],
    },
  ],
  zigzag: [
    {
      id: 'cassiopeia',
      name: '仙后',
      points: [
        { x: 0, y: 0.2 },
        { x: 1.1, y: 1.1 },
        { x: 2.2, y: 0.15 },
        { x: 3.3, y: 1.2 },
        { x: 4.4, y: 0.1 },
      ],
    },
  ],
  dipper: [
    {
      id: 'dipper',
      name: '北斗',
      points: [
        { x: 0, y: 0 },
        { x: 1.2, y: 0.35 },
        { x: 2.5, y: 0.85 },
        { x: 3.5, y: 1.7 },
        { x: 3.1, y: 2.7 },
        { x: 1.9, y: 2.45 },
        { x: 1.15, y: 1.55 },
      ],
    },
    {
      id: 'orion',
      name: '猎户',
      points: [
        { x: -0.9, y: 1.6 },
        { x: 0.9, y: 1.45 },
        { x: -0.35, y: 0.35 },
        { x: 0, y: 0 },
        { x: 0.4, y: -0.35 },
        { x: -0.7, y: -1.5 },
        { x: 0.85, y: -1.35 },
      ],
    },
  ],
}

export function templateForArchetype(archetype: Archetype, domainId: string): ConstellationTemplate {
  const list = TEMPLATES[archetype]
  return list[hashString(domainId) % list.length]!
}

export function applyTemplateReward(layout: ConstellationLayout, template: ConstellationTemplate): ConstellationLayout {
  const order = walkStars(layout)
  if (order.length === 0) return layout
  const samples = resample(template.points, order.length)
  const origin = samples[0] ?? { x: 0, y: 0 }
  const shifted = samples.map((p) => ({ x: p.x - origin.x, y: p.y - origin.y }))
  let maxR = 0
  for (const p of shifted) maxR = Math.max(maxR, Math.hypot(p.x, p.y))
  const scale = maxR < 1e-6 ? 1 : layout.radius / maxR
  const root = layout.stars.find((star) => star.role === 'domain') ?? layout.stars[0]
  const at = new Map<string, { x: number; y: number }>()
  order.forEach((id, i) => {
    const p = shifted[i] ?? { x: 0, y: 0 }
    at.set(id, {
      x: (root?.x ?? 0) + p.x * scale,
      y: (root?.y ?? 0) + p.y * scale,
    })
  })
  const stars: StarPlacement[] = layout.stars.map((star) => {
    const p = at.get(star.id) ?? star
    return { ...star, x: p.x, y: p.y }
  })
  const segments = order.slice(1).map((id, i) => ({
    from: order[i]!,
    to: id,
    kind: 'spine' as const,
  }))
  return { ...layout, stars, segments }
}

function walkStars(layout: ConstellationLayout): string[] {
  const root = layout.stars.find((star) => star.role === 'domain')?.id
  if (!root) return layout.stars.map((star) => star.id)
  const adj = new Map<string, string[]>()
  for (const seg of layout.segments) {
    adj.set(seg.from, [...(adj.get(seg.from) ?? []), seg.to])
    adj.set(seg.to, [...(adj.get(seg.to) ?? []), seg.from])
  }
  const seen = new Set<string>()
  const out: string[] = []
  const stack = [root]
  while (stack.length) {
    const id = stack.pop()!
    if (seen.has(id)) continue
    seen.add(id)
    out.push(id)
    const next = (adj.get(id) ?? []).slice().sort()
    for (let i = next.length - 1; i >= 0; i--) stack.push(next[i]!)
  }
  for (const star of layout.stars) if (!seen.has(star.id)) out.push(star.id)
  return out
}

function resample(points: Array<{ x: number; y: number }>, count: number): Array<{ x: number; y: number }> {
  if (count <= 0) return []
  if (points.length === 0) return Array.from({ length: count }, () => ({ x: 0, y: 0 }))
  if (count === 1) return [{ ...points[0]! }]
  const lengths = [0]
  for (let i = 1; i < points.length; i++) {
    const a = points[i - 1]!
    const b = points[i]!
    lengths.push(lengths[i - 1]! + Math.hypot(b.x - a.x, b.y - a.y))
  }
  const total = lengths[lengths.length - 1] || 1
  const out: Array<{ x: number; y: number }> = []
  for (let i = 0; i < count; i++) {
    const target = (total * i) / (count - 1)
    let seg = 1
    while (seg < lengths.length - 1 && lengths[seg]! < target) seg++
    const start = lengths[seg - 1] ?? 0
    const end = lengths[seg] ?? total
    const t = end === start ? 0 : (target - start) / (end - start)
    const a = points[seg - 1] ?? points[0]!
    const b = points[seg] ?? a
    out.push({ x: a.x + (b.x - a.x) * t, y: a.y + (b.y - a.y) * t })
  }
  return out
}
