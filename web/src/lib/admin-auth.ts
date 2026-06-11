const ADMIN_TOKEN_KEY = 'regulus:adminToken'

export function getAdminToken(): string {
  return sessionStorage.getItem(ADMIN_TOKEN_KEY) ?? ''
}

export function setAdminToken(token: string): void {
  const t = token.trim()
  if (t) sessionStorage.setItem(ADMIN_TOKEN_KEY, t)
  else sessionStorage.removeItem(ADMIN_TOKEN_KEY)
}

export function clearAdminToken(): void {
  sessionStorage.removeItem(ADMIN_TOKEN_KEY)
}

export async function adminRequest<T>(path: string, options?: RequestInit): Promise<T> {
  const token = getAdminToken()
  if (!token) throw new Error('未登录管理员')
  const res = await fetch(path, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
      ...options?.headers,
    },
  })
  const data = await res.json().catch(() => ({}))
  if (res.status === 401) {
    clearAdminToken()
    throw new Error('管理员 Token 无效')
  }
  if (!res.ok) {
    throw new Error((data as { error?: string }).error ?? `请求失败 (${res.status})`)
  }
  return data as T
}
