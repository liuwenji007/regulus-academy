import { beforeAll, describe, expect, it } from 'vitest'
import { buildAssistantPlanPanelHtml } from './assistant-render'
import type { PlanningResult } from './api'

beforeAll(() => {
  // escapeHtml 依赖 document；本仓库 vitest 无 DOM 环境，做最小 stub。
  if (typeof document === 'undefined') {
    ;(globalThis as unknown as { document: Document }).document = {
      createElement: () => {
        let text = ''
        return {
          set textContent(v: string) {
            text = String(v)
          },
          get innerHTML() {
            return text
              .replace(/&/g, '&amp;')
              .replace(/</g, '&lt;')
              .replace(/>/g, '&gt;')
              .replace(/"/g, '&quot;')
          },
        }
      },
    } as unknown as Document
  }
})

function samplePlan(): PlanningResult {
  return {
    situation_summary: '事务过多学不动',
    focus: {
      north_star: '把 Go 并发线钉住',
      why: '其余是噪声',
      week_wedge: 'Go 并发',
      today_learning: {
        title: 'channel 基础',
        minutes: 15,
        matched_domain_id: 'd1',
        matched_node_key: 'channel-basics',
        matched_node_title: 'channel 基础',
      },
    },
    clear_first: [{ title: '回一封邮件', next_step: '打开邮箱', minutes: 10 }],
    matrix: {
      important_urgent: [{ title: '周报' }],
      important_not_urgent: [{ title: 'Go' }],
      quick_wins: [{ title: '回邮件' }],
      defer_or_drop: [{ title: '刷新闻', reason: '不贡献节奏' }],
    },
    action_plan: {
      today: [{ title: 'channel 基础', minutes: 15, kind: 'learning' }],
      this_week: [],
    },
    learning_focus: [],
    mindset_note: '先做一件小事',
    ui_state: { north_star_pinned: false, checked: {} },
  }
}

describe('assistant-render focus pin', () => {
  it('折叠态展示北星、清障与今日学习，不展开四象限', () => {
    const html = buildAssistantPlanPanelHtml(samplePlan(), false)
    expect(html).toContain('聚焦位')
    expect(html).toContain('把 Go 并发线钉住')
    expect(html).toContain('先清障')
    expect(html).toContain('回一封邮件')
    expect(html).toContain('今日学习')
    expect(html).toContain('开始 15 分钟')
    expect(html).toContain('钉住这个目标')
    expect(html).toContain('展开详情')
    expect(html).not.toContain('重要且紧急')
    expect(html).not.toContain('建议暂缓')
  })

  it('展开态显示四象限与本周楔子', () => {
    const html = buildAssistantPlanPanelHtml(samplePlan(), true)
    expect(html).toContain('本周楔子')
    expect(html).toContain('重要且紧急')
    expect(html).toContain('建议暂缓')
    expect(html).toContain('收起详情')
  })
})
