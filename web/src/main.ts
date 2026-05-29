import './style.css'
import { renderHome } from './pages/home'
import { renderTree } from './pages/tree'
import { renderCoach } from './pages/coach'

const app = document.querySelector<HTMLDivElement>('#app')!

function route(): void {
  const hash = location.hash.slice(1) || '/'

  const treeMatch = hash.match(/^\/tree\/([^/]+)$/)
  if (treeMatch) {
    void renderTree(app, treeMatch[1])
    return
  }

  const coachMatch = hash.match(/^\/coach\/([^/]+)$/)
  if (coachMatch) {
    void renderCoach(app, coachMatch[1])
    return
  }

  renderHome(app)
}

window.addEventListener('hashchange', route)
route()
