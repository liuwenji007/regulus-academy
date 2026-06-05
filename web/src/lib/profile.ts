export interface UserProfile {
  id: string
  displayName: string
  profileSummary?: string
  onboardedAt?: string
}

const STORAGE_KEY = 'regulus:activeProfile'

let activeProfile: UserProfile | null = null
let profileChangeListeners: Array<(profile: UserProfile) => void> = []

export function getActiveProfile(): UserProfile | null {
  if (activeProfile) return activeProfile
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw) as UserProfile
    if (!parsed?.id || !parsed?.displayName) return null
    activeProfile = parsed
    return activeProfile
  } catch {
    return null
  }
}

export function getActiveUserId(): string {
  return getActiveProfile()?.id ?? ''
}

export function setActiveProfile(profile: UserProfile): void {
  activeProfile = profile
  localStorage.setItem(STORAGE_KEY, JSON.stringify(profile))
  profileChangeListeners.forEach((fn) => fn(profile))
}

export function clearActiveProfile(): void {
  activeProfile = null
  localStorage.removeItem(STORAGE_KEY)
}

export function onProfileChange(listener: (profile: UserProfile) => void): () => void {
  profileChangeListeners.push(listener)
  return () => {
    profileChangeListeners = profileChangeListeners.filter((fn) => fn !== listener)
  }
}

export function hasActiveProfile(): boolean {
  return getActiveProfile() !== null
}
