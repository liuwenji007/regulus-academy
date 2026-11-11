import { listDomainFields, updateDomainField } from '../lib/api'
import { showDomainFieldNameDialog } from './domain-field-name'

function option(value: string, label: string, selected: boolean): string {
  return `<option value="${escapeAttr(value)}"${selected ? ' selected' : ''}>${escapeText(label)}</option>`
}

function escapeText(value: string): string {
  return value.replace(/[&<>]/g, (ch) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;' })[ch] ?? ch)
}

function escapeAttr(value: string): string {
  return escapeText(value).replace(/"/g, '&quot;')
}

async function resolveFieldNames(current: string): Promise<string[]> {
  const { fields } = await listDomainFields()
  const names = [...fields]
  const value = current.trim()
  if (value && !names.includes(value)) names.push(value)
  return names
}

async function saveDomainField(
  domainId: string,
  current: string,
  next: string
): Promise<string | null> {
  const value = current.trim()
  let field = next.trim()
  if (field === '__custom') {
    const typed = await showDomainFieldNameDialog(value)
    if (!typed) return null
    field = typed
  }
  // 后端要求领域名 1～12 字，空值（未分类）不能通过 PATCH 写入
  if (!field || field === value) return null
  await updateDomainField(domainId, field)
  return field
}

/** 填入领域下拉：已有领域、当前值、以及自定义输入 */
export async function mountDomainFieldSelect(
  select: HTMLSelectElement,
  domainId: string,
  current: string,
  onSaved?: () => void
): Promise<void> {
  let value = current.trim()
  const paint = async () => {
    const names = await resolveFieldNames(value)
    select.innerHTML = [
      option('', '未分类', value === ''),
      ...names.map((name) => option(name, name, name === value)),
      option('__custom', '自定义…', false),
    ].join('')
    select.value = value
  }
  await paint()
  select.addEventListener('change', () => {
    void (async () => {
      select.disabled = true
      try {
        const saved = await saveDomainField(domainId, value, select.value)
        if (!saved) {
          select.value = value
          return
        }
        value = saved
        await paint()
        onSaved?.()
      } catch (err) {
        select.value = value
        window.alert(err instanceof Error ? err.message : '保存领域失败')
      } finally {
        select.disabled = false
      }
    })()
  })
}

/** 在菜单内渲染领域选项列表（避免原生 select 弹层风格割裂） */
export async function mountDomainFieldPicker(
  host: HTMLElement,
  domainId: string,
  current: string,
  onSaved?: () => void
): Promise<void> {
  let value = current.trim()
  let busy = false

  const paint = async () => {
    const names = await resolveFieldNames(value)
    host.innerHTML = `
      <p class="course-card-menu-label">所属领域</p>
      <div class="course-card-field-list" role="group" aria-label="所属领域">
        ${names
          .map((name) => {
            const on = name === value
            return `<button type="button" class="course-card-menu-item course-card-field-option${on ? ' is-active' : ''}" data-field="${escapeAttr(name)}" role="menuitemradio" aria-checked="${on ? 'true' : 'false'}">
              <span class="course-card-field-check" aria-hidden="true">${on ? '✓' : ''}</span>
              <span>${escapeText(name)}</span>
            </button>`
          })
          .join('')}
        <button type="button" class="course-card-menu-item course-card-field-option" data-field="__custom" role="menuitem">
          <span class="course-card-field-check" aria-hidden="true"></span>
          <span>自定义…</span>
        </button>
      </div>
    `
  }

  await paint()
  host.addEventListener('click', (e) => {
    const btn = (e.target as HTMLElement | null)?.closest<HTMLButtonElement>('button[data-field]')
    if (!btn || busy) return
    e.preventDefault()
    e.stopPropagation()
    void (async () => {
      busy = true
      host.querySelectorAll('button').forEach((b) => {
        b.disabled = true
      })
      try {
        const saved = await saveDomainField(domainId, value, btn.dataset.field ?? '')
        if (!saved) return
        value = saved
        onSaved?.()
      } catch (err) {
        window.alert(err instanceof Error ? err.message : '保存领域失败')
      } finally {
        busy = false
        host.querySelectorAll('button').forEach((b) => {
          b.disabled = false
        })
      }
    })()
  })
}

