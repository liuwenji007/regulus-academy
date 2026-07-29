import { getActiveUserId } from './profile'

const API_BASE = ''

export interface UserProfile {
  id: string
  displayName: string
  profileSummary?: string
  profileBackground?: string
  profileGoal?: string
  profilePreference?: string
  domainProfiles?: DomainProfileEntry[]
  onboardedAt?: string
}

export interface DomainProfileEntry {
  userId: string
  domainId: string
  domainName?: string
  summary: string
  updatedAt?: string
}

export interface OnboardingPayload {
  role: string
  background: string
  goal?: string
  skip?: boolean
}

export interface TreeNode {
  key: string
  title: string
  requires?: string[]
}

export interface TreeLayer {
  key: string
  label: string
  time: string
  goal: string
  nodes: TreeNode[]
}

export interface TreeModule {
  key: string
  label: string
  goal?: string
  order?: number
  nodes: string[]
}

export interface KnowledgeTree {
  domainId: string
  domainName: string
  layers: TreeLayer[]
  modules?: TreeModule[]
}

export interface DomainSummary {
  id: string
  name: string
  slug?: string
  parentSlug?: string
  source?: string
  createdAt: string
  nodeTotal: number
  completed: number
}

export interface CourseLinkParent {
  domainId: string
  name: string
  slug?: string
}

export interface CourseDerivation {
  childDomainId: string
  childName: string
  childSlug?: string
  afterNodeKey?: string
  afterModuleKey?: string
  layerKey: string
  label: string
}

export interface CourseLinks {
  parent?: CourseLinkParent
  derivations?: CourseDerivation[]
}

export interface PublicDomainEntry {
  slug: string
  name: string
  description: string
  version: number
  nodeCount: number
}

export interface SkillExportMeta {
  slug: string
  filename: string
}

export interface IntentResult {
  slug: string
  displayName: string
  confidence: number
  reason: string
  source: 'skill_pack' | 'generated'
  scopeBreadth?: 'narrow' | 'moderate' | 'broad'
}

export interface AutoAuditSummary {
  score: number
  grade: string
  failCount: number
  warnCount: number
  infoCount: number
  headline: string
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
  progressKept?: number
  progressSkipped?: number
  /** 建课完成后规则体检摘要（无 LLM） */
  autoAudit?: AutoAuditSummary
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
  exercise?: SessionExercise | null
  nextNodeKey?: string
  nextNodeTitle?: string
}

export type AnswerFormat = 'text' | 'json' | 'choice'

export interface SessionExercise {
  answerFormat: AnswerFormat
  questionType?: string
  choices?: string[]
  choiceMode?: 'single' | 'multiple'
}

