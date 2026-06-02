export type AnswerFormat = 'text' | 'json' | 'choice'

export interface SessionExercise {
  answerFormat: AnswerFormat
  choices?: string[]
  choiceMode?: 'single' | 'multiple'
}

export interface ExerciseDraft {
  text: string
  selectedChoices: string[]
}

export function normalizeSessionExercise(raw: unknown): SessionExercise | null {
  if (!raw || typeof raw !== 'object') return null
  const o = raw as Record<string, unknown>
  const format = o.answerFormat
  if (format !== 'text' && format !== 'json' && format !== 'choice') return null
  const choices = Array.isArray(o.choices)
    ? o.choices.filter((c): c is string => typeof c === 'string' && c.trim() !== '')
    : undefined
  const choiceMode = o.choiceMode === 'multiple' ? 'multiple' : 'single'
  return {
    answerFormat: format,
    choices: choices?.length ? choices : undefined,
    choiceMode,
  }
}

export function exerciseFormatLabel(format: AnswerFormat): string {
  const map: Record<AnswerFormat, string> = {
    text: '文字作答',
    json: 'JSON / 代码',
    choice: '选择题',
  }
  return map[format]
}

export function exercisePlaceholder(format: AnswerFormat): string {
  const map: Record<AnswerFormat, string> = {
    text: '写下你的答案，可分点说明…',
    json: '粘贴或编写 JSON / 代码…',
    choice: '',
  }
  return map[format]
}

export function exerciseComposerHint(format: AnswerFormat, choiceMode?: 'single' | 'multiple'): string {
  if (format === 'json') return 'Enter 提交 · Shift+Enter 换行 · 可点「格式化」'
  if (format === 'choice') {
    return choiceMode === 'multiple' ? '可多选 · 选好后点「提交答案」' : '单选 · 选好后点「提交答案」'
  }
  return 'Enter 提交 · Shift+Enter 换行'
}

export function formatChoiceSubmission(
  selected: string[],
  choices: string[],
  mode: 'single' | 'multiple'
): string {
  const labels = selected
    .map((value) => {
      const idx = choices.indexOf(value)
      const letter = idx >= 0 ? String.fromCharCode(65 + idx) : '?'
      return `${letter}. ${value}`
    })
    .join(mode === 'multiple' ? '；' : '')
  return mode === 'multiple' ? `我选择：${labels}` : `我选择：${labels}`
}

export function collectExerciseAnswer(
  container: HTMLElement,
  exercise: SessionExercise
): { ok: true; text: string } | { ok: false; message: string } {
  if (exercise.answerFormat === 'choice') {
    const selected = Array.from(
      container.querySelectorAll<HTMLInputElement>('.coach-choice-input:checked')
    ).map((el) => el.value)
    const choices = exercise.choices ?? []
    if (selected.length === 0) {
      return { ok: false, message: '请先选择一个选项' }
    }
    if (exercise.choiceMode !== 'multiple' && selected.length > 1) {
      return { ok: false, message: '本题为单选题，只能选一个' }
    }
    return {
      ok: true,
      text: formatChoiceSubmission(selected, choices, exercise.choiceMode ?? 'single'),
    }
  }

  const input = container.querySelector<HTMLTextAreaElement>('#msg-input')
  const text = input?.value.trim() ?? ''
  if (!text) {
    return { ok: false, message: '请先写下你的答案' }
  }
  return { ok: true, text }
}

export function restoreExerciseDraft(
  container: HTMLElement,
  draft: ExerciseDraft,
  exercise: SessionExercise | null
): void {
  if (!exercise) return
  if (exercise.answerFormat === 'choice') {
    for (const value of draft.selectedChoices) {
      const el = container.querySelector<HTMLInputElement>(
        `.coach-choice-input[value="${CSS.escape(value)}"]`
      )
      if (el) el.checked = true
    }
    return
  }
  const input = container.querySelector<HTMLTextAreaElement>('#msg-input')
  if (input && draft.text) input.value = draft.text
}

