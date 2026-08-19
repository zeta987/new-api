/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
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
