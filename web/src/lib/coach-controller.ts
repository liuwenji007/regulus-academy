import {
  getSession,
  getDomainTree,
  getUserProgress,
  sendMessage,
  startNextSession,
  ApiError,
  type SessionDetail,
  type KnowledgeTree,
} from './api'
import { clearAppBusyIf, setAppBusy } from './app-busy'
import { fadeClearTreeSessionOverlay } from './session-loading-overlay'
import { navigateToCoach } from './navigate'
import { setNodeSessionOverlay } from './start-node-session'
import {
  collectExerciseAnswer,
  formatChoiceSubmission,
  normalizeCoachReply,
  normalizeSessionExercise,
  readExerciseDraft,
  tryFormatJsonInTextarea,
  type ExerciseDraft,
} from './coach-exercise'
import { nodeLayerKeyMap } from './tree-normalize'
import { scrollChatMessages } from './chat-scroll'
import {
  buildDisplayMessages,
  deriveCoachViewState,
  mergeSessionDetail,
  REAL_WORLD_CASE_PROMPT,
  type BootstrapPreview,
  type CoachViewState,
  type PendingTurn,
} from './coach-view-state'
import { isExerciseSubmitPrompt } from './coach-exercise'
import {
  renderCoachView,
  getMsgInput,
  type CoachRenderChrome,
} from './coach-render'
import {
  clearSessionBootstrap,
  peekSessionBootstrap,
  stashSessionBootstrap,
} from './session-bootstrap'

function findNextUncompletedNode(
  tree: KnowledgeTree | null,
  nodeKey: string,
  completedKeys: ReadonlySet<string>
): { key: string; title: string; layer: string } | null {
  if (!tree?.layers?.length || !nodeKey) return null
  let found = false
  for (const layer of tree.layers) {
    for (const node of layer.nodes ?? []) {
      if (found) {
        if (completedKeys.has(node.key)) continue
        return { key: node.key, title: node.title, layer: layer.key || 'entry' }
      }
      if (node.key === nodeKey) found = true
    }
  }
  return null
}

async function loadSessionResilient(
  sessionId: string,
  isStale: () => boolean
): Promise<SessionDetail> {
  const waits = [0, 300, 700, 1200]
  let lastErr: unknown
  for (const ms of waits) {
    if (isStale()) throw new DOMException('stale', 'AbortError')
    if (ms > 0) await new Promise((r) => setTimeout(r, ms))
    try {
      return await getSession(sessionId)
    } catch (e) {
      lastErr = e
      if (e instanceof ApiError && e.message.includes('无权')) throw e
    }
  }
  throw lastErr
}

function formatLoadError(e: unknown): string {
  if (e instanceof ApiError) return e.message
  if (e instanceof DOMException && e.name === 'AbortError') return ''
  if (e instanceof Error && e.message) return e.message
  return '加载失败，请稍后重试'
}

export type CoachChangeListener = () => void

export interface CoachControllerOpts {
  container: HTMLElement
  sessionId: string
  isAlive: () => boolean
  onChromeUpdate?: () => void
}

let nextNodeHandoffSessionId: string | null = null

export class CoachController {
  private readonly container: HTMLElement
  private readonly isAlive: () => boolean
  private readonly onChromeUpdate?: () => void

  private sessionId: string
  private server: SessionDetail | null = null
  private bootstrap: BootstrapPreview | null = null
  private pending: PendingTurn | null = null
  private sending = false
  private error = ''
  private toastHtml = ''
  private draft: ExerciseDraft = { text: '', selectedChoices: [] }

  private courseTree: KnowledgeTree | null = null
  private domainName = ''
  private domainNodeTotal = 0
  private domainCompleted = 0
  private currentNodeKey = ''
  private pendingNextTitle = ''
  private pendingNextNodeKey = ''
  private completedNodeKeys = new Set<string>()

  private preferReadableOnce = false
  private loadGeneration = 0
  private reconcileGeneration = 0

  private listeners: CoachChangeListener[] = []

