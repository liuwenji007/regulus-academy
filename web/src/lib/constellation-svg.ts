import type { KnowledgeTree } from './api'
import { buildConstellation, type ConstellationLayout } from './constellation-layout'

export interface SvgPanel {
  title: string
  domainId: string
  tree: KnowledgeTree
  salt?: number
}

export function renderConstellationSheet(panels: SvgPanel[]): string {
  const cell = 560
  const cols = Math.min(3, Math.max(1, panels.length))
  const rows = Math.ceil(panels.length / cols)
  const width = cols * cell
  const height = rows * cell
  const parts: string[] = []
  panels.forEach((panel, i) => {
    const layout = buildConstellation(panel.domainId, panel.tree, panel.salt ?? 0)
    const col = i % cols
    const row = Math.floor(i / cols)
    parts.push(renderPanel(layout, panel.title, col * cell, row * cell, cell))
  })
  return `<?xml version="1.0" encoding="UTF-8"?>\n<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}">\n<rect width="100%" height="100%" fill="#0b1226"/>\n${parts.join('\n')}\n</svg>\n`
}

function renderPanel(layout: ConstellationLayout, title: string, ox: number, oy: number, cell: number): string {
  const pad = 48
  let minX = Infinity
  let minY = Infinity
  let maxX = -Infinity
  let maxY = -Infinity
  for (const s of layout.stars) {
    minX = Math.min(minX, s.x)
    minY = Math.min(minY, s.y)
    maxX = Math.max(maxX, s.x)
    maxY = Math.max(maxY, s.y)
  }
  const span = Math.max(maxX - minX, maxY - minY, 1)
  const scale = (cell - pad * 2) / span
  const at = (x: number, y: number) => {
    const px = ox + pad + (x - minX) * scale
    const py = oy + pad + (y - minY) * scale
    return `${px.toFixed(1)},${py.toFixed(1)}`
  }
  const byId = new Map(layout.stars.map((s) => [s.id, s]))
  const lines = layout.segments
    .map((seg) => {
      const a = byId.get(seg.from)!
      const b = byId.get(seg.to)!
      return `<line x1="${at(a.x, a.y).split(',')[0]}" y1="${at(a.x, a.y).split(',')[1]}" x2="${at(b.x, b.y).split(',')[0]}" y2="${at(b.x, b.y).split(',')[1]}" stroke="${seg.kind === 'spine' ? '#d7e4ff' : '#7f93b8'}" stroke-width="1.2"/>`
    })
    .join('')
  const dots = layout.stars
    .map((s) => {
      const [x, y] = at(s.x, s.y).split(',')
      const r = s.role === 'domain' ? 5.5 : s.role === 'module' ? 3.6 : 2.2
      const fill = s.role === 'domain' ? '#f5dc6a' : s.role === 'module' ? '#9fc3ff' : '#d5def0'
      return `<circle cx="${x}" cy="${y}" r="${r}" fill="${fill}"/>`
    })
    .join('')
  return `<g><text x="${ox + 16}" y="${oy + 24}" fill="#c5d2ea" font-size="14" font-family="sans-serif">${escapeXml(title)} · ${layout.archetype}</text>${lines}${dots}</g>`
}

function escapeXml(s: string): string {
  return s.replace(/[&<>]/g, (ch) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;' })[ch] ?? ch)
}
