<script lang="ts">
  import { onMount } from 'svelte'
  import {
    api,
    type APIKey,
    type ProviderConnection,
    type ProviderNode,
    type Settings
  } from '../api/client'
  import SummaryKpiCards from './analytics/SummaryKpiCards.svelte'
  import RequestDetailsTab from './analytics/RequestDetailsTab.svelte'
  import QuotaTrackerView from './QuotaTrackerView.svelte'
  import UsageChart from './UsageChart.svelte'
  import Card from '../lib/ui/Card.svelte'
  import Toggle from '../lib/ui/Toggle.svelte'
  import { fmtCost, timeAgo, type Period, type RecentRequestItem, type RequestDetailItem, type StatsData } from './analytics/types'
  import { PROVIDER_CATALOG } from '../lib/providers'

  interface Props {
    connections?: ProviderConnection[]
    providerNodes?: ProviderNode[]
    apiKeys?: APIKey[]
    settings?: Settings
    onRefresh?: () => void
  }

  let {
    connections = [],
    providerNodes = [],
    apiKeys: initialKeys = [],
    settings: initialSettings = {},
    onRefresh
  }: Props = $props()

  // Tab router
  type DashboardTab = 'overview' | 'requests' | 'quota'
  let activeSubTab = $state<DashboardTab>('overview')
  let period = $state<Period>('7d')

  // Overview stats & recent requests
  let stats = $state<StatsData>({})
  let recentRequests = $state<RecentRequestItem[]>([])

  // Request details tab state
  let details = $state<RequestDetailItem[]>([])
  let detailsTotal = $state(0)
  let detailsPage = $state(1)
  let detailsLoading = $state(false)

  // API Key manager state
  let keys = $state<APIKey[]>([])
  let keysLoading = $state(true)
  let requireApiKey = $state(false)
  let baseUrl = $state('/v1')
  let copiedId = $state<string | null>(null)
  let visibleKeyIds = $state<Set<string>>(new Set())
  let showCreateKeyModal = $state(false)
  let newKeyName = $state('')
  let isCreatingKey = $state(false)
  let createdKeySecret = $state<string | null>(null)
  let deleteKeyConfirm = $state<APIKey | null>(null)
  let pauseKeyConfirm = $state<APIKey | null>(null)

  // URL query param synchronization
  onMount(() => {
    if (typeof window !== 'undefined') {
      baseUrl = `${window.location.origin}/v1`
      const params = new URLSearchParams(window.location.search)
      const tabParam = params.get('tab')
      if (tabParam === 'requests' || tabParam === 'quota') {
        activeSubTab = tabParam
      }
    }
  })

  function handleTabChange(tab: DashboardTab) {
    activeSubTab = tab
    if (typeof window !== 'undefined') {
      const url = new URL(window.location.href)
      if (tab === 'overview') {
        url.searchParams.delete('tab')
      } else {
        url.searchParams.set('tab', tab)
      }
      window.history.replaceState({}, '', url.pathname + url.search)
    }
  }

  // Load overview stats
  async function loadOverviewStats(targetPeriod: Period) {
    try {
      const res = await api.getUsageStats(targetPeriod)
      if (res) {
        stats = res
        if (Array.isArray(res.recentRequests)) {
          recentRequests = res.recentRequests
        }
      }
    } catch (e) {
      console.error('Failed to load usage stats:', e)
    }
  }

  $effect(() => {
    loadOverviewStats(period)
  })

  // SSE for live recentRequests
  $effect(() => {
    let es: EventSource | null = null
    try {
      es = new EventSource('/api/usage/stream')
      es.onmessage = (e) => {
        try {
          const payload = JSON.parse(e.data)
          if (Array.isArray(payload.recentRequests) && payload.recentRequests.length > 0) {
            recentRequests = payload.recentRequests
          }
        } catch {}
      }
    } catch {}

    return () => {
      es?.close()
    }
  })

  // Load request details tab data
  async function loadDetails(page = 1) {
    detailsLoading = true
    try {
      const limit = 50
      const offset = (page - 1) * limit
      const res = await api.getRequestDetails(limit, offset)
      if (res && Array.isArray(res.details)) {
        details = res.details
        detailsTotal = res.total || 0
        detailsPage = page
      }
    } catch (err) {
      console.error('Failed to load request details:', err)
    } finally {
      detailsLoading = false
    }
  }

  $effect(() => {
    if (activeSubTab === 'requests') {
      loadDetails(detailsPage)
    }
  })

  // API Key management functions
  async function loadKeys() {
    keysLoading = true
    try {
      const res = await api.getApiKeys()
      keys = Array.isArray(res) ? res : []
    } catch {
      keys = []
    } finally {
      keysLoading = false
    }
  }

  $effect(() => {
    if (initialKeys && initialKeys.length > 0) {
      keys = initialKeys
      keysLoading = false
    } else {
      loadKeys()
    }
  })

  $effect(() => {
    if (initialSettings) {
      requireApiKey = !!initialSettings.requireApiKey
    }
  })

  async function handleCreateKey() {
    if (!newKeyName.trim() || isCreatingKey) return
    isCreatingKey = true
    try {
      const res = await api.createApiKey({ name: newKeyName.trim() })
      if (res && res.key) {
        createdKeySecret = res.key
        newKeyName = ''
        showCreateKeyModal = false
        await loadKeys()
        onRefresh?.()
      }
    } catch (err) {
      console.error('Failed to create key:', err)
    } finally {
      isCreatingKey = false
    }
  }

  async function handleDeleteKey(id: string) {
    try {
      await api.deleteApiKey(id)
      keys = keys.filter((k) => k.id !== id)
      deleteKeyConfirm = null
      onRefresh?.()
    } catch (err) {
      console.error('Failed to delete key:', err)
    }
  }

  async function handleToggleKey(key: APIKey, nextState: boolean) {
    try {
      await api.toggleApiKey(key.id)
      keys = keys.map((k) => (k.id === key.id ? { ...k, isActive: nextState ? 1 : 0 } : k))
      pauseKeyConfirm = null
      onRefresh?.()
    } catch (err) {
      console.error('Failed to toggle key:', err)
    }
  }

  async function handleRequireApiKey(next: boolean) {
    requireApiKey = next
    try {
      await api.updateSettings({ requireApiKey: next })
      onRefresh?.()
    } catch (err) {
      console.error('Failed to update requireApiKey:', err)
    }
  }

  function copyText(text: string, id: string) {
    try {
      navigator.clipboard.writeText(text)
      copiedId = id
      setTimeout(() => {
        if (copiedId === id) copiedId = null
      }, 2000)
    } catch {}
  }

  function maskKey(k?: string): string {
    if (!k || k.length <= 10) return k || ''
    return k.slice(0, 6) + '•'.repeat(Math.max(4, k.length - 10)) + k.slice(-4)
  }

  function toggleVisibility(id: string) {
    const next = new Set(visibleKeyIds)
    if (next.has(id)) next.delete(id)
    else next.add(id)
    visibleKeyIds = next
  }

  function getProviderDisplayName(providerId?: string): string {
    if (!providerId) return ''
    const node = providerNodes.find((n) => n.id === providerId)
    if (node) return node.name || node.prefix || node.id
    const cat = PROVIDER_CATALOG.find((p) => p.id === providerId || p.alias === providerId)
    if (cat?.name) return cat.name
    return providerId
  }

  function getStatusEmoji(conn: ProviderConnection): string {
    if (conn.isActive === 0) return '⏸️'
    if (conn.testStatus === 'error' || conn.errorCode) return '❌'
    if (conn.testStatus === 'ok') return '✅'
    if (conn.expiresAt && new Date(conn.expiresAt as string).getTime() < Date.now()) return '⚠️'
    return '✅'
  }

  function getAccountLabel(conn: ProviderConnection): string {
    if (conn.email) return conn.email
    if (conn.displayName) return conn.displayName
    if (conn.name) return conn.name.length > 20 ? conn.name.slice(0, 8) + '…' : conn.name
    return ''
  }

  const fmtReqNumber = (n?: number) => (n || 0).toLocaleString()
  const fmtCompact = (n?: number) => ((n || 0) >= 1000 ? `${((n || 0) / 1000).toFixed(1)}k` : String(n || 0))

  let sortedModels = $derived.by(() => {
    if (!stats.byModel) return []
    return Object.entries(stats.byModel)
      .sort(([, a], [, b]) => (b.requests || 0) - (a.requests || 0))
      .slice(0, 8)
  })
