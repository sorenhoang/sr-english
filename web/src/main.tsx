import { createClient } from '@supabase/supabase-js'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import App from './App.tsx'
import './index.css'
import { createApi } from './lib/api'
import { supabaseAuth } from './lib/auth'

const { VITE_SUPABASE_URL, VITE_SUPABASE_ANON_KEY, VITE_API_URL } = import.meta.env
if (!VITE_SUPABASE_URL || !VITE_SUPABASE_ANON_KEY) {
  throw new Error('Missing VITE_SUPABASE_URL or VITE_SUPABASE_ANON_KEY (see web/.env.example)')
}
const supabase = createClient(VITE_SUPABASE_URL, VITE_SUPABASE_ANON_KEY)
const auth = supabaseAuth(supabase)
const api = createApi({
  // Empty means same origin: on Vercel, web/vercel.json proxies /api/* to the API project (no CORS needed).
  baseUrl: VITE_API_URL ?? '',
  getToken: async () => (await auth.getSession())?.accessToken ?? null,
})

const root = document.getElementById('root')
if (!root) throw new Error('missing #root')
createRoot(root).render(
  <StrictMode>
    <App auth={auth} api={api} />
  </StrictMode>,
)
