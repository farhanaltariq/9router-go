<script lang="ts">
  import { api } from '../api/client'
  import { TAB_ROUTES, type ActiveTab } from '../lib/router'

  export type { ActiveTab }

  let {
    activeTab = $bindable('dashboard'),
    navigate = (tab: ActiveTab) => {
      activeTab = tab
    },
    onClose,
  }: {
    activeTab: ActiveTab
    navigate?: (tab: ActiveTab, replace?: boolean) => void
    activeConnections?: number
    totalConnections?: number
    onClose?: () => void
  } = $props()

  let version = $state('')
  let isDisconnected = $state(false)

  $effect(() => {
    api
      .getSystemVersion()
      .then((v) => {
        if (v?.currentVersion) {
          version = v.currentVersion
        }
      })
      .catch(() => {})
  })

  function handleNav(tab: ActiveTab, e?: MouseEvent) {
    if (e) {
      if (e.ctrlKey || e.metaKey || e.shiftKey || e.altKey || e.button !== 0) return
      e.preventDefault()
    }
    navigate(tab)
    onClose?.()
  }

  const navItems = [
    { tab: 'dashboard' as ActiveTab, label: 'Dashboard', icon: 'dashboard' },
    { tab: 'connections' as ActiveTab, label: 'Providers', icon: 'dns' },
    { tab: 'combos' as ActiveTab, label: 'Combo & Vision Adapter', icon: 'layers' },
    { tab: 'token-saver' as ActiveTab, label: 'Token Saver', icon: 'savings' },
  ] as const

  const systemItems = [
    { tab: 'proxy-pools' as ActiveTab, label: 'Proxy Pools', icon: 'lan' },
    { tab: 'console-log' as ActiveTab, label: 'Console Log', icon: 'terminal' },
    { tab: 'settings' as ActiveTab, label: 'Settings', icon: 'settings' },
  ] as const

  function isLinkActive(tab: ActiveTab): boolean {
    if (tab === 'dashboard') {
      return (
        activeTab === 'dashboard' ||
        activeTab === 'endpoint' ||
        activeTab === 'analytics' ||
        activeTab === 'quota' ||
        activeTab === 'cli-tools' ||
        activeTab === 'keys'
      )
    }
    if (tab === 'console-log') {
      return activeTab === 'console-log' || activeTab === 'terminal'
    }
    return activeTab === tab
  }
</script>

<aside
  class="flex w-72 flex-col border-r border-border-subtle bg-vibrancy backdrop-blur-xl transition-colors duration-300 min-h-full shrink-0 select-none z-30"
>
  <!-- Brand logo header -->
  <div class="px-6 py-4 flex flex-col gap-2">
    <a
      href={TAB_ROUTES.dashboard}
      onclick={(e) => handleNav('dashboard', e)}
      class="flex items-center gap-3 cursor-pointer group"
    >
      <div
        class="flex items-center justify-center size-9 rounded-[10px] bg-linear-to-br from-brand-500 to-brand-700 shadow-[var(--shadow-warm)] shrink-0 group-hover:scale-105 transition-transform text-white"
      >
        <span class="material-symbols-outlined text-[20px]">hub</span>
      </div>
      <div class="flex flex-col min-w-0">
        <h1 class="text-lg font-semibold tracking-tight text-text-main truncate leading-snug">
          9router-go
        </h1>
        <span class="text-xs text-text-muted leading-tight">
          {version ? `v${version}` : 'v1.9.1'}
        </span>
      </div>
    </a>
  </div>

  <!-- Navigation -->
  <nav class="flex-1 px-4 py-2 space-y-0.5 overflow-y-auto custom-scrollbar">
    {#each navItems as item (item.tab)}
      {@const active = isLinkActive(item.tab)}
      <a
        href={TAB_ROUTES[item.tab]}
        onclick={(e) => handleNav(item.tab, e)}
        class="flex items-center gap-3 px-3 py-1 rounded-lg transition-all group cursor-pointer {active
          ? 'bg-primary/10 text-primary font-medium'
          : 'text-text-muted hover:bg-surface-2 hover:text-text-main'}"
      >
        <span
          class="material-symbols-outlined text-[18px] {active
            ? 'fill-1'
            : 'group-hover:text-primary transition-colors'}"
        >
          {item.icon}
        </span>
        <span class="text-[13px] font-medium">{item.label}</span>
      </a>
    {/each}

    <!-- System section -->
    <div class="pt-3 mt-2 space-y-0.5">
      <p class="px-4 text-xs font-semibold text-text-muted/60 uppercase tracking-wider mb-2">
        System
      </p>

      {#each systemItems as item (item.tab)}
        {@const active = isLinkActive(item.tab)}
        <a
          href={TAB_ROUTES[item.tab]}
          onclick={(e) => handleNav(item.tab, e)}
          class="flex items-center gap-3 px-3 py-1 rounded-lg transition-all group cursor-pointer {active
            ? 'bg-primary/10 text-primary font-medium'
            : 'text-text-muted hover:bg-surface-2 hover:text-text-main'}"
        >
          <span
            class="material-symbols-outlined text-[18px] {active
              ? 'fill-1'
              : 'group-hover:text-primary transition-colors'}"
          >
            {item.icon}
          </span>
          <span class="text-[13px] font-medium">{item.label}</span>
        </a>
      {/each}
    </div>
  </nav>
</aside>

<!-- Server Disconnected Overlay -->
{#if isDisconnected}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm p-6">
    <div class="text-center p-8 max-w-sm rounded-2xl bg-surface border border-border-subtle shadow-2xl">
      <div class="flex items-center justify-center size-16 rounded-full bg-red-500/20 text-red-500 mx-auto mb-4">
        <span class="material-symbols-outlined text-4xl">power_off</span>
      </div>
      <h2 class="text-xl font-semibold text-text-main mb-2">Server Disconnected</h2>
      <p class="text-text-muted text-sm mb-6">The gateway server has stopped or restarted.</p>
      <button
        type="button"
        onclick={() => globalThis.location.reload()}
        class="w-full py-2 px-4 rounded-lg bg-primary hover:bg-primary-hover text-white text-sm font-semibold transition-colors cursor-pointer"
      >
        Reload Page
      </button>
    </div>
  </div>
{/if}
