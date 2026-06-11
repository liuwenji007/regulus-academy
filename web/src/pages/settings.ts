import { iconChannels, iconModel, iconSettings, iconSparkles, iconChevronRight } from '../lib/icons'
import { setBreadcrumb, updateSidebar } from '../components/layout'
import { fetchCloudInfo, isCloudDeployment, type CloudInfo } from '../lib/cloud'
import { escapeHtml } from '../lib/utils'

function settingsRowLink(opts: {
  href: string
  icon: string
  title: string
  desc: string
  badge?: string
}): string {
  const badge = opts.badge
    ? `<span class="settings-row-badge settings-row-badge--demo">${escapeHtml(opts.badge)}</span>`
    : ''
  return `
    <a href="${opts.href}" class="settings-row card">
      <span class="settings-row-icon" aria-hidden="true">${opts.icon}</span>
      <span class="settings-row-body">
        <span class="settings-row-head">
          <span class="settings-row-title">${escapeHtml(opts.title)}</span>
          ${badge}
        </span>
        <span class="settings-row-desc">${escapeHtml(opts.desc)}</span>
      </span>
      <span class="settings-row-chevron" aria-hidden="true">${iconChevronRight()}</span>
    </a>
  `
}

function settingsRowDisabled(opts: {
  icon: string
  title: string
  desc: string
  badge: string
}): string {
  return `
    <div class="settings-row card settings-row--disabled" aria-disabled="true">
      <span class="settings-row-icon settings-row-icon--muted" aria-hidden="true">${opts.icon}</span>
      <span class="settings-row-body">
        <span class="settings-row-head">
          <span class="settings-row-title">${escapeHtml(opts.title)}</span>
          <span class="settings-row-badge settings-row-badge--demo">${escapeHtml(opts.badge)}</span>
        </span>
        <span class="settings-row-desc">${escapeHtml(opts.desc)}</span>
      </span>
    </div>
  `
}

function cloudBanner(info: CloudInfo): string {
  const docs = info.docsUrl
    ? `<a href="${escapeHtml(info.docsUrl)}" target="_blank" rel="noopener noreferrer">自托管文档</a>`
    : '本地 Docker 自托管'
  return `
    <div class="settings-cloud-banner card" role="note">
      <p class="settings-cloud-banner__title">在线演示模式</p>
      <p class="settings-cloud-banner__text">核心学习功能可用；IM 机器人需在你自己的服务器上部署，${docs} 可查看完整能力。</p>
    </div>
  `
}

export async function renderSettings(container: HTMLElement): Promise<void> {
  void updateSidebar({ active: 'settings' })
  setBreadcrumb([{ label: '开始学习', href: '#/' }, { label: '设置' }])

  const info = await fetchCloudInfo()
  const cloud = isCloudDeployment(info)

  const imRow = cloud
    ? settingsRowDisabled({
        icon: iconChannels(),
        title: 'IM 频道',
        badge: '演示模式不可用',
        desc: 'Telegram、钉钉、飞书等需本地 Docker 部署后配置；在线 Demo 无法运行长连接 Gateway',
      })
    : settingsRowLink({
        href: '#/settings/channels',
        icon: iconChannels(),
        title: 'IM 频道',
        desc: '在 Telegram、钉钉、飞书等中与教练对话，进度与 Web 同步',
      })

  const adminSection = cloud
    ? `
      <p class="settings-section-label">在线体验版</p>
      <nav class="settings-list" aria-label="在线体验版">
        ${settingsRowLink({
          href: '#/admin',
          icon: iconSettings(),
          title: '管理员控制台',
          desc: 'Token 消耗、用户配额与运维统计（需 ADMIN_TOKEN）',
        })}
      </nav>
    `
    : ''

  container.innerHTML = `
    <section class="page page-settings">
      <header class="page-header page-header-compact">
        <h1 class="page-title">设置</h1>
        <p class="page-sub">连接方式与进阶选项，日常学习请从「开始学习」进入即可。</p>
      </header>

      ${cloud ? cloudBanner(info) : ''}

      <nav class="settings-list" aria-label="设置项">
        ${settingsRowLink({
          href: '#/settings/model',
          icon: iconModel(),
          title: 'AI 模型',
          desc: 'API Key、提供商与模型名称；左下角可快速切换',
        })}
        ${settingsRowLink({
          href: '#/settings/profile',
          icon: iconSparkles(),
          title: '学习画像',
          desc: '查看与编辑学生画像，影响课程规划与讲解风格',
        })}
        ${imRow}
      </nav>
      ${adminSection}
    </section>
  `
}
