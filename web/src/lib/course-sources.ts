import { addDomainSource, listDomainSources, type DomainSourceMeta } from './api'
import { fetchCloudInfo, isCloudDeployment } from './cloud'

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

export async function mountCourseSources(root: HTMLElement, domainId: string): Promise<void> {
  const slot = root.querySelector<HTMLElement>('#course-sources')
  if (!slot) return
  let cloud = false
  try {
    cloud = isCloudDeployment(await fetchCloudInfo())
  } catch {
    cloud = false
  }
  const paint = (sources: DomainSourceMeta[], message = '') => {
    const items = sources.length
      ? `<ul class="course-source-list">${sources
          .map(
            (s) =>
              `<li>${escapeHtml(sourceLabel(s))} <span class="page-sub">${s.chunkCount} 段</span></li>`
          )
          .join('')}</ul>`
      : `<p class="page-sub">还没有挂资料。内部文档优先上传 Markdown / 文本或粘贴，内网链接通常抓不到。</p>`
    slot.innerHTML = `
      <section class="card course-sources">
        <h2 class="section-title">课程资料</h2>
        <p class="page-sub">资料会绑定到节点，供主线讲练和助教引用。${
          cloud ? '在线体验版请勿上传公司机密。' : ''
        }</p>
        ${items}
        ${message ? `<p class="page-sub">${escapeHtml(message)}</p>` : ''}
        <form class="course-source-form" id="course-source-form">
          <input class="input" type="file" name="file" accept=".pdf,.md,.txt,.markdown,text/plain" />
          <input class="input" type="url" name="url" placeholder="公开网页 URL（可选）" />
          <textarea class="input" name="text" rows="3" placeholder="或粘贴内部文档片段"></textarea>
          <button type="submit" class="btn btn-ghost btn-sm">追加资料</button>
        </form>
      </section>`
    slot.querySelector('#course-source-form')?.addEventListener('submit', (ev) => {
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
            paint(sources, '请选择文件、填写 URL 或粘贴正文')
            return
          }
          const next =
            file instanceof File && file.size > 0
              ? await addDomainSource(domainId, { file })
              : await addDomainSource(domainId, { url, text })
          form.reset()
          paint(next, '已追加，并重新绑定到节点')
        } catch (err) {
          paint(sources, err instanceof Error ? err.message : '追加失败')
        }
      })()
    })
  }
  try {
    paint(await listDomainSources(domainId))
  } catch (err) {
    slot.innerHTML = `<p class="page-sub">${escapeHtml(err instanceof Error ? err.message : '资料加载失败')}</p>`
  }
}
