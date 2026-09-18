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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { getCoreRowModel, useReactTable } from '@tanstack/react-table'
import { cleanup, render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { DataTablePage } from '@/components/data-table'

import { usageLogSchema, type UsageLog } from '../../data/schema'
import { useCommonLogsColumns } from '../columns/common-logs-columns'
import { UsageLogsProvider } from '../usage-logs-provider'

// Decorative icon assets use directory imports that Node cannot load as ESM.
vi.mock('@lobehub/icons', () => ({}))

function LogTable(props: { logs: UsageLog[]; fetching: boolean }) {
  const columns = useCommonLogsColumns(false, false).filter(
    (column) => 'accessorKey' in column && column.accessorKey === 'content'
  )
  const table = useReactTable({
    data: props.logs,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getRowId: (row) => String(row.id),
  })
  return (
    <DataTablePage
      table={table}
      columns={columns}
      isFetching={props.fetching}
      interactiveWhileFetching
      showPagination={false}
      fixedHeight={false}
    />
  )
}

const firstLog = usageLogSchema.parse({
  id: 1,
  user_id: 1,
  created_at: 1,
  type: 3,
  content: 'original audit entry',
})
const secondLog = usageLogSchema.parse({
  id: 2,
  user_id: 1,
  created_at: 2,
  type: 3,
  content: 'new audit entry',
})

describe('usage log details during background updates', () => {
  let queryClient: QueryClient
  let stylesheet: HTMLStyleElement

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false, staleTime: Infinity } },
    })
    queryClient.setQueryData(['status'], {})
    // jsdom does not load the app's Tailwind stylesheet. Preserve pointer semantics.
    stylesheet = document.createElement('style')
    stylesheet.textContent = '.pointer-events-none { pointer-events: none; }'
    document.head.append(stylesheet)
  })

  afterEach(() => {
    cleanup()
    queryClient.clear()
    stylesheet.remove()
  })

  function page(logs: UsageLog[], fetching: boolean) {
    return (
      <QueryClientProvider client={queryClient}>
        <UsageLogsProvider>
          <LogTable logs={logs} fetching={fetching} />
        </UsageLogsProvider>
      </QueryClientProvider>
    )
  }

  test('allows opening details while the existing rows are refreshing', async () => {
    const user = userEvent.setup()
    render(page([firstLog], true))

    await user.click(screen.getByTitle('Click to view full details'))

    const dialog = screen.getByRole('dialog')
    expect(within(dialog).getByText('original audit entry')).toBeInTheDocument()
  })

  test('keeps the selected log visible when a refresh replaces its table row', async () => {
    const user = userEvent.setup()
    const view = render(page([firstLog], false))
    await user.click(screen.getByTitle('Click to view full details'))
    expect(screen.getByRole('dialog')).toBeInTheDocument()

    view.rerender(page([secondLog], false))

    const dialog = screen.getByRole('dialog')
    expect(within(dialog).getByText('original audit entry')).toBeInTheDocument()
    expect(
      within(dialog).queryByText('new audit entry')
    ).not.toBeInTheDocument()
  })
})
