import './style.css'
import { mountAppShell, setBreadcrumb, updateSidebar, navFromHash, invalidateSidebarCourses } from './components/layout'
import { navigateHash } from './lib/navigate'
import { ensureProfile, showProfilePicker } from './components/profile-picker'
import { onProfileChange } from './lib/profile'
import { renderHome } from './pages/home'
import { renderTree } from './pages/tree'
import { renderCoach } from './pages/coach'
import { renderChannels } from './pages/channels'
import { renderSettings } from './pages/settings'

let content: HTMLElement | null = null
let treeRouteRaf = 0
let treeRouteId: string | null = null
let coachRouteRaf = 0
let coachRouteId: string | null = null

function route(): void {
  if (!content) return

  const hash = location.hash.slice(1) || '/'
  const nav = navFromHash(hash)

  const treeMatch = hash.match(/^\/tree\/([^/]+)$/)
  if (treeMatch) {
    const domainId = treeMatch[1]
    treeRouteId = domainId
    cancelAnimationFrame(treeRouteRaf)
    treeRouteRaf = requestAnimationFrame(() => {
      if (!content || treeRouteId !== domainId) return
      void renderTree(content, domainId, nav)
    })
    return
  }
  treeRouteId = null

  const coachMatch = hash.match(/^\/coach\/([^/]+)$/)
  if (coachMatch) {
    const sessionId = coachMatch[1]
    coachRouteId = sessionId
    cancelAnimationFrame(coachRouteRaf)
    coachRouteRaf = requestAnimationFrame(() => {
      if (!content || coachRouteId !== sessionId) return
      void renderCoach(content, sessionId)
    })
    return
  }
  coachRouteId = null

  if (hash === '/channels') {
    navigateHash('/settings/channels')
    return
  }

  if (hash === '/settings/channels') {
    void renderChannels(content)
    return
  }

  if (hash === '/settings') {
    void renderSettings(content)
    return
  }

  renderHome(content)
  void updateSidebar({ active: 'home' })
  setBreadcrumb([{ label: '开始学习' }])
}

async function boot(): Promise<void> {
  const app = document.querySelector<HTMLDivElement>('#app')!
  await ensureProfile()
  content = mountAppShell(app)
  window.addEventListener('hashchange', route)
  route()
}

onProfileChange(() => {
  if (!content) return
  invalidateSidebarCourses()
  navigateHash('/')
})

document.addEventListener('click', (e) => {
  const btn = (e.target as HTMLElement).closest<HTMLButtonElement>('#switch-profile-btn')
  if (!btn) return
  e.preventDefault()
  void showProfilePicker({ required: false })
})

void boot()
