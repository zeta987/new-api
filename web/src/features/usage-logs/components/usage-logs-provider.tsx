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
/* eslint-disable react-refresh/only-export-components */
import { useQueryClient } from '@tanstack/react-query'
import {
  createContext,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from 'react'

import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import type { UsageLog } from '../data/schema'
import { createUsageLogsRefreshController } from '../lib/background-refresh'
import { subscribeUsageLogStream } from '../lib/log-stream'
import { subscribeUsageLogsChanged } from '../lib/refresh-events'
import type { ChannelAffinityInfo } from '../types'
import { DetailsDialog } from './dialogs/details-dialog'

export type LogsViewScope = 'all' | 'self'
export type LogsViewAccess = 'self' | 'admin' | 'root'

interface LogDetailsSelection {
  log: UsageLog
  isAdmin: boolean
  isRoot: boolean
}

export function resolveLogsViewAccess(
  role: number,
  viewScope: LogsViewScope
): LogsViewAccess {
  if (viewScope !== 'all' || role < ROLE.ADMIN) return 'self'
  return role === ROLE.SUPER_ADMIN ? 'root' : 'admin'
}

interface UsageLogsContextValue {
  showLogDetails: (selection: LogDetailsSelection) => void
  selectedUserId: number | null
  setSelectedUserId: (userId: number | null) => void
  userInfoDialogOpen: boolean
  setUserInfoDialogOpen: (open: boolean) => void
  affinityTarget: ChannelAffinityInfo | null
  setAffinityTarget: (target: ChannelAffinityInfo | null) => void
  affinityDialogOpen: boolean
  setAffinityDialogOpen: (open: boolean) => void
  sensitiveVisible: boolean
  setSensitiveVisible: (visible: boolean) => void
  viewScope: LogsViewScope
  setViewScope: (scope: LogsViewScope) => void
}

const UsageLogsContext = createContext<UsageLogsContextValue | undefined>(
  undefined
)

export function UsageLogsProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient()
  const accessToken = useAuthStore((state) => state.auth.accessToken)
  const [detailsSelection, showLogDetails] =
    useState<LogDetailsSelection | null>(null)
  const [selectedUserId, setSelectedUserId] = useState<number | null>(null)
  const [userInfoDialogOpen, setUserInfoDialogOpen] = useState(false)
  const [affinityTarget, setAffinityTarget] =
    useState<ChannelAffinityInfo | null>(null)
  const [affinityDialogOpen, setAffinityDialogOpen] = useState(false)
  const [sensitiveVisible, setSensitiveVisible] = useState(true)
  const [viewScope, setViewScope] = useState<LogsViewScope>('all')

  useEffect(() => {
    const refresh = createUsageLogsRefreshController(queryClient)
    const unsubscribeBrowserEvents = subscribeUsageLogsChanged(refresh.request)
    let unsubscribeServerEvents: (() => void) | undefined
    let connectionTimer: ReturnType<typeof setTimeout> | undefined

    const connectServerEvents = () => {
      connectionTimer = undefined
      if (
        document.visibilityState === 'hidden' ||
        !accessToken ||
        unsubscribeServerEvents
      ) {
        return
      }
      const delay = refresh.retryAfter()
      if (delay > 0) {
        connectionTimer = setTimeout(connectServerEvents, delay)
        return
      }
      unsubscribeServerEvents = subscribeUsageLogStream(
        refresh.request,
        undefined,
        refresh.defer
      )
    }

    const updateVisibility = () => {
      if (connectionTimer !== undefined) clearTimeout(connectionTimer)
      connectionTimer = undefined
      const hidden = document.visibilityState === 'hidden'
      refresh.setPaused(hidden)
      if (hidden) {
        unsubscribeServerEvents?.()
        unsubscribeServerEvents = undefined
      } else {
        connectServerEvents()
      }
    }
    updateVisibility()
    document.addEventListener('visibilitychange', updateVisibility)

    return () => {
      document.removeEventListener('visibilitychange', updateVisibility)
      if (connectionTimer !== undefined) clearTimeout(connectionTimer)
      unsubscribeBrowserEvents()
      unsubscribeServerEvents?.()
      refresh.dispose()
    }
  }, [accessToken, queryClient])

  return (
    <UsageLogsContext.Provider
      value={{
        showLogDetails,
        selectedUserId,
        setSelectedUserId,
        userInfoDialogOpen,
        setUserInfoDialogOpen,
        affinityTarget,
        setAffinityTarget,
        affinityDialogOpen,
        setAffinityDialogOpen,
        sensitiveVisible,
        setSensitiveVisible,
        viewScope,
        setViewScope,
      }}
    >
      {children}
      {detailsSelection && (
        <DetailsDialog
          log={detailsSelection.log}
          isAdmin={detailsSelection.isAdmin}
          isRoot={detailsSelection.isRoot}
          open
          onOpenChange={(open) => {
            if (!open) showLogDetails(null)
          }}
        />
      )}
    </UsageLogsContext.Provider>
  )
}

export function useUsageLogsContext() {
  const context = useContext(UsageLogsContext)
  if (!context) {
    throw new Error('useUsageLogsContext must be used within UsageLogsProvider')
  }
  return context
}

/**
 * Resolves the effective admin scope for usage logs: whether the current
 * user is allowed to view all users' logs (`canManageScope`), and whether
 * their current view preference (`viewScope`) has that scope active
 * (`isAdminView`). Data fetching and admin-only UI should key off
 * `isAdminView` rather than raw role, so an admin who switches to "only
 * mine" is treated exactly like a regular user for that view.
 */
export function useLogsViewScope() {
  const role = useAuthStore((state) => state.auth.user?.role ?? ROLE.GUEST)
  const { viewScope, setViewScope } = useUsageLogsContext()
  const canManageScope = role >= ROLE.ADMIN
  const viewAccess = resolveLogsViewAccess(role, viewScope)
  const isAdminView = viewAccess !== 'self'
  const isRootView = viewAccess === 'root'

  return {
    canManageScope,
    viewScope,
    setViewScope,
    isAdminView,
    isRootView,
    viewAccess,
  }
}
