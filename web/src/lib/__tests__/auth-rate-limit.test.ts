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
  AxiosError,
  AxiosHeaders,
  type AxiosAdapter,
  type AxiosStatic,
} from 'axios'
import { afterEach, beforeEach, expect, test, vi } from 'vitest'

import { useAuthStore, type AuthBundle } from '../../stores/auth-store'
import {
  clearAuthentication,
  refreshAuthentication,
  resolveAuthentication,
} from '../auth-session'

const { adapter } = vi.hoisted(() => ({ adapter: vi.fn<AxiosAdapter>() }))

vi.mock('axios', async (importOriginal) => {
  const actual = await importOriginal<typeof import('axios')>()
  return {
    ...actual,
    default: {
      ...actual.default,
      create: (config: Parameters<AxiosStatic['create']>[0]) =>
        actual.default.create({ ...config, adapter }),
    },
  }
})

const bundle: AuthBundle = {
  access_token: 'preserved-access-token',
  token_type: 'Bearer',
  access_expires_at: 2_000_000_000,
  user: { id: 42, username: 'test-user', role: 1 },
  session: {
    sid: 'preserved-session',
    current: true,
    login_method: 'password',
    ip: '127.0.0.1',
    user_agent: 'test',
    created_at: 100,
    last_active_at: 100,
    expires_at: 2_000_000_000,
  },
}

function rateLimited(errorConfig: Parameters<AxiosAdapter>[0]): never {
  throw new AxiosError(
    'rate limited',
    'ERR_BAD_REQUEST',
    errorConfig,
    undefined,
    {
      status: 429,
      statusText: 'Too Many Requests',
      data: '',
      headers: new AxiosHeaders({ 'retry-after': '60' }),
      config: errorConfig,
    }
  )
}

beforeEach(() => {
  vi.useFakeTimers()
  clearAuthentication(false, 'idle')
  adapter.mockReset()
})

afterEach(() => {
  clearAuthentication(false, 'idle')
  vi.useRealTimers()
})

test('a cold route bootstrap surfaces a temporary 429 instead of resolving anonymous', async () => {
  adapter.mockImplementation(async (config) => rateLimited(config))

  await expect(resolveAuthentication()).rejects.toMatchObject({
    response: { status: 429 },
  })
  expect(useAuthStore.getState().auth.bootstrapState).toBe('idle')
  expect(useAuthStore.getState().auth.user).toBeNull()
})

test('refresh callers respect Retry-After and recover after the window', async () => {
  adapter
    .mockImplementationOnce(async (config) => rateLimited(config))
    .mockImplementationOnce(async (config) => ({
      status: 200,
      statusText: 'OK',
      data: { success: true, data: bundle },
      headers: new AxiosHeaders(),
      config,
    }))

  expect((await refreshAuthentication()).kind).toBe('transient_error')
  expect((await refreshAuthentication()).kind).toBe('transient_error')
  expect(adapter).toHaveBeenCalledTimes(1)

  await vi.advanceTimersByTimeAsync(60_000)

  expect((await refreshAuthentication()).kind).toBe('authenticated')
  expect(adapter).toHaveBeenCalledTimes(2)
  expect(useAuthStore.getState().auth.accessToken).toBe(bundle.access_token)
  expect(useAuthStore.getState().auth.session?.sid).toBe(bundle.session.sid)
})

test('a rate-limited refresh preserves a still-valid in-memory session', async () => {
  useAuthStore.getState().auth.setBundle(bundle)
  adapter.mockImplementation(async (config) => rateLimited(config))

  expect((await refreshAuthentication()).kind).toBe('transient_error')

  const auth = useAuthStore.getState().auth
  expect(auth.user).toEqual(bundle.user)
  expect(auth.accessToken).toBe(bundle.access_token)
  expect(auth.session).toEqual(bundle.session)
  expect(auth.bootstrapState).toBe('idle')
})
