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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, render } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { useAuthStore, type AuthBundle } from '@/stores/auth-store'

import { UsageLogsProvider } from '../usage-logs-provider'

const streamTestState = vi.hoisted(() => ({
  createdStreams: [] as Array<{ headers: Record<string, string> }>,
  events: [] as string[],
}))

vi.mock('sse.js', () => ({
  SSE: class {
    authorization: string

    constructor(_url: string, options: { headers: Record<string, string> }) {
      this.authorization = options.headers.Authorization ?? 'none'
      streamTestState.createdStreams.push({ headers: options.headers })
      streamTestState.events.push(`create:${this.authorization}`)
    }

    addEventListener() {}

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
