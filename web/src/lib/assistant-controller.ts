import { getActiveUserId } from './profile'
import {
  getPlanningSession,
  sendPlanningMessage,
  startPlanning,
  ApiError,
  QuotaExceededError,
  type PlanningMessage,
  type PlanningResult,
  type PlanningSessionDetail,
} from './api'
import { showByokModal } from '../components/byok-modal'
import { tryRecoverFromQuotaError } from './cloud'
import { renderAssistantView, getAssistantInput, type AssistantViewState } from './assistant-render'
import { isAssistantRoute } from './assistant-route'
import { navigateHash } from './navigate'
import { startNodeSession } from './start-node-session'

export class AssistantController {
  private container: HTMLElement
  private sessionId: string
  private ownerUserId: string
  private isAlive: () => boolean
  private listeners = new Set<() => void>()
  private state: AssistantViewState

  constructor(opts: {
    container: HTMLElement
    sessionId: string
    ownerUserId: string
    isAlive: () => boolean
    initial?: Partial<AssistantViewState>
  }) {
    this.container = opts.container
    this.sessionId = opts.sessionId
    this.ownerUserId = opts.ownerUserId
    this.isAlive = opts.isAlive
    this.state = {
      sessionId: opts.sessionId,
      phase: 'intake',
      messages: [],
      plan: null,
      sending: false,
      synthesizing: false,
      error: '',
      ...opts.initial,
    }
  }

  subscribe(fn: () => void): () => void {
    this.listeners.add(fn)
    return () => this.listeners.delete(fn)
  }

  private emit(): void {
    for (const fn of this.listeners) fn()
  }

  paint(): void {
    if (!this.isAlive() || !isAssistantRoute()) return
    renderAssistantView(this.container, this.state)
  }

  handleClick(target: HTMLElement): boolean {
    if (!this.isAlive() || !isAssistantRoute()) return false
    if (target.id === 'send-btn' || target.closest('#send-btn')) {
      void this.submitMessage()
      return true
    }
    const quick = target.closest<HTMLElement>('[data-quick]')
    if (quick?.dataset.quick) {
      void this.submitText(quick.dataset.quick)
      return true
    }
    if (target.id === 'assistant-new-btn' || target.closest('#assistant-new-btn')) {
      navigateHash('/assistant?new=1')
      return true
    }
    const learnBtn = target.closest<HTMLElement>('.assistant-start-learn')
    if (learnBtn?.dataset.domainId && learnBtn.dataset.nodeKey) {
      void startNodeSession({
        domainId: learnBtn.dataset.domainId,
        nodeKey: learnBtn.dataset.nodeKey,
        layer: 'entry',
        nodeTitle: learnBtn.dataset.nodeTitle ?? '学习节点',
        pageEl: this.container.querySelector<HTMLElement>('.page-assistant'),
        onError: (msg) => {
          this.state.error = msg
          this.emit()
          this.paint()
        },
      })
      return true
    }
    return false
  }

  handleKeydown(e: KeyboardEvent): void {
    if (!this.isAlive() || !isAssistantRoute()) return
    const input = e.target as HTMLElement
    if (input.id !== 'msg-input') return
    if (e.key !== 'Enter' || e.shiftKey || input.dataset.composing === '1') return
    e.preventDefault()
    void this.submitMessage()
  }

  async load(): Promise<void> {
    const detail = await getPlanningSession(this.sessionId)
    if (!this.isAlive()) return
    this.applyDetail(detail)
    this.emit()
    this.paint()
  }

  private applyDetail(detail: PlanningSessionDetail): void {
    this.state.sessionId = detail.sessionId
    this.state.phase = detail.phase
    this.state.messages = detail.messages ?? []
    this.state.plan = detail.plan ?? null
  }

  private async submitMessage(): Promise<void> {
    const input = getAssistantInput(this.container)
    const text = input?.value.trim() ?? ''
    if (!text || this.state.sending) return
    await this.submitText(text)
    if (input) input.value = ''
  }

