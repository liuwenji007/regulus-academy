import './style.css'
import { mountAppShell, setBreadcrumb, updateSidebar, navFromHash } from './components/layout'
import { renderHome } from './pages/home'
import { renderTree } from './pages/tree'
import { renderCoach } from './pages/coach'

const content = mountAppShell(document.querySelector<HTMLDivElement>('#app')!)

function route(): void {
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

  renderHome(content)
  updateSidebar({ active: 'home' })
  setBreadcrumb([{ label: '开始学习' }])
}

window.addEventListener('hashchange', route)
route()
