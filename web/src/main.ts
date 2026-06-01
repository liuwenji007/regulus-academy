import './style.css'
import { mountAppShell, setBreadcrumb, updateSidebar, navFromHash, invalidateSidebarCourses } from './components/layout'
import { navigateHash } from './lib/navigate'
import { ensureProfile, showProfilePicker } from './components/profile-picker'
import { onProfileChange, setActiveProfile, type UserProfile } from './lib/profile'
import { renderHome } from './pages/home'
import { renderTree } from './pages/tree'
import { renderGraph } from './pages/graph'
import { renderCourses } from './pages/courses'
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

  if (hash === '/graph') {
    void renderGraph(content)
    return
  }

  if (hash === '/courses') {
    void renderCourses(content)
    return
  }

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

function applyDevProfileSeed(): void {
  if (!import.meta.env.DEV) return
  const params = new URLSearchParams(location.search)
  const raw = params.get('seedProfile')
  if (!raw) return
  try {
    const profile = JSON.parse(raw) as UserProfile
    if (profile?.id && profile?.displayName) {
      setActiveProfile(profile)
      params.delete('seedProfile')
      const q = params.toString()
      const next = location.pathname + (q ? `?${q}` : '') + location.hash
      history.replaceState(null, '', next)
    }
  } catch {
    /* ignore invalid seed */
  }
}

async function boot(): Promise<void> {
  const app = document.querySelector<HTMLDivElement>('#app')!
  applyDevProfileSeed()
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
