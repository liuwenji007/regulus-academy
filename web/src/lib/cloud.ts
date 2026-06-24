import { getActiveUserId } from './profile'
import { ApiError, QuotaExceededError } from './api'
import { showByokModal } from '../components/byok-modal'

export { QuotaExceededError }

export interface CloudInfo {
  deployment: string
  githubUrl: string
  docsUrl: string
  demoUrl: string
  demoLabel: string
  selfHostHint: string
  quotaDaily: number
  quotaDailyBuilds: number
}

export interface CloudStats {
  totalLearners: number
  activeLast7Days: number
  platformTokensToday: number
  asOf: string
}

export interface LLMQuota {
  used: number
  limit: number
  remaining: number
  buildUsed: number
  buildLimit: number
  buildRemaining: number
  hasByok: boolean
  promptTokensToday: number
  completionTokensToday: number
}

let cachedInfo: CloudInfo | null = null

export function isCloudDeployment(info?: CloudInfo | null): boolean {
  return (info ?? cachedInfo)?.deployment === 'cloud'
}

export async function fetchCloudInfo(): Promise<CloudInfo> {
  const res = await fetch('/api/cloud/info')
  if (!res.ok) {
    return { deployment: 'selfhosted', githubUrl: '', docsUrl: '', demoUrl: '', demoLabel: '', selfHostHint: '', quotaDaily: 0, quotaDailyBuilds: 0 }
  }
  const data = (await res.json()) as CloudInfo
  cachedInfo = data
  return data
}

export function getCachedCloudInfo(): CloudInfo | null {
  return cachedInfo
}

export async function fetchCloudStats(): Promise<CloudStats> {
  const res = await fetch('/api/cloud/stats')
  if (!res.ok) throw new ApiError('无法加载共学统计')
  return res.json() as Promise<CloudStats>
}

export async function fetchLLMQuota(): Promise<LLMQuota> {
  const userId = getActiveUserId()
  if (!userId) throw new ApiError('未选择学习角色')
  const res = await fetch('/api/user/llm-quota', {
    headers: { 'X-User-Id': userId },
  })
  const data = await res.json()
  if (!res.ok) throw new ApiError((data as { error?: string }).error ?? '无法加载额度')
  return data as LLMQuota
}

export async function saveUserLLMKey(payload: {
  provider: string
  apiKey: string
  baseUrl?: string
  model?: string
}): Promise<void> {
  const userId = getActiveUserId()
  const res = await fetch('/api/user/llm-key', {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      ...(userId ? { 'X-User-Id': userId } : {}),
    },
    body: JSON.stringify(payload),
  })
  const data = await res.json()
  if (!res.ok) throw new ApiError((data as { error?: string }).error ?? '保存失败')
}

export function isQuotaExceededError(err: unknown): err is QuotaExceededError {
  return err instanceof QuotaExceededError
}

/** 额度用尽时弹出 BYOK；返回是否已保存 Key */
export async function tryRecoverFromQuotaError(err: unknown): Promise<boolean> {
  if (!isQuotaExceededError(err)) return false
  return showByokModal(err.code === 'build_quota_exceeded' ? 'build' : 'message')
}
