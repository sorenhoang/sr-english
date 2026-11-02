import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { createApi, type Task } from '../lib/api'
import { Today } from '../pages/Today'
import { sampleItem } from '../test/fakes'

describe('Today extra study', () => {
  it('should fetch 10 extra new words and start studying them when the daily plan is done', async () => {
    const requested: string[] = []
    const fetchFn = async (input: RequestInfo | URL) => {
      const url = new URL(String(input))
      requested.push(url.pathname + url.search)
      if (url.pathname === '/api/v1/stats') return new Response(JSON.stringify({ streak: 3, today_new: 10, today_reviews: 0 }))
      const extra = url.searchParams.get('extra')
      const items = extra ? [sampleItem('bonus1'), sampleItem('bonus2')] : []
      return new Response(JSON.stringify({ tasks: [{ type: 'vocab_review', items: [] }, { type: 'vocab_new', items }] }))
    }
    const api = createApi({ baseUrl: 'http://api.test', getToken: async () => 't', fetchFn })
    let started: Task[] | null = null
    render(<Today api={api} onStart={(t) => (started = t)} />)

    fireEvent.click(await screen.findByRole('button', { name: 'Học thêm 10 từ mới' }))
    await waitFor(() => expect(started).not.toBeNull())
    expect(requested).toContain('/api/v1/session/today?extra=10')
    const fresh = (started as unknown as Task[]).find((t) => t.type === 'vocab_new')?.items ?? []
    expect(fresh.map((i) => i.headword)).toEqual(['bonus1', 'bonus2'])
  })
})
