import {
  getDomainBuildJobStatus,
  parseBuildDomainPollResult,
  pollDomainBuildJob,
  pollDomainJob,
  type BuildDomainResult,
  type DomainBuildJobPoll,
  type KnowledgeTree,
} from './api'
import { clearAppBusyIfAfter, setAppBusy } from './app-busy'
import { stashPrefetchTree } from './course-prefetch'

export type DomainBuildPhase = 'analyzing' | 'generating' | 'success' | 'error'
export type DomainBuildJobKind = 'build' | 'extend'

export interface DomainBuildJob {
  kind: DomainBuildJobKind
  topic: string
  domainId?: string
  phase: DomainBuildPhase
  message: string
  error?: string
  resultDomainId?: string
  resultMessage?: string
}

const PENDING_BUILD_KEY = 'regulus:pendingDomainBuild'

export interface PendingDomainJob {
  jobId: string
  topic: string
  kind?: DomainBuildJobKind
  domainId?: string
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
  if (phase === 'extend') return 'generating'
  return 'generating'
}

function busyReasonForKind(kind: DomainBuildJobKind): string {
  return kind === 'extend' ? 'extend' : 'build'
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

export function savePendingBuild(pending: PendingDomainJob): void {
  sessionStorage.setItem(PENDING_BUILD_KEY, JSON.stringify(pending))
}

export function clearPendingBuild(): void {
  sessionStorage.removeItem(PENDING_BUILD_KEY)
}

function loadPendingBuild(): PendingDomainJob | null {
  const raw = sessionStorage.getItem(PENDING_BUILD_KEY)
  if (!raw) return null
  try {
    const parsed = JSON.parse(raw) as PendingDomainJob
    if (parsed.jobId && parsed.topic) return parsed
  } catch {
    /* ignore */
  }
  return null
}

/** 若已有建课/扩展进行中则返回 false */
export function tryStartDomainBuildJob(
  topic: string,
  opts?: { kind?: DomainBuildJobKind; domainId?: string }
): boolean {
  if (isDomainBuildRunning()) return false
  const kind = opts?.kind ?? 'build'
  job = {
    kind,
    topic,
    domainId: opts?.domainId,
    phase: 'analyzing',
    message: kind === 'extend' ? '正在生成进阶节点…' : '任务已创建…',
  }
  setAppBusy(true, busyReasonForKind(kind))
  emit()
  return true
}

export function applyServerBuildProgress(status: DomainBuildJobPoll): void {
  if (!job || !isDomainBuildRunning()) return
  const message = status.message?.trim() || job.message
  const phase = mapServerBuildPhase(status.phase)
  if (job.phase === phase && job.message === message) return
  job.phase = phase
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
  const kind = job.kind
  job.phase = 'success'
  job.message =
    opts.message?.trim() ||
    (kind === 'extend'
      ? `「${job.topic}」进阶路径已解锁`
      : `「${job.topic}」课程已就绪`)
  job.resultDomainId = opts.domainId
  job.resultMessage = opts.message
  emit()
  clearAppBusyIfAfter(busyReasonForKind(kind), onReleased)
}

export function finishDomainBuildJobError(message: string, onReleased?: () => void): void {
  if (!job) return
  const kind = job.kind
  const fallback = kind === 'extend' ? '纵深扩展失败，请稍后重试' : '建课失败，请稍后重试'
  const err = message.trim() || fallback
  job.phase = 'error'
  job.message = err
  job.error = err
  emit()
  clearAppBusyIfAfter(busyReasonForKind(kind), onReleased)
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

function finishFromExtendResult(result: {
  tree?: KnowledgeTree
  message?: string
}): void {
  if (result.tree?.domainId) {
    stashPrefetchTree(result.tree)
    finishDomainBuildJobSuccess({
      domainId: result.tree.domainId,
      message: result.message,
    })
    return
  }
  finishDomainBuildJobError(result.message ?? '纵深扩展失败')
}

async function pollPendingBuild(
  pending: PendingDomainJob,
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
    const pending = loadPendingBuild()
    if (!pending) return

    // 内存里已有另一主题的建课：session pending 已过时，清掉避免每次刷新重试
    if (isDomainBuildRunning() && job?.topic !== pending.topic) {
      clearPendingBuild()
      return
    }

    try {
      const status = await getDomainBuildJobStatus(pending.jobId)
      const pendingKind = pending.kind ?? 'build'
      const startOpts =
        pendingKind === 'extend' && pending.domainId
          ? { kind: 'extend' as const, domainId: pending.domainId }
          : undefined

      if (status.status === 'done') {
        if (!status.result) {
          const msg =
            pendingKind === 'extend'
              ? '扩展已完成但未能加载结果，请刷新课程页查看'
              : '建课已完成但未能加载结果，请到「我的课程」查看或重新创建'
          tryStartDomainBuildJob(pending.topic, startOpts)
          finishDomainBuildJobError(msg, opts?.onReleased)
          clearPendingBuild()
          return
        }
        tryStartDomainBuildJob(pending.topic, startOpts)
        if (pendingKind === 'extend') {
          finishFromExtendResult(status.result as { tree?: KnowledgeTree; message?: string })
        } else {
          finishFromBuildResult(parseBuildDomainPollResult(status.result))
        }
        clearPendingBuild()
        opts?.onReleased?.()
        return
      }
      if (status.status === 'failed') {
        clearPendingBuild()
        const msg =
          status.error?.trim() ||
          status.message?.trim() ||
          (pendingKind === 'extend' ? '纵深扩展失败' : '建课失败')
        job = {
          kind: pendingKind,
          topic: pending.topic,
          domainId: pending.domainId,
          phase: 'error',
          message: msg,
          error: msg,
        }
        emit()
        clearAppBusyIfAfter(busyReasonForKind(pendingKind), opts?.onReleased)
        return
      }

      if (!isDomainBuildRunning() && !tryStartDomainBuildJob(pending.topic, startOpts)) {
        clearPendingBuild()
        return
      }
      applyServerBuildProgress(status)
      if (pendingKind === 'extend') {
        const final = await pollDomainJob(pending.jobId, (s) => applyServerBuildProgress(s))
        clearPendingBuild()
        if (final.status === 'done' && final.result) {
          finishFromExtendResult(final.result as { tree?: KnowledgeTree; message?: string })
        } else {
          finishDomainBuildJobError(
            final.error ?? final.message ?? '纵深扩展失败',
            opts?.onReleased
          )
        }
        opts?.onReleased?.()
        return
      }
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
