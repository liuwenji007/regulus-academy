import { marked } from 'marked'

marked.setOptions({ breaks: true, gfm: true })

/** 将助手消息的 Markdown 转为 HTML（仅用于受信任的服务端 LLM 输出） */
export function renderMarkdown(text: string): string {
  return marked.parse(text, { async: false }) as string
}
