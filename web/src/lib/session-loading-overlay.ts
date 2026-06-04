import { fadeOutAndRemove } from './loading-transition'

const OVERLAY_ID = 'tree-session-overlay'

/** 全屏 handoff loading（挂 body，避免 main-content 内 fixed 错位） */

export interface TreeSessionOverlayOpts {
  nodeTitle: string
  message: string
  hint?: string
}

function escapeHtml(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}

export function getMainContentEl(): HTMLElement | null {
  return document.getElementById('main-content')
}

function getOverlayEl(): HTMLDivElement | null {
  return document.getElementById(OVERLAY_ID) as HTMLDivElement | null
}

export function setTreeSessionOverlay(active: boolean, opts?: TreeSessionOverlayOpts): void {
  const host = getMainContentEl()

  if (!active) {
    void fadeClearTreeSessionOverlay()
    return
  }
  if (!opts) return

  host?.scrollTo({ top: 0 })
  host?.classList.add('has-tree-session-loading')
  document.body.classList.add('has-tree-session-overlay')

  let overlay = getOverlayEl()
  if (!overlay) {
    overlay = document.createElement('div')
    overlay.id = OVERLAY_ID
    overlay.className = 'tree-session-overlay'
    overlay.setAttribute('role', 'alertdialog')
    overlay.setAttribute('aria-modal', 'true')
    overlay.setAttribute('aria-busy', 'true')
    overlay.setAttribute('aria-live', 'polite')
    document.body.appendChild(overlay)
  }

  overlay.innerHTML = `
    <div class="tree-session-overlay-card card">
      <div class="spinner tree-session-spinner" aria-hidden="true"></div>
      <p class="tree-session-node">${escapeHtml(opts.nodeTitle)}</p>
      <p class="tree-session-message">${escapeHtml(opts.message)}</p>
      ${opts.hint ? `<p class="tree-session-hint">${escapeHtml(opts.hint)}</p>` : ''}
    </div>
  `
}

export async function fadeClearTreeSessionOverlay(): Promise<void> {
  const host = getMainContentEl()
  host?.classList.remove('has-tree-session-loading')
  document.body.classList.remove('has-tree-session-overlay')

  const overlay = getOverlayEl()
  if (overlay) await fadeOutAndRemove(overlay)
}

/** @deprecated 优先使用 fadeClearTreeSessionOverlay */
export function clearTreeSessionOverlay(): void {
  const host = getMainContentEl()
  host?.classList.remove('has-tree-session-loading')
  document.body.classList.remove('has-tree-session-overlay')
  getOverlayEl()?.remove()
}
