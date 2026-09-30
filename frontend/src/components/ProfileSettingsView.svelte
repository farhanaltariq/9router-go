<script lang="ts">
  import { onMount } from 'svelte'
  import Card from '../lib/ui/Card.svelte'
  import Toggle from '../lib/ui/Toggle.svelte'
  import Modal from '../lib/ui/Modal.svelte'
  import Button from '../lib/ui/Button.svelte'
  import { api, type Settings } from '../api/client'

  interface Props {
    settings?: Settings
    onRefresh?: () => void
  }

  let {
    settings = {},
    onRefresh
  }: Props = $props()

  // Theme state ('light' | 'dark' | 'system')
  let currentTheme = $state<'light' | 'dark' | 'system'>('system')

  function initTheme() {
    if (typeof window !== 'undefined') {
      const stored = (localStorage.getItem('9router-theme') || localStorage.getItem('theme')) as 'light' | 'dark' | 'system' | null
      if (stored === 'light' || stored === 'dark' || stored === 'system') {
        currentTheme = stored
      }
    }
  }

  function setTheme(t: 'light' | 'dark' | 'system') {
    currentTheme = t
    localStorage.setItem('9router-theme', t)
    localStorage.setItem('theme', t)
    if (t === 'system') {
      const dark = window.matchMedia('(prefers-color-scheme: dark)').matches
      document.documentElement.classList.toggle('dark', dark)
      document.documentElement.classList.toggle('light', !dark)
    } else {
      document.documentElement.classList.toggle('dark', t === 'dark')
      document.documentElement.classList.toggle('light', t === 'light')
    }
  }

  // Database Backup / Restore State
  let isDownloadingBackup = $state(false)
  let isImportingBackup = $state(false)
  let fileInput: HTMLInputElement | null = $state(null)
  let dbStatus = $state<{ type: 'success' | 'error' | ''; message: string }>({ type: '', message: '' })
  let dbAuthModalOpen = $state(false)
  let dbAuthMode = $state<'export' | 'import'>('export')
  let dbAuthPassword = $state('')
  let pendingImportFile = $state<File | null>(null)

  // Security / Password State
  let requireLogin = $state(false)
  let currentPassword = $state('')
  let newPassword = $state('')
  let confirmNewPassword = $state('')
  let isUpdatingPassword = $state(false)
  let passStatus = $state<{ type: 'success' | 'error' | ''; message: string }>({ type: '', message: '' })

  // Routing Strategy State
  let fallbackStrategy = $state('failover')
  let stickyRoundRobinLimit = $state(3)
  let comboStrategy = $state('first-model')
  let comboStickyRoundRobinLimit = $state(1)

  // Observability State
  let enableObservability = $state(false)

  $effect(() => {
    if (settings) {
      requireLogin = !!settings.requireLogin
      enableObservability = !!settings.enableObservability
      if (typeof settings.sessionTimeout === 'string') {}
      if (typeof settings.fallbackStrategy === 'string') fallbackStrategy = settings.fallbackStrategy
      if (typeof settings.stickyRoundRobinLimit === 'number') stickyRoundRobinLimit = settings.stickyRoundRobinLimit
      if (typeof settings.comboStrategy === 'string') comboStrategy = settings.comboStrategy
      if (typeof settings.comboStickyRoundRobinLimit === 'number') comboStickyRoundRobinLimit = settings.comboStickyRoundRobinLimit
    }
  })

  onMount(() => {
    initTheme()
  })

  // -------------------------------------------------------------
  // DATABASE BACKUP & RESTORE
  // -------------------------------------------------------------
  function handleDownloadBackupClick() {
    dbStatus = { type: '', message: '' }
    dbAuthMode = 'export'
    dbAuthPassword = ''
    dbAuthModalOpen = true
  }

  async function executeDownloadBackup(password?: string) {
    isDownloadingBackup = true
    dbStatus = { type: '', message: '' }
    try {
      const headers: Record<string, string> = {}
      if (password) headers['x-9r-password'] = password

      const res = await fetch('/api/settings/database', { headers })
      const data = await res.json().catch(() => ({}))
      if (!res.ok) {
        const errorMsg =
          typeof data?.error === 'object' && data.error?.message
            ? data.error.message
            : typeof data?.error === 'string'
              ? data.error
              : data?.message || `Export failed with status ${res.status}`
        throw new Error(errorMsg)
      }

      const content = JSON.stringify(data, null, 2)
      const blob = new Blob([content], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      const stamp = new Date().toISOString().replace(/[.:]/g, '-')
      a.href = url
      a.download = `9router-backup-${stamp}.json`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)

      dbStatus = { type: 'success', message: 'Database backup downloaded' }
    } catch (err) {
      dbStatus = { type: 'error', message: err instanceof Error ? err.message : String(err) }
    } finally {
      isDownloadingBackup = false
    }
  }

  function handleFileSelected(event: Event) {
    const target = event.target as HTMLInputElement
    const file = target.files?.[0]
    if (target) target.value = ''
    if (!file) return

    pendingImportFile = file
    dbStatus = { type: '', message: '' }
    dbAuthMode = 'import'
    dbAuthPassword = ''
    dbAuthModalOpen = true
  }

  async function executeImportBackup(password?: string) {
    if (!pendingImportFile) return
    isImportingBackup = true
    dbStatus = { type: '', message: '' }
    try {
      const raw = await pendingImportFile.text()
      let payload: Record<string, any>
      try {
        payload = JSON.parse(raw)
      } catch {
        throw new Error('Invalid JSON backup file')
      }

      if (password) {
        payload.password = password
      }

      const res = await fetch('/api/settings/database', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })

      const data = await res.json().catch(() => ({}))
      if (!res.ok) {
        const errorMsg =
          typeof data?.error === 'object' && data.error?.message
            ? data.error.message
            : typeof data?.error === 'string'
              ? data.error
              : data?.message || `Import failed with status ${res.status}`
        throw new Error(errorMsg)
      }

      dbStatus = { type: 'success', message: 'Database backup imported successfully! Reloading...' }
      if (onRefresh) {
        await onRefresh()
      }
      setTimeout(() => {
        window.location.reload()
      }, 1000)
    } catch (err) {
      dbStatus = { type: 'error', message: err instanceof Error ? err.message : String(err) }
    } finally {
      isImportingBackup = false
      pendingImportFile = null
    }
  }

  async function handleDbAuthConfirm() {
    const mode = dbAuthMode
    const pwd = dbAuthPassword
    dbAuthModalOpen = false
    if (mode === 'export') {
      await executeDownloadBackup(pwd)
    } else {
      await executeImportBackup(pwd)
    }
  }

  // -------------------------------------------------------------
  // SECURITY & PASSWORD
  // -------------------------------------------------------------
  async function updateRequireLogin(newVal: boolean) {
    requireLogin = newVal
    try {
      await api.updateSettings({ requireLogin: newVal })
      if (onRefresh) onRefresh()
    } catch (err) {
      alert(`Failed to update require login setting: ${err instanceof Error ? err.message : String(err)}`)
    }
  }

  async function handlePasswordChange(e: Event) {
    e.preventDefault()
    passStatus = { type: '', message: '' }

    if (newPassword.length < 6) {
      passStatus = { type: 'error', message: 'New password must be at least 6 characters' }
      return
    }
    if (newPassword !== confirmNewPassword) {
      passStatus = { type: 'error', message: 'New passwords do not match' }
      return
    }

    isUpdatingPassword = true
    try {
      await api.updateSettings({
        currentPassword,
        newPassword,
      })
      passStatus = { type: 'success', message: 'Password updated successfully!' }
      currentPassword = ''
      newPassword = ''
      confirmNewPassword = ''
      if (onRefresh) onRefresh()
    } catch (err) {
      passStatus = { type: 'error', message: err instanceof Error ? err.message : 'Failed to update password' }
    } finally {
      isUpdatingPassword = false
    }
  }

  // -------------------------------------------------------------
  // ROUTING STRATEGY
  // -------------------------------------------------------------
  async function updateFallbackStrategy(strat: string) {
    fallbackStrategy = strat
    try {
      await api.updateSettings({ fallbackStrategy: strat })
      if (onRefresh) onRefresh()
    } catch (err) {
      console.error('Failed to update fallback strategy', err)
    }
  }

  async function updateStickyLimit(limitStr: string) {
    const lim = parseInt(limitStr, 10) || 1
    stickyRoundRobinLimit = lim
    try {
      await api.updateSettings({ stickyRoundRobinLimit: lim })
      if (onRefresh) onRefresh()
    } catch (err) {
      console.error('Failed to update sticky limit', err)
    }
  }

  async function updateComboStrategy(strat: string) {
    comboStrategy = strat
    try {
      await api.updateSettings({ comboStrategy: strat })
      if (onRefresh) onRefresh()
    } catch (err) {
      console.error('Failed to update combo strategy', err)
    }
  }

  async function updateComboStickyLimit(limitStr: string) {
    const lim = parseInt(limitStr, 10) || 1
    comboStickyRoundRobinLimit = lim
    try {
      await api.updateSettings({ comboStickyRoundRobinLimit: lim })
      if (onRefresh) onRefresh()
    } catch (err) {
      console.error('Failed to update combo sticky limit', err)
    }
  }

  // -------------------------------------------------------------
  // -------------------------------------------------------------
  // OBSERVABILITY
  // -------------------------------------------------------------
  async function updateObservability(enabled: boolean) {
    enableObservability = enabled
    try {
      await api.updateSettings({ enableObservability: enabled })
      if (onRefresh) onRefresh()
    } catch (err) {
      console.error('Failed to update observability', err)
    }
  }

  // -------------------------------------------------------------
  // LOGOUT
  // -------------------------------------------------------------
  async function handleLogout() {
    try {
      await api.logout()
      window.location.assign('/login')
    } catch {
      window.location.assign('/login')
    }
  }
