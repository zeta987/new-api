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
import { describe, expect, test } from 'vitest'

import {
  notifyUsageLogsChanged,
  subscribeUsageLogsChanged,
} from './refresh-events'

type RefreshEventTarget = NonNullable<
  Parameters<typeof notifyUsageLogsChanged>[0]
>

function createRefreshEventTarget(
  onStorageWrite: (key: string, value: string) => void = () => undefined
): RefreshEventTarget {
  const events = new EventTarget()

  return {
    addEventListener: events.addEventListener.bind(events),
    removeEventListener: events.removeEventListener.bind(events),
    dispatchEvent: events.dispatchEvent.bind(events),
    localStorage: {
      setItem: onStorageWrite,
    },
  } as RefreshEventTarget
}

function dispatchStorageEvent(
  target: RefreshEventTarget,
  key: string,
  value: string
) {
  const event = new Event('storage')
  Object.defineProperties(event, {
    key: { value: key },
    newValue: { value },
  })
  target.dispatchEvent(event)
}

describe('usage logs refresh events', () => {
  test('notifies an open logs tab when another tab reports a new log', () => {
    const receiver = createRefreshEventTarget()
    const sender = createRefreshEventTarget((key, value) => {
      dispatchStorageEvent(receiver, key, value)
    })
    let notifications = 0
    const unsubscribe = subscribeUsageLogsChanged(() => {
      notifications += 1
    }, receiver)

    notifyUsageLogsChanged(sender)

    expect(notifications).toBe(1)
    unsubscribe()
    notifyUsageLogsChanged(sender)
    expect(notifications).toBe(1)
  })

  test('notifies the current tab when browser storage is unavailable', () => {
    const currentTab = createRefreshEventTarget(() => {
      throw new Error('storage unavailable')
    })
    let notifications = 0
    const unsubscribe = subscribeUsageLogsChanged(() => {
      notifications += 1
    }, currentTab)

    notifyUsageLogsChanged(currentTab)

    expect(notifications).toBe(1)
    unsubscribe()
  })
})
