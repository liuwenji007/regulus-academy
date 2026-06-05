import { fadeOutAndRemove } from './loading-transition'
import { getDomainBuildJob, isDomainBuildRunning } from './domain-build-job'
import { pageLoadingHtml } from './page-loading'

const BUILD_HINT = 'AI 正在规划学习路径，通常需要 30 秒～2 分钟；可切换其他页面，进度显示在右上角'

/** 从其它页面返回首页时，若建课仍在进行则恢复遮罩 */
export function syncHomeBuildOverlay(container: HTMLElement): void {
  const job = getDomainBuildJob()
  if (!job || !isDomainBuildRunning()) return
  void setHomeBuildLoading(container, true, job.message)
}

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
