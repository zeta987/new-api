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

import { subscribeUsageLogStream } from './log-stream'

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

    expect(stream.started).toBe(true)

    stream.dispatchEvent(new Event('ready'))
    stream.dispatchEvent(new Event('log'))

    expect(notifications).toBe(2)

    unsubscribe()
    expect(stream.closed).toBe(true)
  })
})
