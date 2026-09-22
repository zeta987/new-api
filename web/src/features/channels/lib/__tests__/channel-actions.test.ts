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
import { afterEach, beforeEach, expect, test } from 'vitest'

import { subscribeUsageLogsChanged } from '@/features/usage-logs/lib/refresh-events'
import { api } from '@/lib/api'

import { handleTestChannel } from '../channel-actions'

let originalAdapter: typeof api.defaults.adapter

beforeEach(() => {
  originalAdapter = api.defaults.adapter
})

afterEach(() => {
  api.defaults.adapter = originalAdapter
})

test.each([true, false])(
  'notifies usage-log listeners after a resolved channel test (success=%s)',
  async (success) => {
    api.defaults.adapter = async (config) => ({
      config,
      data: { success, message: success ? undefined : 'upstream failed' },
      headers: {},
      status: 200,
      statusText: 'OK',
    })
    let notifications = 0
    const unsubscribe = subscribeUsageLogsChanged(() => {
      notifications += 1
    })

    await handleTestChannel(7, { silent: true })

    expect(notifications).toBe(1)
    unsubscribe()
  }
)
