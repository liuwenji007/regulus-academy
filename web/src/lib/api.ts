import { getActiveUserId } from './profile'

const API_BASE = ''

export interface UserProfile {
  id: string
  displayName: string
}

export interface TreeNode {
  key: string
  title: string
}

export interface TreeLayer {
  key: string
  label: string
  time: string
  goal: string
  nodes: TreeNode[]
}

export interface KnowledgeTree {
  domainId: string
  domainName: string
  layers: TreeLayer[]
}

export interface DomainSummary {
  id: string
  name: string
  slug?: string
  source?: string
  createdAt: string
  nodeTotal: number
  completed: number
}

export interface PublicDomainEntry {
  slug: string
  name: string
  description: string
  version: number
  nodeCount: number
}

export interface DomainExportPackage {
  slug: string
  domainName: string
  description: string
  version: number
  source: string
  files: Record<string, string>
}

export interface IntentResult {
  slug: string
  displayName: string
  confidence: number
  reason: string
  source: 'skill_pack' | 'generated'
  scopeBreadth?: 'narrow' | 'moderate' | 'broad'
}

export interface BuildDomainResult {
  status: 'ready' | 'error' | 'related'
  message?: string
  relation?: string
  existingDomain?: DomainSummary
  intent?: IntentResult
  tree?: KnowledgeTree
  generated?: boolean
  personalized?: boolean
  reason?: string
  redirected?: boolean
  reused?: boolean
  focusNodeKeys?: string[]
  focusLabel?: string
}

export interface UserProgress {
  userId: string
  domainId: string
  nodeKey: string
  layer: string
  status: string
  mastery: number
}

export interface SessionMessage {
  id: number
  sessionId: string
  role: string
  content: string
}

export interface SessionDetail {
  sessionId: string
  domainId: string
  nodeKey: string
  nodeTitle: string
  phase: string
  messages: SessionMessage[]
}

export interface MessageResponse {
  role: string
  content: string
  phase: string
  nodeCompleted?: boolean
  progressUpdated?: boolean
}

export interface StartSessionResponse {
  sessionId: string
  nodeKey: string
  domainId: string
  phase: string
  content?: string
  resumed?: boolean
}

export interface ActiveSessionResponse {
  sessionId: string | null
  phase?: string
  nodeKey?: string
  domainId?: string
}

export interface LLMInfo {
  provider: string
  model: string
  configured: boolean
  presets?: string[]
}

export type GatewayPlatformStatus = 'disabled' | 'pending' | 'waiting' | 'ready'

export interface GatewayPlatform {
  id: string
  name: string
  /** 平台开关（用户配置） */
  platformEnabled?: boolean
  /** 运行时是否生效（Gateway 总开关 + 平台开关） */
  enabled: boolean
  configured: boolean
  status: GatewayPlatformStatus
  connection?: string
  mode?: string
  webhookUrl?: string
  needsPublicHttps?: boolean
  envVars?: string[]
  setupHint?: string
  setupSteps?: string[]
  runtime?: {
    connected?: boolean
    lastEventAt?: string | null
    lastError?: string
  }
}

export interface ChannelBinding {
  platform: string
  platformUserId: string
  userId: string
  displayNameSnapshot?: string
  createdAt: string
}

export interface GatewayCommand {
  command: string
  description: string
}

export interface GatewayInfo {
  enabled: boolean
  activePlatforms: number
  publicBaseUrl: string
  platforms: GatewayPlatform[]
  bindings: ChannelBinding[]
  commands: GatewayCommand[]
  settings: GatewaySettingsView
  needsRestart?: boolean
  runtime?: {
    platformHealth?: Record<string, { connected?: boolean; lastEventAt?: string; lastError?: string }>
  }
}

