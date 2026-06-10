import { invalidateSidebarCourses } from '../components/layout'
import { setAppBusy } from './app-busy'
import { stashPrefetchTree } from './course-prefetch'
import { navigateHash } from './navigate'
import { extendDomain, type BuildDomainResult, type DomainBuildJobPoll, type ExtendDomainResult } from './api'
import {
  applyServerBuildProgress,
  savePendingBuild,
} from './domain-build-job'

const LAST_DOMAIN_KEY = 'regulus:lastDomainId'
const REGENERATE_TOAST_KEY = 'regulus:regenerateToast'

export async function handleDomainDelete(domainId: string): Promise<void> {
  if (localStorage.getItem(LAST_DOMAIN_KEY) === domainId) {
    localStorage.removeItem(LAST_DOMAIN_KEY)
  }
  invalidateSidebarCourses()
  navigateHash('/courses')
}

export async function handleDomainRegenerate(
  domainId: string,
  newDomainId: string,
  result?: BuildDomainResult
): Promise<void> {
  if (localStorage.getItem(LAST_DOMAIN_KEY) === domainId) {
    localStorage.setItem(LAST_DOMAIN_KEY, newDomainId)
  }
  invalidateSidebarCourses()
  if (result?.tree) {
    stashPrefetchTree(result.tree)
  }
  const kept = result?.progressKept ?? 0
  if (kept > 0) {
    const skipped = result?.progressSkipped ?? 0
    let msg = `课程已按当前学习画像重新规划，已保留 ${kept} 个已掌握节点`
    if (skipped > 0) {
      msg += `（${skipped} 个因新路径未包含而未迁移）`
    }
    sessionStorage.setItem(REGENERATE_TOAST_KEY, msg)
  } else if (result?.message?.trim()) {
    sessionStorage.setItem(REGENERATE_TOAST_KEY, result.message.trim())
  }
  setAppBusy(true, 'build')
  navigateHash(`/tree/${newDomainId}`, { reload: true })
}

export async function handleDomainExtend(
  domainId: string,
  topic: string,
  goal?: string,
  onProgress?: (status: DomainBuildJobPoll) => void
): Promise<ExtendDomainResult> {
  const result = await extendDomain(domainId, {
    goal: goal?.trim() || undefined,
    onJobAccepted: (jobId) =>
      savePendingBuild({ jobId, topic, kind: 'extend', domainId }),
    onProgress: (status) => {
      applyServerBuildProgress(status)
      onProgress?.(status)
    },
  })
  if (result.tree) {
    stashPrefetchTree(result.tree)
  }
  invalidateSidebarCourses()
  return result
}

export function consumeRegenerateToast(): string | null {
  const msg = sessionStorage.getItem(REGENERATE_TOAST_KEY)
  if (msg) {
    sessionStorage.removeItem(REGENERATE_TOAST_KEY)
    return msg
  }
  return null
}
