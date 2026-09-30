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
    class?: string
    style?: string
  }

  let { period = '7d', class: klass = '', style = '' }: Props = $props()

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

  const width = 800
  const height = 340
  const padLeft = 56
  const padRight = 24
  const padTop = 20
  const padBottom = 34
  const plotWidth = width - padLeft - padRight
  const plotHeight = height - padTop - padBottom

  let maxVal = $derived.by(() => {
    if (!data.length) return 1
    const vals = data.map((d) => (viewMode === 'tokens' ? d.tokens || 0 : d.cost || 0))
    const m = Math.max(...vals, 0)
    return m === 0 ? 1 : m
  })

  // Y axis ticks (0, 25%, 50%, 75%, 100%)
  let yTicks = $derived([
    { val: maxVal, y: padTop },
    { val: maxVal * 0.75, y: padTop + plotHeight * 0.25 },
    { val: maxVal * 0.5, y: padTop + plotHeight * 0.5 },
    { val: maxVal * 0.25, y: padTop + plotHeight * 0.75 },
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

  function shouldShowXLabel(i: number, total: number): boolean {
    if (i === 0 || i === total - 1) return true
    if (total <= 8) return true // 7d: show all days
    if (total <= 24) return i % 4 === 0 // 24h / today: every 4 hours
    if (total <= 31) return i % 5 === 0 // 30d: every 5 days
    return i % 10 === 0 // 60d: every 10 days
  }
</script>

<Card
  padding="none"
  class="flex min-w-0 flex-col overflow-hidden bg-surface border border-border-subtle rounded-brand-lg shadow-(--shadow-soft) h-full {klass}"
  style={style}
>
  <div class="px-4 py-3 border-b border-border-subtle flex items-center justify-between shrink-0">
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

  <div class="p-4 flex-1 flex flex-col justify-between min-h-0 relative">
    {#if loading}
      <div class="flex-1 min-h-75 flex items-center justify-center text-text-muted text-sm gap-2">
        <span class="w-4 h-4 border-2 border-primary border-t-transparent rounded-full animate-spin"></span>
        <span>Loading chart...</span>
      </div>
    {:else if !hasData}
      <div class="flex-1 min-h-75 flex flex-col items-center justify-center text-text-muted text-sm gap-1.5">
        <span class="material-symbols-outlined text-[28px] opacity-40">query_stats</span>
        <span>No usage data recorded for this period</span>
      </div>
    {:else}
      <div
        class="relative w-full h-full flex-1 flex flex-col justify-center overflow-hidden select-none"
        onmousemove={handleMouseMove}
        onmouseleave={handleMouseLeave}
        role="region"
        aria-label="Usage chart"
      >
        <svg viewBox="0 0 {width} {height}" class="w-full h-full min-h-75 overflow-visible">
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

          <!-- X axis labels (evenly distributed) -->
          {#if points.length > 0}
            {#each points as p, i}
              {#if shouldShowXLabel(i, points.length)}
                <text
                  x={p.x}
                  y={height - 8}
                  text-anchor={i === 0 ? 'start' : i === points.length - 1 ? 'end' : 'middle'}
                  class="text-[10px] fill-text-muted/70 font-mono select-none"
                >
                  {p.label}
                </text>
              {/if}
            {/each}
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
          {@const isNearTop = hp.y < 60}
          {@const isLeftEdge = hoverIdx === 0}
          {@const isRightEdge = hoverIdx === points.length - 1}
          {@const xTransform = isLeftEdge ? 'translate-x-1' : isRightEdge ? '-translate-x-[calc(100%-4px)]' : '-translate-x-1/2'}
          <div
            class="absolute pointer-events-none z-20 px-2.5 py-1.5 rounded-lg bg-surface border border-border shadow-xl text-xs flex flex-col gap-0.5 {xTransform} {isNearTop ? 'translate-y-3' : '-translate-y-full -mt-2'}"
            style="left: {(hp.x / width) * 100}%; top: {(hp.y / height) * 100}%;"
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
