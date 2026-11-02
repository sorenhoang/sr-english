import { useState } from 'react'
import type { AuthClient } from '../lib/auth'

export function Login({ auth }: { auth?: AuthClient }) {
  const [error, setError] = useState<string | null>(null)
  const signIn = async () => {
    if (!auth) return
    try {
      setError(null)
      await auth.signInWithGoogle()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Đăng nhập thất bại.')
    }
  }
  return (
    <main className="screen center">
      <h1 className="brand">sr-english</h1>
      <p className="muted">Học từ vựng mỗi ngày theo CEFR.</p>
      <button type="button" className="btn primary" onClick={signIn} disabled={!auth}>
        Đăng nhập với Google
      </button>
      {error && <p role="alert" className="error">{error}</p>}
    </main>
  )
}
