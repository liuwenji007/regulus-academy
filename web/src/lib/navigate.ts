/** 仅改 hash；浏览器在 hash 变化时会自动触发 hashchange */

export function navigateHash(hash: string): void {
  const next = hash.startsWith('#') ? hash : `#${hash}`
  if (location.hash === next) return
  location.hash = next
}