</script>

<div class="max-w-2xl mx-auto px-4 sm:px-0">
  <div class="flex flex-col gap-6">

    <!-- CARD 1: LOCAL MODE & DATABASE -->
    <Card>
      <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-4">
        <div class="flex items-center gap-3 sm:gap-4">
          <div class="size-10 sm:size-12 rounded-lg bg-green-500/10 text-green-500 flex items-center justify-center shrink-0">
            <span class="material-symbols-outlined text-xl sm:text-2xl">computer</span>
          </div>
          <div>
            <h2 class="text-lg sm:text-xl font-semibold text-text-main">Local Mode</h2>
            <p class="text-sm text-text-muted">Running on your machine</p>
          </div>
        </div>

        <!-- Theme pills -->
        <div class="inline-flex p-1 rounded-lg bg-black/5 dark:bg-white/5 w-full sm:w-auto border border-border">
          <button
            type="button"
            onclick={() => setTheme('light')}
            class="flex items-center justify-center gap-1 sm:gap-1.5 px-2 sm:px-3 py-1.5 rounded-md font-medium text-xs sm:text-sm transition-all flex-1 sm:flex-initial cursor-pointer {currentTheme === 'light' ? 'bg-surface text-text-main shadow-sm' : 'text-text-muted hover:text-text-main'}"
          >
            <span class="material-symbols-outlined text-[18px]">light_mode</span>
            <span>Light</span>
          </button>
          <button
            type="button"
            onclick={() => setTheme('dark')}
            class="flex items-center justify-center gap-1 sm:gap-1.5 px-2 sm:px-3 py-1.5 rounded-md font-medium text-xs sm:text-sm transition-all flex-1 sm:flex-initial cursor-pointer {currentTheme === 'dark' ? 'bg-surface text-text-main shadow-sm' : 'text-text-muted hover:text-text-main'}"
          >
            <span class="material-symbols-outlined text-[18px]">dark_mode</span>
            <span>Dark</span>
          </button>
          <button
            type="button"
            onclick={() => setTheme('system')}
            class="flex items-center justify-center gap-1 sm:gap-1.5 px-2 sm:px-3 py-1.5 rounded-md font-medium text-xs sm:text-sm transition-all flex-1 sm:flex-initial cursor-pointer {currentTheme === 'system' ? 'bg-surface text-text-main shadow-sm' : 'text-text-muted hover:text-text-main'}"
          >
            <span class="material-symbols-outlined text-[18px]">contrast</span>
            <span>System</span>
          </button>
        </div>
      </div>

      <div class="flex flex-col gap-3 pt-4 border-t border-border">
        <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between p-3 rounded-lg bg-bg border border-border gap-2">
          <div>
            <p class="font-medium text-sm sm:text-base text-text-main">Database Location</p>
            <p class="text-xs sm:text-sm text-text-muted font-mono break-all">~/.9router/db/data.sqlite</p>
          </div>
        </div>

        <div class="flex flex-col sm:flex-row gap-2">
          <Button
            variant="secondary"
            icon="download"
            onclick={handleDownloadBackupClick}
            loading={isDownloadingBackup}
            class="w-full sm:w-auto"
          >
            Download Backup
          </Button>

          <Button
            variant="outline"
            icon="upload"
            onclick={() => fileInput?.click()}
            disabled={isImportingBackup}
            class="w-full sm:w-auto"
          >
            {isImportingBackup ? 'Importing...' : 'Import Backup'}
          </Button>

          <input
            bind:this={fileInput}
            type="file"
            accept="application/json,.json"
            class="hidden"
            onchange={handleFileSelected}
          />
        </div>

        {#if dbStatus.message}
          <p class="text-sm font-medium {dbStatus.type === 'error' ? 'text-red-500' : 'text-green-600 dark:text-green-400'}">
            {dbStatus.message}
          </p>
        {/if}
      </div>
    </Card>

    <!-- CARD 2: SECURITY -->
    <Card>
      <div class="flex items-center gap-3 mb-4">
        <div class="p-2 rounded-lg bg-brand-500/10 text-brand-500 shrink-0">
          <span class="material-symbols-outlined text-[20px]">shield</span>
        </div>
        <h3 class="text-base sm:text-lg font-semibold text-text-main">Security</h3>
      </div>

      <div class="flex flex-col gap-4">
        <div class="flex items-start sm:items-center justify-between gap-4">
          <div class="flex-1 min-w-0">
            <p class="font-medium text-sm sm:text-base text-text-main">Require login</p>
            <p class="text-xs sm:text-sm text-text-muted">
              When ON, dashboard requires password. When OFF, access without login.
            </p>
          </div>
          <Toggle
            checked={requireLogin}
            onChange={(val) => updateRequireLogin(val)}
          />
        </div>

        {#if requireLogin}
          <form onsubmit={handlePasswordChange} class="flex flex-col gap-4 pt-4 border-t border-border/50">
            <div class="flex flex-col gap-2">
              <label for="curr-pwd" class="text-xs sm:text-sm font-medium text-text-main">Current Password</label>
              <input
                id="curr-pwd"
                type="password"
                placeholder="Enter current password"
                bind:value={currentPassword}
                required
                class="w-full px-3 py-2 text-sm text-text-main bg-bg rounded-lg border border-border focus:outline-none focus:border-brand-500 transition-colors"
              />
            </div>

            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div class="flex flex-col gap-2">
                <label for="new-pwd" class="text-xs sm:text-sm font-medium text-text-main">New Password</label>
                <input
                  id="new-pwd"
                  type="password"
                  placeholder="Enter new password"
                  bind:value={newPassword}
                  required
                  class="w-full px-3 py-2 text-sm text-text-main bg-bg rounded-lg border border-border focus:outline-none focus:border-brand-500 transition-colors"
                />
              </div>

              <div class="flex flex-col gap-2">
                <label for="conf-pwd" class="text-xs sm:text-sm font-medium text-text-main">Confirm New Password</label>
                <input
                  id="conf-pwd"
                  type="password"
                  placeholder="Confirm new password"
                  bind:value={confirmNewPassword}
                  required
                  class="w-full px-3 py-2 text-sm text-text-main bg-bg rounded-lg border border-border focus:outline-none focus:border-brand-500 transition-colors"
                />
              </div>
            </div>

            {#if passStatus.message}
              <p class="text-xs sm:text-sm font-medium {passStatus.type === 'error' ? 'text-red-500' : 'text-green-500'}">
                {passStatus.message}
              </p>
            {/if}

            <div class="pt-2">
              <Button type="submit" variant="primary" loading={isUpdatingPassword} class="w-full sm:w-auto">
                Update Password
              </Button>
            </div>
          </form>
        {/if}
      </div>
    </Card>

    <!-- CARD 4: ROUTING STRATEGY -->
    <Card>
      <div class="flex items-center gap-3 mb-4">
        <div class="p-2 rounded-lg bg-blue-500/10 text-blue-500 shrink-0">
          <span class="material-symbols-outlined text-[20px]">route</span>
        </div>
        <h3 class="text-base sm:text-lg font-semibold text-text-main">Routing Strategy</h3>
      </div>

      <div class="flex flex-col gap-4">
        <div class="flex items-start sm:items-center justify-between gap-4">
          <div class="flex-1 min-w-0">
            <p class="font-medium text-sm sm:text-base text-text-main">Round Robin</p>
            <p class="text-xs sm:text-sm text-text-muted">
              Cycle through accounts to distribute load
            </p>
          </div>
          <Toggle
            checked={fallbackStrategy === 'round-robin'}
            onChange={(val) => updateFallbackStrategy(val ? 'round-robin' : 'fill-first')}
          />
        </div>

        {#if fallbackStrategy === 'round-robin'}
          <div class="flex items-start sm:items-center justify-between gap-4 pt-2 border-t border-border/50">
            <div class="flex-1 min-w-0">
              <p class="font-medium text-sm sm:text-base text-text-main">Sticky Limit</p>
              <p class="text-xs sm:text-sm text-text-muted">
                Calls per account before switching
              </p>
            </div>
            <input
              type="number"
              min="1"
              max="10"
              value={stickyRoundRobinLimit}
              onchange={(e) => updateStickyLimit((e.target as HTMLInputElement).value)}
              class="w-16 sm:w-20 px-2 py-1 text-center text-sm text-text-main bg-bg rounded-lg border border-border focus:outline-none focus:border-brand-500 font-mono"
            />
          </div>
        {/if}

        <div class="flex items-start sm:items-center justify-between gap-4 pt-4 border-t border-border/50">
          <div class="flex-1 min-w-0">
            <p class="font-medium text-sm sm:text-base text-text-main">Combo Round Robin</p>
            <p class="text-xs sm:text-sm text-text-muted">
              Cycle through providers in combos instead of always starting with first
            </p>
          </div>
          <Toggle
            checked={comboStrategy === 'round-robin'}
            onChange={(val) => updateComboStrategy(val ? 'round-robin' : 'fallback')}
          />
        </div>

        {#if comboStrategy === 'round-robin'}
          <div class="flex items-center justify-between pt-2 border-t border-border/50">
            <div>
              <p class="font-medium text-sm sm:text-base text-text-main">Combo Sticky Limit</p>
              <p class="text-xs sm:text-sm text-text-muted">
                Calls per combo model before switching
              </p>
            </div>
            <input
              type="number"
              min="1"
              max="100"
              value={comboStickyRoundRobinLimit}
              onchange={(e) => updateComboStickyLimit((e.target as HTMLInputElement).value)}
              class="w-20 px-2 py-1 text-center text-sm text-text-main bg-bg rounded-lg border border-border focus:outline-none focus:border-brand-500 font-mono"
            />
          </div>
        {/if}

        <p class="text-xs text-text-muted italic pt-2 border-t border-border/50">
          {fallbackStrategy === 'round-robin'
            ? `Currently distributing requests across all available accounts with ${stickyRoundRobinLimit} calls per account.`
            : 'Currently using accounts in priority order (Fill First).'}
          {comboStrategy === 'round-robin'
            ? ` Combos rotate after ${comboStickyRoundRobinLimit} call${comboStickyRoundRobinLimit === 1 ? '' : 's'} per model.`
            : ' Combos always start with their first model.'}
        </p>
      </div>
    </Card>

    <!-- CARD 4: OBSERVABILITY -->
    <Card>
      <div class="flex items-center gap-3 mb-4">
        <div class="p-2 rounded-lg bg-orange-500/10 text-orange-500 shrink-0">
          <span class="material-symbols-outlined text-[20px]">monitoring</span>
        </div>
        <h3 class="text-base sm:text-lg font-semibold text-text-main">Observability</h3>
      </div>

      <div class="flex items-start sm:items-center justify-between gap-4">
        <div class="flex-1 min-w-0">
          <p class="font-medium text-sm sm:text-base text-text-main">Enable Observability</p>
          <p class="text-xs sm:text-sm text-text-muted">
            Record request details for inspection in the logs view
          </p>
        </div>
        <Toggle
          checked={enableObservability}
          onChange={(val) => updateObservability(val)}
        />
      </div>
    </Card>

    <!-- BOTTOM ACTIONS -->
    <div class="flex flex-col sm:flex-row gap-2 pt-2">
      <Button
        variant="outline"
        fullWidth
        icon="logout"
        onclick={handleLogout}
      >
        Logout
      </Button>
    </div>

    <!-- APP INFO FOOTER -->
    <div class="text-center text-xs sm:text-sm text-text-muted py-4">
      <p>9router-go v1.9.1</p>
      <p class="mt-1">Local Mode — All data stored on your machine</p>
    </div>

  </div>

  <!-- CONFIRM PASSWORD MODAL -->
  <Modal
    isOpen={dbAuthModalOpen}
    onClose={() => {
      dbAuthModalOpen = false
      pendingImportFile = null
    }}
    title="Confirm Password"
    size="sm"
  >
    {#snippet children()}
      <div class="space-y-3">
        <p class="text-text-muted text-xs sm:text-sm">
          Enter your current password to {dbAuthMode === 'export' ? 'export' : 'import'} the database.
        </p>
        <input
          type="password"
          bind:value={dbAuthPassword}
          onkeydown={(e) => {
            if (e.key === 'Enter') handleDbAuthConfirm()
          }}
          placeholder="Current password (default: 123456)"
          class="w-full px-3 py-2 text-sm text-text-main bg-bg rounded-lg border border-border focus:outline-none focus:border-brand-500 font-mono transition-colors"
        />
      </div>
    {/snippet}

    {#snippet footer()}
      <div class="flex items-center justify-end gap-2">
        <button
          type="button"
          onclick={() => {
            dbAuthModalOpen = false
            pendingImportFile = null
          }}
          class="px-3 py-1.5 rounded-lg bg-surface-2 hover:bg-surface-3 border border-border text-xs sm:text-sm font-semibold text-text-main transition cursor-pointer"
        >
          Cancel
        </button>
        <button
          type="button"
          onclick={handleDbAuthConfirm}
          class="px-3 py-1.5 rounded-lg bg-brand-500 hover:bg-brand-600 text-white font-semibold text-xs sm:text-sm transition cursor-pointer"
        >
          Confirm
        </button>
      </div>
    {/snippet}
  </Modal>

</div>
