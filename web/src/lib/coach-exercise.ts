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

const LETTERED_CHOICE_LINE = /^[ \t]*([A-Da-d])[\.、．\)\:]?[ \t]+(.+?)[ \t]*$/gm

/** 从题干正文中识别 A–D 选项（与后端 ParseLetteredChoices 对齐，兼容 LLM 未填 choices 的旧题） */
export function inferChoiceFromQuestionText(question: string): SessionExercise | null {
  const byLetter = new Map<number, string>()
  let maxIdx = -1
  let m: RegExpExecArray | null
  LETTERED_CHOICE_LINE.lastIndex = 0
  while ((m = LETTERED_CHOICE_LINE.exec(question)) !== null) {
    const letter = m[1].toUpperCase()
    if (letter.length !== 1 || letter < 'A' || letter > 'D') continue
    const idx = letter.charCodeAt(0) - 65
    if (byLetter.has(idx)) continue
    const text = m[2].trim()
    if (!text) continue
    byLetter.set(idx, text)
    if (idx > maxIdx) maxIdx = idx
  }
  if (byLetter.size < 2) return null
  const choices = Array.from({ length: maxIdx + 1 }, (_, i) => byLetter.get(i) ?? '')
  return { answerFormat: 'choice', choices, choiceMode: 'single' }
}

export function nonEmptyChoiceCount(choices?: string[]): number {
  return choices?.filter((c) => typeof c === 'string' && c.trim() !== '').length ?? 0
}

/** 至少 2 个非空选项才算可渲染的选择题 */
export function isValidChoiceExercise(ex: SessionExercise | null | undefined): boolean {
  return ex?.answerFormat === 'choice' && nonEmptyChoiceCount(ex.choices) >= 2
}

export function inferChoiceFromAssistantContent(content: string): SessionExercise | null {
  const embedded = extractEmbeddedExercise(content)
  if (
    embedded.exercise?.answerFormat === 'choice' &&
    nonEmptyChoiceCount(embedded.exercise.choices) >= 2
  ) {
    return embedded.exercise
  }
  return inferChoiceFromQuestionText(content)
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
    text: '文字 / 代码',
    json: 'JSON',
    choice: '选择题',
  }
  return map[format]
}

export function exercisePlaceholder(format: AnswerFormat): string {
  const map: Record<AnswerFormat, string> = {
    text: '写下答案，可粘贴代码或分点说明…',
    json: '粘贴合法 JSON（对象或数组）…',
    choice: '',
  }
  return map[format]
}

export function exerciseComposerHint(
  format: AnswerFormat,
  choiceMode?: 'single' | 'multiple',
  opts?: { prefilled?: boolean }
): string {
  if (opts?.prefilled) {
    if (format === 'json') return '已填入题干片段 · 直接修改后提交 · 可点「格式化」'
    return '已填入题干代码 · 直接修改后提交 · Ctrl+Enter 提交'
  }
  if (format === 'json') return '须为合法 JSON · Enter 换行 · Ctrl+Enter 提交 · 可点「格式化」'
  if (format === 'choice') {
    return choiceMode === 'multiple' ? '可多选 · 选好后点「提交答案」' : '单选 · 选好后点「提交答案」'
  }
  return 'Enter 换行 · Ctrl+Enter 提交'
}

/** 从题干中提取最大的 Markdown 代码块正文（优先取 `---` 后最新题干）。 */
export function extractExerciseStarterCode(content: string): string {
  const parts = content.split(/\n---\n/)
  const searchIn = parts[parts.length - 1] ?? content

  const fenced = extractFencedCodeBlocks(searchIn)
  if (fenced) return fenced

  const indented = extractIndentedCodeBlock(searchIn)
  if (indented) return indented

  return extractLikelyDockerfileBlock(searchIn)
}

