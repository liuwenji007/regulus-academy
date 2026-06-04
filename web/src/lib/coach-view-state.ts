import type { SessionDetail, SessionExercise } from './api'
import {
  extractEmbeddedExercise,
  exercisePlaceholder,
  isExerciseSubmitPrompt,
  normalizeCoachReply,
  normalizeSessionExercise,
  type ExerciseDraft,
} from './coach-exercise'

export interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
}

export type ComposerMode = 'chat' | 'exercise_text' | 'exercise_choice' | 'completed'

export type ScrollMode = 'readable' | 'bottom'

export const EXERCISE_MARKER = '做完后直接把答案发给我'
export const REAL_WORLD_CASE_PROMPT = '实际案例'
/** 与后端 skip_mastery 触发词一致 */
export const SKIP_MASTERY_PROMPT = '已经掌握，下一节'

const PRACTICE_INVITE_PATTERNS = [
  '开始练习',
  '再来一道',
  '再练一题',
  '再练一遍',
  '练一道题',
  '练一题',
  '再出一题',
  '继续练习',
]

export interface PendingTurn {
  userContent: string
  assistantContent?: string
  phase?: string
  exercise?: SessionExercise | null
}

export interface BootstrapPreview {
  messages: ChatMessage[]
  phase: string
  domainId: string
  nodeKey: string
}

export interface CoachViewState {
  sessionId: string
  messages: ChatMessage[]
  phase: string
  exercise: SessionExercise | null
  composerMode: ComposerMode
  practiceLabel: string
  hasAttemptedExercise: boolean
  showInlinePracticeOnLast: boolean
  showInlineMasteryOnLast: boolean
  showInlineCaseOnLast: boolean
  placeholder: string
  sending: boolean
  error: string
  toastHtml: string
  nodeTitle: string
  domainId: string
  pendingNextTitle: string
  scrollMode: ScrollMode
  completedNextTitle: string
}

export function maxMessageId(messages: { id?: number }[] | undefined): number {
  if (!messages?.length) return 0
  return messages.reduce((max, m) => Math.max(max, m.id ?? 0), 0)
}

export function invitesPractice(content: string): boolean {
  return PRACTICE_INVITE_PATTERNS.some((p) => content.includes(p))
}

export function shouldShowPracticeCTA(content: string, inExercise: boolean): boolean {
  if (isExerciseSubmitPrompt(content)) return false
  if (!invitesPractice(content)) return false
  if (!inExercise) return true
  return (
    content.includes('再来一道') ||
    content.includes('继续练习') ||
    (content.includes('点击') && content.includes('开始练习'))
  )
}

export function isAnsweringExercise(phase: string, lastAssistantContent: string): boolean {
  if (!lastAssistantContent.trim()) return false
  if (shouldShowPracticeCTA(lastAssistantContent, phase === 'exercise')) return false
  if (phase === 'exercise') return true
  return isExerciseSubmitPrompt(lastAssistantContent)
}

export function hasExerciseInHistory(messages: ChatMessage[]): boolean {
  return messages.some(
    (m) => m.role === 'assistant' && isExerciseSubmitPrompt(m.content)
  )
}

/** explain/review 最后一条助手消息后固定展示练习/掌握度出口（不依赖正文是否含「开始练习」） */
export function shouldShowInlineExitActions(
  phase: string,
  opts: { sending: boolean; completed: boolean; answering: boolean; lastRole: string }
): boolean {
  if (opts.completed || opts.sending || opts.answering) return false
  if (opts.lastRole !== 'assistant') return false
  return phase === 'explain' || phase === 'review'
}

/** 从服务端 detail 解析展示用消息与 phase/exercise（历史消息 fallback 规范化） */
export function messagesFromDetail(detail: SessionDetail): {
  messages: ChatMessage[]
  phase: string
  exercise: SessionExercise | null
} {
  const raw = detail.messages ?? []
  let phase = detail.phase ?? 'explain'
  let exercise = normalizeSessionExercise(detail.exercise)
  const out: ChatMessage[] = []
  for (const m of raw) {
    if (m.role === 'user') {
      out.push({ role: 'user', content: m.content ?? '' })
      continue
    }
    const normalized = normalizeCoachReply(m.content ?? '', phase, exercise)
    out.push({ role: 'assistant', content: normalized.content })
    phase = normalized.phase
    exercise = normalized.exercise
  }
  if (phase === 'exercise' && !exercise) {
    exercise = normalizeSessionExercise(detail.exercise)
  }
  return { messages: out, phase, exercise }
}

/** 优先 API exercise；练习阶段且无 meta 时从最后一条助手消息 fallback */
export function resolveExercise(
  phase: string,
  serverExercise: SessionExercise | null | undefined,
  messages: ChatMessage[]
): SessionExercise | null {
  const fromApi = normalizeSessionExercise(serverExercise)
  if (fromApi) return fromApi
  if (phase !== 'exercise') return null
  for (let i = messages.length - 1; i >= 0; i--) {
    if (messages[i].role !== 'assistant') continue
    const { exercise } = extractEmbeddedExercise(messages[i].content)
    if (exercise) return exercise
    break
  }
  return null
}

/**
 * 合并服务端快照：有 pending 时不缩短消息列表；无 pending 时仅接受 id 不更旧的快照。
 */
