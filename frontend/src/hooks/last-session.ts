const KEY = "nexa.last-session-id"

/** Last workspace session for this tab; survives refresh, not new tabs. */
export function readLastSessionId(): string | null {
  try {
    return sessionStorage.getItem(KEY)
  } catch {
    return null
  }
}

/** Persists or clears the last session id used by this tab. */
export function writeLastSessionId(id: string | null): void {
  try {
    if (id) sessionStorage.setItem(KEY, id)
    else sessionStorage.removeItem(KEY)
  } catch {
    // ignore quota / private mode
  }
}
