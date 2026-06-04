import { fadeOutAndRemove } from './loading-transition'
import { pageLoadingHtml } from './page-loading'

const BUILD_HINT = 'AI 正在规划学习路径，通常需要 30 秒～2 分钟，请稍候'

export async function setHomeBuildLoading(
  container: HTMLElement,
  active: boolean,
  title = '正在准备课程…'
): Promise<void> {
  const page = container.querySelector<HTMLElement>('.page-home')
  if (!page) return

  const existing = page.querySelector<HTMLElement>('.home-build-overlay')
  if (!active) {
    if (existing) await fadeOutAndRemove(existing)
    page.classList.remove('is-building')
    page.removeAttribute('aria-busy')
    return
  }

  page.classList.add('is-building')
  page.setAttribute('aria-busy', 'true')

  const inner = pageLoadingHtml(title, BUILD_HINT)
  if (existing) {
    existing.innerHTML = inner
    return
  }

  const overlay = document.createElement('div')
  overlay.className = 'home-build-overlay'
  overlay.innerHTML = inner
  page.appendChild(overlay)
}
