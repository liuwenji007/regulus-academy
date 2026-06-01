/** 知识树启动会话时的全屏 loading（挂在 #main-content 上） */

export function clearTreeSessionOverlay(): void {
  const host = document.getElementById('main-content')
  host?.classList.remove('has-tree-session-loading')
  host?.querySelector('#tree-session-overlay')?.remove()
}
