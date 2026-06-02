/** 仅改 hash；浏览器在 hash 变化时会自动触发 hashchange */

export function navigateHash(hash: string, opts?: { reload?: boolean }): void {
  const next = hash.startsWith('#') ? hash : `#${hash}`
  if (location.hash === next) {
    if (opts?.reload) window.dispatchEvent(new HashChangeEvent('hashchange'))
    return
  }
  location.hash = next
}
