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
import { SSE } from 'sse.js'

import {
  getCommonHeaders,
  refreshAuthentication,
  type RefreshOutcome,
} from '@/lib/api'

import { usageLogsRetryAfterDelay } from './background-refresh'

interface UsageLogStream {
  addEventListener(type: string, listener: (event: Event) => void): void
  stream(): void
  close(): void
  reconnectDelay?: number
  xhr?: Pick<XMLHttpRequest, 'getResponseHeader'> | null
}

type UsageLogStreamFactory = () => UsageLogStream
type AuthRefresh = () => Promise<RefreshOutcome>

const RECONNECT_DELAY_MS = 10_000

function createUsageLogStream(): UsageLogStream {
  return new SSE('/api/log/stream', {
    autoReconnect: true,
    reconnectDelay: RECONNECT_DELAY_MS,
    start: false,
    headers: {
      ...getCommonHeaders(),
      Accept: 'text/event-stream',
    },
    method: 'GET',
  })
}

export function subscribeUsageLogStream(
  listener: () => void,
  createStream: UsageLogStreamFactory = createUsageLogStream,
  onRateLimit?: (delay: number) => void,
  refreshAuth: AuthRefresh = refreshAuthentication
): () => void {
  let disposed = false
  let refreshingAuth = false
  let source: UsageLogStream | undefined
  let retryTimer: ReturnType<typeof setTimeout> | undefined

  const clearRetryTimer = () => {
    if (retryTimer !== undefined) clearTimeout(retryTimer)
    retryTimer = undefined
  }

  const schedule = (callback: () => void, delay: number) => {
    clearRetryTimer()
    if (disposed) return
    retryTimer = setTimeout(() => {
      retryTimer = undefined
      callback()
    }, delay)
  }

  const recoverAuthentication = async () => {
    if (disposed || refreshingAuth) return
    refreshingAuth = true
    let outcome: RefreshOutcome
    try {
      outcome = await refreshAuth()
    } catch (error) {
      outcome = { kind: 'transient_error', error }
    }
    refreshingAuth = false
    if (disposed) return

    if (outcome.kind === 'authenticated') {
      schedule(connect, 0)
    } else if (outcome.kind === 'transient_error') {
      schedule(recoverAuthentication, RECONNECT_DELAY_MS)
    }
  }

  function connect() {
    if (disposed || source) return
    const nextSource = createStream()
    source = nextSource
    nextSource.addEventListener('ready', () => {
      if (source !== nextSource) return
      nextSource.reconnectDelay = RECONNECT_DELAY_MS
      listener()
    })
    nextSource.addEventListener('log', () => {
      if (source === nextSource) listener()
    })
    nextSource.addEventListener('error', (event) => {
      if (source !== nextSource || !('responseCode' in event)) return
      if (event.responseCode === 401) {
        source = undefined
        nextSource.close()
        void recoverAuthentication()
        return
      }
      if (event.responseCode !== 429) return
      const delay = usageLogsRetryAfterDelay(
        nextSource.xhr?.getResponseHeader('Retry-After')
      )
      nextSource.reconnectDelay = Math.max(RECONNECT_DELAY_MS, delay)
      onRateLimit?.(delay)
    })
    nextSource.stream()
  }

  connect()

  return () => {
    disposed = true
    clearRetryTimer()
    source?.close()
    source = undefined
  }
}