export interface GatewaySettingsView {
  enabled: boolean
  publicUrl: string
  telegramEnabled: boolean
  telegramBotTokenSet: boolean
  telegramAllowedUsers: string
  dingtalkEnabled: boolean
  dingtalkClientId: string
  dingtalkClientSecretSet: boolean
  feishuEnabled: boolean
  feishuMode: string
  feishuAppId: string
  feishuAppSecretSet: boolean
  feishuAllowedUsers: string
  wecomEnabled: boolean
  wecomCorpId: string
  wecomAgentId: string
  wecomSecretSet: boolean
  wecomTokenSet: boolean
  wecomEncodingAesKeySet: boolean
  wecomAllowedUsers: string
}

export interface GatewaySettingsPayload {
  enabled: boolean
  publicUrl: string
  telegramEnabled: boolean
  telegramBotToken?: string
  telegramAllowedUsers: string
  dingtalkEnabled: boolean
  dingtalkClientId: string
  dingtalkClientSecret?: string
  feishuEnabled: boolean
  feishuMode: string
  feishuAppId: string
  feishuAppSecret?: string
  feishuAllowedUsers: string
  wecomEnabled: boolean
  wecomCorpId: string
  wecomAgentId: string
  wecomSecret?: string
  wecomToken?: string
  wecomEncodingAesKey?: string
  wecomAllowedUsers: string
}

export class ApiError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'ApiError'
  }
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const userId = getActiveUserId()
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(userId ? { 'X-User-Id': userId } : {}),
      ...options?.headers,
    },
  })
  const contentType = res.headers.get('content-type') ?? ''
  if (!contentType.includes('application/json')) {
    throw new ApiError(
      '接口返回了页面而非数据，请硬刷新（Cmd+Shift+R）或清除站点缓存后重试'
    )
  }
  const data = await res.json().catch(() => {
    throw new ApiError('无法解析服务器响应')
  })
  if (!res.ok) {
    const msg = (data as { error?: string }).error ?? `请求失败 (${res.status})`
    throw new ApiError(msg)
  }
  return data as T
}

export async function getLLMInfo(): Promise<LLMInfo> {
  return request<LLMInfo>('/api/llm/info')
}

export async function getGatewayInfo(): Promise<GatewayInfo> {
  return request<GatewayInfo>('/api/gateway/info')
}

export async function saveGatewayConfig(payload: GatewaySettingsPayload): Promise<GatewayInfo> {
  return request<GatewayInfo>('/api/gateway/config', {
    method: 'PUT',
    body: JSON.stringify(payload),
  })
}

export interface ChannelBindCode {
  code: string
  expiresAt: string
  hint: string
}

export async function createChannelBindCode(): Promise<ChannelBindCode> {
  return request<ChannelBindCode>('/api/channel/bind-code', { method: 'POST' })
}

export async function updateUserProfile(profileSummary: string): Promise<UserProfile> {
  return request<UserProfile>('/api/users/profile', {
    method: 'PATCH',
    body: JSON.stringify({ profileSummary }),
  })
}

export async function listUsers(): Promise<UserProfile[]> {
  const data = await request<{ users?: UserProfile[] }>('/api/users')
  return data.users ?? []
}

export async function createUser(displayName: string): Promise<UserProfile> {
  return request<UserProfile>('/api/users', {
    method: 'POST',
    body: JSON.stringify({ displayName }),
  })
}

export async function deleteUser(id: string, confirmName: string): Promise<void> {
  await request<{ status: string }>(`/api/users/${encodeURIComponent(id)}`, {
    method: 'DELETE',
    body: JSON.stringify({ confirmName }),
  })
}

export async function getDomains(): Promise<DomainSummary[]> {
  const data = await request<{ domains?: unknown }>('/api/domains')
  if (!Array.isArray(data.domains)) {
    throw new ApiError('课程列表格式异常')
  }
  return data.domains as DomainSummary[]
}

export async function getPublicDomains(): Promise<PublicDomainEntry[]> {
  const data = await request<{ domains?: unknown }>('/api/domains/public')
  if (!Array.isArray(data.domains)) {
    throw new ApiError('公共知识库格式异常')
  }
  return data.domains as PublicDomainEntry[]
}

