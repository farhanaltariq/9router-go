import assert from 'node:assert'
import { describe, it } from 'node:test'
import {
  getProvidersByKind,
  isChatProvider,
  MEDIA_PROVIDER_KINDS,
  PROVIDER_CATALOG,
} from './providers'

describe('providers & media separation', () => {
  it('correctly classifies chat vs pure media providers', () => {
    const antigravity = PROVIDER_CATALOG.find((p) => p.id === 'antigravity')
    const github = PROVIDER_CATALOG.find((p) => p.id === 'github')
    const codex = PROVIDER_CATALOG.find((p) => p.id === 'codex')
    const commandcode = PROVIDER_CATALOG.find((p) => p.id === 'commandcode')
    const opencode = PROVIDER_CATALOG.find((p) => p.id === 'opencode')

    assert.ok(antigravity && isChatProvider(antigravity))
    assert.ok(github && isChatProvider(github))
    assert.ok(codex && isChatProvider(codex))
    assert.ok(commandcode && isChatProvider(commandcode))
    assert.ok(opencode && isChatProvider(opencode))
  })

  it('retrieves providers by kind', () => {
    const imageProviders = getProvidersByKind('image')
    assert.ok(imageProviders.some((p) => p.id === 'antigravity'))
    assert.ok(imageProviders.some((p) => p.id === 'cloudflare-ai'))
    assert.ok(imageProviders.some((p) => p.id === 'codex'))

    const llmProviders = getProvidersByKind('llm')
    assert.ok(llmProviders.some((p) => p.id === 'antigravity'))
    assert.ok(llmProviders.some((p) => p.id === 'github'))
    assert.ok(llmProviders.some((p) => p.id === 'commandcode'))
    assert.ok(llmProviders.some((p) => p.id === 'opencode'))

    assert.ok(MEDIA_PROVIDER_KINDS.length >= 6)
  })

  it('has exactly the curated provider set', () => {
    const ids = PROVIDER_CATALOG.map((p) => p.id).sort()
    const expected = [
      'antigravity', 'cloudflare-ai', 'codebuddy-cn', 'codebuddy-intl',
      'codex', 'commandcode', 'github', 'nvidia', 'ollama', 'opencode',
    ].sort()
    assert.deepStrictEqual(ids, expected)
  })
})