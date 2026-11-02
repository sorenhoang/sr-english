import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import App from '../App'
import { createApi } from '../lib/api'
import { fakeAuth, fakeFetch, sampleItem } from '../test/fakes'

const session = { accessToken: 'token-123', email: 'learner@example.com' }

function renderApp(auth: ReturnType<typeof fakeAuth>, fetchFn: ReturnType<typeof fakeFetch>) {
  const api = createApi({ baseUrl: 'http://api.test', getToken: async () => (await auth.getSession())?.accessToken ?? null, fetchFn })
  return render(<App auth={auth} api={api} />)
}

const okRoutes = {
  'GET /api/v1/session/today': {
    body: {
      tasks: [
        { type: 'vocab_review', items: [sampleItem('older'), sampleItem('newer')] },
        { type: 'vocab_new', items: [sampleItem('fresh1'), sampleItem('fresh2'), sampleItem('fresh3')] },
      ],
    },
  },
  'GET /api/v1/stats': { body: { streak: 12, today_new: 0, today_reviews: 0 } },
}

describe('Today screen', () => {
  it('should show Google sign-in and call no API when signed out', async () => {
    const fetchFn = fakeFetch(okRoutes)
    renderApp(fakeAuth(null), fetchFn)
    expect(await screen.findByRole('button', { name: 'Đăng nhập với Google' })).toBeInTheDocument()
    expect(fetchFn.calls).toHaveLength(0)
  })

  it('should show today tasks and streak from the API when signed in', async () => {
    const fetchFn = fakeFetch(okRoutes)
    renderApp(fakeAuth(session), fetchFn)
    expect(await screen.findByText('2 từ đến hạn ôn')).toBeInTheDocument()
    expect(screen.getByText('3 từ mới')).toBeInTheDocument()
    expect(screen.getByText('12 ngày')).toBeInTheDocument()
    expect(fetchFn.calls.every((c) => c.auth === 'Bearer token-123')).toBe(true)
  })

  it('should explain access is denied when the API returns 403', async () => {
    const fetchFn = fakeFetch({
      'GET /api/v1/session/today': { status: 403, body: { error: { code: 'forbidden', message: 'account is not allowed' } } },
      'GET /api/v1/stats': { status: 403, body: { error: { code: 'forbidden', message: 'account is not allowed' } } },
    })
    renderApp(fakeAuth(session), fetchFn)
    expect(await screen.findByText('Tài khoản này không có quyền truy cập')).toBeInTheDocument()
  })

  it('should show the server error message instead of a blank screen when the API fails', async () => {
    const fetchFn = fakeFetch({
      'GET /api/v1/session/today': { status: 500, body: { error: { code: 'internal', message: 'database unavailable' } } },
      'GET /api/v1/stats': okRoutes['GET /api/v1/stats'],
    })
    renderApp(fakeAuth(session), fetchFn)
    expect(await screen.findByText('database unavailable')).toBeInTheDocument()
  })
})
