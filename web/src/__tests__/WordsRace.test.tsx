import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { createApi } from '../lib/api'
import { Words } from '../pages/Words'

const word = (headword: string, level: string) => ({
  card_id: `card-${headword}`, sense_id: `sense-${headword}`, headword, pos: 'noun', cefr_level: level,
  vi_meaning: `nghĩa ${headword}`, due: '2026-11-03T00:00:00Z', reps: 1, state: 2,
})

describe('Words page races', () => {
  it('should ignore a slow "load more" that resolves after the level filter changed', async () => {
    let releaseSlow: (r: Response) => void = () => {}
    const fetchFn = async (_input: RequestInfo | URL, init?: RequestInit) => {
      const body = JSON.parse(String(init?.body)) as { level?: string; offset: number }
      if (body.level === 'B2') {
        return new Response(JSON.stringify({ items: [word('bravo', 'B2')], total: 1, offset: 0, limit: 20, has_more: false }))
      }
      if (body.offset === 0) {
        return new Response(JSON.stringify({ items: [word('alpha', 'A1')], total: 2, offset: 0, limit: 1, has_more: true }))
      }
      return new Promise<Response>((resolve) => {
        releaseSlow = resolve
      })
    }
    const api = createApi({ baseUrl: 'http://api.test', getToken: async () => 't', fetchFn })
    render(<Words api={api} />)
    expect(await screen.findByText('alpha')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Xem thêm' }))
    fireEvent.change(screen.getByLabelText('Cấp độ'), { target: { value: 'B2' } })
    expect(await screen.findByText('bravo')).toBeInTheDocument()

    releaseSlow(new Response(JSON.stringify({ items: [word('stale', 'A1')], total: 2, offset: 1, limit: 1, has_more: false })))
    await new Promise((r) => setTimeout(r, 20))
    await waitFor(() => expect(screen.queryByText('stale')).not.toBeInTheDocument())
    expect(screen.queryByText('alpha')).not.toBeInTheDocument()
    expect(screen.getByText('bravo')).toBeInTheDocument()
  })
})
