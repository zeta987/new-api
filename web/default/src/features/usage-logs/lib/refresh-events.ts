const USAGE_LOGS_REFRESH_EVENT = 'new-api:usage-logs-changed'
const USAGE_LOGS_REFRESH_STORAGE_KEY = 'new-api:usage-logs-changed'

type UsageLogsRefreshTarget = Pick<
  Window,
  'addEventListener' | 'removeEventListener' | 'dispatchEvent' | 'localStorage'
>

export function notifyUsageLogsChanged(
  target: UsageLogsRefreshTarget = window
): void {
  target.dispatchEvent(new Event(USAGE_LOGS_REFRESH_EVENT))

  try {
    target.localStorage.setItem(
      USAGE_LOGS_REFRESH_STORAGE_KEY,
      `${Date.now()}:${crypto.randomUUID()}`
    )
  } catch {
    // The same-tab event still works when browser storage is unavailable.
  }
}

export function subscribeUsageLogsChanged(
  listener: () => void,
  target: UsageLogsRefreshTarget = window
): () => void {
  const handleLocalRefresh = () => listener()
  const handleStorageRefresh = (event: Event) => {
    const storageEvent = event as StorageEvent
    if (
      storageEvent.key === USAGE_LOGS_REFRESH_STORAGE_KEY &&
      storageEvent.newValue
    ) {
      listener()
    }
  }

  target.addEventListener(USAGE_LOGS_REFRESH_EVENT, handleLocalRefresh)
  target.addEventListener('storage', handleStorageRefresh)

  return () => {
    target.removeEventListener(USAGE_LOGS_REFRESH_EVENT, handleLocalRefresh)
    target.removeEventListener('storage', handleStorageRefresh)
  }
}
