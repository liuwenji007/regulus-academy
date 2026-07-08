import { AssistantController } from '../lib/assistant-controller'
import { assistantLoadingHtml } from '../lib/assistant-render'
import { isAssistantRoute } from '../lib/assistant-route'
import { setBreadcrumb, updateSidebar } from '../components/layout'

let assistantRenderGen = 0

const assistantControllers = new WeakMap<HTMLElement, AssistantController>()

type AssistantContainer = HTMLElement & { __assistantEventsBound?: boolean }

function assistantEventsActive(container: HTMLElement): boolean {
  return Boolean(assistantControllers.get(container) && container.querySelector('.page-assistant'))
}

function bindAssistantEvents(container: AssistantContainer): void {
  if (container.__assistantEventsBound) return
  container.__assistantEventsBound = true

  container.addEventListener('click', (e) => {
    const ctrl = assistantControllers.get(container)
    if (!ctrl || !isAssistantRoute()) return
    if (ctrl.handleClick(e.target as HTMLElement)) {
      e.preventDefault()
    }
  })

  container.addEventListener('keydown', (e) => {
    const ctrl = assistantControllers.get(container)
    if (!ctrl || !isAssistantRoute()) return
    ctrl.handleKeydown(e)
  })

  container.addEventListener('input', (e) => {
    if (!assistantEventsActive(container)) return
    const input = e.target as HTMLElement
    if (input instanceof HTMLTextAreaElement && input.id === 'msg-input') {
      input.style.height = 'auto'
      const max = Math.min(window.innerHeight * 0.38, 320)
      input.style.height = `${Math.min(input.scrollHeight, max)}px`
    }
  })

  container.addEventListener('compositionstart', (e) => {
    if (!assistantEventsActive(container)) return
    const input = e.target as HTMLElement
    if (input.id === 'msg-input') input.dataset.composing = '1'
  })

  container.addEventListener('compositionend', (e) => {
    if (!assistantEventsActive(container)) return
    const input = e.target as HTMLElement
    if (input.id === 'msg-input') delete input.dataset.composing
  })
}

/** 切换学习角色时丢弃进行中的行动助手渲染 */
export function resetAssistantOnProfileChange(): void {
  cancelAssistantRender()
}

/** 离开行动助手路由时取消进行中的渲染并解绑控制器 */
export function cancelAssistantRender(container?: HTMLElement): void {
  assistantRenderGen++
  if (container) assistantControllers.delete(container)
}

/** @deprecated 请使用 cancelAssistantRender(container) */
export function detachAssistantController(container: HTMLElement): void {
  cancelAssistantRender(container)
}

export async function renderAssistant(container: HTMLElement, sessionId?: string): Promise<void> {
  const gen = ++assistantRenderGen
  const stale = () => gen !== assistantRenderGen

  const hashQuery = location.hash.includes('?') ? location.hash.split('?')[1] : ''
  const params = new URLSearchParams(hashQuery)
  const forceNew = params.get('new') === '1'

  container.innerHTML = assistantLoadingHtml('准备好后，随便说说你现在的情况')
  assistantControllers.delete(container)

  void updateSidebar({ active: 'assistant' })
  setBreadcrumb([
    { label: '开始学习', href: '#/' },
    { label: '行动助手' },
  ])

  bindAssistantEvents(container as AssistantContainer)

  const ctrl = await AssistantController.bootstrap(
    container,
    sessionId ?? null,
    forceNew,
    () => !stale() && isAssistantRoute(),
  )
  if (stale()) return
  if (!ctrl) return

  assistantControllers.set(container, ctrl)
  ctrl.subscribe(() => {
    if (!stale()) ctrl.paint()
  })
  ctrl.paint()
}

export async function renderAssistantEntry(container: HTMLElement): Promise<void> {
  await renderAssistant(container)
}
