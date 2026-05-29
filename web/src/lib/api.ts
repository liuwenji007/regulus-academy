const API_BASE = ''

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

export interface IntentResult {
  slug: string
  displayName: string
  confidence: number
  reason: string
  source: 'skill_pack' | 'generated'
}

export interface BuildDomainResult {
  status: 'ready' | 'error'
  message?: string
  intent?: IntentResult
  tree?: KnowledgeTree
  generated?: boolean
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

export class ApiError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'ApiError'
  }
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
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

export async function getDomains(): Promise<DomainSummary[]> {
  const data = await request<{ domains?: unknown }>('/api/domains')
  if (!Array.isArray(data.domains)) {
    throw new ApiError('课程列表格式异常')
  }
  return data.domains as DomainSummary[]
}

export async function buildDomain(name: string): Promise<BuildDomainResult> {
  const data = await request<Record<string, unknown>>('/api/domain/build', {
    method: 'POST',
    body: JSON.stringify({ name }),
  })

  if (data.status === 'ready' && data.tree) {
    return {
      status: 'ready',
      intent: data.intent as IntentResult | undefined,
      tree: data.tree as KnowledgeTree,
      generated: data.generated as boolean | undefined,
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
