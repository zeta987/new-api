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

interface UsageLogStream {
  addEventListener(type: string, listener: () => void): void
  stream(): void
  close(): void
}

type UsageLogStreamFactory = () => UsageLogStream

function createUsageLogStream(): UsageLogStream {
  return new SSE('/api/log/stream', {
    autoReconnect: true,
    headers: {
      ...getCommonHeaders(),
      Accept: 'text/event-stream',
    },
    method: 'GET',
  })
}

export function subscribeUsageLogStream(
  listener: () => void,
  createStream: UsageLogStreamFactory = createUsageLogStream
): () => void {
  const source = createStream()
  source.addEventListener('ready', listener)
  source.addEventListener('log', listener)
  source.stream()

  return () => source.close()
}