function extractFencedCodeBlocks(searchIn: string): string {
  const re = /```[^\n`]*\r?\n([\s\S]*?)```/g
  let best = ''
  let m: RegExpExecArray | null
  while ((m = re.exec(searchIn)) !== null) {
    const body = m[1].replace(/\n+$/, '')
    if (body.trim().length >= best.trim().length) best = body
  }
  return best
}

/** CommonMark 缩进代码块（连续 ≥2 行以 4 空格或 tab 开头）。 */
function extractIndentedCodeBlock(searchIn: string): string {
  const lines = searchIn.replace(/\r\n/g, '\n').split('\n')
  let best = ''
  let buf: string[] = []
  const flush = () => {
    const body = buf.map((l) => l.replace(/^(?: {4}|\t)/, '')).join('\n').replace(/\n+$/, '')
    if (body.trim().split('\n').filter((x) => x.trim()).length >= 2) {
      if (body.trim().length >= best.trim().length) best = body
    }
    buf = []
  }
  for (const line of lines) {
    if (/^(?: {4}|\t)/.test(line) || (buf.length > 0 && line.trim() === '')) {
      buf.push(line)
    } else {
      flush()
    }
  }
  flush()
  return best
}

/** 无 fence 时识别题干中的 Dockerfile 原文。 */
function extractLikelyDockerfileBlock(searchIn: string): string {
  if (!/dockerfile|docker\s*file/i.test(searchIn) && !/FROM\s+\S+/m.test(searchIn)) {
    return ''
  }
  const lines = searchIn.replace(/\r\n/g, '\n').split('\n')
  const start = lines.findIndex((l) => /^\s*FROM\s+\S+/i.test(l))
  if (start < 0) return ''
  const out: string[] = []
  for (let i = start; i < lines.length; i++) {
    const l = lines[i]
    if (/^做完后|^请写出|^要求[:：]/.test(l.trim())) break
    if (out.length > 0 && l.trim() === '' && !/^\s*(FROM|RUN|COPY|CMD|WORKDIR|ENV|ARG|EXPOSE|ENTRYPOINT|#)/i.test(lines[i + 1] ?? '')) {
      break
    }
    out.push(l.replace(/^\s{0,3}/, ''))
  }
  const body = out.join('\n').replace(/\n+$/, '')
  return body.trim().split('\n').filter((x) => x.trim()).length >= 2 ? body : ''
}

/** 是否应将题干代码回显到作答框（补全 / 找 bug / 配置片段）。 */
export function shouldPrefillExerciseStarter(
  questionContent: string,
  exercise: SessionExercise | null,
  starter: string
): boolean {
  if (!starter.trim()) return false
  if (exercise?.answerFormat === 'choice') return false

  // 「写出/写下」 alone 太宽（会命中「写出输出结果」）；只认改/补代码意图。
  // 「写出以下代码的输出结果」应排除：代码后紧跟「的输出/的运行」不算补全。
  const looksLikeFill =
    /补全|填空|修正|优化|改写|找出|错误|TODO|完整代码|完整\s*Dockerfile|写(出|下).{0,16}(完整\s*)?(代码|函数|方法|类型|接口|实现|Dockerfile|配置)(?!的输出|的运行)|实现\s*[`「]?[\w.]+|声明|bug[_ ]?find|code[_ ]?fill|Dockerfile/i.test(
      questionContent
    )

  // 读代码报输出 / 推结果：答案是文字结果，不能回显题干代码。
  // 注意：不要用「程序输出」——会误伤「使程序输出 '6'」这类补全题。
  const predictOutput =
    /(写出|写下|给出|求).{0,24}(输出结果|运行结果|执行结果)|输出结果是什么|运行结果是什么|会输出什么|打印(出|什么|哪些)|what\s+(does\s+it\s+)?print|what\s+is\s+the\s+output/i.test(
      questionContent
    )
  // 补全意图优先：题干说「补全…使程序输出」时仍应回显 starter。
  if (predictOutput && !looksLikeFill) return false

  // 说明/辨析类概念题：即使文中提到镜像名或旧题代码残留，也不回显。
  const conceptual =
    /请说明|主要优势|优缺点|指出一个|为什么|二者区别|请解释|用一句话|简述/.test(questionContent) &&
    !/补全|填空|修正|找出.*错误|TODO|完整代码|写出.*完整|改写.*Dockerfile/.test(questionContent)
  if (conceptual) return false

  if (exercise?.answerFormat === 'json') return true

  const lines = starter.split('\n').filter((l) => l.trim().length > 0)
  // 必须是「改/补代码」意图；禁止仅因多行代码块就回显。
  if (!looksLikeFill) return false
  return lines.length >= 1 && starter.trim().length >= 20
}

