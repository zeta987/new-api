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

import { getCommonHeaders } from '@/lib/api'

import { usageLogsRetryAfterDelay } from './background-refresh'

interface UsageLogStream {
  addEventListener(type: string, listener: (event: Event) => void): void
  stream(): void
  close(): void
  reconnectDelay?: number
  xhr?: Pick<XMLHttpRequest, 'getResponseHeader'> | null
}

type UsageLogStreamFactory = () => UsageLogStream

function createUsageLogStream(): UsageLogStream {
  return new SSE('/api/log/stream', {
    autoReconnect: true,
    reconnectDelay: 10_000,
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
  onRateLimit?: (delay: number) => void
): () => void {
  const source = createStream()
  source.addEventListener('ready', () => {
    source.reconnectDelay = 10_000
    listener()
  })
  source.addEventListener('log', listener)
  source.addEventListener('error', (event) => {
    if (!('responseCode' in event) || event.responseCode !== 429) return
    const delay = usageLogsRetryAfterDelay(
      source.xhr?.getResponseHeader('Retry-After')
    )
    source.reconnectDelay = Math.max(10_000, delay)
    onRateLimit?.(delay)
  })
  source.stream()

  return () => source.close()
}
