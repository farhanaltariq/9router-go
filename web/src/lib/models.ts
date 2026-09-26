// Curated model catalog matching upstream 9router
export interface ProviderModel {
  id: string
  name?: string
  upstreamModelId?: string
  kind?: string
  type?: string
  thinking?: boolean
  imageGen?: boolean
  capabilities?: string[]
  isFree?: boolean
  contextLength?: number
  [key: string]: unknown
}

export const PROVIDER_ID_TO_ALIAS: Record<string, string> = {
  antigravity: 'ag',
  github: 'gh',
  codex: 'cx',
  'codebuddy-intl': 'cbai',
  'codebuddy-cn': 'cd',
  nvidia: 'nv',
  ollama: 'ol',
  'cloudflare-ai': 'cf',
  commandcode: 'cmc',
  opencode: 'oc',
}

const ANTIGRAVITY_MODELS: ProviderModel[] = [
  { id: 'gemini-3.8-flash-high', name: 'Gemini 3.8 Flash (High)', upstreamModelId: 'gemini-3.8-flash-high(high)' },
  { id: 'gemini-3.8-flash-medium', name: 'Gemini 3.8 Flash (Medium)', upstreamModelId: 'gemini-3.8-flash-medium(medium)' },
  { id: 'gemini-3.8-flash-low', name: 'Gemini 3.8 Flash (Low)', upstreamModelId: 'gemini-3.8-flash-low(low)' },
  { id: 'gemini-3.8-flash', name: 'Gemini 3.8 Flash', upstreamModelId: 'gemini-3.8-flash-medium(medium)' },
  { id: 'gemini-3.7-flash-high', name: 'Gemini 3.7 Flash (High)', upstreamModelId: 'gemini-3.7-flash-high(high)' },
  { id: 'gemini-3.7-flash-medium', name: 'Gemini 3.7 Flash (Medium)', upstreamModelId: 'gemini-3.7-flash-medium(medium)' },
  { id: 'gemini-3.7-flash-low', name: 'Gemini 3.7 Flash (Low)', upstreamModelId: 'gemini-3.7-flash-low(low)' },
  { id: 'gemini-3.6-flash-high', name: 'Gemini 3.6 Flash (High)', upstreamModelId: 'gemini-3.6-flash-high(high)' },
  { id: 'gemini-3.6-flash-medium', name: 'Gemini 3.6 Flash (Medium)', upstreamModelId: 'gemini-3.6-flash-medium(medium)' },
  { id: 'gemini-3.6-flash-low', name: 'Gemini 3.6 Flash (Low)', upstreamModelId: 'gemini-3.6-flash-low(low)' },
  { id: 'gemini-3.5-flash-high', name: 'Gemini 3.5 Flash (High)', upstreamModelId: 'gemini-3.5-flash-high(high)' },
  { id: 'gemini-3-flash-agent', name: 'Gemini 3 Flash Agent', upstreamModelId: 'gemini-3-flash-agent(high)' },
  { id: 'gemini-3.5-flash-low', name: 'Gemini 3.5 Flash (Low)', upstreamModelId: 'gemini-3.5-flash-low(low)' },
  { id: 'gemini-3.5-flash-extra-low', name: 'Gemini 3.5 Flash (Extra Low)', upstreamModelId: 'gemini-3.5-flash-extra-low(extra-low)' },
  { id: 'gemini-pro-agent', name: 'Gemini Pro Agent', upstreamModelId: 'gemini-pro-agent(medium)' },
  { id: 'gemini-3.1-pro-low', name: 'Gemini 3.1 Pro (Low)', upstreamModelId: 'gemini-3.1-pro-low(low)' },
  { id: 'claude-sonnet-4-6', name: 'Claude Sonnet 4.6', upstreamModelId: 'claude-sonnet-4-6(medium)' },
  { id: 'claude-opus-4-6-thinking', name: 'Claude Opus 4.6 Thinking', upstreamModelId: 'claude-opus-4-6-thinking(medium)' },
  { id: 'gpt-oss-120b-medium', name: 'GPT-OSS 120B (Medium)', upstreamModelId: 'gpt-oss-120b-medium(medium)' },
  { id: 'gemini-2.5-flash', name: 'Gemini 2.5 Flash', upstreamModelId: 'gemini-2.5-flash' },
  { id: 'gemini-2.5-pro', name: 'Gemini 2.5 Pro', upstreamModelId: 'gemini-2.5-pro' },
  { id: 'gemini-2.5-flash-thinking', name: 'Gemini 2.5 Flash Thinking', upstreamModelId: 'gemini-2.5-flash-thinking' },
  { id: 'gemini-3-flash', name: 'Gemini 3 Flash', upstreamModelId: 'gemini-3-flash' },
  { id: 'gemini-3.1-flash-image', name: 'Gemini 3.1 Flash Image', upstreamModelId: 'gemini-3.1-flash-image' },
]

