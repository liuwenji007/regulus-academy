import './style.css'
import { mountAppShell, setBreadcrumb, updateSidebar, navFromHash, invalidateSidebarCourses } from './components/layout'
import { ensureProfile, showProfilePicker } from './components/profile-picker'
import { onProfileChange } from './lib/profile'
import { renderHome } from './pages/home'
import { renderTree } from './pages/tree'
import { renderCoach } from './pages/coach'
import { renderChannels } from './pages/channels'

let content: HTMLElement | null = null

function route(): void {
  if (!content) return

  const hash = location.hash.slice(1) || '/'
  const nav = navFromHash(hash)

  const treeMatch = hash.match(/^\/tree\/([^/]+)$/)
  if (treeMatch) {
    void renderTree(content, treeMatch[1], nav)
    return
  }

  const coachMatch = hash.match(/^\/coach\/([^/]+)$/)
  if (coachMatch) {
    void renderCoach(content, coachMatch[1])
    return
  }

  if (hash === '/channels') {
    void renderChannels(content)
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
  location.hash = '#/'
  route()
})

document.addEventListener('click', (e) => {
  const btn = (e.target as HTMLElement).closest<HTMLButtonElement>('#switch-profile-btn')
  if (!btn) return
  e.preventDefault()
  void showProfilePicker({ required: false })
})

void boot()
