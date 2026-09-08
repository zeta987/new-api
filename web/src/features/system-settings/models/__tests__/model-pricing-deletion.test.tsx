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
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useForm } from 'react-hook-form'
import { afterEach, describe, expect, test, vi } from 'vitest'

import { ModelRatioForm } from '../model-ratio-form'

const defaultValues = {
  ModelPrice: '{"delete-a":1,"delete-b":2,"delete-c":3,"keep-model":4}',
  ModelRatio: '{}',
  CacheRatio: '{}',
  CreateCacheRatio: '{}',
  CompletionRatio: '{}',
  ImageRatio: '{}',
  AudioRatio: '{}',
  AudioCompletionRatio: '{}',
  ExposeRatioEnabled: false,
  BillingMode: '{}',
  BillingExpr: '{}',
}

function PricingFormFixture(props: {
  onSave: (values: typeof defaultValues) => Promise<void>
  initialValues?: typeof defaultValues
}) {
  const initialValues = props.initialValues ?? defaultValues
  const form = useForm({ defaultValues: initialValues })
  return (
    <ModelRatioForm
      form={form}
      savedValues={initialValues}
      onSave={props.onSave}
      onReset={() => undefined}
      isSaving={false}
      isResetting={false}
    />
  )
}

const queryClients: QueryClient[] = []

function renderPricingForm(initialValues = defaultValues) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { enabled: false, retry: false } },
  })
  queryClients.push(queryClient)
  queryClient.setQueryData(['pricing'], { data: [], vendors: [] })
  queryClient.setQueryData(['status'], { price: 1 })
  const onSave = vi
    .fn<(values: typeof defaultValues) => Promise<void>>()
    .mockResolvedValue(undefined)
  const user = userEvent.setup()

  render(
    <QueryClientProvider client={queryClient}>
      <PricingFormFixture onSave={onSave} initialValues={initialValues} />
    </QueryClientProvider>
  )
  return { user, onSave }
}

describe('model pricing deletion', () => {
  afterEach(() => {
    localStorage.clear()
    for (const client of queryClients) client.clear()
    queryClients.length = 0
  })

  test.each([
    {
      names: ['delete-a'],
      expected: { 'delete-b': 2, 'delete-c': 3, 'keep-model': 4 },
    },
    {
      names: ['delete-a', 'delete-b', 'delete-c'],
      expected: { 'keep-model': 4 },
    },
  ])(
    'saving after deleting $names excludes every deleted name',
    async ({ names, expected }) => {
      const { user, onSave } = renderPricingForm()
      await user.click(screen.getByText('keep-model'))

      for (const name of names) {
        const row = screen.getByRole('row', { name: new RegExp(name) })
        await user.click(within(row).getByRole('button', { name: 'Open menu' }))
        await user.click(
          await screen.findByRole('menuitem', { name: 'Delete' })
        )
        expect(
          screen.queryByRole('row', { name: new RegExp(name) })
        ).not.toBeInTheDocument()
      }

      await user.click(
        screen.getAllByRole('button', { name: 'Save model prices' })[0]
      )
      await waitFor(() => expect(onSave).toHaveBeenCalled())
      expect(JSON.parse(onSave.mock.calls[0][0].ModelPrice)).toEqual(expected)
      expect(screen.getByRole('textbox', { name: 'Model name' })).toHaveValue(
        'keep-model'
      )
    }
  )

  test('deleting every model without opening an editor still allows saving empty pricing', async () => {
    const { user, onSave } = renderPricingForm()
    for (const name of ['delete-a', 'delete-b', 'delete-c', 'keep-model']) {
      const row = screen.getByRole('row', { name: new RegExp(name) })
      await user.click(within(row).getByRole('button', { name: 'Open menu' }))
      await user.click(await screen.findByRole('menuitem', { name: 'Delete' }))
    }
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Save model prices' }))
    await waitFor(() => expect(onSave).toHaveBeenCalled())
    expect(JSON.parse(onSave.mock.calls[0][0].ModelPrice)).toEqual({})
    expect(
      screen.queryByRole('textbox', { name: 'Model name' })
    ).not.toBeInTheDocument()
  })

  test('deleting the currently edited model clears all its pricing fields when saved', async () => {
    const { user, onSave } = renderPricingForm({
      ...defaultValues,
      ModelPrice: '{}',
      ModelRatio: '{"delete-a":1}',
      CompletionRatio: '{"delete-a":2}',
      CacheRatio: '{"delete-a":0.1}',
      CreateCacheRatio: '{"delete-a":1.25}',
      ImageRatio: '{"delete-a":1}',
      AudioRatio: '{"delete-a":2}',
      AudioCompletionRatio: '{"delete-a":3}',
      BillingMode: '{"delete-a":"per-token"}',
      BillingExpr: '{"delete-a":"p * 2"}',
    })
    await user.click(screen.getByText('delete-a'))
    await user.click(screen.getByRole('button', { name: 'Open menu' }))
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }))
    await user.click(screen.getByRole('button', { name: 'Save model prices' }))
    await waitFor(() => expect(onSave).toHaveBeenCalled())
    for (const [field, value] of Object.entries(onSave.mock.calls[0][0])) {
      if (field !== 'ExposeRatioEnabled') {
        expect(JSON.parse(String(value))).toEqual({})
      }
    }
  })

  test('keyboard activation of Delete does not replace the editor with the deleted model', async () => {
    const { user, onSave } = renderPricingForm()
    await user.click(screen.getByText('keep-model'))
    const row = screen.getByRole('row', { name: /delete-a/ })
    await user.click(within(row).getByRole('button', { name: 'Open menu' }))
    await user.keyboard('{ArrowDown}{Enter}')
    expect(
      screen.queryByRole('row', { name: /delete-a/ })
    ).not.toBeInTheDocument()
    await user.click(
      screen.getAllByRole('button', { name: 'Save model prices' })[0]
    )
    await waitFor(() => expect(onSave).toHaveBeenCalled())
    expect(JSON.parse(onSave.mock.calls[0][0].ModelPrice)).toEqual({
      'delete-b': 2,
      'delete-c': 3,
      'keep-model': 4,
    })
  })
})