/** 取助手消息中的当前题干（`---` 后为换题连发时的新题）。 */
export function currentExercisePromptContent(content: string): string {
  const parts = content.split(/\n---\n/)
  return (parts[parts.length - 1] ?? content).trim()
}

/**
 * 仅从「当前正在作答的题」提取回显代码。
 * 找到带提交提示的当前题后无论成败都停止，避免串到上一题的 Dockerfile。
 */
export function findExerciseStarterPrefill(
  messages: { role: string; content: string }[],
  exercise: SessionExercise | null
): string {
  for (let i = messages.length - 1; i >= 0; i--) {
    if (messages[i].role !== 'assistant') continue
    const full = messages[i].content
    const prompt = currentExercisePromptContent(full)
    const isCurrentPrompt =
      isExerciseSubmitPrompt(prompt) || isExerciseSubmitPrompt(full)
    if (!isCurrentPrompt) {
      // 纯反馈气泡：继续往前找同一道题的题干
      continue
    }
    const starter = extractExerciseStarterCode(full)
    if (shouldPrefillExerciseStarter(prompt, exercise, starter)) {
      return starter
    }
    return ''
  }
  return ''
}

/** 展示用字母：跳过空槽后紧凑编号（与后端 formatChoicesForPrompt 一致） */
export function displayLetterForChoiceIndex(choices: string[], idx: number): string {
  let n = 0
  for (let i = 0; i < choices.length; i++) {
    const c = choices[i]?.trim() ?? ''
    if (!c) continue
    if (i === idx) return String.fromCharCode(65 + n)
    n++
  }
  return String.fromCharCode(65 + idx)
}

