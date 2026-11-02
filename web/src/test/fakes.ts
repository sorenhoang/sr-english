// Test doubles: a fake auth client and a fetch that answers per path.
import type { AuthClient, Session } from '../lib/auth'

export function fakeAuth(session: Session | null): AuthClient & { signIns: number } {
  const auth = {
    signIns: 0,
    async getSession() {
      return session
    },
    onChange() {
      return () => {}
    },
    async signInWithGoogle() {
      auth.signIns++
    },
    async signOut() {},
  }
  return auth
}

export type Route = { status?: number; body: unknown }

/** fakeFetch answers "METHOD /path" keys; unknown paths return 404. `calls` records every request. */
export function fakeFetch(routes: Record<string, Route>) {
  const calls: { method: string; path: string; body?: unknown; auth?: string }[] = []
  const fn = async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = new URL(String(input), 'http://api.test')
    const method = init?.method ?? 'GET'
    const headers = new Headers(init?.headers)
    calls.push({
      method,
      path: url.pathname,
      body: init?.body ? JSON.parse(String(init.body)) : undefined,
      auth: headers.get('Authorization') ?? undefined,
    })
    const route = routes[`${method} ${url.pathname}`]
    if (!route) {
      return new Response(JSON.stringify({ error: { code: 'not_found', message: 'not found' } }), { status: 404 })
    }
    return new Response(JSON.stringify(route.body), {
      status: route.status ?? 200,
      headers: { 'Content-Type': 'application/json' },
    })
  }
  return Object.assign(fn, { calls })
}

export const sampleItem = (headword: string, extra: Record<string, unknown> = {}) => ({
  sense_id: `sense-${headword}`,
  card_id: null,
  headword,
  pos: 'noun',
  cefr_level: 'B1',
  ipa_uk: '/uk/',
  ipa_us: '/us/',
  vi_meaning: `nghĩa ${headword}`,
  en_definition: `definition of ${headword}`,
  example_en: `An example with ${headword}.`,
  example_vi: `Ví dụ với ${headword}.`,
  distractors: ['sai một', 'sai hai', 'sai ba'],
  ...extra,
})
