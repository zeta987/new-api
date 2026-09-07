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
import { AxiosError, type AxiosAdapter, type AxiosStatic } from 'axios'
import { afterEach, beforeEach, expect, test, vi } from 'vitest'

import { useAuthStore } from '../../stores/auth-store'
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

beforeEach(() => {
  vi.useFakeTimers()
  clearAuthentication(false, 'idle')
  adapter.mockImplementation(async (config) => {
    throw new AxiosError('rate limited', 'ERR_BAD_REQUEST', config, undefined, {
      status: 429,
      statusText: 'Too Many Requests',
      data: '',
      headers: { 'retry-after': '60' },
      config,
    })
  })
})

afterEach(() => {
  clearAuthentication(false, 'idle')
  vi.useRealTimers()
})

test('a cold route bootstrap throws a temporary 429 instead of resolving anonymous', async () => {
  await expect(resolveAuthentication()).rejects.toMatchObject({
    response: { status: 429 },
  })
  expect(useAuthStore.getState().auth.bootstrapState).toBe('idle')
})

test('subsequent refresh callers respect Retry-After and recover after the window', async () => {
  expect((await refreshAuthentication()).kind).toBe('transient_error')
  expect((await refreshAuthentication()).kind).toBe('transient_error')
  expect(adapter).toHaveBeenCalledTimes(1)
  await vi.advanceTimersByTimeAsync(60_000)
  await refreshAuthentication()
  expect(adapter).toHaveBeenCalledTimes(2)
})
