/** 将文本转义为安全 HTML，防止 XSS */
export function escapeHtml(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}