export async function buildDomain(name: string, options?: { goal?: string; force?: boolean }): Promise<BuildDomainResult> {
  const data = await request<Record<string, unknown>>('/api/domain/build', {
    method: 'POST',
    body: JSON.stringify({
      name,
      ...(options?.goal ? { goal: options.goal } : {}),
      ...(options?.force ? { force: true } : {}),
    }),
  })

  if (data.status === 'related') {
    return {
      status: 'related',
      message: data.message as string | undefined,
      relation: data.relation as string | undefined,
      existingDomain: data.existingDomain as DomainSummary | undefined,
      intent: data.intent as IntentResult | undefined,
    }
  }

  if (data.status === 'ready' && data.tree) {
    return {
      status: 'ready',
      intent: data.intent as IntentResult | undefined,
      tree: data.tree as KnowledgeTree,
      generated: data.generated as boolean | undefined,
      personalized: data.personalized as boolean | undefined,
      reason: data.reason as string | undefined,
      redirected: data.redirected as boolean | undefined,
      message: data.message as string | undefined,
      reused: data.reused as boolean | undefined,
      focusNodeKeys: data.focusNodeKeys as string[] | undefined,
      focusLabel: data.focusLabel as string | undefined,
    }
  }

  // 兼容旧版扁平结构
  if (data.domainId) {
    return { status: 'ready', tree: data as unknown as KnowledgeTree }
  }

  return {
    status: 'error',
    message: (data.message as string | undefined) ?? '无法解析课程加载结果',
  }
}

export async function getDomainTree(domainId: string): Promise<KnowledgeTree> {
  return request<KnowledgeTree>(`/api/domain/${domainId}/tree`)
}

export async function exportDomain(domainId: string): Promise<DomainExportPackage> {
  return request<DomainExportPackage>(`/api/domain/${domainId}/export`)
}

export async function deleteDomain(id: string, confirmName: string): Promise<void> {
  await request<{ status: string }>(`/api/domain/${encodeURIComponent(id)}`, {
    method: 'DELETE',
    body: JSON.stringify({ confirmName }),
  })
}

export async function regenerateDomain(
  id: string,
  confirmName: string
): Promise<BuildDomainResult> {
  const data = await request<Record<string, unknown>>(
    `/api/domain/${encodeURIComponent(id)}/regenerate`,
    {
      method: 'POST',
      body: JSON.stringify({ confirmName }),
    }
  )
  if (data.status === 'ready' && data.tree) {
    return {
      status: 'ready',
      intent: data.intent as IntentResult | undefined,
      tree: data.tree as KnowledgeTree,
      generated: data.generated as boolean | undefined,
    }
  }
  return {
    status: 'error',
    message: (data.message as string | undefined) ?? '重新生成失败',
  }
}

export async function getUserProgress(domainId?: string): Promise<UserProgress[]> {
  const q = domainId ? `?domainId=${encodeURIComponent(domainId)}` : ''
  const data = await request<{ progress: UserProgress[] }>(`/api/user/progress${q}`)
  return data.progress ?? []
}

export async function getActiveSession(
  domainId: string,
  nodeKey: string
): Promise<ActiveSessionResponse> {
  const q = `?domainId=${encodeURIComponent(domainId)}&nodeKey=${encodeURIComponent(nodeKey)}`
  return request<ActiveSessionResponse>(`/api/sessions/active${q}`)
}

export async function startSession(
  domainId: string,
  nodeKey: string,
  layer: string
): Promise<StartSessionResponse> {
  return request<StartSessionResponse>('/api/session/start', {
    method: 'POST',
    body: JSON.stringify({ domainId, nodeKey, layer }),
  })
}

export async function getSession(sessionId: string): Promise<SessionDetail> {
  return request<SessionDetail>(`/api/session/${sessionId}`)
}

export async function sendMessage(
  sessionId: string,
  content: string
): Promise<MessageResponse> {
  return request<MessageResponse>('/api/session/message', {
    method: 'POST',
    body: JSON.stringify({ sessionId, content }),
  })
}

export function phaseLabel(phase: string): string {
  const map: Record<string, string> = {
    explain: '讲解',
    exercise: '练习',
    review: '巩固',
    completed: '已完成',
  }
  return map[phase] ?? phase
}