export function formatChoiceSubmission(
  selected: string[],
  choices: string[],
  mode: 'single' | 'multiple'
): string {
  const labels = selected
    .map((value) => {
      const idx = choices.indexOf(value)
      const letter = idx >= 0 ? displayLetterForChoiceIndex(choices, idx) : '?'
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
  draftText?: string
}): string {
  const { exercise, placeholder, sending, quickActionsHtml, draftText = '' } = opts
  const prefilled = draftText.trim().length > 0
  const label = exerciseFormatLabel(exercise.answerFormat)
  const hint = exerciseComposerHint(exercise.answerFormat, exercise.choiceMode, { prefilled })
  const disabled = sending ? 'disabled' : ''
  const draftLines = draftText.split('\n').length
  const rows = Math.min(
    16,
    Math.max(exercise.answerFormat === 'json' ? 8 : 5, prefilled ? Math.max(6, draftLines) : 5)
  )

  if (exercise.answerFormat === 'choice' && exercise.choices?.length) {
    const multiple = exercise.choiceMode === 'multiple'
    const inputType = multiple ? 'checkbox' : 'radio'
    const nameAttr = multiple ? '' : ' name="coach-choice"'
    let choiceLetter = 0
    const options = exercise.choices
      .map((choice) => choice.trim())
      .filter((choice) => choice.length > 0)
      .map((choice) => {
        const letter = String.fromCharCode(65 + choiceLetter++)
        return { choice, letter }
      })
      .map(({ choice, letter }) => {
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
          rows="${rows}"
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

const EXERCISE_SUBMIT_SUFFIX = '做完后直接把答案发给我。'

const EXERCISE_SUBMIT_MARKERS = [
  '做完后直接把答案发给我',
  '做完直接把答案发给我',
  '做完后把答案发给我',
  '做完把答案发给我',
  '直接把答案发给我',
] as const

function compactExercisePromptText(s: string): string {
  return s.replace(/\s/g, '')
}

/** 助手消息是否已进入「请用户作答」态（含「做完直接把…」等 LLM 措辞变体） */
export function isExerciseSubmitPrompt(content: string): boolean {
  if (!content.trim()) return false
  const compact = compactExercisePromptText(content)
  return EXERCISE_SUBMIT_MARKERS.some((m) => compact.includes(compactExercisePromptText(m)))
}

/** 从助手消息中剥离误输出的出题 JSON（历史 fallback，非主路径） */
export function extractEmbeddedExercise(content: string): {
  displayContent: string
  exercise: SessionExercise | null
} {
  const trimmed = content.trim()
  const jsonStart = trimmed.indexOf('{')
  if (jsonStart < 0) {
    return { displayContent: content, exercise: null }
  }

  const intro = trimmed.slice(0, jsonStart).trim()
  let jsonPart = trimmed.slice(jsonStart)
  const jsonEnd = jsonPart.lastIndexOf('}')
  if (jsonEnd < 0) {
    return { displayContent: content, exercise: null }
  }
  jsonPart = jsonPart.slice(0, jsonEnd + 1)

  try {
    const o = JSON.parse(jsonPart) as Record<string, unknown>
    const question = o.question
    if (typeof question !== 'string' || !question.trim()) {
      return { displayContent: content, exercise: null }
    }
    const rawFormat = o.answer_format ?? o.answerFormat
    const format =
      rawFormat === 'text' || rawFormat === 'json' || rawFormat === 'choice' ? rawFormat : undefined
    const exercise = normalizeSessionExercise({
      answerFormat: format,
      choices: o.choices,
      choiceMode: o.choice_mode ?? o.choiceMode,
    })
    if (!exercise) {
      return { displayContent: content, exercise: null }
    }
    const body = `${question.trim()}\n\n${EXERCISE_SUBMIT_SUFFIX}`
    const displayContent = intro ? `${intro}\n\n${body}` : body
    return { displayContent, exercise }
  } catch {
    return { displayContent: content, exercise: null }
  }
}

function extractJSONObjectText(content: string): string | null {
  const trimmed = content.trim()
  const start = trimmed.indexOf('{')
  if (start < 0) return null
  let jsonPart = trimmed.slice(start)
  const end = jsonPart.lastIndexOf('}')
  if (end < 0) return null
  return jsonPart.slice(0, end + 1)
}

/** 从助手消息中剥离误输出的批改 JSON */
export function extractEmbeddedGrade(content: string): {
  displayContent: string
  phase?: string
} | null {
  const jsonPart = extractJSONObjectText(content)
  if (!jsonPart) return null
  try {
    const o = JSON.parse(jsonPart) as Record<string, unknown>
    if (typeof o.passed !== 'boolean') return null
    let feedback = typeof o.feedback === 'string' ? o.feedback.trim() : ''
    if (!feedback) {
      feedback = o.passed ? '回答正确，很好。' : '这轮还没完全过关，建议再巩固一下。'
    }
    const intro = content.trim().slice(0, content.indexOf('{')).trim()
    const displayContent = intro ? `${intro}\n\n${feedback}` : feedback
    return {
      displayContent,
      phase: o.passed ? undefined : 'review',
    }
  } catch {
    return null
  }
}

/**
 * 规范化助手回复正文（加载历史消息时的 fallback）。
 * 实时对话应优先使用 API 返回的 phase + exercise；仅当历史记录内嵌 JSON 时使用本函数。
 */
export function normalizeCoachReply(
  content: string,
  phase: string,
  exercise: SessionExercise | null | undefined
): { content: string; phase: string; exercise: SessionExercise | null } {
  const grade = extractEmbeddedGrade(content)
  if (grade) {
    return {
      content: grade.displayContent,
      phase: grade.phase ?? phase,
      exercise: null,
    }
  }
  if (exercise && phase === 'exercise') {
    return { content, phase, exercise }
  }
  const extracted = extractEmbeddedExercise(content)
  if (extracted.exercise) {
    return {
      content: extracted.displayContent,
      phase: 'exercise',
      exercise: extracted.exercise,
    }
  }
  return { content, phase, exercise: exercise ?? null }
}

/** @deprecated 使用 normalizeCoachReply */
export function normalizeAssistantExerciseReply(
  content: string,
  phase: string,
  exercise: SessionExercise | null | undefined
): { content: string; phase: string; exercise: SessionExercise | null } {
  return normalizeCoachReply(content, phase, exercise)
}
