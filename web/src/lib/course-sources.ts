import { addDomainSource, listDomainSources, type DomainSourceMeta } from './api'
import { fetchCloudInfo, isCloudDeployment } from './cloud'
import { setAsideOpen } from '../components/aside-panel'
import { iconBookOpen, iconX } from './icons'

let teardown: (() => void) | null = null

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

function sourceLabel(s: DomainSourceMeta): string {
  return s.filename || s.url || s.kind
}

function syncTriggerLabel(btn: HTMLButtonElement, count: number): void {
  btn.textContent = count > 0 ? `资料 · ${count}` : '资料'
  btn.setAttribute('aria-label', count > 0 ? `资料管理，已挂 ${count} 份` : '资料管理')
}

function syncEmptyHint(meta: HTMLElement | null, count: number, open: () => void, signal: AbortSignal): void {
  if (!meta) return
  meta.querySelector('#course-sources-hint')?.remove()
  if (count > 0) return
  const hint = document.createElement('button')
  hint.type = 'button'
  hint.id = 'course-sources-hint'
  hint.className = 'page-tree-sources-hint'
  hint.textContent = '可在资料中挂载文档'
  hint.addEventListener('click', open, { signal })
  meta.appendChild(hint)
}

/** 课程树页：顶栏「资料」+ 主栏内右侧 overlay 抽屉（与助教互斥） */
export async function mountCourseSources(root: HTMLElement, domainId: string): Promise<void> {
  teardown?.()
  teardown = null

  const page = root.querySelector<HTMLElement>('.page-tree')
  const actions = root.querySelector<HTMLElement>('.domain-actions')
  const meta = root.querySelector<HTMLElement>('.page-tree-meta')
  const mainPanel = document.querySelector<HTMLElement>('.main-panel')
  if (!page || !actions || !mainPanel) return

  const ac = new AbortController()
  const { signal } = ac

  const btn = document.createElement('button')
  btn.type = 'button'
  btn.className = 'btn btn-ghost btn-sm'
  btn.id = 'domain-sources-btn'
  btn.title = '管理课程资料'
  btn.textContent = '资料'
  const deleteBtn = actions.querySelector('#domain-delete-btn')
  if (deleteBtn) actions.insertBefore(btn, deleteBtn)
  else actions.appendChild(btn)

  const host = document.createElement('div')
  host.className = 'course-sources-host'
  host.hidden = true
  host.innerHTML = `
    <div class="course-sources-backdrop" id="course-sources-backdrop" hidden></div>
    <aside class="course-sources-drawer" id="course-sources-drawer" aria-label="资料管理" hidden>
      <header class="course-sources-drawer__header">
        <div class="course-sources-drawer__title">
          <span class="course-sources-drawer__icon">${iconBookOpen()}</span>
          <div>
            <strong>资料管理</strong>
            <p class="course-sources-drawer__subtitle" id="course-sources-subtitle"></p>
          </div>
        </div>
        <button type="button" class="course-sources-drawer__close" id="course-sources-close" aria-label="关闭">${iconX()}</button>
      </header>
      <div class="course-sources-drawer__body" id="course-sources-body"></div>
    </aside>
  `
  mainPanel.appendChild(host)

  let open = false
  let cloud = false
  let sources: DomainSourceMeta[] = []

  try {
    cloud = isCloudDeployment(await fetchCloudInfo())
  } catch {
    cloud = false
  }

  const subtitle = host.querySelector<HTMLElement>('#course-sources-subtitle')
  if (subtitle) {
    subtitle.textContent = cloud
      ? '绑定到节点，供讲练与助教引用。在线体验版请勿上传公司机密。'
      : '绑定到节点，供主线讲练和助教引用。'
  }

  const setOpen = (next: boolean) => {
    open = next
    host.hidden = !open
    host.classList.toggle('is-open', open)
    const drawer = host.querySelector<HTMLElement>('#course-sources-drawer')
    const backdrop = host.querySelector<HTMLElement>('#course-sources-backdrop')
    if (drawer) {
      if (open) drawer.removeAttribute('hidden')
      else drawer.setAttribute('hidden', '')
    }
    if (backdrop) {
      if (open) backdrop.removeAttribute('hidden')
      else backdrop.setAttribute('hidden', '')
    }
    btn.classList.toggle('is-active', open)
    if (open) setAsideOpen(false)
  }

  const paintBody = (message = '') => {
    const body = host.querySelector<HTMLElement>('#course-sources-body')
    if (!body) return
    const items = sources.length
      ? `<ul class="course-source-list">${sources
          .map(
            (s) =>
              `<li>${escapeHtml(sourceLabel(s))} <span class="page-sub">${s.chunkCount} 段</span></li>`
          )
          .join('')}</ul>`
      : `<p class="page-sub">还没有挂资料。内部文档优先上传 Markdown / 文本或粘贴，内网链接通常抓不到。</p>`
    body.innerHTML = `
      ${items}
      ${message ? `<p class="page-sub course-sources-msg">${escapeHtml(message)}</p>` : ''}
      <form class="course-source-form" id="course-source-form">
        <input class="input" type="file" name="file" accept=".pdf,.md,.txt,.markdown,text/plain" />
        <input class="input" type="url" name="url" placeholder="公开网页 URL（可选）" />
        <textarea class="input" name="text" rows="3" placeholder="或粘贴内部文档片段"></textarea>
        <button type="submit" class="btn btn-ghost btn-sm">追加资料</button>
      </form>`
    body.querySelector('#course-source-form')?.addEventListener(
      'submit',
      (ev) => {
        ev.preventDefault()
        const form = ev.currentTarget
        if (!(form instanceof HTMLFormElement)) return
        void (async () => {
          const data = new FormData(form)
          const file = data.get('file')
          const url = String(data.get('url') ?? '').trim()
          const text = String(data.get('text') ?? '').trim()
          try {
            if (!(file instanceof File && file.size > 0) && !url && !text) {
              paintBody('请选择文件、填写 URL 或粘贴正文')
              return
            }
            sources =
              file instanceof File && file.size > 0
                ? await addDomainSource(domainId, { file })
                : await addDomainSource(domainId, { url, text })
            form.reset()
            syncTriggerLabel(btn, sources.length)
            syncEmptyHint(meta, sources.length, () => setOpen(true), signal)
            paintBody('已追加，并重新绑定到节点')
          } catch (err) {
            paintBody(err instanceof Error ? err.message : '追加失败')
          }
        })()
      },
      { signal }
    )
  }

  const openDrawer = () => {
    paintBody()
    setOpen(true)
  }

  btn.addEventListener('click', () => {
    if (open) setOpen(false)
    else openDrawer()
  }, { signal })

  host.addEventListener(
    'click',
    (e) => {
      const t = e.target as HTMLElement
      if (t.closest('#course-sources-close') || t.closest('#course-sources-backdrop')) {
        setOpen(false)
      }
    },
    { signal }
  )

  document.addEventListener(
    'keydown',
    (e) => {
      if (e.key === 'Escape' && open) setOpen(false)
    },
    { signal }
  )

  window.addEventListener(
    'regulus:aside-open',
    () => {
      if (open) setOpen(false)
    },
    { signal }
  )

  window.addEventListener(
    'hashchange',
    () => {
      if (!location.hash.includes(`/tree/${domainId}`)) teardown?.()
    },
    { signal }
  )

  teardown = () => {
    ac.abort()
    host.remove()
    btn.remove()
    meta?.querySelector('#course-sources-hint')?.remove()
    teardown = null
  }

  try {
    sources = await listDomainSources(domainId)
  } catch (err) {
    syncTriggerLabel(btn, 0)
    paintBody(err instanceof Error ? err.message : '资料加载失败')
    return
  }

  syncTriggerLabel(btn, sources.length)
  syncEmptyHint(meta, sources.length, openDrawer, signal)
  paintBody()
}
