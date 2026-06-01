import { invalidateSidebarCourses } from '../components/layout'
import { setAppBusy } from './app-busy'
import { stashPrefetchTree } from './course-prefetch'
import { navigateHash } from './navigate'
import type { BuildDomainResult } from './api'

const LAST_DOMAIN_KEY = 'regulus:lastDomainId'

export async function handleDomainDelete(domainId: string): Promise<void> {
  if (localStorage.getItem(LAST_DOMAIN_KEY) === domainId) {
    localStorage.removeItem(LAST_DOMAIN_KEY)
  }
  invalidateSidebarCourses()
  navigateHash('/')
}

export async function handleDomainRegenerate(
  domainId: string,
  newDomainId: string,
  result?: BuildDomainResult
): Promise<void> {
  if (localStorage.getItem(LAST_DOMAIN_KEY) === domainId) {
    localStorage.setItem(LAST_DOMAIN_KEY, newDomainId)
  }
  invalidateSidebarCourses()
  if (result?.tree) {
    stashPrefetchTree(result.tree)
  }
  setAppBusy(true, 'build')
  navigateHash(`/tree/${newDomainId}`)
}
