<script lang="ts">
  import { api } from '../api/client'
  import Card from '../lib/ui/Card.svelte'

  interface ChartPoint {
    label: string
    tokens: number
    cost: number
  }

  interface Props {
    period?: string
  }

  let { period = '7d' }: Props = $props()

  let data = $state<ChartPoint[]>([])
  let loading = $state(true)
  let viewMode = $state<'tokens' | 'cost'>('tokens')
  let hoverIdx = $state<number | null>(null)

  function fmtTokens(n: number): string {
    if (n >= 1000000) return `${(n / 1000000).toFixed(1)}M`
    if (n >= 1000) return `${(n / 1000).toFixed(1)}K`
    return String(n || 0)
  }

  function fmtCost(n: number): string {
    return `$${(n || 0).toFixed(4)}`
  }

  $effect(() => {
    let cancelled = false
    loading = true
    api
      .getUsageChart(period)
      .then((res) => {
        if (!cancelled) {
          data = Array.isArray(res) ? res : []
        }
      })
      .catch(() => {
        if (!cancelled) {
          data = []
        }
      })
      .finally(() => {
        if (!cancelled) {
          loading = false
        }
      })
    return () => {
      cancelled = true
    }
  })

  let hasData = $derived(data.some((d) => (d.tokens || 0) > 0 || (d.cost || 0) > 0))

  const width = 600
  const height = 180
  const padLeft = 48
  const padRight = 16
  const padTop = 16
  const padBottom = 28
  const plotWidth = width - padLeft - padRight
  const plotHeight = height - padTop - padBottom

  let maxVal = $derived.by(() => {
    if (!data.length) return 1
    const vals = data.map((d) => (viewMode === 'tokens' ? d.tokens || 0 : d.cost || 0))
    const m = Math.max(...vals, 0)
    return m === 0 ? 1 : m
  })

  // Y axis ticks (0, 33%, 66%, 100%)
  let yTicks = $derived([
    { val: maxVal, y: padTop },
    { val: maxVal * 0.66, y: padTop + plotHeight * 0.34 },
    { val: maxVal * 0.33, y: padTop + plotHeight * 0.67 },
    { val: 0, y: padTop + plotHeight },
  ])

  // Points coordinates
  let points = $derived.by(() => {
    if (!data.length) return []
    const step = data.length > 1 ? plotWidth / (data.length - 1) : plotWidth
    return data.map((d, i) => {
      const v = viewMode === 'tokens' ? d.tokens || 0 : d.cost || 0
      const x = padLeft + (data.length > 1 ? i * step : plotWidth / 2)
      const y = padTop + plotHeight - (v / maxVal) * plotHeight
      return { x, y, ...d }
    })
  })

  // Smooth SVG path
  let pathD = $derived.by(() => {
    if (points.length < 2) return ''
    let d = `M ${points[0].x} ${points[0].y}`
    for (let i = 0; i < points.length - 1; i++) {
      const p0 = points[i]
      const p1 = points[i + 1]
      const mx = (p0.x + p1.x) / 2
      d += ` C ${mx} ${p0.y}, ${mx} ${p1.y}, ${p1.x} ${p1.y}`
    }
    return d
  })

  let areaD = $derived.by(() => {
    if (!pathD || points.length < 2) return ''
    const lastX = points[points.length - 1].x
    const firstX = points[0].x
    const baseY = padTop + plotHeight
    return `${pathD} L ${lastX} ${baseY} L ${firstX} ${baseY} Z`
  })

  function handleMouseMove(e: MouseEvent) {
    if (!points.length) return
    const svg = (e.currentTarget as HTMLElement).getBoundingClientRect()
    const mouseX = ((e.clientX - svg.left) / svg.width) * width
    let closestIdx = 0
    let closestDist = Infinity
    points.forEach((p, idx) => {
      const dist = Math.abs(p.x - mouseX)
      if (dist < closestDist) {
        closestDist = dist
        closestIdx = idx
      }
    })
    hoverIdx = closestIdx
  }

  function handleMouseLeave() {
    hoverIdx = null
  }
</script>

