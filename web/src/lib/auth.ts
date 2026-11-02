import type { SupabaseClient } from '@supabase/supabase-js'

export type Session = { accessToken: string; email: string }

/** The slice of Supabase auth the app uses; tests pass a fake. */
export interface AuthClient {
  getSession(): Promise<Session | null>
  onChange(cb: (session: Session | null) => void): () => void
  signInWithGoogle(): Promise<void>
  signOut(): Promise<void>
}

export function supabaseAuth(client: SupabaseClient): AuthClient {
  const toSession = (s: { access_token: string; user: { email?: string } } | null): Session | null =>
    s ? { accessToken: s.access_token, email: s.user.email ?? '' } : null
  return {
    async getSession() {
      const { data, error } = await client.auth.getSession()
      if (error) throw error
      return toSession(data.session)
    },
    onChange(cb) {
      const { data } = client.auth.onAuthStateChange((_event, s) => cb(toSession(s)))
      return () => data.subscription.unsubscribe()
    },
    async signInWithGoogle() {
      const { error } = await client.auth.signInWithOAuth({
        provider: 'google',
        options: { redirectTo: window.location.origin },
      })
      if (error) throw error
    },
    async signOut() {
      const { error } = await client.auth.signOut()
      if (error) throw error
    },
  }
}
