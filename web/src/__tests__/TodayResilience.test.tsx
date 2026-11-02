import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import App from '../App'
import { createApi } from '../lib/api'
import { fakeAuth, fakeFetch } from '../test/fakes'

describe('Today screen resilience', () => {
  it('should still render when stats and task items come back empty', async () => {
    const auth = fakeAuth({ accessToken: 't', email: 'learner@example.com' })
    const fetchFn = fakeFetch({
      'GET /api/v1/session/today': { body: { tasks: [{ type: 'vocab_review', items: null }] } },
      'GET /api/v1/stats': { body: null },
    })
    const api = createApi({ baseUrl: 'http://api.test', getToken: async () => 't', fetchFn })
    render(<App auth={auth} api={api} />)
    expect(await screen.findByText('Kế hoạch hôm nay')).toBeInTheDocument()
    expect(screen.getByText('0 từ đến hạn ôn')).toBeInTheDocument()
    expect(screen.getByText('0 ngày')).toBeInTheDocument()
  })
})
