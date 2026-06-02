import { getActiveSession, startSession, ApiError } from './api'
import { clearAppBusyIf, setAppBusy } from './app-busy'
import { navigateToCoach } from './navigate'
import { stashSessionBootstrap } from './session-bootstrap'
import { clearTreeSessionOverlay } from './session-loading-overlay'

export interface NodeSessionOverlayOpts {
  nodeTitle: string
  message: string
  hint?: string
}

function getScrollHost(): HTMLElement | null {
  return document.getElementById('main-content')
}

export function setNodeSessionOverlay(
  pageEl: HTMLElement | null,
  active: boolean,
  opts?: NodeSessionOverlayOpts
): void {
  const scrollHost = getScrollHost()
  if (!scrollHost) return

  if (!active) {
    pageEl?.classList.remove('is-session-loading')
    clearTreeSessionOverlay()
    return
  }

  pageEl?.classList.add('is-session-loading')
  scrollHost.classList.add('has-tree-session-loading')
  let overlay = scrollHost.querySelector<HTMLDivElement>('#tree-session-overlay')
  if (!overlay) {
    overlay = document.createElement('div')
    overlay.id = 'tree-session-overlay'
    overlay.className = 'tree-session-overlay'
    overlay.setAttribute('role', 'alertdialog')
    overlay.setAttribute('aria-modal', 'true')
    overlay.setAttribute('aria-busy', 'true')
    overlay.setAttribute('aria-live', 'polite')
    scrollHost.appendChild(overlay)
  }
  overlay.innerHTML = `
    <div class="tree-session-overlay-card card">
      <div class="spinner tree-session-spinner" aria-hidden="true"></div>
      <p class="tree-session-node">${escapeHtml(opts!.nodeTitle)}</p>
      <p class="tree-session-message">${escapeHtml(opts!.message)}</p>
      ${opts!.hint ? `<p class="tree-session-hint">${escapeHtml(opts!.hint)}</p>` : ''}
    </div>
  `
}

let handoffInFlight: string | null = null

export async function startNodeSession(opts: {
  domainId: string
  nodeKey: string
  layer: string
  nodeTitle: string
  pageEl?: HTMLElement | null
  onError: (message: string) => void
}): Promise<void> {
  const { domainId, nodeKey, layer, nodeTitle, pageEl, onError } = opts
  const handoffKey = `${domainId}:${nodeKey}`
  if (handoffInFlight === handoffKey) return
  handoffInFlight = handoffKey

  setNodeSessionOverlay(pageEl ?? null, true, {
    nodeTitle,
    message: '正在检查学习记录…',
    hint: '若该节点曾学过，将直接进入对话',
  })
  setAppBusy(true, 'session')
  try {
    const active = await getActiveSession(domainId, nodeKey)
    if (active.sessionId) {
      setNodeSessionOverlay(pageEl ?? null, true, {
        nodeTitle,
        message: '正在打开教练对话…',
      })
      navigateToCoach(active.sessionId)
      return
    }
    setNodeSessionOverlay(pageEl ?? null, true, {
      nodeTitle,
      message: 'AI 正在准备首条讲解…',
      hint: '首次约需 30–60 秒，请勿关闭或刷新页面',
    })
    const res = await startSession(domainId, nodeKey, layer)
    setNodeSessionOverlay(pageEl ?? null, true, {
      nodeTitle,
      message: '讲解已就绪，正在进入对话…',
    })
    stashSessionBootstrap(res.sessionId, res)
    navigateToCoach(res.sessionId)
  } catch (e) {
    onError(e instanceof ApiError ? e.message : '启动会话失败')
  } finally {
    setNodeSessionOverlay(pageEl ?? null, false)
    clearAppBusyIf('session')
    if (handoffInFlight === handoffKey) handoffInFlight = null
  }
}

function escapeHtml(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}
