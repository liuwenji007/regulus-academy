import { getActiveSession, startSession, ApiError } from './api'
import { clearAppBusyIf, setAppBusy } from './app-busy'
import { navigateToCoach } from './navigate'
import { stashSessionBootstrap } from './session-bootstrap'
import {
  fadeClearTreeSessionOverlay,
  setTreeSessionOverlay,
  type TreeSessionOverlayOpts,
} from './session-loading-overlay'

export type NodeSessionOverlayOpts = TreeSessionOverlayOpts

export function setNodeSessionOverlay(
  pageEl: HTMLElement | null,
  active: boolean,
  opts?: NodeSessionOverlayOpts
): void {
  if (!active) {
    pageEl?.classList.remove('is-session-loading')
    void fadeClearTreeSessionOverlay()
    return
  }
  pageEl?.classList.add('is-session-loading')
  setTreeSessionOverlay(true, opts)
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
  let handedOff = false
  try {
    const active = await getActiveSession(domainId, nodeKey)
    if (active.sessionId) {
      setNodeSessionOverlay(pageEl ?? null, true, {
        nodeTitle,
        message: '正在打开教练对话…',
      })
      navigateToCoach(active.sessionId)
      handedOff = true
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
    handedOff = true
  } catch (e) {
    onError(e instanceof ApiError ? e.message : '启动会话失败')
  } finally {
    if (!handedOff) {
      setNodeSessionOverlay(pageEl ?? null, false)
      clearAppBusyIf('session')
    }
    if (handoffInFlight === handoffKey) handoffInFlight = null
  }
}