export interface MessageResponse {
  role: string
  content: string
  phase: string
  exercise?: SessionExercise | null
  nodeCompleted?: boolean
  progressUpdated?: boolean
  nextSessionId?: string
  nextNodeKey?: string
  nextNodeTitle?: string
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

export interface LLMPreset {
  id: string
  name: string
  baseUrl?: string
  defaultModel?: string
}

export interface LLMSettingsView {
  provider: string
  apiKeySet: boolean
  baseUrl: string
  model: string
  displayName: string
}

export interface LLMSettingsPayload {
  provider: string
  apiKey?: string
  baseUrl?: string
  model?: string
}

export interface LLMInfo {
  provider: string
  providerId?: string
  model: string
  configured: boolean
  presets?: LLMPreset[]
  settings?: LLMSettingsView
}

export interface LLMProfileView {
  id: string
  name: string
  provider: string
  baseUrl?: string
  model: string
  apiKeySet?: boolean
}

export interface LLMProfileInput {
  id?: string
  name: string
  provider: string
  baseUrl?: string
  model: string
  apiKey?: string
}

export interface LLMProfilesPayload {
  activeId: string
  asideProfileId?: string
  profiles: LLMProfileInput[]
}

export interface LLMConfigResponse extends LLMInfo {
  needsRestart?: boolean
  profiles?: LLMProfileView[]
  activeProfileId?: string
  asideProfileId?: string
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

export class QuotaExceededError extends ApiError {
  readonly needsByok = true
  readonly code: 'quota_exceeded' | 'build_quota_exceeded'
  constructor(message: string, code: 'quota_exceeded' | 'build_quota_exceeded' = 'quota_exceeded') {
    super(message)
    this.name = 'QuotaExceededError'
    this.code = code
  }
}

export class SessionBusyError extends ApiError {
  constructor(message = '正在回复上一条消息，请稍候…') {
    super(message)
    this.name = 'SessionBusyError'
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
    const body = data as { error?: string; code?: string }
    const msg = body.error ?? `请求失败 (${res.status})`
    if (res.status === 402 || body.code === 'quota_exceeded' || body.code === 'build_quota_exceeded') {
      const code = body.code === 'build_quota_exceeded' ? 'build_quota_exceeded' : 'quota_exceeded'
      throw new QuotaExceededError(msg, code)
    }
    throw new ApiError(msg)
  }
  return data as T
}

export async function getLLMInfo(): Promise<LLMInfo> {
  return request<LLMInfo>('/api/llm/info')
}

export async function getLLMConfig(): Promise<LLMConfigResponse> {
  return request<LLMConfigResponse>('/api/llm/config')
}

export async function saveLLMConfig(payload: LLMSettingsPayload): Promise<LLMConfigResponse> {
  return request<LLMConfigResponse>('/api/llm/config', {
    method: 'PUT',
    body: JSON.stringify(payload),
  })
}

export async function saveLLMProfiles(payload: LLMProfilesPayload): Promise<LLMConfigResponse> {
  return request<LLMConfigResponse>('/api/llm/profiles', {
    method: 'PUT',
    body: JSON.stringify(payload),
  })
}

export async function activateLLMProfile(id: string): Promise<LLMConfigResponse> {
  return request<LLMConfigResponse>('/api/llm/active', {
    method: 'PUT',
    body: JSON.stringify({ id }),
  })
}

export async function pingLLM(): Promise<{ status: string; message: string }> {
  return request<{ status: string; message: string }>('/api/llm/ping')
}

/** 按卡片内填写的配置探测连接（不必先保存） */
export async function pingLLMProfile(
  payload: LLMSettingsPayload
): Promise<{ status: string; message: string }> {
  return request<{ status: string; message: string }>('/api/llm/ping', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
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

export async function updateUserProfile(
  payload: string | { profileSummary?: string; profileBackground?: string; profileGoal?: string; profilePreference?: string },
): Promise<UserProfile> {
  const body = typeof payload === 'string' ? { profileSummary: payload } : payload
  return request<UserProfile>('/api/users/profile', {
    method: 'PATCH',
    body: JSON.stringify(body),
  })
}

export async function refineUserProfile(supplement: string): Promise<UserProfile> {
  return request<UserProfile>('/api/users/profile/refine', {
    method: 'POST',
    body: JSON.stringify({ supplement }),
  })
}

export async function migrateUserProfile(): Promise<UserProfile> {
  return request<UserProfile>('/api/users/profile/migrate', { method: 'POST' })
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

export async function submitOnboarding(userId: string, payload: OnboardingPayload): Promise<UserProfile> {
  // 切换角色时 active 可能仍是旧用户，须显式带上目标角色的 X-User-Id
  return request<UserProfile>(`/api/users/${encodeURIComponent(userId)}/onboarding`, {
    method: 'POST',
    body: JSON.stringify(payload),
    headers: { 'X-User-Id': userId },
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

export interface LastLessonShortcut {
  domainId: string
  domainName: string
  nodeKey: string
  nodeTitle: string
  sessionId: string
  phase: string
  status: string
  lastActiveAt: string
  canResume: boolean
}

export interface ShortcutRecommendation {
  source: 'planning' | 'progress' | 'gap' | string
  domainId: string
  domainName: string
  title?: string
  nodeKey?: string
  nodeTitle?: string
  minutes?: number
  completed: number
  nodeTotal: number
  sessionId?: string
  canResume?: boolean
  reason?: string
}

export interface LearningShortcuts {
  lastLesson: LastLessonShortcut | null
  recommendations: ShortcutRecommendation[]
  hasCourses: boolean
}

export async function getLearningShortcuts(): Promise<LearningShortcuts> {
  const data = await request<{
    lastLesson?: LastLessonShortcut | null
    recommendations?: ShortcutRecommendation[]
    hasCourses?: boolean
  }>('/api/learning/shortcuts')
  return {
    lastLesson: data.lastLesson ?? null,
    recommendations: Array.isArray(data.recommendations) ? data.recommendations : [],
    hasCourses: Boolean(data.hasCourses),
  }
}

export async function getPublicDomains(): Promise<PublicDomainEntry[]> {
  const data = await request<{ domains?: unknown }>('/api/domains/public')
  if (!Array.isArray(data.domains)) {
    throw new ApiError('公共知识库格式异常')
  }
  return data.domains as PublicDomainEntry[]
}

export interface DomainBuildJobPoll {
  status: 'running' | 'done' | 'failed'
  phase: string
  message: string
  topic?: string
  jobKind?: string
  domainId?: string
  result?: Record<string, unknown>
  error?: string
}

const DOMAIN_BUILD_POLL_MS = 3000
const DOMAIN_BUILD_POLL_MAX_MS = 6 * 60 * 1000

export async function submitDomainBuildJob(
  name: string,
  options?: { goal?: string; force?: boolean; action?: 'merge' | 'separate' }
): Promise<{ jobId: string }> {
  const data = await request<{ status?: string; jobId?: string }>('/api/domain/build', {
    method: 'POST',
    body: JSON.stringify({
      name,
      ...(options?.goal ? { goal: options.goal } : {}),
      ...(options?.force ? { force: true } : {}),
      ...(options?.action ? { action: options.action } : {}),
    }),
  })
  if (data.status !== 'accepted' || !data.jobId) {
    throw new ApiError('建课任务创建失败')
  }
  return { jobId: data.jobId }
}

export async function getDomainBuildJobStatus(jobId: string): Promise<DomainBuildJobPoll> {
  return request<DomainBuildJobPoll>(`/api/domain/build/jobs/${encodeURIComponent(jobId)}`)
}

export function parseBuildDomainPollResult(data: Record<string, unknown>): BuildDomainResult {
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
      autoAudit: data.autoAudit as AutoAuditSummary | undefined,
    }
  }

  if (data.domainId) {
    return { status: 'ready', tree: data as unknown as KnowledgeTree }
  }

  return {
    status: 'error',
    message: (data.message as string | undefined) ?? '无法解析课程加载结果',
  }
}

export async function pollDomainJob(
  jobId: string,
  onUpdate?: (status: DomainBuildJobPoll) => void,
  timeoutMessage = '任务超时，请稍后刷新查看是否已完成'
): Promise<DomainBuildJobPoll> {
  const started = Date.now()
  for (;;) {
    const status = await getDomainBuildJobStatus(jobId)
    onUpdate?.(status)
    if (status.status === 'done' || status.status === 'failed') {
      return status
    }
    if (Date.now() - started > DOMAIN_BUILD_POLL_MAX_MS) {
      throw new ApiError(timeoutMessage)
    }
    await new Promise((r) => setTimeout(r, DOMAIN_BUILD_POLL_MS))
  }
}

export async function pollDomainBuildJob(
  jobId: string,
  onUpdate?: (status: DomainBuildJobPoll) => void
): Promise<BuildDomainResult> {
  const status = await pollDomainJob(
    jobId,
    onUpdate,
    '建课超时，请稍后在课程列表查看是否已生成'
  )
  if (status.status === 'failed') {
    throw new ApiError(status.error?.trim() || status.message?.trim() || '建课失败')
  }
  if (!status.result) {
    throw new ApiError('建课完成但缺少结果')
  }
  return parseBuildDomainPollResult(status.result)
}

export async function buildDomain(
  name: string,
  options?: {
    goal?: string
    force?: boolean
    action?: 'merge' | 'separate'
    onProgress?: (status: DomainBuildJobPoll) => void
    onJobAccepted?: (jobId: string) => void
  }
): Promise<BuildDomainResult> {
  const { jobId } = await submitDomainBuildJob(name, options)
  options?.onJobAccepted?.(jobId)
  return pollDomainBuildJob(jobId, options?.onProgress)
}

export async function submitDomainBuildFromSource(
  input: { file?: File; url?: string },
  options?: { name?: string; goal?: string; force?: boolean }
): Promise<{ jobId: string }> {
  const userId = getActiveUserId()
  let res: Response
  if (input.file) {
    const form = new FormData()
    form.append('file', input.file)
    if (options?.name?.trim()) form.append('name', options.name.trim())
    if (options?.goal?.trim()) form.append('goal', options.goal.trim())
    if (options?.force) form.append('force', 'true')
    res = await fetch(`${API_BASE}/api/domain/build/from-source`, {
      method: 'POST',
      headers: userId ? { 'X-User-Id': userId } : {},
      body: form,
    })
  } else if (input.url?.trim()) {
    res = await fetch(`${API_BASE}/api/domain/build/from-source`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...(userId ? { 'X-User-Id': userId } : {}),
      },
      body: JSON.stringify({
        url: input.url.trim(),
        ...(options?.name?.trim() ? { name: options.name.trim() } : {}),
        ...(options?.goal?.trim() ? { goal: options.goal.trim() } : {}),
        ...(options?.force ? { force: true } : {}),
      }),
    })
  } else {
    throw new ApiError('请上传 PDF 或填写网页 URL')
  }

  const contentType = res.headers.get('content-type') ?? ''
  if (!contentType.includes('application/json')) {
    throw new ApiError('接口返回了页面而非数据，请硬刷新后重试')
  }
  const data = await res.json().catch(() => {
    throw new ApiError('无法解析服务器响应')
  })
  if (!res.ok) {
    const msg = (data as { error?: string }).error ?? `请求失败 (${res.status})`
    throw new ApiError(msg)
  }
  if (data.status !== 'accepted' || !data.jobId) {
    throw new ApiError('导入建课任务创建失败')
  }
  return { jobId: data.jobId as string }
}

export async function buildDomainFromSource(
  input: { file?: File; url?: string },
  options?: {
    name?: string
    goal?: string
    force?: boolean
    onProgress?: (status: DomainBuildJobPoll) => void
    onJobAccepted?: (jobId: string) => void
  }
): Promise<BuildDomainResult> {
  const { jobId } = await submitDomainBuildFromSource(input, options)
  options?.onJobAccepted?.(jobId)
  return pollDomainBuildJob(jobId, options?.onProgress)
}

export interface ExtendEligibility {
  eligible: boolean
  completed: number
  total: number
  minRatio: number
  reason?: string
  treeVersion?: number
}

export interface ExtendDomainResult {
  tree: KnowledgeTree
  addedNodeKeys: string[]
  treeVersion: number
  message?: string
}

export async function getExtendEligibility(domainId: string): Promise<ExtendEligibility> {
  return request<ExtendEligibility>(`/api/domain/${encodeURIComponent(domainId)}/extend/eligibility`)
}

export async function extendDomain(
  domainId: string,
  options?: {
    goal?: string
    onProgress?: (status: DomainBuildJobPoll) => void
    onJobAccepted?: (jobId: string) => void
  }
): Promise<ExtendDomainResult> {
  const data = await request<{ status?: string; jobId?: string }>(
    `/api/domain/${encodeURIComponent(domainId)}/extend`,
    {
      method: 'POST',
      body: JSON.stringify({
        confirm: true,
        ...(options?.goal?.trim() ? { goal: options.goal.trim() } : {}),
      }),
    }
  )
  if (data.status !== 'accepted' || !data.jobId) {
    throw new ApiError('扩展任务创建失败')
  }
  options?.onJobAccepted?.(data.jobId)
  const status = await pollDomainJob(
    data.jobId,
    options?.onProgress,
    '纵深扩展超时，请稍后刷新查看是否已生成'
  )
  if (status.status === 'failed') {
    throw new ApiError(status.error?.trim() || status.message?.trim() || '纵深扩展失败')
  }
  if (!status.result) {
    throw new ApiError('扩展完成但缺少结果')
  }
  return status.result as unknown as ExtendDomainResult
}

export async function getDomainTree(domainId: string): Promise<KnowledgeTree> {
  return request<KnowledgeTree>(`/api/domain/${domainId}/tree`)
}

export async function getCourseLinks(domainId: string): Promise<CourseLinks> {
  return request<CourseLinks>(`/api/domain/${encodeURIComponent(domainId)}/course-links`)
}

export interface DomainNoteItem {
  nodeKey: string
  contentMd: string
}

export interface DomainMistakeItem {
  nodeKey: string
  concepts: string[]
}

/** 读取课程节点学习笔记；可传 nodeKey 只取单节点 */
export async function getDomainNotes(
  domainId: string,
  nodeKey?: string
): Promise<DomainNoteItem[]> {
  const q = nodeKey ? `?nodeKey=${encodeURIComponent(nodeKey)}` : ''
  const data = await request<{ notes: DomainNoteItem[] }>(
    `/api/domain/${encodeURIComponent(domainId)}/notes${q}`
  )
  return data.notes ?? []
}

/** 读取课程节点踩坑概念；可传 nodeKey 只取单节点 */
export async function getDomainMistakes(
  domainId: string,
  nodeKey?: string
): Promise<DomainMistakeItem[]> {
  const q = nodeKey ? `?nodeKey=${encodeURIComponent(nodeKey)}` : ''
  const data = await request<{ mistakes: DomainMistakeItem[] }>(
    `/api/domain/${encodeURIComponent(domainId)}/mistakes${q}`
  )
  return data.mistakes ?? []
}

/** 从 Content-Disposition 头解析文件名，优先读 RFC 5987 的 filename* 参数 */
function parseDispositionFilename(disposition: string, fallback: string): string {
  // filename*=UTF-8''...（RFC 5987）
  const rfc5987 = disposition.match(/filename\*=UTF-8''([^;]+)/i)
  if (rfc5987) {
    try { return decodeURIComponent(rfc5987[1].trim()) } catch { /* fall through */ }
  }
  // filename="..."
  const plain = disposition.match(/filename="([^"]+)"/)
  if (plain) return plain[1]
  return fallback
}

export async function exportCoachSkillZip(): Promise<string> {
  const res = await fetch(`${API_BASE}/api/coach/export`)
  if (!res.ok) {
    throw new ApiError(`下载失败 (${res.status})`)
  }
  const blob = await res.blob()
  const disposition = res.headers.get('content-disposition') ?? ''
  const filename = parseDispositionFilename(disposition, 'regulus-coach.zip')
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
  return filename
}

/** 检测当前平台标识，供 CLI 下载 API 使用 */
export async function detectCLIPlatform(): Promise<string> {
  const ua = navigator.userAgent.toLowerCase()
  const platform = navigator.platform?.toLowerCase() ?? ''
  const uaData = (
    navigator as Navigator & {
      userAgentData?: {
        getHighEntropyValues?: (hints: string[]) => Promise<{ architecture?: string }>
      }
    }
  ).userAgentData

  if (platform.includes('mac') || ua.includes('mac')) {
    if (uaData?.getHighEntropyValues) {
      try {
        const { architecture } = await uaData.getHighEntropyValues(['architecture'])
        if (architecture === 'arm') return 'darwin_arm64'
        if (architecture === 'x86') return 'darwin_amd64'
      } catch {
        // fall through to UA heuristics
      }
    }
    if (ua.includes('arm64') || ua.includes('aarch64')) return 'darwin_arm64'
    return 'darwin_amd64'
  }
  if (platform.includes('linux') || ua.includes('linux')) {
    return 'linux_amd64'
  }
  if (platform.includes('win') || ua.includes('win')) {
    return 'windows_amd64'
  }
  return 'linux_amd64'
}

export async function exportCoachCLI(platform?: string): Promise<string> {
  const p = platform ?? (await detectCLIPlatform())
  const res = await fetch(`${API_BASE}/api/coach/cli?platform=${encodeURIComponent(p)}`)
  if (!res.ok) {
    throw new ApiError(`CLI 下载失败 (${res.status})，请从 GitHub Releases 获取`)
  }
  const blob = await res.blob()
  const disposition = res.headers.get('content-disposition') ?? ''
  const filename = parseDispositionFilename(disposition, `regulus-${p}`)
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
  return filename
}

export async function exportDomainZip(domainId: string): Promise<SkillExportMeta> {
  const userId = getActiveUserId()
  const res = await fetch(`${API_BASE}/api/domain/${domainId}/export`, {
    headers: userId ? { 'X-User-Id': userId } : {},
  })
  if (!res.ok) {
    const ct = res.headers.get('content-type') ?? ''
    if (ct.includes('application/json')) {
      const data = await res.json().catch(() => ({}))
      throw new ApiError((data as { error?: string }).error ?? `导出失败 (${res.status})`)
    }
    throw new ApiError(`导出失败 (${res.status})`)
  }
  const blob = await res.blob()
  const disposition = res.headers.get('content-disposition') ?? ''
  const filename = parseDispositionFilename(disposition, `${domainId}-domain.zip`)
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
  const slugMatch = filename.match(/^(.+)-domain\.zip$/)
  return { slug: slugMatch ? slugMatch[1] : domainId, filename }
}

/** 导出 Obsidian vault zip，触发浏览器下载，返回文件名 */
export async function exportDomainVault(domainId: string): Promise<string> {
  const userId = getActiveUserId()
  const res = await fetch(`${API_BASE}/api/domain/${domainId}/export/vault`, {
    headers: userId ? { 'X-User-Id': userId } : {},
  })
  if (!res.ok) {
    const ct = res.headers.get('content-type') ?? ''
    if (ct.includes('application/json')) {
      const data = await res.json().catch(() => ({}))
      throw new ApiError((data as { error?: string }).error ?? `导出失败 (${res.status})`)
    }
    throw new ApiError(`导出失败 (${res.status})`)
  }
  const blob = await res.blob()
  const disposition = res.headers.get('content-disposition') ?? ''
  const filename = parseDispositionFilename(disposition, `${domainId}-vault.zip`)
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
  return filename
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
      message: data.message as string | undefined,
      progressKept: data.progressKept as number | undefined,
      progressSkipped: data.progressSkipped as number | undefined,
      autoAudit: data.autoAudit as AutoAuditSummary | undefined,
    }
  }
  return {
    status: 'error',
    message: (data.message as string | undefined) ?? '重新生成失败',
  }
}

export interface AuditFinding {
  id: string
  severity: 'fail' | 'warn' | 'info'
  dimension: string
  nodeKey?: string
  code: string
  message: string
  suggestion: string
  autoFixable: boolean
  fixKind: string
}

export interface CourseAuditReport {
  version: number
  domainId: string
  domainName: string
  source: string
  treeVersion: number
  auditedAt: string
  summary: {
    score: number
    grade: string
    failCount: number
    warnCount: number
    infoCount: number
    headline: string
  }
  dimensions: Record<string, { score: number; findingCount: number }>
  findings: AuditFinding[]
  llmCritique?: { severity: string; feedback: string }
}

export interface OptimizePatchItem {
  id: string
  findingId: string
  nodeKey: string
  nodeTitle?: string
  before: Record<string, unknown>
  after: Record<string, unknown>
  summary: string
  benefits?: string[]
}

export interface OptimizePatch {
  domainId: string
  baseTreeVersion: number
  headline?: string
  patches: OptimizePatchItem[]
}

export async function submitDomainAuditJob(domainId: string): Promise<{ jobId: string }> {
  const data = await request<{ status?: string; jobId?: string }>(
    `/api/domain/${encodeURIComponent(domainId)}/audit`,
    { method: 'POST', body: '{}' }
  )
  if (data.status !== 'accepted' || !data.jobId) {
    throw new ApiError('体检任务创建失败')
  }
  return { jobId: data.jobId }
}

export async function submitDomainOptimizeJob(
  domainId: string,
  findingIds: string[],
  auditJobId?: string
): Promise<{ jobId: string }> {
  const data = await request<{ status?: string; jobId?: string }>(
    `/api/domain/${encodeURIComponent(domainId)}/optimize`,
    {
      method: 'POST',
      body: JSON.stringify({ findingIds, auditJobId: auditJobId ?? '' }),
    }
  )
  if (data.status !== 'accepted' || !data.jobId) {
    throw new ApiError('优化任务创建失败')
  }
  return { jobId: data.jobId }
}

export async function applyDomainOptimizePatches(
  domainId: string,
  jobId: string,
  patchIds: string[]
): Promise<{ tree: KnowledgeTree; treeVersion: number; message?: string }> {
  return request(`/api/domain/${encodeURIComponent(domainId)}/optimize/apply`, {
    method: 'POST',
    body: JSON.stringify({ jobId, patchIds, confirm: true }),
  })
}

export async function pollDomainAuditJob(
  jobId: string,
  onUpdate?: (status: DomainBuildJobPoll) => void
): Promise<CourseAuditReport> {
  const status = await pollDomainJob(jobId, onUpdate, '课程体检超时，请稍后重试')
  if (status.status === 'failed') {
    throw new ApiError(status.error?.trim() || status.message?.trim() || '课程体检失败')
  }
  if (!status.result) {
    throw new ApiError('体检完成但缺少报告')
  }
  return status.result as unknown as CourseAuditReport
}

export async function pollDomainOptimizeJob(
  jobId: string,
  onUpdate?: (status: DomainBuildJobPoll) => void
): Promise<OptimizePatch> {
  const status = await pollDomainJob(jobId, onUpdate, '课程优化超时，请稍后重试')
  if (status.status === 'failed') {
    throw new ApiError(status.error?.trim() || status.message?.trim() || '课程优化失败')
  }
  if (!status.result) {
    throw new ApiError('优化完成但缺少结果')
  }
  return status.result as unknown as OptimizePatch
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

/** 已完成节点点击「继续 · 下一节」：进入下一未完成节点；已有会话则恢复，否则生成开场讲解 */
export async function startNextSession(completedSessionId: string): Promise<StartSessionResponse> {
  return request<StartSessionResponse>('/api/session/next', {
    method: 'POST',
    body: JSON.stringify({ sessionId: completedSessionId }),
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

export type CoachStreamStage = 'thinking' | 'grading' | 'mastery' | 'exercise' | string

export interface SendMessageStreamHandlers {
  /** SSE 响应头已成功、可读 body（请求已进入服务端流式处理） */
  onOpen?: () => void
  onStage?: (stage: CoachStreamStage) => void
  onDelta?: (text: string) => void
}

interface CoachStreamSSEEvent {
  type: 'stage' | 'delta' | 'message' | 'error'
  stage?: string
  text?: string
  message?: MessageResponse
  code?: string
  error?: string
}

/** 流式发送教练消息；未收到 message 尾包时抛错，由调用方降级。 */
export async function sendMessageStream(
  sessionId: string,
  content: string,
  handlers?: SendMessageStreamHandlers
): Promise<MessageResponse> {
  const userId = getActiveUserId()
  const res = await fetch(`${API_BASE}/api/session/message/stream`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'text/event-stream',
      ...(userId ? { 'X-User-Id': userId } : {}),
    },
    body: JSON.stringify({ sessionId, content }),
  })

  const contentType = res.headers.get('content-type') ?? ''
  if (!res.ok) {
    if (contentType.includes('application/json')) {
      const data = (await res.json().catch(() => ({}))) as { error?: string; code?: string }
      const msg = data.error ?? `请求失败 (${res.status})`
      if (res.status === 402 || data.code === 'quota_exceeded' || data.code === 'build_quota_exceeded') {
        const code = data.code === 'build_quota_exceeded' ? 'build_quota_exceeded' : 'quota_exceeded'
        throw new QuotaExceededError(msg, code)
      }
      throw new ApiError(msg)
    }
    throw new ApiError(`流式请求失败 (${res.status})`)
  }

  if (!res.body) {
    throw new ApiError('流式响应为空')
  }

  handlers?.onOpen?.()

  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  let finalMessage: MessageResponse | null = null
  let streamError: Error | null = null

  const handleEvent = (raw: string) => {
    const dataLines = raw
      .split('\n')
      .filter((line) => line.startsWith('data:'))
      .map((line) => line.slice(5).trimStart())
    if (!dataLines.length) return
    const payload = dataLines.join('\n')
    if (!payload) return
    let ev: CoachStreamSSEEvent
    try {
      ev = JSON.parse(payload) as CoachStreamSSEEvent
    } catch {
      return
    }
    switch (ev.type) {
      case 'stage':
        if (ev.stage) handlers?.onStage?.(ev.stage)
        break
      case 'delta':
        if (ev.text) handlers?.onDelta?.(ev.text)
        break
      case 'message':
        if (ev.message) finalMessage = ev.message
        break
      case 'error': {
        const msg = ev.error || '流式回复失败'
        if (ev.code === 'busy') {
          streamError = new ApiError(msg)
          ;(streamError as ApiError & { code?: string }).code = 'busy'
        } else if (ev.code === 'quota_exceeded' || ev.code === 'build_quota_exceeded') {
          streamError = new QuotaExceededError(
            msg,
            ev.code === 'build_quota_exceeded' ? 'build_quota_exceeded' : 'quota_exceeded'
          )
        } else {
          streamError = new ApiError(msg)
        }
        break
      }
    }
  }

  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    let sep: number
    while ((sep = buffer.indexOf('\n\n')) >= 0) {
      const chunk = buffer.slice(0, sep)
      buffer = buffer.slice(sep + 2)
      handleEvent(chunk)
      if (streamError) throw streamError
      if (finalMessage) return finalMessage
    }
  }
  if (buffer.trim()) {
    handleEvent(buffer)
  }
  if (streamError) throw streamError
  if (finalMessage) return finalMessage
  throw new ApiError('流式中断：未收到完整回复')
}


export function phaseLabel(phase: string): string {
  const map: Record<string, string> = {
    explain: '讲解',
    exercise: '练习',
    review: '巩固',
    completed: '已完成',
    intake: '倾听中',
    plan_ready: '已出方案',
  }
  return map[phase] ?? phase
}

export interface PlanningMatrixItem {
  title: string
  why?: string
  minutes?: number
  next_step?: string
  reason?: string
}

export interface PlanningActionItem {
  title: string
  minutes: number
  kind: 'task' | 'learning'
  reason?: string
}

export interface PlanningLearningFocus {
  area: string
  rationale: string
  suggested_minutes: number
  matched_domain_id?: string
  matched_node_key?: string
  matched_node_title?: string
}

export interface PlanningFocusTodayLearning {
  title: string
  minutes: number
  matched_domain_id?: string
  matched_node_key?: string
  matched_node_title?: string
}

export interface PlanningFocus {
  north_star: string
  why?: string
  week_wedge?: string
  today_learning?: PlanningFocusTodayLearning | null
}

export interface PlanningClearItem {
  title: string
  next_step?: string
  minutes?: number
}

export interface PlanningUIState {
  north_star_pinned: boolean
  checked?: Record<string, boolean>
}

export interface PlanningResult {
  situation_summary: string
  focus?: PlanningFocus | null
  clear_first?: PlanningClearItem[]
  matrix: {
    important_urgent: PlanningMatrixItem[]
    important_not_urgent: PlanningMatrixItem[]
    quick_wins: PlanningMatrixItem[]
    defer_or_drop: PlanningMatrixItem[]
  }
  action_plan: {
    today: PlanningActionItem[]
    this_week: PlanningActionItem[]
  }
  learning_focus: PlanningLearningFocus[]
  mindset_note: string
  ui_state?: PlanningUIState | null
}

export interface PlanningMessage {
  id: number
  sessionId: string
  role: string
  content: string
}

export interface PlanningSessionDetail {
  sessionId: string
  phase: string
  status?: string
  messages: PlanningMessage[]
  plan?: PlanningResult | null
}

export interface StartPlanningResponse {
  sessionId: string
  phase: string
  resumed: boolean
  content?: string
  messages: PlanningMessage[]
  plan?: PlanningResult | null
}

export interface PlanningMessageResponse {
  role: string
  content: string
  phase: string
  synthesized?: boolean
  plan?: PlanningResult | null
}

export async function startPlanning(forceNew = false): Promise<StartPlanningResponse> {
  return request<StartPlanningResponse>('/api/planning/start', {
    method: 'POST',
    body: JSON.stringify({ forceNew }),
  })
}

export async function getPlanningSession(sessionId: string): Promise<PlanningSessionDetail> {
  return request<PlanningSessionDetail>(`/api/planning/${encodeURIComponent(sessionId)}`)
}

export async function getActivePlanningSession(): Promise<{ sessionId: string | null; phase?: string }> {
  return request<{ sessionId: string | null; phase?: string }>('/api/planning/active')
}

export async function sendPlanningMessage(
  sessionId: string,
  content: string
): Promise<PlanningMessageResponse> {
  return request<PlanningMessageResponse>('/api/planning/message', {
    method: 'POST',
    body: JSON.stringify({ sessionId, content }),
  })
}

export interface PlanningFocusPatch {
  north_star_pinned?: boolean
  north_star?: string
  checked?: Record<string, boolean>
}

export async function patchPlanningFocus(
  sessionId: string,
  patch: PlanningFocusPatch
): Promise<{ sessionId: string; plan: PlanningResult }> {
  return request<{ sessionId: string; plan: PlanningResult }>(
    `/api/planning/${encodeURIComponent(sessionId)}/focus`,
    {
      method: 'PATCH',
      body: JSON.stringify(patch),
    }
  )
}

// —— 学习旁路助手 ——

export type AsideIntent = 'what' | 'reading' | 'expand' | 'ask'

export interface TermCardPayload {
  term: string
  originalText: string
  ipa?: string
  readingCn?: string
  oneLiner?: string
  explanation?: string
  analogy?: string
  relationToLesson?: string
  prerequisites?: string[]
  confidenceHint?: string
  redirectHint?: string
}

export interface AsideExplainResult {
  cached: boolean
  card: TermCardPayload
  markdown: string
  hitCount: number
}

export interface AsideTermItem {
  id: number
  domainId: string
  nodeKey?: string
  normalizedTerm: string
  originalText: string
  hitCount: number
  lastHitAt: string
  term?: string
  oneLiner?: string
  card?: TermCardPayload
}

export interface KnowledgeGapItem {
  id: number
  userId: string
  domainId: string
  nodeKey?: string
  concept: string
  source: string
  hitCount: number
  severity: number
  matchedDomainId?: string
  matchedNodeKey?: string
  reason?: string
  lastHitAt: string
}

export interface AsideMessageItem {
  id: number
  userId: string
  domainId: string
  nodeKey?: string
  role: string
  content: string
  anchorText?: string
  intent?: string
  createdAt: string
}

export async function asideExplain(body: {
  domainId?: string
  nodeKey?: string
  coachSessionId?: string
  anchorText: string
  intent?: AsideIntent
}): Promise<AsideExplainResult> {
  return request<AsideExplainResult>('/api/aside/explain', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export async function listAsideTerms(domainId?: string): Promise<AsideTermItem[]> {
  const q = domainId ? `?domainId=${encodeURIComponent(domainId)}` : ''
  const data = await request<{ terms?: AsideTermItem[] }>(`/api/aside/terms${q}`)
  return Array.isArray(data.terms) ? data.terms : []
}

export async function listAsideMessages(domainId?: string): Promise<AsideMessageItem[]> {
  const q = domainId ? `?domainId=${encodeURIComponent(domainId)}` : ''
  const data = await request<{ messages?: AsideMessageItem[] }>(`/api/aside/messages${q}`)
  return Array.isArray(data.messages) ? data.messages : []
}

export async function listKnowledgeGaps(domainId?: string): Promise<KnowledgeGapItem[]> {
  const q = domainId ? `?domainId=${encodeURIComponent(domainId)}` : ''
  const data = await request<{ gaps?: KnowledgeGapItem[] }>(`/api/aside/gaps${q}`)
  return Array.isArray(data.gaps) ? data.gaps : []
}

export async function resolveKnowledgeGap(id: number): Promise<void> {
  await request<{ status: string }>(`/api/aside/gaps/${id}/resolve`, { method: 'POST' })
}

export interface AsideAskStreamHandlers {
  onDelta?: (text: string) => void
  onDone?: (content: string) => void
  onError?: (err: Error) => void
}

/** 旁路自由问答 SSE */
export async function asideAskStream(
  body: {
    domainId?: string
    nodeKey?: string
    coachSessionId?: string
    anchorText?: string
    question: string
  },
  handlers?: AsideAskStreamHandlers
): Promise<string> {
  const userId = getActiveUserId()
  const res = await fetch(`${API_BASE}/api/aside/ask/stream`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'text/event-stream',
      ...(userId ? { 'X-User-Id': userId } : {}),
    },
    body: JSON.stringify(body),
  })

  if (!res.ok) {
    const contentType = res.headers.get('content-type') ?? ''
    if (contentType.includes('application/json')) {
      const data = (await res.json().catch(() => ({}))) as { error?: string }
      throw new ApiError(data.error ?? `请求失败 (${res.status})`)
    }
    throw new ApiError(`流式请求失败 (${res.status})`)
  }
  if (!res.body) throw new ApiError('流式响应为空')

  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  let finalContent = ''
  let streamError: Error | null = null

  const handleEvent = (raw: string) => {
    const dataLines = raw
      .split('\n')
      .filter((line) => line.startsWith('data:'))
      .map((line) => line.slice(5).trimStart())
    if (!dataLines.length) return
    const payload = dataLines.join('\n')
    if (!payload) return
    let ev: { type?: string; text?: string; content?: string; error?: string }
    try {
      ev = JSON.parse(payload)
    } catch {
      return
    }
    if (ev.type === 'delta' && ev.text) {
      handlers?.onDelta?.(ev.text)
    } else if (ev.type === 'message' && ev.content) {
      finalContent = ev.content
      handlers?.onDone?.(ev.content)
    } else if (ev.type === 'error') {
      streamError = new ApiError(ev.error || '旁路回复失败')
      handlers?.onError?.(streamError)
    }
  }

  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    let sep: number
    while ((sep = buffer.indexOf('\n\n')) >= 0) {
      const chunk = buffer.slice(0, sep)
      buffer = buffer.slice(sep + 2)
      handleEvent(chunk)
    }
  }
  if (buffer.trim()) handleEvent(buffer)

  if (streamError) throw streamError
  return finalContent
}
