import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  notifyUsageLogsChanged,
  subscribeUsageLogsChanged,
} from './refresh-events.ts'

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

    assert.equal(notifications, 1)
    unsubscribe()
    notifyUsageLogsChanged(sender)
    assert.equal(notifications, 1)
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

    assert.equal(notifications, 1)
    unsubscribe()
  })
})
