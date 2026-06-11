import { adminRequest, clearAdminToken, getAdminToken, setAdminToken } from '../lib/admin-auth'
import { fetchCloudInfo, isCloudDeployment } from '../lib/cloud'
import { setBreadcrumb, updateSidebar } from '../components/layout'

interface AdminStats {
  totalLearners: number
  activeLast7Days: number
  platformTokensToday: number
  platformTokensTotal: number
  newUsersToday: number
  runningBuildJobs: number
  asOf: string
}

interface AdminUser {
  id: string
  displayName: string
  createdAt: string
  lastSeenAt?: string
  messagesToday: number
  tokensToday: number
  hasByok: boolean
}

export async function renderAdmin(container: HTMLElement): Promise<void> {
  const info = await fetchCloudInfo()
  if (!isCloudDeployment(info)) {
    container.innerHTML = '<section class="page"><p class="alert alert-error">管理员控制台仅在 Cloud 部署可用。</p></section>'
    return
  }

  void updateSidebar({ active: 'settings' })
  setBreadcrumb([{ label: '管理员控制台' }])

  if (!getAdminToken()) {
    renderLogin(container)
    return
  }

  try {
    await renderDashboard(container)
  } catch (e) {
    container.innerHTML = `<section class="page"><p class="alert alert-error">${escapeHtml(e instanceof Error ? e.message : '加载失败')}</p></section>`
  }
}

function renderLogin(container: HTMLElement): void {
  container.innerHTML = `
    <section class="page page-admin">
      <h1 class="page-title">管理员登录</h1>
      <p class="page-sub">输入 Railway 环境变量中的 <code class="inline-code">ADMIN_TOKEN</code></p>
      <div class="card" style="max-width:420px">
        <label class="field-label" for="admin-token">Admin Token</label>
        <input class="input" id="admin-token" type="password" autocomplete="off" />
        <button type="button" class="btn btn-primary" id="admin-login-btn" style="margin-top:1rem">登录</button>
        <div id="admin-login-error"></div>
      </div>
    </section>
  `
  const errEl = container.querySelector<HTMLDivElement>('#admin-login-error')!
  container.querySelector('#admin-login-btn')?.addEventListener('click', () => {
    const token = container.querySelector<HTMLInputElement>('#admin-token')!.value.trim()
    if (!token) {
      errEl.innerHTML = '<div class="alert alert-error">请输入 Token</div>'
      return
    }
    setAdminToken(token)
    void renderAdmin(container)
  })
}

async function renderDashboard(container: HTMLElement): Promise<void> {
  const stats = await adminRequest<AdminStats>('/api/admin/stats')
  const usersRes = await adminRequest<{ users: AdminUser[] }>('/api/admin/users')
  const usageRes = await adminRequest<{ byDay: Array<{ date: string; platformTokens: number; byokTokens: number }> }>('/api/admin/usage')

  container.innerHTML = `
    <section class="page page-admin">
      <div class="page-hero" style="display:flex;justify-content:space-between;align-items:flex-start;gap:1rem;flex-wrap:wrap">
        <div>
          <h1 class="page-title">运营控制台</h1>
          <p class="page-sub">更新于 ${escapeHtml(stats.asOf)}</p>
        </div>
        <button type="button" class="btn btn-ghost btn-sm" id="admin-logout">退出登录</button>
      </div>
      <div class="admin-stats-grid">
        <div class="card"><p class="field-label">共学人数</p><p class="admin-stat-num">${stats.totalLearners}</p></div>
        <div class="card"><p class="field-label">近 7 天活跃</p><p class="admin-stat-num">${stats.activeLast7Days}</p></div>
        <div class="card"><p class="field-label">今日新用户</p><p class="admin-stat-num">${stats.newUsersToday}</p></div>
        <div class="card"><p class="field-label">平台 Token 今日</p><p class="admin-stat-num">${stats.platformTokensToday.toLocaleString()}</p></div>
        <div class="card"><p class="field-label">平台 Token 累计</p><p class="admin-stat-num">${stats.platformTokensTotal.toLocaleString()}</p></div>
        <div class="card"><p class="field-label">建课进行中</p><p class="admin-stat-num">${stats.runningBuildJobs}</p></div>
      </div>
      <h2 class="page-title" style="font-size:1.1rem;margin-top:2rem">用户列表</h2>
      <div class="card admin-users-table-wrap">
        <table class="admin-users-table">
          <thead><tr><th>昵称</th><th>今日消息</th><th>今日 Token</th><th>BYOK</th><th>最后活跃</th></tr></thead>
          <tbody>
            ${usersRes.users.map((u) => `
              <tr>
                <td>${escapeHtml(u.displayName)}</td>
                <td>${u.messagesToday}</td>
                <td>${u.tokensToday}</td>
                <td>${u.hasByok ? '是' : '否'}</td>
                <td>${u.lastSeenAt ? escapeHtml(u.lastSeenAt.slice(0, 16)) : '—'}</td>
              </tr>
            `).join('')}
          </tbody>
        </table>
      </div>
      <h2 class="page-title" style="font-size:1.1rem;margin-top:2rem">Token 趋势（14 天）</h2>
      <div class="card">
        <ul class="admin-usage-list">
          ${usageRes.byDay.map((row) => `<li><span>${escapeHtml(row.date)}</span> 平台 ${row.platformTokens.toLocaleString()} · BYOK ${row.byokTokens.toLocaleString()}</li>`).join('')}
        </ul>
      </div>
    </section>
  `
  container.querySelector('#admin-logout')?.addEventListener('click', () => {
    clearAdminToken()
    void renderAdmin(container)
  })
}

function escapeHtml(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}
