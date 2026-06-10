import DOMPurify from 'dompurify'
import { marked } from 'marked'

marked.setOptions({ breaks: true, gfm: true })

/** 将助手消息的 Markdown 转为 HTML，经 DOMPurify 消毒后安全插入 DOM */
export function renderMarkdown(text: string): string {
  const raw = marked.parse(text, { async: false }) as string
  return DOMPurify.sanitize(raw)
}
