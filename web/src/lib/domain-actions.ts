import { invalidateSidebarCourses } from '../components/layout'

const LAST_DOMAIN_KEY = 'regulus:lastDomainId'

export async function handleDomainDelete(domainId: string): Promise<void> {
  if (localStorage.getItem(LAST_DOMAIN_KEY) === domainId) {
    localStorage.removeItem(LAST_DOMAIN_KEY)
  }
  invalidateSidebarCourses()
  location.hash = '#/'
  window.dispatchEvent(new HashChangeEvent('hashchange'))
}

export async function handleDomainRegenerate(
  domainId: string,
  newDomainId: string
): Promise<void> {
  if (localStorage.getItem(LAST_DOMAIN_KEY) === domainId) {
    localStorage.setItem(LAST_DOMAIN_KEY, newDomainId)
  }
  invalidateSidebarCourses()
  location.hash = `#/tree/${newDomainId}`
  window.dispatchEvent(new HashChangeEvent('hashchange'))
}
