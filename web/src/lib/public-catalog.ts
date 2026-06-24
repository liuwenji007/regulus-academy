import { buildDomain, ApiError, type PublicDomainEntry } from './api'
import {
  applyServerBuildProgress,
  clearPendingBuild,
  finishDomainBuildJobError,
  finishDomainBuildJobSuccess,
  savePendingBuild,
  tryStartDomainBuildJob,
} from './domain-build-job'
import { setPageBuildLoading } from './home-build-loading'
import { stashPrefetchTree } from './course-prefetch'
import { navigateHash } from './navigate'
import { invalidateSidebarCourses, refreshLLMStatusAfterBusy } from '../components/layout'
import { showRelatedBuildConfirm } from '../components/related-build-confirm'
import { tryRecoverFromQuotaError } from './cloud'

export const PUBLIC_CATALOG_PREVIEW = 2

const LAST_DOMAIN_KEY = 'regulus:lastDomainId'
const TREE_FOCUS_PREFIX = 'regulus:treeFocus:'

export interface PublicDomainStartOptions {
  errEl?: HTMLElement | null
  toastEl?: HTMLElement | null
  pageContainer?: HTMLElement
  input?: HTMLInputElement | null
  action?: 'merge' | 'separate'
}

function escapeHtml(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}

function saveTreeFocus(domainId: string, focusNodeKeys?: string[], focusLabel?: string): void {
  if (!focusNodeKeys?.length) {
    sessionStorage.removeItem(TREE_FOCUS_PREFIX + domainId)
    return
  }
  sessionStorage.setItem(
    TREE_FOCUS_PREFIX + domainId,
    JSON.stringify({ keys: focusNodeKeys, label: focusLabel ?? '' })
  )
}

function navigateToTree(
  domainId: string,
  result?: { focusNodeKeys?: string[]; focusLabel?: string; message?: string },
  toastEl?: HTMLElement | null
): void {
  saveTreeFocus(domainId, result?.focusNodeKeys, result?.focusLabel)
  if (result?.message && toastEl) {
    toastEl.innerHTML = `<div class="alert alert-success">${escapeHtml(result.message)}</div>`
  }
  localStorage.setItem(LAST_DOMAIN_KEY, domainId)
  invalidateSidebarCourses()
  navigateHash(`/tree/${domainId}`)
}

export function renderPublicCard(d: PublicDomainEntry): string {
  return `
    <article class="public-card card">
      <div class="public-card-head">
        <h3 class="public-card-title">${escapeHtml(d.name)}</h3>
        <span class="badge badge-muted">v${d.version}</span>
      </div>
      <p class="public-card-desc">${escapeHtml(d.description || '社区维护的标准学习路径')}</p>
      <p class="public-card-meta">${d.nodeCount} 个节点 · 建课后可能按用户能力裁剪</p>
      <button type="button" class="btn btn-secondary btn-sm public-card-btn" data-public-start data-public-name="${escapeHtml(d.name)}">开始学习</button>
    </article>
  `
}

export function bindPublicDomainStarts(root: HTMLElement, options: Omit<PublicDomainStartOptions, 'action'>): void {
  root.querySelectorAll<HTMLButtonElement>('[data-public-start]').forEach((btn) => {
    btn.addEventListener('click', () => {
      void startPublicDomain(btn, options)
    })
  })
}

export async function startPublicDomain(
  btn: HTMLButtonElement,
  options: PublicDomainStartOptions = {}
): Promise<void> {
  const name = btn.dataset.publicName?.trim()
  if (!name) return
  const { errEl, toastEl, pageContainer, input } = options
  if (input) input.value = name
  const page = pageContainer ?? btn.closest<HTMLElement>('.page')?.parentElement ?? undefined
  const isRetry = Boolean(options.action)
  if (!isRetry && !tryStartDomainBuildJob(name)) {
    if (errEl) {
      errEl.innerHTML = '<div class="alert alert-error">已有课程正在创建，请稍候或查看右上角进度</div>'
    }
    return
  }
  btn.disabled = true
  const prev = btn.textContent
  btn.textContent = '加载中…'
  if (errEl) errEl.innerHTML = ''
  if (toastEl) toastEl.innerHTML = ''
  if (page) await setPageBuildLoading(page, true, '任务已创建…')
  let handoffToTree = false
  try {
    const result = await buildDomain(name, {
      action: options.action,
      onJobAccepted: (jobId) => savePendingBuild({ jobId, topic: name }),
      onProgress: (status) => {
        applyServerBuildProgress(status)
        if (page) void setPageBuildLoading(page, true, status.message || '正在创建课程…')
      },
    })
    clearPendingBuild()
    if (result.status === 'related' && result.existingDomain) {
      if (page) await setPageBuildLoading(page, false)
      const choice = await showRelatedBuildConfirm({
        message: result.message,
        relation: result.relation,
        existingDomain: result.existingDomain,
        newCourseName: result.intent?.displayName ?? name,
      })
      btn.disabled = false
      btn.textContent = prev ?? '开始学习'
      if (choice === 'merge') {
        await startPublicDomain(btn, { ...options, action: 'merge' })
        return
      }
      if (choice === 'separate') {
        await startPublicDomain(btn, { ...options, action: 'separate' })
        return
      }
      finishDomainBuildJobError('已取消建课', refreshLLMStatusAfterBusy)
      return
    }
    if (result.status !== 'ready' || !result.tree) {
      const msg = result.message ?? '无法加载学习路径'
      if (errEl) {
        errEl.innerHTML = `<div class="alert alert-error">${escapeHtml(msg)}</div>`
      }
      finishDomainBuildJobError(msg, refreshLLMStatusAfterBusy)
      return
    }
    if (result.personalized && toastEl) {
      toastEl.innerHTML = '<div class="alert alert-success">已根据你的背景裁剪学习路径</div>'
    }
    handoffToTree = true
    stashPrefetchTree(result.tree)
    finishDomainBuildJobSuccess(
      { domainId: result.tree.domainId, message: result.message },
      refreshLLMStatusAfterBusy
    )
    navigateToTree(result.tree.domainId, result, toastEl)
  } catch (e) {
    clearPendingBuild()
    if (await tryRecoverFromQuotaError(e)) {
      finishDomainBuildJobError('已取消或已配置 Key', refreshLLMStatusAfterBusy)
      return
    }
    const msg = e instanceof ApiError ? e.message : '网络错误，请稍后重试'
    if (errEl) {
      errEl.innerHTML = `<div class="alert alert-error">${escapeHtml(msg)}</div>`
    }
    finishDomainBuildJobError(msg, refreshLLMStatusAfterBusy)
  } finally {
    if (!handoffToTree && page) {
      await setPageBuildLoading(page, false)
    }
    btn.disabled = false
    btn.textContent = prev ?? '开始学习'
  }
}
