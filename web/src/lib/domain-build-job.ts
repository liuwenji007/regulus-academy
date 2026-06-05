import { clearAppBusyIfAfter, setAppBusy } from './app-busy'

export type DomainBuildPhase = 'analyzing' | 'generating' | 'success' | 'error'

export interface DomainBuildJob {
  topic: string
  phase: DomainBuildPhase
  message: string
  error?: string
  resultDomainId?: string
  resultMessage?: string
}

let job: DomainBuildJob | null = null
const listeners = new Set<() => void>()

function emit(): void {
  for (const fn of listeners) fn()
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

/** 若已有建课进行中则返回 false */
export function tryStartDomainBuildJob(topic: string): boolean {
  if (isDomainBuildRunning()) return false
  job = { topic, phase: 'analyzing', message: '正在分析学习目标…' }
  setAppBusy(true, 'build')
  emit()
  return true
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
