import { buildDomainFromSource, ApiError } from '../lib/api'
import {
  applyServerBuildProgress,
  clearPendingBuild,
  finishDomainBuildJobError,
  finishDomainBuildJobSuccess,
  getDomainBuildJob,
  isDomainBuildRunning,
  onDomainBuildJobChange,
  savePendingBuild,
  tryStartDomainBuildJob,
} from '../lib/domain-build-job'
import { setHomeBuildLoading, syncHomeBuildOverlay } from '../lib/home-build-loading'
import { stashPrefetchTree } from '../lib/course-prefetch'
import { navigateHash } from '../lib/navigate'
import {
  invalidateSidebarCourses,
  refreshLLMStatusAfterBusy,
  setBreadcrumb,
  updateSidebar,
} from '../components/layout'

const LAST_DOMAIN_KEY = 'regulus:lastDomainId'

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

export function renderImport(container: HTMLElement): void {
  void updateSidebar({ active: 'home' })
  setBreadcrumb([{ label: '开始学习', href: '#/' }, { label: '导入材料' }])

  container.innerHTML = `
    <section class="page page-import">
      <div class="page-hero">
        <p class="page-eyebrow">从材料建课</p>
        <h1 class="page-title">导入 PDF 或网页</h1>
        <p class="page-sub">上传 PDF 或粘贴文章链接，系统会蒸馏材料大纲并生成学习路径。扫描版 PDF 可能无法提取文字。</p>
      </div>

      <div class="card card-elevated home-form-card">
        <div id="import-toast"></div>
        <div id="import-error"></div>

        <label class="field-label" for="import-file">PDF 文件</label>
        <input class="input" id="import-file" type="file" accept="application/pdf,.pdf" />

        <p class="field-hint" style="margin: 1rem 0 0.5rem; text-align: center; opacity: 0.7;">或</p>

        <label class="field-label" for="import-url">网页 URL</label>
        <input class="input input-lg" id="import-url" type="url" placeholder="https://example.com/article" autocomplete="off" />

        <label class="field-label" for="import-name" style="margin-top: 1rem;">课程名称（可选）</label>
        <input class="input" id="import-name" type="text" placeholder="留空则使用材料主题" autocomplete="off" />

        <label class="field-label" for="import-goal">学习目标（可选）</label>
        <input class="input" id="import-goal" type="text" placeholder="例如：快速掌握文中核心概念" autocomplete="off" />

        <button class="btn btn-primary btn-lg" id="import-submit-btn" style="margin-top: 1rem;">开始导入</button>
        <p class="home-courses-link"><a href="#/">返回首页</a></p>
      </div>
    </section>
  `

  const fileInput = container.querySelector<HTMLInputElement>('#import-file')!
  const urlInput = container.querySelector<HTMLInputElement>('#import-url')!
  const nameInput = container.querySelector<HTMLInputElement>('#import-name')!
  const goalInput = container.querySelector<HTMLInputElement>('#import-goal')!
  const btn = container.querySelector<HTMLButtonElement>('#import-submit-btn')!
  const errEl = container.querySelector<HTMLDivElement>('#import-error')!
  const toastEl = container.querySelector<HTMLDivElement>('#import-toast')!

  let submitting = false

  const unsub = onDomainBuildJobChange(() => {
    const job = getDomainBuildJob()
    if (!job || !isDomainBuildRunning()) {
      void setHomeBuildLoading(container, false)
      return
    }
    void setHomeBuildLoading(container, true, job.message)
  })

  syncHomeBuildOverlay(container)

  const submit = async (): Promise<void> => {
    if (submitting || isDomainBuildRunning()) return
    errEl.innerHTML = ''
    toastEl.innerHTML = ''

    const file = fileInput.files?.[0]
    const url = urlInput.value.trim()
    if (!file && !url) {
      errEl.innerHTML = '<div class="alert alert-error">请上传 PDF 或填写网页 URL</div>'
      return
    }
    if (file && url) {
      errEl.innerHTML = '<div class="alert alert-error">请只选择一种导入方式</div>'
      return
    }

    const topic = nameInput.value.trim() || file?.name || url || '导入课程'
    if (!tryStartDomainBuildJob(topic)) {
      errEl.innerHTML = '<div class="alert alert-error">已有建课任务进行中，请稍候</div>'
      return
    }

    submitting = true
    btn.disabled = true
    await setHomeBuildLoading(container, true, '正在提交导入任务…')

    try {
      const result = await buildDomainFromSource(
        file ? { file } : { url },
        {
          name: nameInput.value.trim() || undefined,
          goal: goalInput.value.trim() || undefined,
          onJobAccepted: (jobId) => savePendingBuild({ jobId, topic }),
          onProgress: (status) => applyServerBuildProgress(status),
        }
      )

      if (result.status === 'ready' && result.tree?.domainId) {
        finishDomainBuildJobSuccess({
          domainId: result.tree.domainId,
          message: result.message,
        })
        stashPrefetchTree(result.tree)
        localStorage.setItem(LAST_DOMAIN_KEY, result.tree.domainId)
        invalidateSidebarCourses()
        if (result.message) {
          toastEl.innerHTML = `<div class="alert alert-success">${escapeHtml(result.message)}</div>`
        }
        navigateHash(`/tree/${result.tree.domainId}`)
        return
      }

      const msg = result.message ?? '导入建课失败'
      finishDomainBuildJobError(msg)
      errEl.innerHTML = `<div class="alert alert-error">${escapeHtml(msg)}</div>`
    } catch (e) {
      const msg = e instanceof ApiError ? e.message : '导入失败，请稍后重试'
      finishDomainBuildJobError(msg)
      clearPendingBuild()
      errEl.innerHTML = `<div class="alert alert-error">${escapeHtml(msg)}</div>`
    } finally {
      submitting = false
      btn.disabled = false
      await setHomeBuildLoading(container, false)
      void refreshLLMStatusAfterBusy()
    }
  }

  btn.addEventListener('click', () => void submit())
  container.addEventListener('destroy', () => unsub(), { once: true })
}