  constructor(opts: CoachControllerOpts) {
    this.container = opts.container
    this.sessionId = opts.sessionId
    this.isAlive = opts.isAlive
    this.onChromeUpdate = opts.onChromeUpdate
  }

  getSessionId(): string {
    return this.sessionId
  }

  getSidebarContext(): {
    active: 'coach'
    domainId?: string
    domainName?: string
    nodeTitle?: string
    domainNodeTotal?: number
    domainCompleted?: number
  } {
    const ctx: {
      active: 'coach'
      domainId?: string
      domainName?: string
      nodeTitle?: string
      domainNodeTotal?: number
      domainCompleted?: number
    } = {
      active: 'coach',
      domainId: this.server?.domainId || this.bootstrap?.domainId,
      nodeTitle: this.server?.nodeTitle || this.bootstrap?.nodeKey,
    }
    if (this.domainName.trim()) ctx.domainName = this.domainName
    if (this.domainNodeTotal > 0) {
      ctx.domainNodeTotal = this.domainNodeTotal
      ctx.domainCompleted = this.domainCompleted
    }
    return ctx
  }

  subscribe(fn: CoachChangeListener): () => void {
    this.listeners.push(fn)
    return () => {
      this.listeners = this.listeners.filter((l) => l !== fn)
    }
  }

  private emit(): void {
    if (!this.isAlive()) return
    for (const fn of this.listeners) {
      fn()
    }
  }

  /** 末条为助手消息时，下一次渲染锚定到该条开头 */
  private markAnchorLastAssistant(): void {
    const messages = buildDisplayMessages(this.server, this.bootstrap, this.pending)
    if (messages.at(-1)?.role === 'assistant') {
      this.preferReadableOnce = true
    }
  }

  private getViewState(): CoachViewState {
    return deriveCoachViewState({
      sessionId: this.sessionId,
      server: this.server,
      bootstrap: this.bootstrap,
      pending: this.pending,
      draft: this.draft,
      sending: this.sending,
      error: this.error,
      toastHtml: this.toastHtml,
      preferReadableOnce: this.preferReadableOnce,
    })
  }

  private resolveNextNode(): { key: string; title: string; layer: string } | null {
    if (this.pendingNextNodeKey && !this.completedNodeKeys.has(this.pendingNextNodeKey)) {
      const layer = this.courseTree
        ? (nodeLayerKeyMap(this.courseTree).get(this.pendingNextNodeKey) ?? 'entry')
        : 'entry'
      const title =
        this.pendingNextTitle ||
        this.courseTree?.layers
          .flatMap((l) => l.nodes)
          .find((n) => n.key === this.pendingNextNodeKey)?.title ||
        this.pendingNextNodeKey
      return { key: this.pendingNextNodeKey, title, layer }
    }
    return findNextUncompletedNode(
      this.courseTree,
      this.currentNodeKey,
      this.completedNodeKeys
    )
  }

  private getRenderChrome(): CoachRenderChrome {
    return {
      completedNextTitle: this.resolveNextNode()?.title ?? '',
    }
  }

  private applyServer(detail: SessionDetail, opts?: { resetScroll?: boolean }): void {
    this.server = mergeSessionDetail(this.server, detail, this.pending)
    this.currentNodeKey = this.server.nodeKey || this.currentNodeKey
    this.pendingNextTitle = this.server.nextNodeTitle ?? this.pendingNextTitle
    this.pendingNextNodeKey = this.server.nextNodeKey ?? this.pendingNextNodeKey
    if (opts?.resetScroll) this.markAnchorLastAssistant()
    this.onChromeUpdate?.()
  }