export function mergeSessionDetail(
  current: SessionDetail | null,
  incoming: SessionDetail,
  pending: PendingTurn | null
): SessionDetail {
  const incMax = maxMessageId(incoming.messages)
  const curMax = current ? maxMessageId(current.messages) : 0

  if (!pending) {
    if (!current || incMax >= curMax) return incoming
    return current
  }

  if (!current) return incoming

  if (incMax >= curMax + (pending.assistantContent ? 2 : 1)) {
    return incoming
  }

  return {
    ...current,
    phase: incoming.phase ?? current.phase,
    exercise: incoming.exercise !== undefined ? incoming.exercise : current.exercise,
    nextNodeKey: incoming.nextNodeKey ?? current.nextNodeKey,
    nextNodeTitle: incoming.nextNodeTitle ?? current.nextNodeTitle,
    nodeTitle: incoming.nodeTitle || current.nodeTitle,
    nodeKey: incoming.nodeKey || current.nodeKey,
    domainId: incoming.domainId || current.domainId,
  }
}

export function buildDisplayMessages(
  server: SessionDetail | null,
  bootstrap: BootstrapPreview | null,
  pending: PendingTurn | null
): ChatMessage[] {
  let base: ChatMessage[] = []

  if (server) {
    const loaded = messagesFromDetail(server)
    base = loaded.messages
  } else if (bootstrap) {
    base = bootstrap.messages
  }

  if (pending?.userContent) {
    base = [...base, { role: 'user', content: pending.userContent }]
  }
  if (pending?.assistantContent) {
    base = [...base, { role: 'assistant', content: pending.assistantContent }]
  }

  return base
}

export function resolvePhaseAndExercise(
  server: SessionDetail | null,
  bootstrap: BootstrapPreview | null,
  pending: PendingTurn | null
): { phase: string; exercise: SessionExercise | null } {
  if (pending?.phase !== undefined) {
    return {
      phase: pending.phase,
      exercise: pending.exercise ?? null,
    }
  }
  if (server) {
    const loaded = messagesFromDetail(server)
    const messages = buildDisplayMessages(server, bootstrap, pending)
    return {
      phase: loaded.phase,
      exercise: resolveExercise(loaded.phase, server.exercise, messages),
    }
  }
  if (bootstrap) {
    return { phase: bootstrap.phase, exercise: null }
  }
  return { phase: 'explain', exercise: null }
}

export interface DeriveCoachViewOpts {
  sessionId: string
  server: SessionDetail | null
  bootstrap: BootstrapPreview | null
  pending: PendingTurn | null
  draft: ExerciseDraft
  sending: boolean
  error: string
  toastHtml: string
  preferReadableOnce: boolean
  sessionHydrating: boolean
  initialScrollDone: boolean
  inputFocused: boolean
}

export function deriveCoachViewState(opts: DeriveCoachViewOpts): CoachViewState {
  const {
    server,
    bootstrap,
    pending,
    sending,
    error,
    toastHtml,
    preferReadableOnce,
    sessionHydrating,
    initialScrollDone,
    inputFocused,
  } = opts

  const messages = buildDisplayMessages(server, bootstrap, pending)
  const { phase, exercise } = resolvePhaseAndExercise(server, bootstrap, pending)

  const nodeTitle =
    server?.nodeTitle?.trim() ||
    bootstrap?.nodeKey ||
    '学习节点'
  const domainId = server?.domainId || bootstrap?.domainId || ''
  const pendingNextTitle = server?.nextNodeTitle ?? ''

  const completed = phase === 'completed'
  const inExercise = phase === 'exercise'
  const hasAttemptedExercise =
    phase === 'review' || phase === 'completed' || hasExerciseInHistory(messages)
  const practiceLabel = hasAttemptedExercise ? '再来一道' : '开始练习'

  const lastIdx = messages.length - 1
  const lastAssistantContent =
    messages[lastIdx]?.role === 'assistant' ? messages[lastIdx].content : ''
  const answering = isAnsweringExercise(phase, lastAssistantContent)

  let composerMode: ComposerMode = 'chat'
  if (completed) {
    composerMode = 'completed'
  } else if (answering) {
    if (exercise?.answerFormat === 'choice' && (exercise.choices?.length ?? 0) > 0) {
      composerMode = 'exercise_choice'
    } else {
      composerMode = 'exercise_text'
    }
  }

  const lastRole = messages[lastIdx]?.role ?? ''
  const showInlineExitOnLast = shouldShowInlineExitActions(phase, {
    sending,
    completed,
    answering,
    lastRole,
  })
  const showInlinePracticeOnLast = showInlineExitOnLast
  const showInlineMasteryOnLast = showInlineExitOnLast
  const showInlineCaseOnLast = showInlineExitOnLast && phase === 'explain'

  const placeholder = completed
    ? '本节点已完成'
    : answering
      ? exercise
        ? exercisePlaceholder(exercise.answerFormat)
        : exercisePlaceholder('text')
      : '有疑问？在这里提问'

  const hasDraft =
    opts.draft.text.trim().length > 0 || opts.draft.selectedChoices.length > 0

  let scrollMode: ScrollMode = 'bottom'
  if (sending || answering || inExercise || phase === 'review' || inputFocused || hasDraft) {
    scrollMode = 'bottom'
  } else if (preferReadableOnce || sessionHydrating || !initialScrollDone) {
    scrollMode = 'readable'
  } else {
    scrollMode = 'bottom'
  }

  return {
    sessionId: opts.sessionId,
    messages,
    phase,
    exercise,
    composerMode,
    practiceLabel,
    hasAttemptedExercise,
    showInlinePracticeOnLast,
    showInlineMasteryOnLast,
    showInlineCaseOnLast,
    placeholder,
    sending,
    error,
    toastHtml,
    nodeTitle,
    domainId,
    pendingNextTitle,
    scrollMode,
    completedNextTitle: pendingNextTitle,
  }
}