  private async submitText(text: string): Promise<void> {
    if (this.state.sending || !this.isAlive() || !isAssistantRoute()) return
    const uid = getActiveUserId()
    if (uid && uid !== this.ownerUserId) {
      if (!this.isAlive()) return
      this.state.error = '已切换学习角色，请刷新行动助手页面'
      this.emit()
      this.paint()
      return
    }
    this.state.sending = true
    this.state.synthesizing = text.includes('整理') || text.includes('方案') || this.state.phase === 'plan_ready'
    this.state.error = ''
    this.emit()
    if (!this.isAlive()) return
    this.paint()

    const pendingUser: PlanningMessage = {
      id: -1,
      sessionId: this.sessionId,
      role: 'user',
      content: text,
    }
    this.state.messages = [...this.state.messages, pendingUser]
    this.emit()
    if (!this.isAlive()) return
    this.paint()

    try {
      const out = await sendPlanningMessage(this.sessionId, text)
      if (!this.isAlive() || !isAssistantRoute()) return
      this.state.phase = out.phase
      if (out.plan) this.state.plan = out.plan
      this.state.messages = [
        ...this.state.messages.filter((m) => m.id !== -1),
        { id: Date.now(), sessionId: this.sessionId, role: 'user', content: text },
        { id: Date.now() + 1, sessionId: this.sessionId, role: 'assistant', content: out.content },
      ]
    } catch (e) {
      if (!this.isAlive() || !isAssistantRoute()) return
      this.state.messages = this.state.messages.filter((m) => m.id !== -1)
      if (e instanceof QuotaExceededError) {
        const recovered = await tryRecoverFromQuotaError(e)
        if (recovered) {
          this.state.sending = false
          this.state.synthesizing = false
          this.emit()
          if (!this.isAlive() || !isAssistantRoute()) return
          this.paint()
          return this.submitText(text)
        }
        void showByokModal()
      }
      this.state.error = e instanceof ApiError ? e.message : '发送失败，请重试'
    } finally {
      if (this.isAlive() && isAssistantRoute()) {
        this.state.sending = false
        this.state.synthesizing = false
        this.emit()
        this.paint()
      }
    }
  }

  static async bootstrap(
    container: HTMLElement,
    sessionId: string | null,
    forceNew: boolean,
    isAlive: () => boolean,
  ): Promise<AssistantController | null> {
    const ownerUserId = getActiveUserId()
    if (!ownerUserId) {
      throw new ApiError('请先选择学习角色')
    }

    try {
      let sid = sessionId
      let initial: Partial<AssistantViewState> | undefined

      if (sid && !forceNew) {
        try {
          const detail = await getPlanningSession(sid)
          if (!isAlive()) return null
          initial = {
            phase: detail.phase,
            messages: detail.messages ?? [],
            plan: detail.plan ?? null,
          }
        } catch (e) {
          if (!(e instanceof ApiError) || !e.message.includes('无权')) {
            throw e
          }
          sid = null
        }
      }

      if (!sid || forceNew) {
        const started = await startPlanning(forceNew)
        if (!isAlive()) return null
        sid = started.sessionId
        initial = {
          phase: started.phase,
          messages: started.messages ?? [],
          plan: started.plan ?? null,
        }
        const assistantHash = `#/assistant/${sid}`
        if (sid && location.hash.startsWith('#/assistant') && location.hash !== assistantHash) {
          history.replaceState(null, '', `${location.pathname}${location.search}${assistantHash}`)
        }
      }

      if (!sid || !isAlive()) return null

      return new AssistantController({
        container,
        sessionId: sid,
        ownerUserId,
        isAlive,
        initial,
      })
    } catch (e) {
      if (!isAlive()) return null
      container.innerHTML = ''
      const { assistantErrorHtml } = await import('./assistant-render')
      container.innerHTML = assistantErrorHtml(e instanceof ApiError ? e.message : '加载失败')
      container.querySelector('#assistant-retry-btn')?.addEventListener('click', () => {
        navigateHash('/assistant')
      })
      return null
    }
  }
}

export type { PlanningResult }
