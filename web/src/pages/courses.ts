import { getDomains, ApiError, type DomainSummary } from '../lib/api'
import { iconTree, iconChevronRight, iconRefresh, iconTrash } from '../lib/icons'
import { setBreadcrumb, updateSidebar } from '../components/layout'
import { showDomainConfirm } from '../components/domain-confirm'
import { handleDomainDelete, handleDomainRegenerate } from '../lib/domain-actions'

const LAST_DOMAIN_KEY = 'regulus:lastDomainId'

export async function renderCourses(container: HTMLElement): Promise<void> {
  void updateSidebar({ active: 'courses' })
  setBreadcrumb([
    { label: '开始学习', href: '#/' },
    { label: '我的课程' },
  ])

  container.innerHTML = `
    <section class="page page-courses">
      <header class="page-header">
        <div class="page-header-main">
          <h1 class="page-title">我的课程</h1>
          <div class="page-tree-meta">
            <p class="page-sub page-tree-hint">按层级浏览学习路径，点击节点开始微训练。</p>
          
          </div>
        </div>
      </header>
      <div id="courses-content">
        <div class="page-loading"><div class="spinner" aria-hidden="true"></div><p>加载课程…</p></div>
      </div>
    </section>
  `

  const contentEl = container.querySelector<HTMLDivElement>('#courses-content')!
  await loadCourses(contentEl)
}

async function loadCourses(el: HTMLElement): Promise<void> {
  try {
    const courses = await getDomains()
    if (courses.length === 0) {
      el.innerHTML = `
        <div class="card courses-empty">
          <p>还没有课程</p>
          <a href="#/" class="btn btn-primary btn-sm">去开始学习</a>
        </div>
      `
      return
    }

    el.innerHTML = `<div class="course-grid">${courses.map(renderCourseCard).join('')}</div>`
    bindCourseCards(el, courses)
  } catch (e) {
    el.innerHTML = `<div class="alert alert-error">${escapeHtml(e instanceof ApiError ? e.message : '加载失败')}</div>`
  }
}

function bindCourseCards(el: HTMLElement, courses: DomainSummary[]): void {
  el.querySelectorAll<HTMLElement>('.course-card').forEach((card) => {
    const id = card.dataset.domainId
    const course = courses.find((c) => c.id === id)
    if (!id || !course) return

    card.querySelector<HTMLAnchorElement>('.course-card-link')?.addEventListener('click', () => {
      localStorage.setItem(LAST_DOMAIN_KEY, id)
    })

    card.querySelector<HTMLButtonElement>('[data-action="regenerate"]')?.addEventListener('click', (e) => {
      e.preventDefault()
      e.stopPropagation()
      void (async () => {
        const outcome = await showDomainConfirm({
          domainId: id,
          domainName: course.name,
          action: 'regenerate',
        })
        if (!outcome.ok) return
        if (outcome.action === 'regenerate') {
          await handleDomainRegenerate(id, outcome.result.tree!.domainId, outcome.result)
        }
      })()
    })

    card.querySelector<HTMLButtonElement>('[data-action="delete"]')?.addEventListener('click', (e) => {
      e.preventDefault()
      e.stopPropagation()
      void (async () => {
        const outcome = await showDomainConfirm({
          domainId: id,
          domainName: course.name,
          action: 'delete',
        })
        if (!outcome.ok) return
        if (outcome.action === 'delete') {
          await handleDomainDelete(id)
          void loadCourses(el)
        }
      })()
    })
  })
}

function renderCourseCard(c: DomainSummary): string {
  const pct = c.nodeTotal > 0 ? Math.round((c.completed / c.nodeTotal) * 100) : 0
  return `
    <article class="course-card card" data-domain-id="${c.id}">
      <div class="course-card-tools">
        <button type="button" class="course-card-tool" data-action="regenerate" title="按当前学习画像重新生成课程" aria-label="按当前学习画像重新生成课程">${iconRefresh()}</button>
        <button type="button" class="course-card-tool course-card-tool--danger" data-action="delete" title="移除课程" aria-label="移除课程">${iconTrash()}</button>
      </div>
      <a href="#/tree/${c.id}" class="course-card-link">
        <div class="course-card-head">
          <span class="course-card-icon" aria-hidden="true">${iconTree()}</span>
          <h3 class="course-card-title">${escapeHtml(c.name)}</h3>
        </div>
        <div class="course-card-progress">
          <div class="course-card-progress-head">
            <p class="course-card-meta">${c.completed} / ${c.nodeTotal} 节点已完成</p>
            <span class="course-card-pct">${pct}%</span>
          </div>
          <div class="progress-bar" role="progressbar" aria-valuenow="${pct}" aria-valuemin="0" aria-valuemax="100">
            <div class="progress-fill" style="width:${pct}%"></div>
          </div>
        </div>
        <span class="course-card-enter">查看课程列表 ${iconChevronRight()}</span>
      </a>
    </article>
  `
}

function escapeHtml(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}
