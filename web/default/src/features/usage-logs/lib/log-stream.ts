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
