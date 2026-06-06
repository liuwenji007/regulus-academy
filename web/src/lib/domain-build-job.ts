import {
  getDomainBuildJobStatus,
  parseBuildDomainPollResult,
  pollDomainBuildJob,
  type BuildDomainResult,
  type DomainBuildJobPoll,
} from './api'
import { clearAppBusyIfAfter, setAppBusy } from './app-busy'
import { stashPrefetchTree } from './course-prefetch'

export type DomainBuildPhase = 'analyzing' | 'generating' | 'success' | 'error'

export interface DomainBuildJob {
  topic: string
  phase: DomainBuildPhase
  message: string
  error?: string
  resultDomainId?: string
  resultMessage?: string
}

const PENDING_BUILD_KEY = 'regulus:pendingDomainBuild'

interface PendingBuild {
  jobId: string
  topic: string
}

let job: DomainBuildJob | null = null
let resumePromise: Promise<void> | null = null
const listeners = new Set<() => void>()

function emit(): void {
  for (const fn of listeners) fn()
}

export function mapServerBuildPhase(phase: string): DomainBuildPhase {
  if (phase === 'failed') return 'error'
  if (phase === 'done') return 'success'
  if (phase === 'starting' || phase === 'intent') return 'analyzing'
  return 'generating'
}

export function onDomainBuildJobChange(fn: () => void): () => void {
  listeners.add(fn)
  return () => listeners.delete(fn)
}

export function getDomainBuildJob(): DomainBuildJob | null {
  return job
}

export function isDomainBuildRunning(): boolean {
  return job !== null && job.phase !== 'success' && job.phase !== 'error'
}

export function savePendingBuild(pending: PendingBuild): void {
  sessionStorage.setItem(PENDING_BUILD_KEY, JSON.stringify(pending))
}

export function clearPendingBuild(): void {
  sessionStorage.removeItem(PENDING_BUILD_KEY)
}

function loadPendingBuild(): PendingBuild | null {
  const raw = sessionStorage.getItem(PENDING_BUILD_KEY)
  if (!raw) return null
  try {
    const parsed = JSON.parse(raw) as PendingBuild
    if (parsed.jobId && parsed.topic) return parsed
  } catch {
    /* ignore */
  }
  return null
}

/** 若已有建课进行中则返回 false */
export function tryStartDomainBuildJob(topic: string): boolean {
  if (isDomainBuildRunning()) return false
  job = { topic, phase: 'analyzing', message: '任务已创建…' }
  setAppBusy(true, 'build')
  emit()
  return true
}

export function applyServerBuildProgress(status: DomainBuildJobPoll): void {
  if (!job || !isDomainBuildRunning()) return
  const message = status.message?.trim() || job.message
  job.phase = mapServerBuildPhase(status.phase)
  job.message = message
  emit()
}

export function setDomainBuildJobPhase(phase: DomainBuildPhase, message: string): void {
  if (!job || !isDomainBuildRunning()) return
  job.phase = phase
  job.message = message
  emit()
}

export function finishDomainBuildJobSuccess(
  opts: { domainId: string; message?: string },
  onReleased?: () => void
): void {
  if (!job) return
  job.phase = 'success'
  job.message = opts.message?.trim() || `「${job.topic}」课程已就绪`
  job.resultDomainId = opts.domainId
  job.resultMessage = opts.message
  emit()
  clearAppBusyIfAfter('build', onReleased)
}

export function finishDomainBuildJobError(message: string, onReleased?: () => void): void {
  if (!job) return
  const err = message.trim() || '建课失败，请稍后重试'
  job.phase = 'error'
  job.message = err
  job.error = err
  emit()
  clearAppBusyIfAfter('build', onReleased)
}

export function dismissDomainBuildJob(): void {
  if (!job) return
  job = null
  emit()
}

function finishFromBuildResult(result: BuildDomainResult): void {
  if (result.status === 'ready' && result.tree?.domainId) {
    stashPrefetchTree(result.tree)
    finishDomainBuildJobSuccess({
      domainId: result.tree.domainId,
      message: result.message,
    })
    return
  }
  const msg = result.message ?? '无法加载学习路径'
  finishDomainBuildJobError(msg)
}

async function pollPendingBuild(
  pending: PendingBuild,
  onProgress?: (status: DomainBuildJobPoll) => void
): Promise<BuildDomainResult> {
  return pollDomainBuildJob(pending.jobId, (status) => {
    applyServerBuildProgress(status)
    onProgress?.(status)
  })
}

/** 刷新页面后恢复未完成的建课轮询 */
export function resumePendingDomainBuildJob(opts?: { onReleased?: () => void }): Promise<void> {
  if (resumePromise) return resumePromise
  resumePromise = (async () => {
    if (isDomainBuildRunning()) return
    const pending = loadPendingBuild()
    if (!pending) return

    try {
      const status = await getDomainBuildJobStatus(pending.jobId)
      if (status.status === 'done') {
        if (!status.result) {
          const msg = '建课已完成但未能加载结果，请到「我的课程」查看或重新创建'
          tryStartDomainBuildJob(pending.topic)
          finishDomainBuildJobError(msg, opts?.onReleased)
          clearPendingBuild()
          return
        }
        tryStartDomainBuildJob(pending.topic)
        finishFromBuildResult(parseBuildDomainPollResult(status.result))
        clearPendingBuild()
        opts?.onReleased?.()
        return
      }
      if (status.status === 'failed') {
        clearPendingBuild()
        const msg = status.error?.trim() || status.message?.trim() || '建课失败'
        job = { topic: pending.topic, phase: 'error', message: msg, error: msg }
        emit()
        clearAppBusyIfAfter('build', opts?.onReleased)
        return
      }

      if (!tryStartDomainBuildJob(pending.topic)) return
      applyServerBuildProgress(status)
      const result = await pollPendingBuild(pending)
      clearPendingBuild()
      finishFromBuildResult(result)
      opts?.onReleased?.()
    } catch (e) {
      clearPendingBuild()
      const msg = e instanceof Error ? e.message : '建课失败，请稍后重试'
      if (job) {
        finishDomainBuildJobError(msg, opts?.onReleased)
      }
    } finally {
      resumePromise = null
    }
  })()
  return resumePromise
}
