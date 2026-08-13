import { getActiveUserId } from './profile'
import {
  getPlanningSession,
  sendPlanningMessage,
  startPlanning,
  patchPlanningFocus,
  ApiError,
  QuotaExceededError,
  type PlanningMessage,
  type PlanningResult,
  type PlanningSessionDetail,
} from './api'
import { showByokModal } from '../components/byok-modal'
import { tryRecoverFromQuotaError } from './cloud'
import {
  renderAssistantView,
  getAssistantInput,
  readPlanExpandedPreference,
  writePlanExpandedPreference,
  type AssistantViewState,
} from './assistant-render'
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
      planExpanded: readPlanExpandedPreference(),
      sending: false,
      synthesizing: false,
      error: '',
      ...opts.initial,
    }
    if (opts.initial?.planExpanded === undefined) {
      this.state.planExpanded = readPlanExpandedPreference()
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
    if (target.id === 'assistant-pin-btn' || target.closest('#assistant-pin-btn')) {
      void this.togglePinNorthStar()
      return true
    }
    if (target.id === 'assistant-expand-btn' || target.closest('#assistant-expand-btn')) {
      this.state.planExpanded = !this.state.planExpanded
      writePlanExpandedPreference(this.state.planExpanded)
      this.emit()
      this.paint()
      return true
    }
    const checkRow = target.closest<HTMLElement>('label.assistant-check-row')
    const check =
      target.closest<HTMLInputElement>('input.assistant-check') ||
      checkRow?.querySelector<HTMLInputElement>('input.assistant-check') ||
      null
    if (check?.dataset.checkKey) {
      // 点 label 时 input.checked 可能尚未翻转，下一帧再读
      const key = check.dataset.checkKey
      queueMicrotask(() => {
        void this.toggleChecked(key, check.checked)
      })
      return false
    }
    const quick = target.closest<HTMLElement>('.assistant-quick-btn')
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

  private async togglePinNorthStar(): Promise<void> {
    if (!this.state.plan || this.state.sending) return
    const next = !this.state.plan.ui_state?.north_star_pinned
    const prev = this.state.plan
    this.state.plan = {
      ...prev,
      ui_state: {
        north_star_pinned: next,
        checked: { ...(prev.ui_state?.checked ?? {}) },
      },
    }
    this.emit()
    this.paint()
    try {
      const out = await patchPlanningFocus(this.sessionId, { north_star_pinned: next })
      if (!this.isAlive()) return
      this.state.plan = out.plan
      this.state.error = ''
    } catch (e) {
      if (!this.isAlive()) return
      this.state.plan = prev
      this.state.error = e instanceof ApiError ? e.message : '钉住失败，请重试'
    }
    this.emit()
    this.paint()
  }

  private async toggleChecked(key: string, checked: boolean): Promise<void> {
    if (!this.state.plan) return
    const prev = this.state.plan
    const nextChecked = { ...(prev.ui_state?.checked ?? {}) }
    if (checked) nextChecked[key] = true
    else delete nextChecked[key]
    this.state.plan = {
      ...prev,
      ui_state: {
        north_star_pinned: Boolean(prev.ui_state?.north_star_pinned),
        checked: nextChecked,
      },
    }
    this.emit()
    this.paint()
    try {
      const out = await patchPlanningFocus(this.sessionId, { checked: { [key]: checked } })
      if (!this.isAlive()) return
      this.state.plan = out.plan
      this.state.error = ''
    } catch (e) {
      if (!this.isAlive()) return
      this.state.plan = prev
      this.state.error = e instanceof ApiError ? e.message : '勾选同步失败'
    }
    this.emit()
    this.paint()
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
      // 兜底：phase 已是 plan_ready 但响应漏了 plan（历史遮蔽 bug / 代理截断）时再拉一次会话
      if (out.phase === 'plan_ready' && !this.state.plan) {
        try {
          const detail = await getPlanningSession(this.sessionId)
          if (this.isAlive() && isAssistantRoute() && detail.plan) {
            this.state.plan = detail.plan
          }
        } catch {
          /* ignore */
        }
      }
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
