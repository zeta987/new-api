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
import {
  QueryCache,
  QueryClient,
  QueryClientProvider,
  useQuery,
} from '@tanstack/react-query'
import { act, cleanup, render, screen } from '@testing-library/react'
import { AxiosError } from 'axios'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'
import { handleQueryError } from '@/lib/handle-server-error'
import { useAuthStore, type AuthBundle } from '@/stores/auth-store'

import { getAllLogs, getLogStats } from '../../api'
import { UsageLogsProvider } from '../usage-logs-provider'

const streamTestState = vi.hoisted(() => ({
  createdStreams: [] as Array<{
    headers: Record<string, string>
    source: EventTarget
  }>,
  events: [] as string[],
}))

vi.mock('sse.js', () => ({
  SSE: class extends EventTarget {
    authorization: string
    reconnectDelay = 10_000
    xhr = { getResponseHeader: () => '180' }

    constructor(_url: string, options: { headers: Record<string, string> }) {
      super()
      this.authorization = options.headers.Authorization ?? 'none'
      streamTestState.createdStreams.push({
        headers: options.headers,
        source: this,
      })
      streamTestState.events.push(`create:${this.authorization}`)
    }

    stream() {}

    close() {
      streamTestState.events.push(`close:${this.authorization}`)
    }
  },
}))

function makeAuthBundle(accessToken: string): AuthBundle {
  return {
    access_token: accessToken,
    token_type: 'Bearer',
    access_expires_at: 1_900_000_000,
    user: {
      id: 42,
      username: 'usage-log-user',
      role: 1,
    },
    session: {
      sid: 'usage-log-session',
      current: true,
      login_method: 'password',
      ip: '127.0.0.1',
      user_agent: 'vitest',
      created_at: 1,
      last_active_at: 1,
      expires_at: 1_900_000_000,
    },
  }
}

describe('usage logs provider stream authentication', () => {
  let queryClient: QueryClient

  beforeEach(() => {
    streamTestState.createdStreams.length = 0
    streamTestState.events.length = 0
    useAuthStore.getState().auth.reset('complete')
    queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
  })

  afterEach(() => {
    queryClient.clear()
    useAuthStore.getState().auth.reset('complete')
  })

  test('replaces the stream with current headers after token rotation', () => {
    useAuthStore.getState().auth.setBundle(makeAuthBundle('token-a'))

    const { unmount } = render(
      <QueryClientProvider client={queryClient}>
        <UsageLogsProvider>
          <div>usage logs</div>
        </UsageLogsProvider>
      </QueryClientProvider>
    )

    expect(streamTestState.createdStreams).toHaveLength(1)
    expect(streamTestState.createdStreams[0]?.headers.Authorization).toBe(
      'Bearer token-a'
    )

    act(() => {
      useAuthStore.getState().auth.setBundle(makeAuthBundle('token-b'))
    })

    expect(streamTestState.events).toEqual([
      'create:Bearer token-a',
      'close:Bearer token-a',
      'create:Bearer token-b',
    ])
    expect(streamTestState.createdStreams).toHaveLength(2)
    expect(streamTestState.createdStreams[1]?.headers.Authorization).toBe(
      'Bearer token-b'
    )

    unmount()
    expect(streamTestState.events).toEqual([
      'create:Bearer token-a',
      'close:Bearer token-a',
      'create:Bearer token-b',
      'close:Bearer token-b',
    ])
  })

  test('closes the stream without reconnecting after sign-out', () => {
    useAuthStore.getState().auth.setBundle(makeAuthBundle('token-b'))

    const { unmount } = render(
      <QueryClientProvider client={queryClient}>
        <UsageLogsProvider>
          <div>usage logs</div>
        </UsageLogsProvider>
      </QueryClientProvider>
    )

    act(() => {
      useAuthStore.getState().auth.reset('complete')
    })

    expect(streamTestState.events).toEqual([
      'create:Bearer token-b',
      'close:Bearer token-b',
    ])
    expect(streamTestState.createdStreams).toHaveLength(1)

    unmount()
    expect(streamTestState.events).toEqual([
      'create:Bearer token-b',
      'close:Bearer token-b',
    ])
  })
})