<Card padding="none" class="flex min-w-0 flex-col overflow-hidden bg-surface border border-border-subtle rounded-[14px]">
  <div class="px-4 py-3 border-b border-border-subtle flex items-center justify-between">
    <div class="flex items-center gap-2">
      <span class="material-symbols-outlined text-[18px] text-primary">show_chart</span>
      <span class="text-sm font-semibold text-text-main">Usage</span>
    </div>
    <div class="flex items-center gap-1 bg-surface-2 p-0.5 rounded-lg border border-border-subtle">
      <button
        type="button"
        onclick={() => (viewMode = 'tokens')}
        class="px-2.5 py-0.5 rounded text-xs font-medium transition-colors cursor-pointer {viewMode === 'tokens'
          ? 'bg-primary text-white shadow-xs'
          : 'text-text-muted hover:text-text-main'}"
      >
        Tokens
      </button>
      <button
        type="button"
        onclick={() => (viewMode = 'cost')}
        class="px-2.5 py-0.5 rounded text-xs font-medium transition-colors cursor-pointer {viewMode === 'cost'
          ? 'bg-primary text-white shadow-xs'
          : 'text-text-muted hover:text-text-main'}"
      >
        Cost
      </button>
    </div>
  </div>

  <div class="p-4 relative">
    {#if loading}
      <div class="h-44 flex items-center justify-center text-text-muted text-sm gap-2">
        <span class="w-4 h-4 border-2 border-primary border-t-transparent rounded-full animate-spin"></span>
        <span>Loading chart...</span>
      </div>
    {:else if !hasData}
      <div class="h-44 flex items-center justify-center text-text-muted text-sm">
        No usage data recorded for this period
      </div>
    {:else}
      <div
        class="relative w-full overflow-hidden select-none"
        onmousemove={handleMouseMove}
        onmouseleave={handleMouseLeave}
        role="region"
        aria-label="Usage chart"
      >
        <svg viewBox="0 0 {width} {height}" class="w-full h-44 overflow-visible">
          <defs>
            <linearGradient id="chartGrad" x1="0" y1="0" x2="0" y2="1">
              <stop
                offset="5%"
                stop-color={viewMode === 'tokens' ? '#6366f1' : '#f59e0b'}
                stop-opacity="0.25"
              />
              <stop
                offset="95%"
                stop-color={viewMode === 'tokens' ? '#6366f1' : '#f59e0b'}
                stop-opacity="0.02"
              />
            </linearGradient>
          </defs>

          <!-- Grid horizontal dashed lines -->
          {#each yTicks as tick}
            <line
              x1={padLeft}
              y1={tick.y}
              x2={width - padRight}
              y2={tick.y}
              stroke="currentColor"
              class="text-border-subtle opacity-40"
              stroke-dasharray="3 3"
            />
            <text
              x={padLeft - 8}
              y={tick.y + 3.5}
              text-anchor="end"
              class="text-[9px] fill-text-muted/70 font-mono"
            >
              {viewMode === 'tokens' ? fmtTokens(tick.val) : fmtCost(tick.val)}
            </text>
          {/each}

          <!-- Filled Area gradient -->
          <path d={areaD} fill="url(#chartGrad)" />

          <!-- Main Stroke Line -->
          <path
            d={pathD}
            fill="none"
            stroke={viewMode === 'tokens' ? '#6366f1' : '#f59e0b'}
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          />

          <!-- X axis labels (first, middle, last) -->
          {#if points.length > 0}
            <text
              x={points[0].x}
              y={height - 6}
              text-anchor="start"
              class="text-[10px] fill-text-muted/70 font-mono"
            >
              {points[0].label}
            </text>
            {#if points.length > 2}
              {@const midIdx = Math.floor(points.length / 2)}
              <text
                x={points[midIdx].x}
                y={height - 6}
                text-anchor="middle"
                class="text-[10px] fill-text-muted/70 font-mono"
              >
                {points[midIdx].label}
              </text>
            {/if}
            <text
              x={points[points.length - 1].x}
              y={height - 6}
              text-anchor="end"
              class="text-[10px] fill-text-muted/70 font-mono"
            >
              {points[points.length - 1].label}
            </text>
          {/if}

          <!-- Hover Indicator Dot -->
          {#if hoverIdx !== null && points[hoverIdx]}
            {@const hp = points[hoverIdx]}
            <line
              x1={hp.x}
              y1={padTop}
              x2={hp.x}
              y2={padTop + plotHeight}
              stroke="currentColor"
              class="text-primary/40"
              stroke-dasharray="2 2"
            />
            <circle
              cx={hp.x}
              cy={hp.y}
              r="4.5"
              fill={viewMode === 'tokens' ? '#6366f1' : '#f59e0b'}
              stroke="white"
              stroke-width="1.5"
            />
          {/if}
        </svg>

        <!-- Hover Tooltip -->
        {#if hoverIdx !== null && points[hoverIdx]}
          {@const hp = points[hoverIdx]}
          <div
            class="absolute pointer-events-none z-20 px-2.5 py-1.5 rounded-lg bg-surface border border-border-subtle shadow-xl text-xs flex flex-col gap-0.5 -translate-x-1/2 -translate-y-full mb-2"
            style="left: {(hp.x / width) * 100}%; top: {Math.max(16, (hp.y / height) * 100)}%;"
          >
            <span class="text-[10px] text-text-muted font-mono">{hp.label}</span>
            <span class="font-semibold text-text-main">
              {viewMode === 'tokens' ? `${fmtTokens(hp.tokens)} tokens` : fmtCost(hp.cost)}
            </span>
          </div>
        {/if}
      </div>
    {/if}
  </div>
</Card>