export function readExerciseDraft(container: HTMLElement, exercise: SessionExercise | null): ExerciseDraft {
  if (!exercise || exercise.answerFormat === 'choice') {
    const selected = exercise
      ? Array.from(
          container.querySelectorAll<HTMLInputElement>('.coach-choice-input:checked')
        ).map((el) => el.value)
      : []
    return { text: '', selectedChoices: selected }
  }
  const input = container.querySelector<HTMLTextAreaElement>('#msg-input')
  return { text: input?.value ?? '', selectedChoices: [] }
}

export function renderExerciseComposer(opts: {
  exercise: SessionExercise
  placeholder: string
  sending: boolean
  quickActionsHtml: string
}): string {
  const { exercise, placeholder, sending, quickActionsHtml } = opts
  const label = exerciseFormatLabel(exercise.answerFormat)
  const hint = exerciseComposerHint(exercise.answerFormat, exercise.choiceMode)
  const disabled = sending ? 'disabled' : ''

  if (exercise.answerFormat === 'choice' && exercise.choices?.length) {
    const multiple = exercise.choiceMode === 'multiple'
    const inputType = multiple ? 'checkbox' : 'radio'
    const nameAttr = multiple ? '' : ' name="coach-choice"'
    const options = exercise.choices
      .map((choice, i) => {
        const letter = String.fromCharCode(65 + i)
        return `
          <label class="coach-choice-option">
            <input
              class="coach-choice-input"
              type="${inputType}"${nameAttr}
              value="${escapeAttr(choice)}"
              ${disabled}
            />
            <span class="coach-choice-marker">${letter}</span>
            <span class="coach-choice-text">${escapeHtml(choice)}</span>
          </label>
        `
      })
      .join('')

    return `
      <div class="coach-composer coach-composer--exercise coach-composer--choice">
        ${quickActionsHtml}
        <div class="coach-composer-head">
          <span class="coach-composer-label">练习作答 · ${label}</span>
          <span class="coach-composer-hint">${hint}</span>
        </div>
        <div class="coach-choice-list" role="${multiple ? 'group' : 'radiogroup'}" aria-label="选择题选项">
          ${options}
        </div>
        <div class="coach-composer-actions">
          <button type="button" class="btn btn-primary coach-send-btn" id="send-btn" ${disabled}>${sending ? '…' : '提交答案'}</button>
        </div>
      </div>
    `
  }

  const jsonTools =
    exercise.answerFormat === 'json'
      ? `<button type="button" class="btn btn-ghost btn-sm coach-json-format-btn" id="json-format-btn" ${disabled}>格式化 JSON</button>`
      : ''

  return `
    <div class="coach-composer coach-composer--exercise coach-composer--${exercise.answerFormat}">
      ${quickActionsHtml}
      <div class="coach-composer-head">
        <span class="coach-composer-label">练习作答 · ${label}</span>
        <span class="coach-composer-hint">${hint}</span>
      </div>
      <div class="coach-composer-body">
        <textarea
          class="input coach-answer-input${exercise.answerFormat === 'json' ? ' coach-answer-input--json' : ''}"
          id="msg-input"
          rows="${exercise.answerFormat === 'json' ? 8 : 5}"
          placeholder="${escapeAttr(placeholder)}"
          autocomplete="off"
          ${disabled}
          aria-label="练习作答"
        ></textarea>
        <div class="coach-composer-side">
          ${jsonTools}
          <button type="button" class="btn btn-primary coach-send-btn" id="send-btn" ${disabled}>${sending ? '…' : '提交答案'}</button>
        </div>
      </div>
    </div>
  `
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

function escapeAttr(s: string): string {
  return escapeHtml(s)
}

export function tryFormatJsonInTextarea(container: HTMLElement): boolean {
  const input = container.querySelector<HTMLTextAreaElement>('#msg-input')
  if (!input) return false
  const raw = input.value.trim()
  if (!raw) return false
  try {
    const parsed = JSON.parse(raw)
    input.value = JSON.stringify(parsed, null, 2)
    input.dispatchEvent(new Event('input', { bubbles: true }))
    return true
  } catch {
    return false
  }
}
