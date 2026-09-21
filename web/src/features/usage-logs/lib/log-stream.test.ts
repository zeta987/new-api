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
import { afterEach, describe, expect, test, vi } from 'vitest'

import { subscribeUsageLogStream } from './log-stream'

class FakeUsageLogStream extends EventTarget {
  started = false
  closed = false
  reconnectDelay = 3000
  xhr = { getResponseHeader: () => '180' }

  stream() {
    this.started = true
  }

  close() {
    this.closed = true
  }
}

afterEach(() => {
  vi.useRealTimers()
})

describe('usage log stream', () => {
  test('honors a rate-limited handshake before reconnecting and refreshing', () => {
    const stream = new FakeUsageLogStream()
    let cooldown = 0
    const unsubscribe = subscribeUsageLogStream(
      () => undefined,
      () => stream,
      (delay) => {
        cooldown = delay
      }
    )

    stream.dispatchEvent(
      Object.assign(new Event('error'), { responseCode: 429 })
    )

    expect(stream.reconnectDelay).toBe(180_000)
    expect(cooldown).toBe(180_000)
    unsubscribe()
  })

  test('refreshes after connecting and whenever the server reports a log', () => {
    const stream = new FakeUsageLogStream()
    let notifications = 0
    const unsubscribe = subscribeUsageLogStream(
      () => {
        notifications += 1
      },
      () => stream
    )

    expect(stream.started).toBe(true)

    stream.dispatchEvent(new Event('ready'))
    stream.dispatchEvent(new Event('log'))

    expect(notifications).toBe(2)

    unsubscribe()
    expect(stream.closed).toBe(true)
  })

  test('refreshes expired credentials after HTTP 401 and resumes with one replacement stream', async () => {
    vi.useFakeTimers()
    const expiredStream = new FakeUsageLogStream()
    const refreshedStream = new FakeUsageLogStream()
    const streams = [expiredStream, refreshedStream]
    let nextStream = 0
    let credentials = 'expired-token'
    const usedCredentials: string[] = []
    const refreshAuthentication = vi.fn().mockImplementation(async () => {
      credentials = 'refreshed-token'
      return {
        kind: 'authenticated',
        bundle: {
          access_token: credentials,
          token_type: 'Bearer',
          access_expires_at: 1_900_000_000,
          user: { id: 42, username: 'usage-log-user', role: 1 },
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
        },
      } as const
    })
    const unsubscribe = subscribeUsageLogStream(
      () => undefined,
      () => {
        usedCredentials.push(credentials)
        const stream = streams[nextStream++]
        if (!stream) throw new Error('Unexpected extra stream connection')
        return stream
      },
      undefined,
      refreshAuthentication
    )

    expiredStream.dispatchEvent(
      Object.assign(new Event('error'), { responseCode: 401 })
    )
    await vi.advanceTimersByTimeAsync(0)

    expect(expiredStream.closed).toBe(true)
    expect(refreshAuthentication).toHaveBeenCalledOnce()
    expect(nextStream).toBe(2)
    expect(usedCredentials).toEqual(['expired-token', 'refreshed-token'])
    expect(refreshedStream.started).toBe(true)

    unsubscribe()
    expect(refreshedStream.closed).toBe(true)
  })

  test('stops reconnecting when HTTP 401 confirms the session is revoked', async () => {
    vi.useFakeTimers()
    const revokedStream = new FakeUsageLogStream()
    const createStream = vi.fn(() => revokedStream)
    const refreshAuthentication = vi
      .fn()
      .mockResolvedValue({ kind: 'anonymous' } as const)
    const unsubscribe = subscribeUsageLogStream(
      () => undefined,
      createStream,
      undefined,
      refreshAuthentication
    )

    revokedStream.dispatchEvent(
      Object.assign(new Event('error'), { responseCode: 401 })
    )
    await vi.advanceTimersByTimeAsync(60_000)

    expect(revokedStream.closed).toBe(true)
    expect(refreshAuthentication).toHaveBeenCalledOnce()
    expect(createStream).toHaveBeenCalledOnce()

    unsubscribe()
  })
})
