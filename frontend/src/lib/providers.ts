// Curated provider catalog matching upstream 9router
export interface ProviderCatalogItem {
  id: string
  name: string
  category: 'oauth' | 'free' | 'freeTier' | 'apikey' | 'webCookie' | 'custom'
  alias: string
  color: string
  icon: string
  noAuth?: boolean
  priority?: number
  mediaPriority?: number
  hiddenKinds?: string[]
  serviceKinds?: string[]
  /** auth modes from upstream registry (e.g. codebuddy ["apikey","oauth"] = dual buttons). */
  authModes?: string[]
  /** hidden from provider list (upstream parity) but detail page stays reachable. */
  hidden?: boolean
  /** Region choices for cluster-specific providers (upstream registry `regions`). */
  regions?: { id: string; label: string }[]
  defaultRegion?: string
  /** Upstream display.website + display.notice (signup/apiKey links). */
  website?: string
  notice?: { text?: string; apiKeyUrl?: string; signupUrl?: string }
  /** Upstream registry authType/authHint (cookie login hint, apikey/oauth/none markers). */
  authType?: string
  authHint?: string
  /** Upstream registry modelsFetcher (public catalog for "Suggested free models"). */
  modelsFetcher?: { url: string; type: string }
  /** Upstream registry systemoneConfig */
  systemoneConfig?: { baseUrl?: string; format?: string; headers?: Record<string, string> }
  searchConfig?: Record<string, any>
  fetchConfig?: Record<string, any>
  searchViaChat?: {
    defaultModel?: string
    endpoint?: string
    pricingUrl?: string
    freeTier?: string
  }
}

export const PROVIDER_CATEGORIES = [
  { id: 'custom', label: 'Custom (OpenAI / Anthropic Compatible)' },
  { id: 'oauth', label: 'OAuth Providers' },
  { id: 'freeTier', label: 'Free Tier Providers' },
  { id: 'apikey', label: 'API Key Providers' },
] as const

export const PROVIDER_CATALOG: ProviderCatalogItem[] = [
  {
    id: 'antigravity',
    name: 'Antigravity',
    category: 'oauth',
    alias: 'ag',
    color: '#F59E0B',
    icon: 'rocket_launch',
    website: 'https://antigravity.google',
    notice: { signupUrl: 'https://antigravity.google' },
    noAuth: false,
    priority: 20,
    serviceKinds: ['llm', 'image', 'webSearch'],
    searchViaChat: {
      defaultModel: 'gemini-2.5-flash',
      endpoint: 'https://daily-cloudcode-pa.googleapis.com/v1internal:generateContent',
      freeTier: 'Free — Google Search grounding through an Antigravity OAuth account.',
    },
  },
  {
    id: 'github',
    name: 'GitHub Models',
    category: 'oauth',
    alias: 'gh',
    color: '#24292F',
    icon: 'code_blocks',
    website: 'https://github.com/marketplace/models',
    notice: { signupUrl: 'https://github.com/marketplace/models' },
    noAuth: false,
    priority: 25,
    serviceKinds: ['llm'],
  },
  {
    id: 'codex',
    name: 'Codex',
    category: 'oauth',
    alias: 'cx',
    color: '#10A37F',
    icon: 'psychology',
    website: 'https://openai.com',
    notice: { signupUrl: 'https://openai.com' },
    noAuth: false,
    priority: 30,
    serviceKinds: ['llm', 'image'],
  },
  {
    id: 'codebuddy-intl',
    name: 'CodeBuddy',
    category: 'oauth',
    alias: 'cbai',
    color: '#165DFF',
    icon: 'code',
    website: 'https://www.codebuddy.ai',
    notice: { apiKeyUrl: 'https://www.codebuddy.ai', signupUrl: 'https://www.codebuddy.ai' },
    noAuth: false,
    priority: 35,
    authModes: ['apikey', 'oauth'],
    serviceKinds: ['llm'],
  },
  {
    id: 'codebuddy-cn',
    name: 'CodeBuddy CN',
    category: 'oauth',
    alias: 'cd',
    color: '#0052D9',
    icon: 'code',
    website: 'https://copilot.tencent.com',
    notice: { apiKeyUrl: 'https://copilot.tencent.com', signupUrl: 'https://copilot.tencent.com' },
    noAuth: false,
    priority: 40,
    authModes: ['apikey', 'oauth'],
    serviceKinds: ['llm'],
  },
  {
    id: 'nvidia',
    name: 'NVIDIA NIM',
    category: 'freeTier',
    alias: 'nv',
    color: '#76B900',
    icon: 'memory',
    website: 'https://build.nvidia.com',
    notice: { apiKeyUrl: 'https://build.nvidia.com', signupUrl: 'https://build.nvidia.com' },
    noAuth: false,
    priority: 45,
    serviceKinds: ['llm'],
  },
  {
    id: 'ollama',
    name: 'Ollama',
    category: 'freeTier',
    alias: 'ol',
    color: '#000000',
    icon: 'terminal',
    website: 'https://ollama.com',
    notice: { signupUrl: 'https://ollama.com' },
    noAuth: false,
    priority: 50,
    serviceKinds: ['llm'],
  },
  {
    id: 'cloudflare-ai',
    name: 'Cloudflare AI',
    category: 'freeTier',
    alias: 'cf',
    color: '#F38020',
    icon: 'cloud',
    website: 'https://ai.cloudflare.com',
    notice: { apiKeyUrl: 'https://dash.cloudflare.com', signupUrl: 'https://dash.cloudflare.com/sign-up' },
    noAuth: false,
    priority: 55,
    serviceKinds: ['llm', 'image'],
  },
  {
    id: 'commandcode',
    name: 'CommandCode',
    category: 'apikey',
    alias: 'cmc',
    color: '#6366F1',
    icon: 'terminal',
    website: 'https://commandcode.ai',
    notice: { apiKeyUrl: 'https://commandcode.ai', signupUrl: 'https://commandcode.ai' },
    noAuth: false,
    priority: 60,
    serviceKinds: ['llm'],
  },
  {
    id: 'opencode',
    name: 'Opencode Free',
    category: 'apikey',
    alias: 'oc',
    color: '#8B5CF6',
    icon: 'code',
    website: 'https://opencode.ai',
    notice: { signupUrl: 'https://opencode.ai' },
    noAuth: true,
    priority: 65,
    serviceKinds: ['llm'],
  },
]

export const PROVIDER_CATALOG_MAP = new Map(PROVIDER_CATALOG.map((p) => [p.id, p]))

export function isChatProvider(p: ProviderCatalogItem): boolean {
  return (p.serviceKinds ?? ['llm']).includes('llm')
}

export function getProvidersByKind(kind: string): ProviderCatalogItem[] {
  return PROVIDER_CATALOG
    .filter((p) => {
      const kinds = p.serviceKinds ?? ['llm']
      if (!kinds.includes(kind)) return false
      if (p.hidden) return false
      if (p.hiddenKinds?.includes(kind)) return false
      return true
    })
    .sort((a, b) => ((a.priority ?? a.mediaPriority ?? 999) - (b.priority ?? b.mediaPriority ?? 999)))
}

export const MEDIA_PROVIDER_KINDS = [
  'embedding',
  'image',
  'tts',
  'stt',
  'video',
  'webSearch',
  'webFetch',
  'web',
] as const
