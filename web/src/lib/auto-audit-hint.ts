import type { AutoAuditSummary } from './api'

const AUTO_AUDIT_KEY = 'regulus:autoAuditHint'

export interface AutoAuditHint extends AutoAuditSummary {
  domainId: string
}

/** 建课成功后暂存规则体检摘要，进入课程树时展示一次 */
export function stashAutoAuditHint(domainId: string, audit?: AutoAuditSummary): void {
  if (!domainId || !audit) return
  const issues = (audit.failCount ?? 0) + (audit.warnCount ?? 0)
  if (issues <= 0) return
  try {
    sessionStorage.setItem(
      AUTO_AUDIT_KEY,
      JSON.stringify({
        domainId,
        score: audit.score ?? 0,
        grade: audit.grade ?? '',
        failCount: audit.failCount ?? 0,
        warnCount: audit.warnCount ?? 0,
        infoCount: audit.infoCount ?? 0,
        headline: audit.headline ?? '',
      } satisfies AutoAuditHint)
    )
  } catch {
    /* ignore quota */
  }
}

/** 读取并清除与当前课程匹配的体检提示 */
export function consumeAutoAuditHint(domainId: string): AutoAuditHint | null {
  try {
    const raw = sessionStorage.getItem(AUTO_AUDIT_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw) as AutoAuditHint
    if (parsed.domainId !== domainId) return null
    sessionStorage.removeItem(AUTO_AUDIT_KEY)
    return parsed
  } catch {
    return null
  }
}