  async load(initialSessionId: string): Promise<{ fatalError?: string; degraded?: boolean }> {
    this.sessionId = initialSessionId
    const boot = peekSessionBootstrap(initialSessionId)

    if (boot?.content && !boot.resumed) {
      this.bootstrap = {
        messages: [{ role: 'assistant', content: boot.content.trim() }],
        phase: boot.phase ?? 'explain',
        domainId: boot.domainId,
        nodeKey: boot.nodeKey,
      }
      this.currentNodeKey = boot.nodeKey
    }

    const loadGen = ++this.loadGeneration
    const staleLoad = () => !this.isAlive() || loadGen !== this.loadGeneration

    const domainId = this.server?.domainId || this.bootstrap?.domainId || ''
    const domainMetaPromise = domainId
      ? Promise.all([
          getDomainTree(domainId).catch(() => null),
          getUserProgress(domainId).catch(() => []),
        ])
      : Promise.resolve([null, []] as const)

    if (this.bootstrap && !staleLoad()) {
      const [tree] = await domainMetaPromise
      if (tree) {
        this.courseTree = tree
        this.domainName = tree.domainName ?? ''
        this.domainNodeTotal =
          tree.layers?.reduce((n, l) => n + (l.nodes?.length ?? 0), 0) ?? 0
        this.currentNodeKey = boot?.nodeKey ?? this.currentNodeKey
      }
      this.markAnchorLastAssistant()
      this.emit()
    }

    try {
      const [[tree, progress], detail] = await Promise.all([
        domainMetaPromise,
        loadSessionResilient(initialSessionId, staleLoad),
      ])
      if (staleLoad()) return {}

      clearSessionBootstrap(initialSessionId)
      this.domainName = tree?.domainName ?? this.domainName
      this.courseTree = tree
      this.domainNodeTotal =
        tree?.layers?.reduce((n, l) => n + (l.nodes?.length ?? 0), 0) ?? 0
      this.domainCompleted = progress.filter((p) => p.status === 'completed').length
      this.completedNodeKeys = new Set(
        progress.filter((p) => p.status === 'completed').map((p) => p.nodeKey)
      )

      if (!this.pending && !this.sending) {
        this.bootstrap = null
        this.applyServer(detail, { resetScroll: true })
      } else if (!this.server) {
        this.applyServer(detail, { resetScroll: false })
      } else {
        this.applyServer(detail)
      }

      this.markAnchorLastAssistant()
      this.emit()
      return {}
    } catch (e) {
      if (staleLoad()) return {}
      const msg = formatLoadError(e)
      if (!msg) return {}

      if (this.bootstrap) {
        clearSessionBootstrap(initialSessionId)
        this.markAnchorLastAssistant()
        this.error = `${msg}（已显示本地缓存的讲解，发送消息将自动重试同步）`
        this.emit()
        return { degraded: true }
      }
      return { fatalError: msg }
    }
  }

  private async reconcile(): Promise<boolean> {
    const gen = ++this.reconcileGeneration
    const targetId = this.sessionId
    try {
      const detail = await getSession(targetId)
      if (gen !== this.reconcileGeneration || this.sessionId !== targetId) return false
      this.pending = null
      this.applyServer(detail, { resetScroll: true })
      this.emit()
      return true
    } catch {
      return false
    }
  }

