import { clearAppBusyIf, setAppBusy } from '../lib/app-busy'
import { CoachController } from '../lib/coach-controller'
import { coachErrorHtml, coachLoadingHtml } from '../lib/coach-render'
import { clearTreeSessionOverlay } from '../lib/session-loading-overlay'
import { peekSessionBootstrap } from '../lib/session-bootstrap'
import { setBreadcrumb, updateSidebar, refreshLLMStatusAfterBusy } from '../components/layout'

/** 每次 hash 进入教练页递增；用于丢弃已切换会话或过期的 load / emit */
let coachRenderGen = 0

const coachControllers = new WeakMap<HTMLElement, CoachController>()

type CoachContainer = HTMLElement & { __coachEventsBound?: boolean }

function bindCoachContainerEvents(container: CoachContainer): void {
  if (container.__coachEventsBound) return
  container.__coachEventsBound = true

  container.addEventListener('click', (e) => {
    const ctrl = coachControllers.get(container)
    if (!ctrl) return
    if (ctrl.handleClick(e.target as HTMLElement)) {
      e.preventDefault()
    }
  })

  container.addEventListener('keydown', (e) => {
    const ctrl = coachControllers.get(container)
    if (!ctrl) return
    ctrl.handleKeydown(e)
  })

  container.addEventListener('input', (e) => {
    const input = e.target as HTMLElement
    if (input instanceof HTMLTextAreaElement && input.id === 'msg-input') {
      input.style.height = 'auto'
      const max = Math.min(window.innerHeight * 0.38, 320)
      input.style.height = `${Math.min(input.scrollHeight, max)}px`
    }
  })

  container.addEventListener('compositionstart', (e) => {
    const input = e.target as HTMLElement
    if (input.id === 'msg-input') input.dataset.composing = '1'
  })

  container.addEventListener('compositionend', (e) => {
    const input = e.target as HTMLElement
    if (input.id === 'msg-input') delete input.dataset.composing
  })
}

export async function renderCoach(container: HTMLElement, sessionId: string): Promise<void> {
  clearTreeSessionOverlay()

  const gen = ++coachRenderGen
  const stale = () => gen !== coachRenderGen

  const bootstrap = peekSessionBootstrap(sessionId)
  container.innerHTML = coachLoadingHtml(
    bootstrap?.content
      ? '正在同步对话记录…'
      : '首次讲解由 AI 生成，可能需要 30–60 秒，请稍候'
  )

  bindCoachContainerEvents(container as CoachContainer)

  const refreshChrome = (ctrl: CoachController) => {
    const ctx = ctrl.getSidebarContext()
    void updateSidebar(ctx)
    setBreadcrumb([
      { label: '开始学习', href: '#/' },
      { label: '我的课程', href: '#/courses' },
      {
        label: ctx.domainName?.trim() || '我的课程',
        href: ctx.domainId ? `#/tree/${ctx.domainId}` : undefined,
      },
      { label: ctx.nodeTitle || '教练对话' },
    ])
  }

  const controller = new CoachController({
    container,
    sessionId,
    isAlive: () => !stale(),
    onChromeUpdate: () => {
      if (!stale()) refreshChrome(controller)
    },
  })

  coachControllers.set(container, controller)

  const paint = () => {
    if (stale()) return
    controller.paint()
  }

  controller.subscribe(() => paint())

  setAppBusy(true, 'session')

  try {
    const result = await controller.load(sessionId)
    if (stale()) return

    if (result.fatalError) {
      const domainId = bootstrap?.domainId ?? ''
      container.innerHTML = coachErrorHtml(result.fatalError, domainId)
      container.querySelector<HTMLButtonElement>('#coach-retry-btn')?.addEventListener('click', () => {
        void renderCoach(container, sessionId)
      })
      return
    }

    refreshChrome(controller)
    paint()
  } finally {
    if (!stale()) {
      if (clearAppBusyIf('session')) refreshLLMStatusAfterBusy()
    }
  }
}
