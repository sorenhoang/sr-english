import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { createApi } from '../lib/api'
import { SettingsPage } from '../pages/Settings'
import { Words } from '../pages/Words'
import type { Route } from '../test/fakes'

const word = (headword: string, level = 'A1') => ({
  card_id: `card-${headword}`, sense_id: `sense-${headword}`, headword, pos: 'noun', cefr_level: level,
  vi_meaning: `nghĩa ${headword}`, due: '2026-11-03T00:00:00Z', reps: 1, state: 2,
})

function apiWith(routes: Record<string, Route | ((body: unknown) => Route)>) {
  const calls: { path: string; body: unknown }[] = []
  const fetchFn = async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = new URL(String(input))
    const body = init?.body ? JSON.parse(String(init.body)) : undefined
    calls.push({ path: url.pathname, body })
    const r = routes[`${init?.method ?? 'GET'} ${url.pathname}`]
    const route = typeof r === 'function' ? r(body) : r
    if (!route) return new Response('{}', { status: 404 })
    return new Response(JSON.stringify(route.body), { status: route.status ?? 200 })
  }
  return { api: createApi({ baseUrl: 'http://api.test', getToken: async () => 't', fetchFn }), calls }
}

describe('Words page', () => {
  it('should page through my words and filter by level', async () => {
    const { api, calls } = apiWith({
      'POST /api/v1/words/list': (body) => {
        const b = body as { level?: string; offset: number }
        if (b.level === 'B2') return { body: { items: [word('three', 'B2')], total: 1, offset: 0, limit: 20, has_more: false } }
        if (b.offset === 0) return { body: { items: [word('one'), word('two')], total: 3, offset: 0, limit: 2, has_more: true } }
        return { body: { items: [word('three', 'B2')], total: 3, offset: 2, limit: 2, has_more: false } }
      },
    })
    render(<Words api={api} />)
    expect(await screen.findByText('one')).toBeInTheDocument()
    expect(screen.getByText('two')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Xem thêm' }))
    expect(await screen.findByText('three')).toBeInTheDocument()
    expect(calls.at(-1)?.body).toMatchObject({ offset: 2 })
    expect(screen.queryByRole('button', { name: 'Xem thêm' })).not.toBeInTheDocument()

    fireEvent.change(screen.getByLabelText('Cấp độ'), { target: { value: 'B2' } })
    await waitFor(() => expect(calls.at(-1)?.body).toMatchObject({ level: 'B2', offset: 0 }))
    await waitFor(() => expect(screen.queryByText('one')).not.toBeInTheDocument())
    expect(screen.getByText('three')).toBeInTheDocument()
  })
})

describe('Settings page', () => {
  it('should save changes and show the values the server returns', async () => {
    const { api, calls } = apiWith({
      'GET /api/v1/settings': { body: { level: 'A1', daily_new: 10, timezone: 'UTC' } },
      'POST /api/v1/settings/update': { body: { level: 'B1', daily_new: 20, timezone: 'Asia/Ho_Chi_Minh' } },
    })
    render(<SettingsPage api={api} />)
    const level = (await screen.findByLabelText('Cấp độ')) as HTMLSelectElement
    await waitFor(() => expect(level.value).toBe('A1'))

    fireEvent.change(level, { target: { value: 'B1' } })
    fireEvent.change(screen.getByLabelText('Số từ mới mỗi ngày'), { target: { value: '15' } })
    fireEvent.change(screen.getByLabelText('Múi giờ'), { target: { value: 'Asia/Ho_Chi_Minh' } })
    fireEvent.click(screen.getByRole('button', { name: 'Lưu' }))

    await waitFor(() =>
      expect(calls.find((c) => c.path === '/api/v1/settings/update')?.body).toEqual({
        level: 'B1', daily_new: 15, timezone: 'Asia/Ho_Chi_Minh',
      }),
    )
    await waitFor(() => expect((screen.getByLabelText('Số từ mới mỗi ngày') as HTMLInputElement).value).toBe('20'))
  })

  it('should show the server validation message next to the form', async () => {
    const { api } = apiWith({
      'GET /api/v1/settings': { body: { level: 'A1', daily_new: 10, timezone: 'UTC' } },
      'POST /api/v1/settings/update': { status: 400, body: { error: { code: 'validation', message: 'timezone is invalid' } } },
    })
    render(<SettingsPage api={api} />)
    await screen.findByLabelText('Múi giờ')
    fireEvent.change(screen.getByLabelText('Múi giờ'), { target: { value: 'Mars/Olympus' } })
    fireEvent.click(screen.getByRole('button', { name: 'Lưu' }))
    expect(await screen.findByText('timezone is invalid')).toBeInTheDocument()
  })
})