  async sendText(text: string): Promise<void> {
    const trimmed = text.trim()
    if (!trimmed || this.sending) return

    this.draft = { text: '', selectedChoices: [] }
    this.pending = { userContent: trimmed }
    this.sending = true
    this.error = ''
    this.emit()

    try {
      const reply = await sendMessage(this.sessionId, trimmed)
      const normalized = normalizeCoachReply(
        reply.content,
        reply.phase,
        normalizeSessionExercise(reply.exercise)
      )
      let phase: string
      if (reply.phase === 'completed') {
        phase = 'completed'
      } else if (
        isExerciseSubmitPrompt(normalized.content) ||
        normalized.phase === 'exercise' ||
        reply.phase === 'exercise'
      ) {
        phase = 'exercise'
      } else if (reply.phase === 'review') {
        phase = 'review'
      } else {
        phase = normalized.phase
      }

      const pendingExercise =
        phase === 'exercise'
          ? normalizeSessionExercise(reply.exercise) ?? normalized.exercise
          : null

      this.pending = {
        userContent: trimmed,
        assistantContent: normalized.content,
        phase,
        exercise: pendingExercise,
      }

      if (reply.nextSessionId?.trim()) {
        const nextId = reply.nextSessionId.trim()
        this.loadGeneration++
        this.pending = null
        stashSessionBootstrap(nextId, {
          sessionId: nextId,
          domainId: this.server?.domainId ?? this.bootstrap?.domainId ?? '',
          nodeKey: reply.nextNodeKey ?? '',
          phase: reply.phase === 'review' || reply.phase === 'completed' ? 'explain' : (reply.phase || 'explain'),
          content: normalized.content,
        })
        navigateToCoach(nextId)
        return
      }

      if (!this.isAlive()) return
      await this.reconcile()

      if (reply.nodeCompleted) {
        // 完成态由底部 coach-completed-dock 承接，避免与聊天区重复的 alert 条
        this.toastHtml = ''
        this.preferReadableOnce = false
        if (this.currentNodeKey) this.completedNodeKeys.add(this.currentNodeKey)
        if (!reply.nextSessionId) {
          if (reply.nextNodeTitle) this.pendingNextTitle = reply.nextNodeTitle
          if (reply.nextNodeKey) this.pendingNextNodeKey = reply.nextNodeKey
        }
      } else if (normalized.content.trim()) {
        this.preferReadableOnce = true
      }
    } catch (err) {
      if (!this.isAlive()) return
      this.pending = null
      this.draft = { text: trimmed, selectedChoices: [] }
      this.error = err instanceof ApiError ? err.message : '发送失败，请重试'
    } finally {
      this.sending = false
      if (!this.isAlive()) return
      this.emit()
      getMsgInput(this.container)?.focus({ preventScroll: true })
    }
  }

  submitAnswer(): void {
    const view = this.getViewState()

    if (view.composerMode === 'exercise_choice' && view.exercise) {
      const result = collectExerciseAnswer(this.container, view.exercise)
      if (!result.ok) {
        this.error = result.message
        this.emit()
        return
      }
      void this.sendText(result.text)
      return
    }

    if (view.composerMode === 'exercise_text' && view.exercise) {
      const result = collectExerciseAnswer(this.container, view.exercise)
      if (!result.ok) {
        this.error = result.message
        this.emit()
        return
      }
      void this.sendText(result.text)
      return
    }

    if (view.composerMode === 'exercise_choice') {
      const checked = this.container.querySelectorAll<HTMLInputElement>(
        '.coach-choice-input:checked'
      )
      if (checked.length > 0) {
        const choices = Array.from(
          this.container.querySelectorAll<HTMLInputElement>('.coach-choice-input')
        ).map((el) => el.value)
        const selected = Array.from(checked).map((el) => el.value)
        const multiple = this.container.querySelector<HTMLInputElement>(
          '.coach-choice-input[type="checkbox"]'
        )
        const mode: 'single' | 'multiple' = multiple ? 'multiple' : 'single'
        if (mode !== 'multiple' && selected.length > 1) {
          this.error = '本题为单选题，只能选一个'
          this.emit()
          return
        }
        void this.sendText(formatChoiceSubmission(selected, choices, mode))
        return
      }
    }

    const input = getMsgInput(this.container)
    if (input) void this.sendText(input.value)
  }

  async goNextNode(): Promise<void> {
    const view = this.getViewState()
    if (this.sending || view.phase !== 'completed') return
    const next = this.resolveNextNode()
    if (!next) {
      this.error = '没有下一节点，或课程信息尚未加载完成'
      this.emit()
      return
    }
    if (nextNodeHandoffSessionId === this.sessionId) return
    nextNodeHandoffSessionId = this.sessionId

    const nextTitle = next.title
    this.sending = true
    this.error = ''
    this.emit()

    const coachPageEl =
      this.container.querySelector<HTMLElement>('.page-coach') ?? this.container
    setNodeSessionOverlay(coachPageEl, true, {
      nodeTitle: nextTitle,
      message: 'AI 正在准备下一节讲解…',
      hint: '首次约需 30–60 秒，请勿关闭或刷新页面',
    })
    setAppBusy(true, 'session')
    let overlayShown = true

    let handedOff = false
    try {
      const res = await startNextSession(this.sessionId)
      if (!res.sessionId?.trim()) {
        throw new ApiError('服务器未返回新会话，请重试')
      }
      if (res.resumed) {
        setNodeSessionOverlay(coachPageEl, false)
        void fadeClearTreeSessionOverlay()
        clearAppBusyIf('session')
        overlayShown = false
      }
      this.loadGeneration++
      stashSessionBootstrap(res.sessionId, res)
      navigateToCoach(res.sessionId)
      handedOff = true
    } catch (err) {
      if (!this.isAlive()) return
      this.error = err instanceof ApiError ? err.message : '进入下一节失败，请重试'
      this.emit()
    } finally {
      if (!handedOff && overlayShown) {
        setNodeSessionOverlay(coachPageEl, false)
        void fadeClearTreeSessionOverlay()
        clearAppBusyIf('session')
      }
      this.sending = false
      if (nextNodeHandoffSessionId === this.sessionId) {
        nextNodeHandoffSessionId = null
      }
      if (this.isAlive()) this.emit()
    }
  }

