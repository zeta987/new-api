import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { subscribeUsageLogStream } from './log-stream.ts'

class FakeUsageLogStream extends EventTarget {
  started = false
  closed = false

  stream() {
    this.started = true
  }

  close() {
    this.closed = true
  }
}

describe('usage log stream', () => {
  test('refreshes after connecting and whenever the server reports a log', () => {
    const stream = new FakeUsageLogStream()
    let notifications = 0
    const unsubscribe = subscribeUsageLogStream(
      () => {
        notifications += 1
      },
      () => stream
    )

    assert.equal(stream.started, true)

    stream.dispatchEvent(new Event('ready'))
    stream.dispatchEvent(new Event('log'))

    assert.equal(notifications, 2)

    unsubscribe()
    assert.equal(stream.closed, true)
  })
})
