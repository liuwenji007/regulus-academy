import { describe, expect, it } from 'vitest'
import type { KnowledgeTree } from './api'
import type { MultiDomainGraphEntry } from './knowledge-graph'
import { computeGraphOutlineSummary, constellationSectionTitle } from './graph-outline'

function makeEntry(
  domainId: string,
  domainName: string,
  nodes: Array<{ key: string; title: string; layerKey: string; status?: string }>
): MultiDomainGraphEntry {
  const layersMap = new Map<string, { key: string; label: string; nodes: Array<{ key: string; title: string }> }>()
  for (const n of nodes) {
    const layer = layersMap.get(n.layerKey) ?? { key: n.layerKey, label: n.layerKey, nodes: [] }
    layer.nodes.push({ key: n.key, title: n.title })
    layersMap.set(n.layerKey, layer)
  }
  const tree: KnowledgeTree = {
    domainId,
    domainName,
    layers: [...layersMap.values()].map((l) => ({
      key: l.key,
      label: l.label,
      time: '',
      goal: '',
      nodes: l.nodes,
    })),
  }
  const progressMap = new Map<string, { status?: string }>()
  for (const n of nodes) {
    if (n.status) progressMap.set(n.key, { status: n.status })
  }
  return {
    domainId,
    slug: domainId,
    tree,
    progressMap: progressMap as MultiDomainGraphEntry['progressMap'],
    focusKeys: new Set(),
  }
}

describe('computeGraphOutlineSummary', () => {
  it('counts all nodes across domains, not truncated at first incomplete', () => {
    const entries = [
      makeEntry('d1', '课程 A', [
        { key: 'a1', title: 'A1', layerKey: 'l1', status: 'completed' },
        { key: 'a2', title: 'A2', layerKey: 'l1' },
      ]),
      makeEntry('d2', '课程 B', [
        { key: 'b1', title: 'B1', layerKey: 'l1', status: 'completed' },
        { key: 'b2', title: 'B2', layerKey: 'l1' },
        { key: 'b3', title: 'B3', layerKey: 'l1' },
      ]),
    ]

    const summary = computeGraphOutlineSummary(entries)
    expect(summary.domainCount).toBe(2)
    expect(summary.totalNodes).toBe(5)
    expect(summary.completedNodes).toBe(2)
  })

  it('points next fields at the first incomplete node in traversal order', () => {
    const entries = [
      makeEntry('d1', '课程 A', [
        { key: 'a1', title: 'A1', layerKey: 'l1', status: 'completed' },
        { key: 'a2', title: 'A2', layerKey: 'l1' },
      ]),
      makeEntry('d2', '课程 B', [
        { key: 'b1', title: 'B1', layerKey: 'l1' },
      ]),
    ]

    const summary = computeGraphOutlineSummary(entries)
    expect(summary.nextDomainId).toBe('d1')
    expect(summary.nextDomainName).toBe('课程 A')
    expect(summary.nextNodeKey).toBe('a2')
    expect(summary.nextNodeTitle).toBe('A2')
    expect(summary.nextLayerKey).toBe('l1')
  })
})

describe('constellationSectionTitle', () => {
  it('shows label for single-domain constellation when multiple groups exist', () => {
    expect(
      constellationSectionTitle(
        { key: 'python', label: 'Python', domainIds: ['c'], nodeCount: 8 },
        2
      )
    ).toBe('Python')
  })

  it('shows count suffix for multi-domain constellation', () => {
    expect(
      constellationSectionTitle(
        { key: 'go', label: 'Go 语言', domainIds: ['a', 'b'], nodeCount: 30 },
        2
      )
    ).toBe('Go 语言 · 2 门')
  })

  it('hides section title when only one constellation group total', () => {
    expect(
      constellationSectionTitle(
        { key: 'go', label: 'Go 语言', domainIds: ['a', 'b'], nodeCount: 30 },
        1
      )
    ).toBe('')
  })
})
