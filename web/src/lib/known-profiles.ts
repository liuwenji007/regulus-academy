import type { UserProfile } from './api'

const STORAGE_KEY = 'regulus:knownProfiles'

export function rememberProfile(profile: UserProfile): void {
  if (!profile?.id) return
  const list = listKnownProfiles().filter((p) => p.id !== profile.id)
  list.unshift({ id: profile.id, displayName: profile.displayName })
  localStorage.setItem(STORAGE_KEY, JSON.stringify(list.slice(0, 20)))
}

export function listKnownProfiles(): UserProfile[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw) as UserProfile[]
    return Array.isArray(parsed) ? parsed.filter((p) => p?.id && p?.displayName) : []
  } catch {
    return []
  }
}

export function mergeProfileLists(fromApi: UserProfile[], cloudMode: boolean): UserProfile[] {
  if (!cloudMode) return fromApi
  const map = new Map<string, UserProfile>()
  for (const p of [...listKnownProfiles(), ...fromApi]) {
    map.set(p.id, p)
  }
  return [...map.values()]
}
