import {
  dismissDomainBuildJob,
  getDomainBuildJob,
  onDomainBuildJobChange,
} from '../lib/domain-build-job'

const AUTO_DISMISS_MS = 8000
let host: HTMLElement | null = null
let dismissTimer = 0

function escapeHtml(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}

function clearDismissTimer(): void {
  if (dismissTimer) {
    window.clearTimeout(dismissTimer)
    dismissTimer = 0
  }
}

function scheduleAutoDismiss(): void {
  clearDismissTimer()
  dismissTimer = window.setTimeout(() => {
    dismissDomainBuildJob()
  }, AUTO_DISMISS_MS)
}

function renderNotification(): void {
  if (!host) return
  const job = getDomainBuildJob()
  if (!job) {
    host.hidden = true
    host.innerHTML = ''
    clearDismissTimer()
    return
  }

  host.hidden = false
  const topic = escapeHtml(job.topic)

  if (job.phase === 'analyzing' || job.phase === 'generating') {
    host.innerHTML = `
      <div class="build-job-notification build-job-notification--running" role="status" aria-live="polite" aria-busy="true">
        <div class="build-job-notification-spinner spinner" aria-hidden="true"></div>
        <div class="build-job-notification-body">
          <p class="build-job-notification-title">正在创建课程</p>
          <p class="build-job-notification-topic">${topic}</p>
          <p class="build-job-notification-hint">${escapeHtml(job.message)}</p>
        </div>
      </div>
    `
    return
  }

  if (job.phase === 'success') {
    const href = job.resultDomainId ? `#/tree/${encodeURIComponent(job.resultDomainId)}` : '#/courses'
    host.innerHTML = `
      <div class="build-job-notification build-job-notification--success" role="status" aria-live="polite">
        <div class="build-job-notification-body">
          <p class="build-job-notification-title">课程已就绪</p>
          <p class="build-job-notification-hint">${escapeHtml(job.message)}</p>
          <a href="${href}" class="build-job-notification-link">查看学习路径</a>
        </div>
        <button type="button" class="build-job-notification-close" aria-label="关闭">×</button>
      </div>
    `
    scheduleAutoDismiss()
    host.querySelector('.build-job-notification-close')?.addEventListener('click', () => {
      dismissDomainBuildJob()
    })
    return
  }

  host.innerHTML = `
    <div class="build-job-notification build-job-notification--error" role="alert">
      <div class="build-job-notification-body">
        <p class="build-job-notification-title">建课失败</p>
        <p class="build-job-notification-hint">${escapeHtml(job.error ?? job.message)}</p>
      </div>
      <button type="button" class="build-job-notification-close" aria-label="关闭">×</button>
    </div>
  `
  host.querySelector('.build-job-notification-close')?.addEventListener('click', () => {
    dismissDomainBuildJob()
  })
}

/** 挂载右上角建课进度通知（全局，跨页面） */
export function mountBuildNotification(root: HTMLElement): void {
  if (host) return
  host = document.createElement('div')
  host.id = 'build-job-notification-host'
  host.className = 'build-job-notification-host'
  host.hidden = true
  root.appendChild(host)
  onDomainBuildJobChange(renderNotification)
  renderNotification()
}
