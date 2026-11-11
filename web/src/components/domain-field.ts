import { listDomainFields, updateDomainField } from '../lib/api'

function option(value: string, label: string, selected: boolean): string {
  return `<option value="${escapeAttr(value)}"${selected ? ' selected' : ''}>${escapeText(label)}</option>`
}

function escapeText(value: string): string {
  return value.replace(/[&<>]/g, (ch) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;' })[ch] ?? ch)
}

function escapeAttr(value: string): string {
  return escapeText(value).replace(/"/g, '&quot;')
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
    const { fields } = await listDomainFields()
    const names = [...fields]
    if (value && !names.includes(value)) names.push(value)
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
      let next = select.value
      if (next === '__custom') {
        const typed = window.prompt('领域名称（1～12 个字）', value)?.trim() ?? ''
        if (!typed || [...typed].length > 12) {
          select.value = value
          return
        }
        next = typed
      }
      if (!next || next === value) {
        select.value = value
        return
      }
      select.disabled = true
      try {
        await updateDomainField(domainId, next)
        value = next
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

