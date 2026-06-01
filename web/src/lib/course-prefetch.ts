import type { KnowledgeTree } from './api'

const PREFIX = 'regulus:prefetchTree:'

function key(domainId: string): string {
  return PREFIX + domainId
}

/** 建课成功后写入，渲染成功后再清除 */
export function stashPrefetchTree(tree: KnowledgeTree): void {
  try {
    sessionStorage.setItem(key(tree.domainId), JSON.stringify(tree))
  } catch {
    /* quota / private mode */
  }
}

/** 读取但不删除，避免并发渲染时第一份消费、第二份失败 */
export function peekPrefetchTree(domainId: string): KnowledgeTree | null {
  try {
    const raw = sessionStorage.getItem(key(domainId))
    if (!raw) return null
    return JSON.parse(raw) as KnowledgeTree
  } catch {
    return null
  }
}

export function clearPrefetchTree(domainId: string): void {
  try {
    sessionStorage.removeItem(key(domainId))
  } catch {
    /* ignore */
  }
}

/** @deprecated 使用 peek + clearPrefetchTree */
export function takePrefetchTree(domainId: string): KnowledgeTree | null {
  const tree = peekPrefetchTree(domainId)
  if (tree) clearPrefetchTree(domainId)
  return tree
}
