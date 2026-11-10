import { mkdirSync, writeFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import type { KnowledgeTree } from './api'
import { renderConstellationSheet } from './constellation-svg'

function course(spec: number[], name: string): KnowledgeTree {
  const modules = spec.map((count, i) => ({
    key: `m${i}`,
    label: `${name}-${i}`,
    order: i + 1,
    nodes: Array.from({ length: count }, (_, j) => `${name}-m${i}n${j}`),
  }))
  return {
    domainId: name,
    domainName: name,
    layers: [
      {
        key: 'intro',
        label: '入门',
        time: '',
        goal: '',
        nodes: modules.flatMap((m) => m.nodes.map((key) => ({ key, title: key }))),
      },
    ],
    modules,
  }
}

describe('renderConstellationSheet', () => {
  it('draws one panel per course', () => {
    const svg = renderConstellationSheet([
      { title: '长链', domainId: 'chain', tree: course([8], 'chain') },
      { title: '双子', domainId: 'twin', tree: course([5, 6], 'twin') },
      { title: '十字', domainId: 'cross', tree: course([3, 3, 3, 3], 'cross') },
      { title: '镰刀', domainId: 'sickle', tree: course([9, 1, 1], 'sickle') },
      { title: 'W 形', domainId: 'zigzag', tree: course([2, 2, 2, 2, 2, 2], 'zigzag') },
      { title: '北斗', domainId: 'dipper', tree: course([4, 4, 1], 'dipper') },
    ])
    expect(svg).toContain('<svg')
    expect(svg.match(/<circle /g)?.length).toBeGreaterThan(20)
    if (process.env.CONSTELLATION_SVG === '1') {
      const out = resolve(process.cwd(), 'tmp/constellations.svg')
      mkdirSync(dirname(out), { recursive: true })
      writeFileSync(out, svg)
    }
  })
})
