/** 遮罩/loading 淡出，避免切换路由时瞬间消失 */

export const LOADING_FADE_MS = 320

export function waitForNextPaint(): Promise<void> {
  return new Promise((resolve) => {
    requestAnimationFrame(() => requestAnimationFrame(() => resolve()))
  })
}

export function delayMs(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

export async function fadeOutAndRemove(el: HTMLElement | null | undefined): Promise<void> {
  if (!el?.isConnected) return
  if (el.dataset.fadingOut === '1') return
  el.dataset.fadingOut = '1'
  el.classList.add('is-fade-out')

  await new Promise<void>((resolve) => {
    const finish = () => {
      el.remove()
      resolve()
    }
    const timer = window.setTimeout(finish, LOADING_FADE_MS + 80)
    const onEnd = (e: TransitionEvent) => {
      if (e.target !== el || e.propertyName !== 'opacity') return
      clearTimeout(timer)
      finish()
    }
    el.addEventListener('transitionend', onEnd, { once: true })
  })
}

/** 先淡出容器内 loading，再写入新 HTML，并等待一帧绘制 */
export async function replaceContainerAfterLoadingFade(
  container: HTMLElement,
  html: string
): Promise<void> {
  const loading =
    container.querySelector<HTMLElement>('.page-loading') ??
    container.querySelector<HTMLElement>('.home-build-overlay')
  if (loading) await fadeOutAndRemove(loading)
  container.innerHTML = html
  await waitForNextPaint()
}
