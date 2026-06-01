import type { StartSessionResponse } from './api'

export interface SessionBootstrap {
  domainId: string
  nodeKey: string
  phase: string
  content?: string
  resumed?: boolean
}

const PREFIX = 'regulus:sessionBoot:'

function key(sessionId: string): string {
  return PREFIX + sessionId
}

/** startSession 成功后写入，Coach 页可立即展示首条讲解，无需再等 getSession */
export function stashSessionBootstrap(sessionId: string, res: StartSessionResponse): void {
  const boot: SessionBootstrap = {
    domainId: res.domainId,
    nodeKey: res.nodeKey,
    phase: res.phase ?? 'explain',
    content: res.content,
    resumed: res.resumed,
  }
  try {
    sessionStorage.setItem(key(sessionId), JSON.stringify(boot))
  } catch {
    /* ignore */
  }
}

export function peekSessionBootstrap(sessionId: string): SessionBootstrap | null {
  try {
    const raw = sessionStorage.getItem(key(sessionId))
    if (!raw) return null
    return JSON.parse(raw) as SessionBootstrap
  } catch {
    return null
  }
}

export function clearSessionBootstrap(sessionId: string): void {
  try {
    sessionStorage.removeItem(key(sessionId))
  } catch {
    /* ignore */
  }
}
