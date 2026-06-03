import { iconChannels, iconModel, iconChevronRight } from '../lib/icons'
import { setBreadcrumb, updateSidebar } from '../components/layout'

export function renderSettings(container: HTMLElement): void {
  void updateSidebar({ active: 'settings' })
  setBreadcrumb([{ label: '开始学习', href: '#/' }, { label: '设置' }])

  container.innerHTML = `
    <section class="page page-settings">
      <header class="page-header">
        <h1 class="page-title">设置</h1>
        <p class="page-sub">连接方式与进阶选项，日常学习请从「开始学习」进入即可。</p>
      </header>

      <nav class="settings-list" aria-label="设置项">
        <a href="#/settings/model" class="settings-row card">
          <span class="settings-row-icon" aria-hidden="true">${iconModel()}</span>
          <span class="settings-row-body">
            <span class="settings-row-title">AI 模型</span>
            <span class="settings-row-desc">API Key、提供商与模型名称；左下角可快速切换</span>
          </span>
          <span class="settings-row-chevron" aria-hidden="true">${iconChevronRight()}</span>
        </a>
        <a href="#/settings/channels" class="settings-row card">
          <span class="settings-row-icon" aria-hidden="true">${iconChannels()}</span>
          <span class="settings-row-body">
            <span class="settings-row-title">IM 频道</span>
            <span class="settings-row-desc">在 Telegram、钉钉、飞书等中与教练对话，进度与 Web 同步</span>
          </span>
          <span class="settings-row-chevron" aria-hidden="true">${iconChevronRight()}</span>
        </a>
      </nav>
    </section>
  `
}
