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
import type { QueryClient } from '@tanstack/react-query'
import { isAxiosError } from 'axios'

const REFRESH_INTERVAL_MS = 10_000
const LOG_QUERY_KEYS = [['logs'], ['usage-logs-stats']]
const cooldowns = new WeakMap<QueryClient, number>()

interface UsageLogsRefreshController {
  request(): void
  defer(delay: number): void
  retryAfter(): number
  setPaused(value: boolean): void
  dispose(): void
}

function isRetryableUsageLogsError(error: unknown): boolean {
  if (!isAxiosError(error) || error.code === 'ERR_CANCELED') return false
  const status = error.response?.status
  return status == null || status === 408 || status === 429 || status >= 500
}

export function usageLogsRetryAfterDelay(
  header: unknown,
  now = Date.now()
): number {
  const value = typeof header === 'string' ? header.trim() : ''
  let delay = 180_000
  if (/^\d+(\.\d+)?$/.test(value)) {
    delay = Number(value) * 1000
  } else if (value && Number.isFinite(Date.parse(value))) {
    delay = Math.max(0, Date.parse(value) - now)
  }
  // Keep timer delays within the platform limit, including malformed responses.
  return Math.min(delay, 2_147_483_647)
}

export function createUsageLogsRefreshController(
  queryClient: QueryClient
): UsageLogsRefreshController {
  let pending = false
  let running = false
  let paused = false
  let disposed = false
  let nextRefreshAt = Date.now() + REFRESH_INTERVAL_MS
  let timer: ReturnType<typeof setTimeout> | undefined
  const retryAfter = () =>
    Math.max(0, (cooldowns.get(queryClient) ?? 0) - Date.now())

  const clearTimer = () => {
    if (timer !== undefined) clearTimeout(timer)
    timer = undefined
  }

  const schedule = () => {
    clearTimer()
    if (disposed || paused || running || !pending) return
    timer = setTimeout(
      refresh,
      Math.max(0, nextRefreshAt - Date.now(), retryAfter())
    )
  }

  const defer = (delay: number) => {
    const retryAt = Math.max(
      cooldowns.get(queryClient) ?? 0,
      Date.now() + delay
    )
    cooldowns.set(queryClient, retryAt)
    nextRefreshAt = Math.max(nextRefreshAt, retryAt)
    pending = true
    schedule()
  }

  async function refresh() {
    timer = undefined
    if (disposed || paused || !pending) return
    if (retryAfter() > 0) {
      schedule()
      return
    }
    nextRefreshAt = Date.now() + REFRESH_INTERVAL_MS

    // Leave manual/initial requests intact and retain a catch-up for newer events.
    if (
      LOG_QUERY_KEYS.some((queryKey) => queryClient.isFetching({ queryKey }))
    ) {
      schedule()
      return
    }

    pending = false
    running = true
    const results = await Promise.allSettled(
      LOG_QUERY_KEYS.map((queryKey) =>
        queryClient.refetchQueries(
          { queryKey, type: 'active' },
          { cancelRefetch: false, throwOnError: true }
        )
      )
    )
    running = false
    if (
      results.some(
        (result) =>
          result.status === 'rejected' &&
          isRetryableUsageLogsError(result.reason)
      )
    ) {
      pending = true
    }
    schedule()
  }

  const cache = queryClient.getQueryCache()
  const handleError = (error: unknown, failedAt: number) => {
    if (!isAxiosError(error) || !isRetryableUsageLogsError(error)) return
    if (error.response?.status !== 429) {
      pending = true
      nextRefreshAt = Math.max(nextRefreshAt, failedAt + REFRESH_INTERVAL_MS)
      schedule()
      return
    }
    const retryAt =
      failedAt +
      usageLogsRetryAfterDelay(error.response.headers['retry-after'], failedAt)
    defer(Math.max(0, retryAt - Date.now()))
  }
  const isLogQuery = (key: readonly unknown[]) =>
    key[0] === 'logs' || key[0] === 'usage-logs-stats'
  // Preserve an existing cooldown across token rotation or a page remount.
  for (const query of cache.findAll()) {
    if (isLogQuery(query.queryKey)) {
      handleError(query.state.error, query.state.errorUpdatedAt)
    }
  }
  const unsubscribe = cache.subscribe((event) => {
    if (
      event.type === 'updated' &&
      event.action.type === 'error' &&
      isLogQuery(event.query.queryKey)
    ) {
      handleError(event.query.state.error, event.query.state.errorUpdatedAt)
    }
  })

  return {
    request() {
      pending = true
      schedule()
    },
    defer,
    retryAfter,
    setPaused(value: boolean) {
      paused = value
      if (!paused) pending = true
      schedule()
    },
    dispose() {
      disposed = true
      clearTimer()
      unsubscribe()
    },
  }
}