const GITHUB_MODELS: ProviderModel[] = [
  { id: 'gpt-5.4', name: 'GPT-5.4' },
  { id: 'gpt-5.4-mini', name: 'GPT-5.4 Mini' },
  { id: 'gpt-5.3-codex', name: 'GPT-5.3 Codex' },
  { id: 'gpt-5.2-codex', name: 'GPT-5.2 Codex' },
  { id: 'gpt-5.2', name: 'GPT-5.2' },
  { id: 'claude-opus-4.7', name: 'Claude Opus 4.7' },
  { id: 'claude-opus-4.6', name: 'Claude Opus 4.6' },
  { id: 'claude-opus-4.5', name: 'Claude Opus 4.5' },
  { id: 'claude-sonnet-4.6', name: 'Claude Sonnet 4.6' },
  { id: 'claude-sonnet-4.5', name: 'Claude Sonnet 4.5' },
  { id: 'claude-haiku-4.5', name: 'Claude Haiku 4.5' },
  { id: 'gemini-3.1-pro-preview', name: 'Gemini 3.1 Pro' },
  { id: 'gemini-3-flash-preview', name: 'Gemini 3 Flash' },
  { id: 'gemini-2.5-pro', name: 'Gemini 2.5 Pro' },
  { id: 'grok-code-fast-1', name: 'Grok Code Fast 1' },
  { id: 'text-embedding-3-small', name: 'Text Embedding 3 Small' },
  { id: 'text-embedding-3-large', name: 'Text Embedding 3 Large' },
]

const CODEX_MODELS: ProviderModel[] = [
  { id: 'gpt-6-astra', name: 'GPT-6 Astra' },
  { id: 'gpt-5.6-sol', name: 'GPT-5.6 Sol' },
  { id: 'gpt-5.6-terra', name: 'GPT-5.6 Terra' },
  { id: 'gpt-5.6-luna', name: 'GPT-5.6 Luna' },
  { id: 'gpt-5.5', name: 'GPT-5.5' },
  { id: 'gpt-5.4', name: 'GPT-5.4' },
  { id: 'gpt-5.4-mini', name: 'GPT-5.4 Mini' },
  { id: 'gpt-5.3-codex-spark', name: 'GPT-5.3 Codex Spark' },
  { id: 'gpt-5.6-sol-image', name: 'GPT-5.6 Sol Image' },
  { id: 'gpt-5.6-terra-image', name: 'GPT-5.6 Terra Image' },
  { id: 'gpt-5.6-luna-image', name: 'GPT-5.6 Luna Image' },
  { id: 'gpt-5.5-image', name: 'GPT-5.5 Image' },
  { id: 'gpt-5.4-image', name: 'GPT-5.4 Image' },
  { id: 'gpt-5.3-image', name: 'GPT-5.3 Image' },
]

const CODEBUDDY_INTL_MODELS: ProviderModel[] = [
  { id: 'glm-5.2', name: 'GLM 5.2' },
  { id: 'glm-5.1', name: 'GLM 5.1' },
  { id: 'glm-5.0', name: 'GLM 5.0' },
  { id: 'glm-5.0-turbo', name: 'GLM 5.0 Turbo' },
  { id: 'glm-5v-turbo', name: 'GLM 5V Turbo' },
  { id: 'glm-4.7', name: 'GLM 4.7' },
  { id: 'minimax-m3', name: 'MiniMax M3' },
  { id: 'minimax-m2.7', name: 'MiniMax M2.7' },
  { id: 'kimi-k2.7', name: 'Kimi K2.7' },
  { id: 'kimi-k2.6', name: 'Kimi K2.6' },
  { id: 'kimi-k2.5', name: 'Kimi K2.5' },
  { id: 'hy3-preview', name: 'Hunyuan 3 Preview' },
  { id: 'deepseek-v4-pro', name: 'DeepSeek V4 Pro' },
  { id: 'deepseek-v4.1-flash', name: 'DeepSeek V4.1 Flash' },
]

const CODEBUDDY_CN_MODELS: ProviderModel[] = [
  { id: 'glm-5.3', name: 'GLM 5.3' },
  { id: 'glm-5.3-flash', name: 'GLM 5.3 Flash' },
  { id: 'glm-5.2', name: 'GLM 5.2' },
  { id: 'glm-5.1', name: 'GLM 5.1' },
  { id: 'glm-5v-turbo', name: 'GLM 5V Turbo' },
  { id: 'minimax-m3', name: 'MiniMax M3' },
  { id: 'kimi-k3-1', name: 'Kimi K3.1' },
  { id: 'kimi-k2.7', name: 'Kimi K2.7' },
  { id: 'kimi-k2.6', name: 'Kimi K2.6' },
  { id: 'hy4-preview', name: 'Hunyuan 4 Preview' },
  { id: 'hy3', name: 'Hunyuan 3' },
  { id: 'deepseek-v4-pro', name: 'DeepSeek V4 Pro' },
  { id: 'deepseek-v4.1-flash', name: 'DeepSeek V4.1 Flash' },
]