</script>

<div class="flex min-w-0 flex-col gap-6">
  <!-- Top Segmented Controls: Tab navigation + Period filter -->
  <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
    <div class="inline-flex p-1 bg-surface-2 border border-border-subtle rounded-xl w-full sm:w-auto">
      <button
        type="button"
        onclick={() => handleTabChange('overview')}
        class="flex-1 sm:flex-none px-4 py-1.5 rounded-lg text-xs font-semibold transition-all cursor-pointer {activeSubTab ===
        'overview'
          ? 'bg-surface text-text-main shadow-xs'
          : 'text-text-muted hover:text-text-main'}"
      >
        Overview
      </button>
      <button
        type="button"
        onclick={() => handleTabChange('requests')}
        class="flex-1 sm:flex-none px-4 py-1.5 rounded-lg text-xs font-semibold transition-all cursor-pointer {activeSubTab ===
        'requests'
          ? 'bg-surface text-text-main shadow-xs'
          : 'text-text-muted hover:text-text-main'}"
      >
        Requests
      </button>
      <button
        type="button"
        onclick={() => handleTabChange('quota')}
        class="flex-1 sm:flex-none px-4 py-1.5 rounded-lg text-xs font-semibold transition-all cursor-pointer {activeSubTab ===
        'quota'
          ? 'bg-surface text-text-main shadow-xs'
          : 'text-text-muted hover:text-text-main'}"
      >
        Quota
      </button>
    </div>

    {#if activeSubTab === 'overview'}
      <div class="inline-flex p-1 bg-surface-2 border border-border-subtle rounded-xl w-full sm:w-auto self-start sm:self-auto">
        {#each [{ id: 'today', label: 'Today' }, { id: '24h', label: '24h' }, { id: '7d', label: '7D' }, { id: '30d', label: '30D' }, { id: '60d', label: '60D' }] as p}
          <button
            type="button"
            onclick={() => (period = p.id as Period)}
            class="flex-1 sm:flex-none px-3 py-1 rounded-lg text-xs font-medium transition-all cursor-pointer {period ===
            p.id
              ? 'bg-surface text-text-main shadow-xs font-semibold'
              : 'text-text-muted hover:text-text-main'}"
          >
            {p.label}
          </button>
        {/each}
      </div>
    {/if}
  </div>

  <!-- Tab 1: Overview -->
  {#if activeSubTab === 'overview'}
    <div class="flex min-w-0 flex-col gap-6 animate-in fade-in duration-200">
      <!-- 1. KPI Cards -->
      <SummaryKpiCards {stats} />

      <!-- 2. Usage chart + Recent requests -->
      <div class="grid grid-cols-1 gap-4 lg:grid-cols-[minmax(0,2fr)_minmax(280px,1fr)]">
        <UsageChart {period} class="h-full" style="height: 480px;" />

        <!-- Recent Requests Card (Matches exact upstream DOM/CSS) -->
        <div
          class="bg-surface border border-border-subtle rounded-brand-lg shadow-(--shadow-soft) p-4 flex min-w-0 flex-col overflow-hidden"
          style="height: 480px;"
        >
          <div class="px-1 py-2 border-b border-border shrink-0 flex items-center justify-between">
            <span class="text-xs font-semibold text-text-muted uppercase tracking-wide">
              Recent Requests
            </span>
            <span class="size-2 rounded-full bg-success animate-pulse" title="Live stream active"></span>
          </div>

          {#if !recentRequests.length}
            <div class="flex-1 flex items-center justify-center text-text-muted text-sm">
              No requests recorded yet.
            </div>
          {:else}
            <div class="flex-1 overflow-y-auto custom-scrollbar">
              <table class="w-full min-w-75 border-collapse text-xs">
                <thead class="sticky top-0 bg-surface z-10">
                  <tr class="border-b border-border">
                    <th class="py-1.5 text-left font-semibold text-text-muted w-2"></th>
                    <th class="py-1.5 text-left font-semibold text-text-muted">Model</th>
                    <th class="py-1.5 text-right font-semibold text-text-muted whitespace-nowrap">In / Out</th>
                    <th class="py-1.5 text-right font-semibold text-text-muted">When</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-border/50">
                  {#each recentRequests as r, i (i)}
                    {@const ok = !r.status || r.status === 'ok' || r.status === 'success'}
                    <tr class="hover:bg-surface-2 transition-colors">
                      <td class="py-1.5">
                        <span
                          class="block w-1.5 h-1.5 rounded-full {ok ? 'bg-success' : 'bg-red-500'}"
                        ></span>
                      </td>
                      <td class="py-1.5 font-mono truncate max-w-30 text-text-main" title={r.model}>
                        {r.model}
                      </td>
                      <td class="py-1.5 text-right whitespace-nowrap">
                        <span class="text-primary">{fmtCompact(r.promptTokens)}↑</span>
                        <span class="text-success">{fmtCompact(r.completionTokens)}↓</span>
                      </td>
                      <td class="py-1.5 text-right text-text-muted whitespace-nowrap">
                        {r.timestamp ? timeAgo(r.timestamp) : 'just now'}
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          {/if}
        </div>
      </div>

      <!-- 3. Top models -->
      {#if sortedModels.length > 0}
        <Card title="Top Models" icon="psychology">
          <div class="flex flex-col -mx-1">
            <div class="flex items-center gap-2 px-1 py-1.5 text-xs font-medium text-text-muted uppercase tracking-wider border-b border-border-subtle">
              <span class="flex-1 min-w-0">Model</span>
              <span class="w-16 text-right shrink-0">Requests</span>
              <span class="w-28 text-right shrink-0">Tokens (In / Out)</span>
              <span class="w-20 text-right shrink-0">Cost</span>
            </div>
            {#each sortedModels as [key, data] (key)}
              <div class="flex items-center gap-2 px-1 py-2.5 rounded-lg hover:bg-surface-2 transition-colors">
                <div class="flex-1 min-w-0">
                  <span class="text-sm font-medium truncate block text-text-main">
                    {data.rawModel || key}
                  </span>
                  {#if data.provider}
                    <span class="text-xs text-text-muted capitalize">{getProviderDisplayName(data.provider)}</span>
                  {/if}
                </div>
                <span class="w-16 text-right text-sm tabular-nums text-text-main shrink-0">
                  {fmtReqNumber(data.requests)}
                </span>
                <span class="w-28 text-right text-xs text-text-muted tabular-nums shrink-0">
                  {fmtCompact(data.promptTokens)} / {fmtCompact(data.completionTokens)}
                </span>
                <span class="w-20 text-right text-xs font-medium text-warning tabular-nums shrink-0">
                  {fmtCost(data.cost)}
                </span>
              </div>
            {/each}
          </div>
        </Card>
      {/if}

      <!-- 4. Two-column bottom grid: API Keys | Quota summary -->
      <div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <!-- Left: API Key Manager -->
        <Card title="API Keys" icon="vpn_key">
          {#snippet action()}
            <button
              type="button"
              onclick={() => (showCreateKeyModal = true)}
              class="flex items-center gap-1 px-2.5 py-1 rounded-lg bg-primary hover:bg-primary-hover text-white text-xs font-medium transition-colors cursor-pointer"
            >
              <span class="material-symbols-outlined text-[16px]">add</span>
              Create
            </button>
          {/snippet}

          <!-- Endpoint URL -->
          <div class="flex items-center gap-2 mb-4">
            <code class="flex-1 min-w-0 truncate rounded-lg bg-surface-2 px-3 py-2 text-xs font-mono text-text-main border border-border-subtle">
              {baseUrl}
            </code>
            <button
              type="button"
              onclick={() => copyText(baseUrl, 'endpoint_url')}
              class="shrink-0 p-2 hover:bg-surface-2 rounded-lg text-text-muted hover:text-primary transition-all cursor-pointer"
              title="Copy endpoint URL"
            >
              <span class="material-symbols-outlined text-[18px]">
                {copiedId === 'endpoint_url' ? 'check' : 'content_copy'}
              </span>
            </button>
          </div>

          <!-- Require API key toggle -->
          <div class="flex items-center justify-between pb-3 mb-3 border-b border-border-subtle">
            <div>
              <p class="text-sm font-medium text-text-main">Require API key</p>
              <p class="text-xs text-text-muted">Reject unauthenticated requests</p>
            </div>
            <Toggle checked={requireApiKey} onChange={(val) => handleRequireApiKey(val)} />
          </div>

          <!-- Keys list -->
          {#if keysLoading}
            <div class="flex items-center justify-center py-6 text-text-muted text-sm gap-2">
              <span class="w-4 h-4 border-2 border-primary border-t-transparent rounded-full animate-spin"></span>
              <span>Loading keys...</span>
            </div>
          {:else if keys.length === 0}
            <div class="text-center py-6">
              <p class="text-sm text-text-muted mb-3">No API keys yet</p>
              <button
                type="button"
                onclick={() => (showCreateKeyModal = true)}
                class="px-3 py-1.5 rounded-lg bg-primary text-white text-xs font-medium cursor-pointer"
              >
                Create Key
              </button>
            </div>
          {:else}
            <div class="flex flex-col -mx-1 divide-y divide-border/30">
              {#each keys as key (key.id)}
                <div
                  class="group flex items-center justify-between gap-3 px-1 py-2.5 rounded-lg hover:bg-surface-2 transition-colors {key.isActive === 0
                    ? 'opacity-50'
                    : ''}"
                >
                  <div class="flex-1 min-w-0">
                    <div class="flex items-center gap-2">
                      <span class="text-sm font-medium text-text-main truncate">{key.name}</span>
                      {#if key.isActive === 0}
                        <span class="text-[10px] font-medium px-1.5 py-0.5 rounded bg-orange-500/10 text-orange-500">
                          Paused
                        </span>
                      {/if}
                    </div>
                    <div class="flex items-center gap-1.5 mt-0.5">
                      <code class="text-xs text-text-muted font-mono">
                        {visibleKeyIds.has(key.id) ? key.key : maskKey(key.key)}
                      </code>
                      <button
                        type="button"
                        onclick={() => toggleVisibility(key.id)}
                        class="p-0.5 hover:text-primary text-text-muted transition-colors cursor-pointer"
                        title={visibleKeyIds.has(key.id) ? 'Hide' : 'Show'}
                      >
                        <span class="material-symbols-outlined text-[13px]">
                          {visibleKeyIds.has(key.id) ? 'visibility_off' : 'visibility'}
                        </span>
                      </button>
                      <button
                        type="button"
                        onclick={() => copyText(key.key, key.id)}
                        class="p-0.5 hover:text-primary text-text-muted transition-colors cursor-pointer"
                        title="Copy key"
                      >
                        <span class="material-symbols-outlined text-[13px]">
                          {copiedId === key.id ? 'check' : 'content_copy'}
                        </span>
                      </button>
                    </div>
                  </div>

                  <div class="flex items-center gap-2 shrink-0">
                    <Toggle
                      size="sm"
                      checked={key.isActive === 1}
                      onChange={(next) => {
                        if (key.isActive === 1 && !next) {
                          pauseKeyConfirm = key
                        } else {
                          handleToggleKey(key, next)
                        }
                      }}
                    />
                    <button
                      type="button"
                      onclick={() => (deleteKeyConfirm = key)}
                      class="p-1.5 rounded text-text-muted hover:text-red-500 hover:bg-red-500/10 opacity-0 group-hover:opacity-100 transition-all cursor-pointer"
                      title="Delete key"
                    >
                      <span class="material-symbols-outlined text-[16px]">delete</span>
                    </button>
                  </div>
                </div>
              {/each}
            </div>
          {/if}
        </Card>

        <!-- Right: Compact Quota Summary -->
        <Card title="Provider Connections" icon="dns">
          {#snippet action()}
            <button
              type="button"
              onclick={() => handleTabChange('quota')}
              class="text-primary hover:underline text-xs font-medium cursor-pointer"
            >
              View All →
            </button>
          {/snippet}

          {#if connections.length === 0}
            <p class="text-sm text-text-muted py-6 text-center">No provider connections yet</p>
          {:else}
            <div class="flex flex-col gap-1 -mx-1">
              {#each connections.slice(0, 6) as conn (conn.id)}
                <div
                  class="flex items-center justify-between rounded-lg px-2 py-2 hover:bg-surface-2 transition-colors"
                >
                  <div class="flex items-center gap-2 min-w-0">
                    <span class="text-sm">{getStatusEmoji(conn)}</span>
                    <span class="text-sm font-medium capitalize text-text-main truncate">
                      {getProviderDisplayName(conn.provider)}
                    </span>
                  </div>
                  <span class="text-xs text-text-muted truncate ml-2">
                    {getAccountLabel(conn)}
                  </span>
                </div>
              {/each}
            </div>
          {/if}
        </Card>
      </div>
    </div>
  {:else if activeSubTab === 'requests'}
    <!-- Tab 2: Requests Details -->
    <div class="animate-in fade-in duration-200">
      <RequestDetailsTab
        {details}
        {detailsTotal}
        {detailsPage}
        {detailsLoading}
        {providerNodes}
        onPageChange={(p) => loadDetails(p)}
        onRefresh={() => loadDetails(detailsPage)}
      />
    </div>
  {:else if activeSubTab === 'quota'}
    <!-- Tab 3: Quota Tracker Table -->
    <div class="animate-in fade-in duration-200">
      <QuotaTrackerView {connections} />
    </div>
  {/if}
</div>

<!-- Modal: Create Key -->
{#if showCreateKeyModal}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4">
    <div
      class="absolute inset-0 bg-black/40 backdrop-blur-sm"
      onclick={() => (showCreateKeyModal = false)}
      onkeydown={(e) => e.key === 'Escape' && (showCreateKeyModal = false)}
      role="button"
      tabindex="-1"
      aria-label="Close background"
    ></div>

    <div class="relative w-full max-w-sm bg-surface border border-border-subtle rounded-2xl shadow-2xl p-6 flex flex-col gap-4 z-10 animate-in fade-in zoom-in-95">
      <h3 class="text-base font-semibold text-text-main">Create API Key</h3>

      <div class="flex flex-col gap-1.5">
        <label for="createKeyNameInput" class="text-xs font-medium text-text-muted">Key Name</label>
        <input
          id="createKeyNameInput"
          type="text"
          bind:value={newKeyName}
          placeholder="Production Key"
          class="w-full px-3 py-2 rounded-lg bg-surface-2 border border-border-subtle text-sm text-text-main focus:outline-none focus:border-primary"
        />
      </div>

      <div class="flex justify-end gap-2 pt-2">
        <button
          type="button"
          onclick={() => {
            showCreateKeyModal = false
            newKeyName = ''
          }}
          class="px-4 py-2 text-sm rounded-lg bg-surface-2 hover:bg-surface-3 text-text-main font-medium transition-colors cursor-pointer"
        >
          Cancel
        </button>
        <button
          type="button"
          disabled={!newKeyName.trim() || isCreatingKey}
          onclick={handleCreateKey}
          class="px-4 py-2 text-sm rounded-lg bg-primary hover:bg-primary-hover text-white font-medium transition-colors cursor-pointer disabled:opacity-50"
        >
          {isCreatingKey ? 'Creating...' : 'Create'}
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Modal: Newly Created Key Secret Alert -->
{#if createdKeySecret}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4">
    <div
      class="absolute inset-0 bg-black/40 backdrop-blur-sm"
      onclick={() => (createdKeySecret = null)}
      onkeydown={(e) => e.key === 'Escape' && (createdKeySecret = null)}
      role="button"
      tabindex="-1"
      aria-label="Close background"
    ></div>

    <div class="relative w-full max-w-md bg-surface border border-border-subtle rounded-2xl shadow-2xl p-6 flex flex-col gap-4 z-10 animate-in fade-in zoom-in-95">
      <h3 class="text-base font-semibold text-text-main">API Key Created</h3>

      <div class="bg-amber-500/10 border border-amber-500/30 rounded-lg p-3 text-amber-700 dark:text-amber-300 text-xs">
        <p class="font-semibold mb-0.5">Save this key now!</p>
        <p>This is the only time you will be able to see the full secret key.</p>
      </div>

      <div class="flex items-center gap-2">
        <input
          type="text"
          readonly
          value={createdKeySecret}
          class="flex-1 font-mono text-xs px-3 py-2 rounded-lg bg-surface-2 border border-border-subtle text-text-main select-all"
        />
        <button
          type="button"
          onclick={() => copyText(createdKeySecret || '', 'created_secret')}
          class="px-3 py-2 rounded-lg bg-surface-2 hover:bg-surface-3 border border-border-subtle text-xs font-semibold text-text-main flex items-center gap-1 cursor-pointer"
        >
          <span class="material-symbols-outlined text-[16px]">
            {copiedId === 'created_secret' ? 'check' : 'content_copy'}
          </span>
          {copiedId === 'created_secret' ? 'Copied!' : 'Copy'}
        </button>
      </div>

      <div class="flex justify-end pt-2">
        <button
          type="button"
          onclick={() => (createdKeySecret = null)}
          class="w-full py-2 text-sm rounded-lg bg-primary hover:bg-primary-hover text-white font-medium transition-colors cursor-pointer"
        >
          Done
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Modal: Confirm Delete Key -->
{#if deleteKeyConfirm}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4">
    <div
      class="absolute inset-0 bg-black/40 backdrop-blur-sm"
      onclick={() => (deleteKeyConfirm = null)}
      onkeydown={(e) => e.key === 'Escape' && (deleteKeyConfirm = null)}
      role="button"
      tabindex="-1"
      aria-label="Close background"
    ></div>

    <div class="relative w-full max-w-sm bg-surface border border-border-subtle rounded-2xl shadow-2xl p-6 flex flex-col gap-4 z-10 animate-in fade-in zoom-in-95">
      <h3 class="text-base font-semibold text-text-main">Delete API Key</h3>
      <p class="text-xs text-text-muted">
        Are you sure you want to delete <strong class="text-text-main">{deleteKeyConfirm.name}</strong>? This action cannot be undone.
      </p>

      <div class="flex justify-end gap-2 pt-2">
        <button
          type="button"
          onclick={() => (deleteKeyConfirm = null)}
          class="px-4 py-2 text-sm rounded-lg bg-surface-2 hover:bg-surface-3 text-text-main font-medium transition-colors cursor-pointer"
        >
          Cancel
        </button>
        <button
          type="button"
          onclick={() => deleteKeyConfirm && handleDeleteKey(deleteKeyConfirm.id)}
          class="px-4 py-2 text-sm rounded-lg bg-red-600 hover:bg-red-700 text-white font-medium transition-colors cursor-pointer"
        >
          Delete
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Modal: Confirm Pause Key -->
{#if pauseKeyConfirm}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4">
    <div
      class="absolute inset-0 bg-black/40 backdrop-blur-sm"
      onclick={() => (pauseKeyConfirm = null)}
      onkeydown={(e) => e.key === 'Escape' && (pauseKeyConfirm = null)}
      role="button"
      tabindex="-1"
      aria-label="Close background"
    ></div>

    <div class="relative w-full max-w-sm bg-surface border border-border-subtle rounded-2xl shadow-2xl p-6 flex flex-col gap-4 z-10 animate-in fade-in zoom-in-95">
      <h3 class="text-base font-semibold text-text-main">Pause API Key</h3>
      <p class="text-xs text-text-muted">
        Pause <strong class="text-text-main">{pauseKeyConfirm.name}</strong>? Requests using this key will be rejected immediately.
      </p>

      <div class="flex justify-end gap-2 pt-2">
        <button
          type="button"
          onclick={() => (pauseKeyConfirm = null)}
          class="px-4 py-2 text-sm rounded-lg bg-surface-2 hover:bg-surface-3 text-text-main font-medium transition-colors cursor-pointer"
        >
          Cancel
        </button>
        <button
          type="button"
          onclick={() => pauseKeyConfirm && handleToggleKey(pauseKeyConfirm, false)}
          class="px-4 py-2 text-sm rounded-lg bg-amber-600 hover:bg-amber-700 text-white font-medium transition-colors cursor-pointer"
        >
          Pause Key
        </button>
      </div>
    </div>
  </div>
{/if}