  captureDraftFromDom(): void {
    const view = this.getViewState()
    this.draft = readExerciseDraft(this.container, view.exercise)
  }

  consumePreferReadable(): boolean {
    if (!this.preferReadableOnce) return false
    this.preferReadableOnce = false
    return true
  }

  /** 进入会话或全屏 loading 结束后，将末条助手消息顶到可视区开头 */
  anchorChatToLastAssistant(): void {
    const messages = buildDisplayMessages(this.server, this.bootstrap, this.pending)
    if (messages.at(-1)?.role !== 'assistant') return
    const msgBox = this.container.querySelector<HTMLDivElement>('#messages')
    if (msgBox) scrollChatMessages(msgBox, 'readable')
  }

  formatJson(): boolean {
    if (!tryFormatJsonInTextarea(this.container)) {
      this.error = '当前内容不是合法 JSON，请检查括号与引号'
      this.emit()
      return false
    }
    this.error = ''
    return true
  }

  handleClick(target: HTMLElement): boolean {
    const nextNodeBtn = target.closest<HTMLButtonElement>('#next-node-btn')
    if (nextNodeBtn && !nextNodeBtn.disabled) {
      void this.goNextNode()
      return true
    }

    const practiceBtn = target.closest<HTMLButtonElement>('.coach-inline-practice')
    if (practiceBtn && !practiceBtn.disabled) {
      void this.sendText(practiceBtn.dataset.practice ?? '开始练习')
      return true
    }

    const caseBtn = target.closest<HTMLButtonElement>('.coach-inline-case')
    if (caseBtn && !caseBtn.disabled) {
      void this.sendText(caseBtn.dataset.case ?? REAL_WORLD_CASE_PROMPT)
      return true
    }

    const quickBtn = target.closest<HTMLButtonElement>('.coach-quick-btn')
    if (quickBtn && !quickBtn.disabled) {
      void this.sendText(quickBtn.dataset.quick ?? '')
      return true
    }

    const sendBtn = target.closest<HTMLButtonElement>('#send-btn')
    if (sendBtn && !sendBtn.disabled) {
      this.captureDraftFromDom()
      this.submitAnswer()
      return true
    }

    const formatBtn = target.closest<HTMLButtonElement>('#json-format-btn')
    if (formatBtn && !formatBtn.disabled) {
      this.formatJson()
      return true
    }

    return false
  }

  handleKeydown(e: KeyboardEvent): boolean {
    const input = e.target as HTMLElement
    if (input.id !== 'msg-input') return false
    if (!(input instanceof HTMLInputElement || input instanceof HTMLTextAreaElement)) return false
    if (e.isComposing || input.dataset.composing === '1') return false

    if (e.key === 'Enter') {
      if (input instanceof HTMLTextAreaElement) {
        if (!(e.ctrlKey || e.metaKey)) return false
      }
      e.preventDefault()
      this.captureDraftFromDom()
      this.submitAnswer()
      return true
    }
    return false
  }

  paint(): void {
    this.captureDraftFromDom()
    const view = this.getViewState()
    const chrome = this.getRenderChrome()
    renderCoachView(this.container, view, chrome, this.draft, {
      consumePreferReadable: () => this.consumePreferReadable(),
    })
  }
}
