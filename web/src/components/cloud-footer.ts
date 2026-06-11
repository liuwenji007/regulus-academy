import type { CloudInfo } from '../lib/cloud'

export function renderCloudFooter(info: CloudInfo): string {
  const links: string[] = []
  if (info.githubUrl) {
    links.push(`<a href="${escapeAttr(info.githubUrl)}" target="_blank" rel="noopener noreferrer">GitHub</a>`)
  }
  if (info.docsUrl) {
    links.push(`<a href="${escapeAttr(info.docsUrl)}" target="_blank" rel="noopener noreferrer">使用文档</a>`)
  }
  if (!links.length) return ''
  return `
    <footer class="cloud-footer" role="contentinfo">
      <p class="cloud-footer-hint">${escapeHtml(info.selfHostHint || '在线体验版 · 数据保存在共享实例')}</p>
      <p class="cloud-footer-links">${links.join(' · ')}</p>
    </footer>
  `
}

export function mountCloudFooter(shell: HTMLElement, info: CloudInfo): void {
  shell.querySelector('.cloud-footer')?.remove()
  const html = renderCloudFooter(info)
  if (!html) return
  shell.insertAdjacentHTML('beforeend', html)
}

function escapeHtml(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

function escapeAttr(s: string): string {
  return escapeHtml(s).replace(/"/g, '&quot;')
}
