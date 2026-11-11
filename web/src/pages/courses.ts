import { getDomains, ApiError, type DomainSummary } from '../lib/api'
import { iconTree, iconChevronRight, iconRefresh, iconTrash, iconMore } from '../lib/icons'
import { setBreadcrumb, updateSidebar } from '../components/layout'
import { showDomainConfirm } from '../components/domain-confirm'
import { mountDomainFieldPicker } from '../components/domain-field'
import { handleDomainDelete, handleDomainRegenerate } from '../lib/domain-actions'
import { PRESET_FIELDS, UNCLASSIFIED_FIELD } from '../lib/constellation-packing'

const LAST_DOMAIN_KEY = 'regulus:lastDomainId'
const FIELD_FILTER_KEY = 'regulus:coursesFieldFilter'
const FILTER_ALL = '__all__'

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
            <p class="page-sub page-tree-hint">按领域分类浏览，点击卡片进入学习路径。</p>
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

async function loadCourses(el: HTMLElement, preferredFilter?: string): Promise<void> {
  disposeCoursesMenuListeners(el)
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

    const fields = collectFields(courses)
    let activeFilter = preferredFilter ?? readStoredFilter()
    if (activeFilter !== FILTER_ALL && !fields.includes(activeFilter)) {
      activeFilter = FILTER_ALL
    }
    writeStoredFilter(activeFilter)

    el.innerHTML = `
      <div class="courses-toolbar">
        <div class="courses-field-filters" role="tablist" aria-label="按领域筛选">
          ${renderFilterChip(FILTER_ALL, '全部', courses.length, activeFilter)}
          ${fields
            .map((field) =>
              renderFilterChip(
                field,
                field,
                courses.filter((c) => fieldOf(c) === field).length,
                activeFilter
              )
            )
            .join('')}
        </div>
      </div>
      <div class="course-grid" id="course-grid"></div>
      <div class="courses-filter-empty card" id="courses-filter-empty" hidden>
        <p>该领域下暂无课程</p>
      </div>
    `

    const grid = el.querySelector<HTMLElement>('#course-grid')!
    const emptyEl = el.querySelector<HTMLElement>('#courses-filter-empty')!
    const paint = (filter: string) => {
      writeStoredFilter(filter)
      el.querySelectorAll<HTMLButtonElement>('.courses-field-filter').forEach((btn) => {
        const on = btn.dataset.field === filter
        btn.classList.toggle('is-active', on)
        btn.setAttribute('aria-selected', on ? 'true' : 'false')
      })
      const visible = filter === FILTER_ALL ? courses : courses.filter((c) => fieldOf(c) === filter)
      emptyEl.hidden = visible.length > 0
      grid.hidden = visible.length === 0
      grid.innerHTML = visible.map(renderCourseCard).join('')
      bindCourseCards(el, courses, () => void loadCourses(el, filter))
    }

    el.querySelectorAll<HTMLButtonElement>('.courses-field-filter').forEach((btn) => {
      btn.addEventListener('click', () => {
        const next = btn.dataset.field || FILTER_ALL
        paint(next)
      })
    })

    paint(activeFilter)
  } catch (e) {
    el.innerHTML = `<div class="alert alert-error">${escapeHtml(e instanceof ApiError ? e.message : '加载失败')}</div>`
  }
}