function LiveLogsProbe() {
  const logs = useQuery({
    queryKey: ['logs', 'common'],
    queryFn: () => getAllLogs(),
    initialData: {
      success: true,
      data: { items: [], total: 0, page: 1, page_size: 20 },
    },
    retry: false,
  })
  const stats = useQuery({
    queryKey: ['usage-logs-stats'],
    queryFn: () => getLogStats(),
    initialData: { success: true, data: { quota: 0, rpm: 0, tpm: 0 } },
    retry: false,
  })

  return (
    <output>
      {logs.data.data?.total}:{stats.data.data?.quota}
    </output>
  )
}

describe('usage logs background refresh', () => {
  let queryClient: QueryClient
  let navigateToErrorPage: ReturnType<typeof vi.fn>
  let originalAdapter: typeof api.defaults.adapter
  let requests: string[]
  let rejectUntil: number
  let rejectionStatus: number
  let serverTotal: number

  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-09-07T00:00:00Z'))
    vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible')
    streamTestState.createdStreams.length = 0
    streamTestState.events.length = 0
    useAuthStore.getState().auth.setBundle(makeAuthBundle('test-token'))
    navigateToErrorPage = vi.fn()
    queryClient = new QueryClient({
      queryCache: new QueryCache({
        onError: (error) => handleQueryError(error, navigateToErrorPage),
      }),
      defaultOptions: {
        queries: { retry: false, staleTime: 10_000, gcTime: Infinity },
      },
    })
    originalAdapter = api.defaults.adapter
    requests = []
    rejectUntil = 0
    rejectionStatus = 429
    serverTotal = 7
    api.defaults.adapter = async (config) => {
      requests.push(config.url ?? '')
      if (Date.now() < rejectUntil) {
        throw new AxiosError(
          'rate limited',
          'ERR_BAD_REQUEST',
          config,
          undefined,
          {
            config,
            data: '',
            headers: { 'retry-after': '180' },
            status: rejectionStatus,
            statusText: 'Too Many Requests',
          }
        )
      }
      return {
        config,
        headers: {},
        status: 200,
        statusText: 'OK',
        data: config.url?.includes('/stat')
          ? { success: true, data: { quota: 42, rpm: 1, tpm: 2 } }
          : {
              success: true,
              data: { items: [], total: serverTotal, page: 1, page_size: 20 },
            },
      }
    }
  })

  afterEach(() => {
    cleanup()
    queryClient.clear()
    api.defaults.adapter = originalAdapter
    useAuthStore.getState().auth.reset('complete')
    vi.clearAllTimers()
    vi.useRealTimers()
  })

  function renderLiveLogs() {
    return render(
      <QueryClientProvider client={queryClient}>
        <UsageLogsProvider>
          <LiveLogsProbe />
        </UsageLogsProvider>
      </QueryClientProvider>
    )
  }

  test('coalesces successive events into one background refresh per interval', async () => {
    renderLiveLogs()
    const source = streamTestState.createdStreams[0].source
    await act(async () => {
      source.dispatchEvent(new Event('ready'))
      await vi.advanceTimersByTimeAsync(1_000)
      source.dispatchEvent(new Event('log'))
      await vi.advanceTimersByTimeAsync(1_000)
      source.dispatchEvent(new Event('log'))
      await vi.advanceTimersByTimeAsync(7_999)
    })
    expect(requests).toHaveLength(0)

    await act(() => vi.advanceTimersByTimeAsync(1))
    expect(requests).toHaveLength(2)
    await act(() => vi.advanceTimersByTimeAsync(1))
    expect(screen.getByText('7:42')).toBeInTheDocument()

    await act(() => vi.advanceTimersByTimeAsync(30_000))
    expect(requests).toHaveLength(2)
  })

  test('waits through Retry-After and catches up without another log event', async () => {
    renderLiveLogs()
    rejectUntil = Date.now() + 190_000
    await act(async () => {
      streamTestState.createdStreams[0].source.dispatchEvent(new Event('log'))
      await vi.advanceTimersByTimeAsync(10_000)
    })
    expect(requests).toHaveLength(2)
    expect(screen.getByText('0:0')).toBeInTheDocument()

    await act(() => vi.advanceTimersByTimeAsync(179_999))
    expect(requests).toHaveLength(2)
    await act(() => vi.advanceTimersByTimeAsync(1))
    expect(requests).toHaveLength(4)
    await act(() => vi.advanceTimersByTimeAsync(1))
    expect(screen.getByText('7:42')).toBeInTheDocument()
  })

  test('keeps hidden tabs idle and refreshes when they become visible', async () => {
    renderLiveLogs()
    const visibility = vi.spyOn(document, 'visibilityState', 'get')
    await act(async () => {
      streamTestState.createdStreams[0].source.dispatchEvent(new Event('log'))
      visibility.mockReturnValue('hidden')
      document.dispatchEvent(new Event('visibilitychange'))
      await vi.advanceTimersByTimeAsync(60_000)
    })
    expect(requests).toHaveLength(0)

    await act(async () => {
      visibility.mockReturnValue('visible')
      document.dispatchEvent(new Event('visibilitychange'))
      await vi.advanceTimersByTimeAsync(1)
    })
    expect(requests).toHaveLength(2)
    expect(screen.getByText('7:42')).toBeInTheDocument()
  })

  test('preserves a rate-limited stream deadline when the tab is hidden and shown', async () => {
    renderLiveLogs()
    const visibility = vi.spyOn(document, 'visibilityState', 'get')
    await act(async () => {
      streamTestState.createdStreams[0].source.dispatchEvent(
        Object.assign(new Event('error'), { responseCode: 429 })
      )
      visibility.mockReturnValue('hidden')
      document.dispatchEvent(new Event('visibilitychange'))
      await vi.advanceTimersByTimeAsync(1_000)
      visibility.mockReturnValue('visible')
      document.dispatchEvent(new Event('visibilitychange'))
    })
    expect(streamTestState.createdStreams).toHaveLength(1)
    expect(requests).toHaveLength(0)

    await act(() => vi.advanceTimersByTimeAsync(178_999))
    expect(streamTestState.createdStreams).toHaveLength(1)
    await act(() => vi.advanceTimersByTimeAsync(1))
    expect(streamTestState.createdStreams).toHaveLength(2)
  })

  test('recovers a failed manual query without waiting for a new log event', async () => {
    renderLiveLogs()
    await act(() => vi.advanceTimersByTimeAsync(10_001))
    expect(screen.getByText('7:42')).toBeInTheDocument()

    rejectionStatus = 502
    rejectUntil = Date.now() + 10_000
    serverTotal = 8
    await act(() => queryClient.invalidateQueries({ queryKey: ['logs'] }))
    expect(requests).toHaveLength(3)

    await act(() => vi.advanceTimersByTimeAsync(10_001))
    expect(requests).toHaveLength(5)
    expect(screen.getByText('8:42')).toBeInTheDocument()
  })

  test('keeps the usage page mounted after HTTP 500 and automatically recovers', async () => {
    renderLiveLogs()
    await act(() => vi.advanceTimersByTimeAsync(10_001))
    expect(screen.getByText('7:42')).toBeInTheDocument()

    rejectionStatus = 500
    rejectUntil = Date.now() + 10_000
    serverTotal = 8
    await act(() => queryClient.invalidateQueries({ queryKey: ['logs'] }))
    expect(navigateToErrorPage).not.toHaveBeenCalled()
    expect(screen.getByText('7:42')).toBeInTheDocument()

    await act(() => vi.advanceTimersByTimeAsync(10_001))
    expect(screen.getByText('8:42')).toBeInTheDocument()
    expect(navigateToErrorPage).not.toHaveBeenCalled()
  })

  test('retains the error-page navigation for an unrelated HTTP 500 query', async () => {
    rejectionStatus = 500
    rejectUntil = Date.now() + 10_000

    await expect(
      queryClient.fetchQuery({
        queryKey: ['unrelated-query'],
        queryFn: () => api.get('/api/unrelated-query'),
      })
    ).rejects.toBeInstanceOf(AxiosError)

    expect(navigateToErrorPage).toHaveBeenCalledOnce()
  })
})
