import { fadeOutAndRemove } from './loading-transition'
import { getDomainBuildJob, isDomainBuildRunning } from './domain-build-job'
import { pageLoadingHtml } from './page-loading'

const BUILD_HINT = 'AI 正在规划学习路径，通常需要 30 秒～2 分钟；可切换其他页面，进度显示在右上角'

function findBuildPage(container: HTMLElement): HTMLElement | null {
  return container.querySelector<HTMLElement>('.page-home, .page-catalog')
}

/** 从其它页面返回时，若建课仍在进行则恢复遮罩 */
export function syncHomeBuildOverlay(container: HTMLElement): void {
  const job = getDomainBuildJob()
  if (!job || !isDomainBuildRunning()) return
  void setPageBuildLoading(container, true, job.message)
}

export async function setPageBuildLoading(
  container: HTMLElement,
  active: boolean,
  title = '正在准备课程…'
): Promise<void> {
  const page = findBuildPage(container)
  if (!page) return

  const existing = page.querySelector<HTMLElement>('.home-build-overlay')
  if (!active) {
    if (existing) await fadeOutAndRemove(existing)
    page.classList.remove('is-building')
    page.removeAttribute('aria-busy')
    page.querySelector<HTMLButtonElement>('#home-coach-export-btn')?.removeAttribute('disabled')
    return
  }

  page.classList.add('is-building')
  page.setAttribute('aria-busy', 'true')
  page.querySelector<HTMLButtonElement>('#home-coach-export-btn')?.setAttribute('disabled', 'disabled')

  if (existing) {
    const titleEl = existing.querySelector<HTMLElement>('.page-loading > p')
    if (titleEl && titleEl.textContent !== title) {
      titleEl.textContent = title
    }
    return
  }

  const inner = pageLoadingHtml(title, BUILD_HINT)

  const overlay = document.createElement('div')
  overlay.className = 'home-build-overlay'
  overlay.innerHTML = inner
  page.appendChild(overlay)
}

/** @deprecated 使用 setPageBuildLoading */
export const setHomeBuildLoading = setPageBuildLoading
