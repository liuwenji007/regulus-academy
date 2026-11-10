import { describe, expect, it } from 'vitest'
import type { KnowledgeTree } from './api'
import { buildConstellation } from './constellation-layout'
import { applyTemplateReward, templateForArchetype } from './constellation-templates'

function tree(): KnowledgeTree {
  return {
    domainId: 'leo',
    domainName: '狮子座',
    layers: [
      {
        key: 'intro',
        label: '入门',
        time: '',
        goal: '',
        nodes: [
          { key: 'a', title: 'A' },
          { key: 'b', title: 'B' },
          { key: 'c', title: 'C' },
        ],
      },
    ],
    modules: [{ key: 'm', label: '模块', order: 1, nodes: ['a', 'b', 'c'] }],
  }
}

describe('applyTemplateReward', () => {
  it('keeps the root in place and every star id', () => {
    const layout = buildConstellation('leo', tree(), 0)
    const rewarded = applyTemplateReward(layout, templateForArchetype(layout.archetype, 'leo'))
    const root = rewarded.stars.find((star) => star.role === 'domain')
    expect(root?.x).toBeCloseTo(0)
    expect(root?.y).toBeCloseTo(0)
    expect(rewarded.stars.map((star) => star.id).sort()).toEqual(layout.stars.map((star) => star.id).sort())
    expect(rewarded.segments).toHaveLength(rewarded.stars.length - 1)
  })
})
