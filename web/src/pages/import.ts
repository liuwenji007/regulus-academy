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
        <p class="page-sub">上传 PDF 或粘贴文章链接，系统会蒸馏材料大纲并生成学习路径。</p>
      </div>

      <div class="card import-tips">
        <p class="import-tips-title">导入说明</p>
        <div class="import-tips-grid">
          <div class="import-tips-block">
            <h3 class="import-tips-heading">PDF 文件</h3>
            <ul class="import-tips-list">
              <li>单文件上限 <strong>200 页</strong>、<strong>20 MB</strong>（需为可选中文字的文字版 PDF，扫描版可能无法识别）</li>
              <li>超过页数时可拆分 PDF 后分次导入，或由管理员提高 <code>REGULUS_INGEST_MAX_PDF_PAGES</code></li>
            </ul>
          </div>
          <div class="import-tips-block">
            <h3 class="import-tips-heading">网页 URL</h3>
            <ul class="import-tips-list">
              <li>适合公开博客、文档站、技术文章等可直接访问的页面</li>
              <li><strong>微信公众号、知乎、小红书</strong> 等站点禁止服务端抓取，链接导入通常会失败</li>
              <li>微信文章建议：在浏览器或微信内打开 → 打印/导出为 PDF → 上传 PDF 导入</li>
            </ul>
          </div>
        </div>
      </div>

      <div class="card card-elevated home-form-card">
        <div id="import-toast"></div>
        <div id="import-error"></div>

        <span class="field-label" id="import-file-label">PDF 文件</span>
        <div class="file-picker" role="group" aria-labelledby="import-file-label">
          <input
            class="file-picker-input"
            id="import-file"
            type="file"
            accept="application/pdf,.pdf"
            aria-describedby="import-file-hint import-file-name"
          />
          <label class="file-picker-trigger" for="import-file">选择 PDF</label>
          <span class="file-picker-name is-empty" id="import-file-name">未选择文件</span>
        </div>
        <p class="field-hint" id="import-file-hint">文字版 PDF，最多 200 页 / 20 MB</p>

        <p class="field-hint import-or-divider">或</p>

        <label class="field-label" for="import-url">网页 URL</label>
        <input class="input input-lg" id="import-url" type="url" placeholder="https://example.com/article" autocomplete="off" />
        <p class="field-hint">微信/知乎等反爬站点请改用 PDF；公开文档链接通常可用</p>

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
  const fileNameEl = container.querySelector<HTMLElement>('#import-file-name')!
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

  const syncFileName = () => {
    const file = fileInput.files?.[0]
    fileNameEl.textContent = file ? file.name : '未选择文件'
    fileNameEl.classList.toggle('is-empty', !file)
  }

  fileInput.addEventListener('change', syncFileName)

  btn.addEventListener('click', () => void submit())
  container.addEventListener('destroy', () => unsub(), { once: true })
}
