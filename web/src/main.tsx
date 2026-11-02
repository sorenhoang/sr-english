import { createClient } from '@supabase/supabase-js'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import App from './App.tsx'
import './index.css'
import { createApi } from './lib/api'
import { supabaseAuth } from './lib/auth'

const { VITE_SUPABASE_URL, VITE_SUPABASE_ANON_KEY, VITE_API_URL } = import.meta.env
if (!VITE_SUPABASE_URL || !VITE_SUPABASE_ANON_KEY || !VITE_API_URL) {
  throw new Error('Missing VITE_SUPABASE_URL, VITE_SUPABASE_ANON_KEY or VITE_API_URL (see web/.env.example)')
}
const supabase = createClient(VITE_SUPABASE_URL, VITE_SUPABASE_ANON_KEY)
const auth = supabaseAuth(supabase)
const api = createApi({
  baseUrl: VITE_API_URL,
  getToken: async () => (await auth.getSession())?.accessToken ?? null,
})

const root = document.getElementById('root')
if (!root) throw new Error('missing #root')
createRoot(root).render(
  <StrictMode>
    <App auth={auth} api={api} />
  </StrictMode>,
)