function bindCourseCards(el: HTMLElement, courses: DomainSummary[], reload: () => void): void {
  el.querySelectorAll<HTMLElement>('.course-card').forEach((card) => {
    const id = card.dataset.domainId
    const course = courses.find((c) => c.id === id)
    if (!id || !course) return

    card.querySelector<HTMLAnchorElement>('.course-card-link')?.addEventListener('click', () => {
      localStorage.setItem(LAST_DOMAIN_KEY, id)
    })

    const menuRoot = card.querySelector<HTMLElement>('.course-card-menu')
    const toggle = card.querySelector<HTMLButtonElement>('[data-action="menu"]')
    const panel = card.querySelector<HTMLElement>('.course-card-menu-panel')
    if (!menuRoot || !toggle || !panel) return

    const closeMenu = () => {
      menuRoot.classList.remove('is-open')
      toggle.setAttribute('aria-expanded', 'false')
      panel.hidden = true
    }

    const openMenu = () => {
      el.querySelectorAll<HTMLElement>('.course-card-menu.is-open').forEach((other) => {
        if (other === menuRoot) return
        other.classList.remove('is-open')
        other.querySelector<HTMLButtonElement>('[data-action="menu"]')?.setAttribute('aria-expanded', 'false')
        const otherPanel = other.querySelector<HTMLElement>('.course-card-menu-panel')
        if (otherPanel) otherPanel.hidden = true
      })
      menuRoot.classList.add('is-open')
      toggle.setAttribute('aria-expanded', 'true')
      panel.hidden = false
    }

    toggle.addEventListener('click', (e) => {
      e.preventDefault()
      e.stopPropagation()
      if (menuRoot.classList.contains('is-open')) closeMenu()
      else openMenu()
    })

    panel.addEventListener('click', (e) => e.stopPropagation())

    const fieldPicker = panel.querySelector<HTMLElement>('[data-action="field-picker"]')
    if (fieldPicker) {
      void mountDomainFieldPicker(fieldPicker, id, course.field ?? '', () => {
        closeMenu()
        reload()
      })
    }

    panel.querySelector<HTMLButtonElement>('[data-action="regenerate"]')?.addEventListener('click', (e) => {
      e.preventDefault()
      e.stopPropagation()
      closeMenu()
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

    panel.querySelector<HTMLButtonElement>('[data-action="delete"]')?.addEventListener('click', (e) => {
      e.preventDefault()
      e.stopPropagation()
      closeMenu()
      void (async () => {
        const outcome = await showDomainConfirm({
          domainId: id,
          domainName: course.name,
          action: 'delete',
        })
        if (!outcome.ok) return
        if (outcome.action === 'delete') {
          await handleDomainDelete(id)
          reload()
        }
      })()
    })
  })

  const onDocPointer = (e: Event) => {
    const target = e.target as Node | null
    el.querySelectorAll<HTMLElement>('.course-card-menu.is-open').forEach((menu) => {
      if (target && menu.contains(target)) return
      menu.classList.remove('is-open')
      menu.querySelector<HTMLButtonElement>('[data-action="menu"]')?.setAttribute('aria-expanded', 'false')
      const panel = menu.querySelector<HTMLElement>('.course-card-menu-panel')
      if (panel) panel.hidden = true
    })
  }
  const onKey = (e: KeyboardEvent) => {
    if (e.key !== 'Escape') return
    el.querySelectorAll<HTMLElement>('.course-card-menu.is-open').forEach((menu) => {
      menu.classList.remove('is-open')
      menu.querySelector<HTMLButtonElement>('[data-action="menu"]')?.setAttribute('aria-expanded', 'false')
      const panel = menu.querySelector<HTMLElement>('.course-card-menu-panel')
      if (panel) panel.hidden = true
    })
  }
  document.addEventListener('pointerdown', onDocPointer)
  document.addEventListener('keydown', onKey)
  disposeCoursesMenuListeners(el)
  ;(el as CoursesEl).__coursesMenuCleanup = () => {
    document.removeEventListener('pointerdown', onDocPointer)
    document.removeEventListener('keydown', onKey)
  }
}

type CoursesEl = HTMLElement & { __coursesMenuCleanup?: () => void }

function disposeCoursesMenuListeners(el: HTMLElement): void {
  const host = el as CoursesEl
  host.__coursesMenuCleanup?.()
  host.__coursesMenuCleanup = undefined
}

function renderCourseCard(c: DomainSummary): string {
  const pct = c.nodeTotal > 0 ? Math.round((c.completed / c.nodeTotal) * 100) : 0
  const field = fieldOf(c)
  return `
    <article class="course-card card" data-domain-id="${c.id}" data-field="${escapeAttr(field)}">
      <div class="course-card-menu">
        <button type="button" class="course-card-tool" data-action="menu" aria-expanded="false" aria-haspopup="menu" title="课程操作" aria-label="${escapeHtml(c.name)}的操作">${iconMore()}</button>
        <div class="course-card-menu-panel" role="menu" hidden>
          <div class="course-card-menu-field-picker" data-action="field-picker">
            <p class="course-card-menu-label">所属领域</p>
            <div class="course-card-field-list"><span class="course-card-field-loading">加载中…</span></div>
          </div>
          <div class="course-card-menu-divider" role="separator"></div>
          <button type="button" class="course-card-menu-item" data-action="regenerate" role="menuitem">
            <span class="course-card-menu-icon" aria-hidden="true">${iconRefresh()}</span>
            重新生成
          </button>
          <button type="button" class="course-card-menu-item course-card-menu-item--danger" data-action="delete" role="menuitem">
            <span class="course-card-menu-icon" aria-hidden="true">${iconTrash()}</span>
            移除课程
          </button>
        </div>
      </div>
      <a href="#/tree/${c.id}" class="course-card-link">
        <span class="course-card-badge">${escapeHtml(field)}</span>
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

function renderFilterChip(value: string, label: string, count: number, active: string): string {
  const on = value === active
  return `<button type="button" class="courses-field-filter${on ? ' is-active' : ''}" role="tab" data-field="${escapeAttr(value)}" aria-selected="${on ? 'true' : 'false'}">${escapeHtml(label)}<span class="courses-field-filter-count">${count}</span></button>`
}

function fieldOf(c: DomainSummary): string {
  const raw = c.field?.trim()
  return raw || UNCLASSIFIED_FIELD
}

function collectFields(courses: DomainSummary[]): string[] {
  const present = new Set(courses.map(fieldOf))
  const ordered: string[] = []
  for (const preset of PRESET_FIELDS) {
    if (present.has(preset)) ordered.push(preset)
  }
  const custom = [...present]
    .filter((name) => name !== UNCLASSIFIED_FIELD && !(PRESET_FIELDS as readonly string[]).includes(name))
    .sort((a, b) => a.localeCompare(b, 'zh'))
  ordered.push(...custom)
  if (present.has(UNCLASSIFIED_FIELD)) ordered.push(UNCLASSIFIED_FIELD)
  return ordered
}

function readStoredFilter(): string {
  try {
    return localStorage.getItem(FIELD_FILTER_KEY) || FILTER_ALL
  } catch {
    return FILTER_ALL
  }
}

function writeStoredFilter(value: string): void {
  try {
    localStorage.setItem(FIELD_FILTER_KEY, value)
  } catch {
    /* ignore */
  }
}

function escapeHtml(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}

function escapeAttr(s: string): string {
  return escapeHtml(s).replace(/"/g, '&quot;')
}
