import type { UserProfile } from './api'

export type ProfileSection = { label: string; body: string }

/** 解析 legacy profile_summary 中的【标签】分段 */
export function parseProfileSections(summary: string): ProfileSection[] {
  const text = summary.trim()
  if (!text) return []
  const re = /【([^】]+)】/g
  const hits: { index: number; label: string; end: number }[] = []
  let m: RegExpExecArray | null
  while ((m = re.exec(text)) !== null) {
    hits.push({ index: m.index, label: m[1].trim(), end: m.index + m[0].length })
  }
  if (hits.length === 0) return [{ label: '', body: text }]
  const sections: ProfileSection[] = []
  for (let i = 0; i < hits.length; i++) {
    const start = hits[i].end
    const end = i + 1 < hits.length ? hits[i + 1].index : text.length
    const body = text.slice(start, end).trim()
    if (body) sections.push({ label: hits[i].label, body })
  }
  return sections.length > 0 ? sections : [{ label: '', body: text }]
}

const PREF_INLINE = '；讲解偏好：'
const PREF_ONLY = '讲解偏好：'

function splitBackgroundPreference(body: string): { background: string; preference: string } {
  const prefIdx = body.indexOf(PREF_INLINE)
  if (prefIdx >= 0) {
    return {
      background: body.slice(0, prefIdx).trim(),
      preference: body.slice(prefIdx + PREF_INLINE.length).trim(),
    }
  }
  if (body.startsWith(PREF_ONLY)) {
    return { background: '', preference: body.slice(PREF_ONLY.length).trim() }
  }
  return { background: body.trim(), preference: '' }
}

/** 编辑表单回显：结构化列优先，缺失时从 profile_summary 解析 */
export function resolveEditableProfileFields(user: UserProfile): {
  background: string
  goal: string
  preference: string
} {
  let background = (user.profileBackground ?? '').trim()
  let goal = (user.profileGoal ?? '').trim()
  let preference = (user.profilePreference ?? '').trim()

  const summary = (user.profileSummary ?? '').trim()
  if (!summary) return { background, goal, preference }

  const sections = parseProfileSections(summary)
  for (const s of sections) {
    if (s.label === '背景') {
      if (!background || !preference) {
        const split = splitBackgroundPreference(s.body)
        if (!background) background = split.background
        if (!preference) preference = split.preference
      }
    } else if (s.label === '目标' && !goal) {
      goal = s.body
    } else if (s.label === '偏好' && !preference) {
      preference = s.body
    } else if (!s.label && !background && !goal) {
      background = s.body
    }
  }

  return { background, goal, preference }
}