const NVIDIA_MODELS: ProviderModel[] = [
  { id: 'minimaxai/minimax-m3', name: 'MiniMax M3' },
  { id: 'minimaxai/minimax-m2.7', name: 'MiniMax M2.7' },
  { id: 'z-ai/glm-5.2', name: 'GLM 5.2' },
  { id: 'deepseek-ai/deepseek-v4-pro', name: 'DeepSeek V4 Pro' },
  { id: 'deepseek-ai/deepseek-v4-flash', name: 'DeepSeek V4 Flash' },
  { id: 'moonshotai/kimi-k2.6', name: 'Kimi K2.6' },
  { id: 'nvidia/nemotron-3-ultra-550b-a55b', name: 'Nemotron 3 Ultra 550B' },
  { id: 'nvidia/nv-embedqa-e5-v5', name: 'NV EmbedQA E5 V5' },
]

const OLLAMA_MODELS: ProviderModel[] = [
  { id: 'gpt-oss:120b', name: 'GPT-OSS 120B' },
  { id: 'kimi-k2.5', name: 'Kimi K2.5' },
  { id: 'glm-5', name: 'GLM 5' },
  { id: 'minimax-m2.5', name: 'MiniMax M2.5' },
  { id: 'glm-4.7-flash', name: 'GLM 4.7 Flash' },
  { id: 'qwen3.5', name: 'Qwen 3.5' },
  { id: 'minimax-m3', name: 'MiniMax M3' },
  { id: 'deepseek-v4.1-flash:cloud', name: 'DeepSeek V4.1 Flash (Cloud)' },
]

const CLOUDFLARE_MODELS: ProviderModel[] = [
  { id: '@cf/meta/llama-3.3-70b-instruct-fp8-fast', name: 'Llama 3.3 70B' },
  { id: '@cf/meta/llama-3.1-70b-instruct-fp8-fast', name: 'Llama 3.1 70B' },
  { id: '@cf/meta/llama-3.1-8b-instruct-fp8-fast', name: 'Llama 3.1 8B' },
  { id: '@cf/mistralai/mistral-small-3.1-24b-instruct', name: 'Mistral Small 3.1 24B' },
  { id: '@cf/deepseek-ai/deepseek-r1-distill-qwen-32b', name: 'DeepSeek R1 Distill Qwen 32B' },
  { id: '@cf/moonshotai/kimi-k2.6', name: 'Kimi K2.6' },
  { id: '@cf/moonshotai/kimi-k2.5', name: 'Kimi K2.5' },
  { id: '@cf/zai-org/glm-4.7-flash', name: 'GLM 4.7 Flash' },
  { id: '@cf/qwen/qwq-32b', name: 'QwQ 32B' },
  { id: '@cf/qwen/qwen2.5-coder-32b-instruct', name: 'Qwen 2.5 Coder 32B' },
  { id: '@cf/black-forest-labs/flux-1-schnell', name: 'Flux 1 Schnell' },
  { id: '@cf/stabilityai/stable-diffusion-xl-base-1.0', name: 'Stable Diffusion XL' },
]

const OPENCODE_MODELS: ProviderModel[] = [
  { id: 'muse-spark-1.2-contributor-free', name: 'Muse Spark 1.2 Free' },
  { id: 'muse-spark-1.3-contributor-free', name: 'Muse Spark 1.3 Free' },
]

const COMMANDCODE_MODELS: ProviderModel[] = [
  { id: 'deepseek/deepseek-v4-pro', name: 'DeepSeek V4 Pro' },
  { id: 'deepseek/deepseek-v4-flash', name: 'DeepSeek V4 Flash' },
  { id: 'moonshotai/Kimi-K2.6', name: 'Kimi K2.6' },
  { id: 'moonshotai/Kimi-K2.5', name: 'Kimi K2.5' },
  { id: 'zai-org/GLM-5.1', name: 'GLM 5.1' },
  { id: 'zai-org/GLM-5', name: 'GLM 5' },
  { id: 'MiniMaxAI/MiniMax-M2.7', name: 'MiniMax M2.7' },
  { id: 'MiniMaxAI/MiniMax-M2.5', name: 'MiniMax M2.5' },
  { id: 'Qwen/Qwen3.6-Max-Preview', name: 'Qwen 3.6 Max' },
  { id: 'Qwen/Qwen3.6-Plus', name: 'Qwen 3.6 Plus' },
  { id: 'stepfun/Step-3.5-Flash', name: 'Step 3.5 Flash' },
]

export const BUILTIN_MODELS_BY_PROVIDER: Record<string, ProviderModel[]> = {
  ag: ANTIGRAVITY_MODELS,
  antigravity: ANTIGRAVITY_MODELS,
  gh: GITHUB_MODELS,
  github: GITHUB_MODELS,
  cx: CODEX_MODELS,
  codex: CODEX_MODELS,
  cbai: CODEBUDDY_INTL_MODELS,
  'codebuddy-intl': CODEBUDDY_INTL_MODELS,
  cd: CODEBUDDY_CN_MODELS,
  cbcn: CODEBUDDY_CN_MODELS,
  'codebuddy-cn': CODEBUDDY_CN_MODELS,
  nv: NVIDIA_MODELS,
  nvidia: NVIDIA_MODELS,
  ol: OLLAMA_MODELS,
  ollama: OLLAMA_MODELS,
  cf: CLOUDFLARE_MODELS,
  'cloudflare-ai': CLOUDFLARE_MODELS,
  cmc: COMMANDCODE_MODELS,
  commandcode: COMMANDCODE_MODELS,
  oc: OPENCODE_MODELS,
  opencode: OPENCODE_MODELS,
}

export function getBuiltinModels(providerId: string): ProviderModel[] {
  if (BUILTIN_MODELS_BY_PROVIDER[providerId]) {
    return BUILTIN_MODELS_BY_PROVIDER[providerId]
  }
  const alias = PROVIDER_ID_TO_ALIAS[providerId]
  if (alias && BUILTIN_MODELS_BY_PROVIDER[alias]) {
    return BUILTIN_MODELS_BY_PROVIDER[alias]
  }
  return []
}

export function resolveModelCapabilities(model: string | ProviderModel, providerId?: string): {
  vision: boolean
  audioInput: boolean
  reasoning: boolean
} {
  const modelId = typeof model === 'string' ? model : model?.id || ''
  const obj = typeof model === 'object' ? model : null
  const idLower = modelId.toLowerCase()
  const nameLower = (obj?.name || '').toLowerCase()
  const upstreamLower = (obj?.upstreamModelId || '').toLowerCase()
  const caps = Array.isArray(obj?.capabilities) ? obj.capabilities : []
  const kind = obj?.kind || obj?.type || ''
  const type = obj?.type || ''

  let vision =
    caps.includes('vision') ||
    caps.includes('image') ||
    idLower.includes('vision') ||
    idLower.includes('image') ||
    idLower.includes('vl') ||
    kind === 'image' ||
    type === 'image'

  if (!vision && providerId === 'antigravity') {
    vision = true
  }

  let audioInput =
    caps.includes('audio') ||
    caps.includes('audioInput') ||
    idLower.includes('audio-in') ||
    idLower.includes('audioinput') ||
    kind === 'stt' ||
    type === 'stt'

  let reasoning =
    caps.includes('reasoning') ||
    caps.includes('thinking') ||
    obj?.thinking === true ||
    idLower.includes('thinking') ||
    idLower.includes('reasoning') ||
    idLower.includes('r1') ||
    idLower.includes('o1') ||
    idLower.includes('o3') ||
    idLower.includes('tiered') ||
    idLower.includes('high') ||
    idLower.includes('medium') ||
    idLower.includes('low') ||
    nameLower.includes('thinking') ||
    nameLower.includes('high') ||
    nameLower.includes('medium') ||
    nameLower.includes('low') ||
    upstreamLower.includes('tiered') ||
    upstreamLower.includes('high') ||
    upstreamLower.includes('medium') ||
    upstreamLower.includes('low')

  if (obj?.thinking === false) {
    reasoning = false
  }

  const objCaps = obj?.caps && typeof obj.caps === 'object' ? (obj.caps as Record<string, unknown>) : null
  if (objCaps) {
    if (objCaps.vision === true) vision = true
    if (objCaps.vision === false) vision = false
    if (objCaps.audioInput === true) audioInput = true
    if (objCaps.audioInput === false) audioInput = false
    if (objCaps.reasoning === true) reasoning = true
    if (objCaps.reasoning === false) reasoning = false
  }
  return { vision, audioInput, reasoning }
}

/** Backward-compat alias used by pickerData.ts, types.ts, quota/types.ts */
export function getModelCaps(modelId: string, _model?: unknown): {
  vision: boolean; audioInput: boolean; reasoning: boolean
} {
  return resolveModelCapabilities(modelId)
}

/** Extract kind/type from a model object. */
export function getModelKind(model: unknown): string {
  if (!model || typeof model !== 'object') return ''
  const obj = model as Record<string, unknown>
  return (obj.kind || obj.type || '') as string
}

/** Get models for a provider by ID or alias. */
export function getModelsByProviderId(providerId: string): ProviderModel[] {
  return getBuiltinModels(providerId)
}
