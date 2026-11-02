import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { createApi } from '../lib/api'
import { Passages } from '../pages/Passages'
import { Practice } from '../pages/Practice'
import { Reader } from '../pages/Reader'

type Call = { method: string; path: string; body?: unknown }

function apiFor(routes: Record<string, (body: unknown) => unknown>) {
  const calls: Call[] = []
  const fetchFn = async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = new URL(String(input))
    const method = init?.method ?? 'GET'
    const body = init?.body ? JSON.parse(String(init.body)) : undefined
    calls.push({ method, path: url.pathname + url.search, body })
    const handler = routes[`${method} ${url.pathname}`]
    if (!handler) return new Response('{"error":{"message":"not found"}}', { status: 404 })
    return new Response(JSON.stringify(handler(body)), { status: 200 })
  }
  return { api: createApi({ baseUrl: 'http://api.test', getToken: async () => 't', fetchFn }), calls }
}

const summary = (title: string, extra: Record<string, unknown> = {}) => ({
  id: `p-${title}`, title, cefr_level: 'B1', topics: ['education'], kinds: ['reading'],
  source_url: `https://learningenglish.voanews.com/a/${title}.html`,
  license: 'Public domain · learningenglish.voanews.com', audio_url: null, published_at: null, ...extra,
})

const passage = (extra: Record<string, unknown> = {}) => ({
  ...summary('birds', extra),
  body: 'Birds fly south.',
  sentences: ['Birds fly south.'],
  tokens: [
    { text: 'Birds', start: 0, end: 5, start16: 0, end16: 5, kind: 'word', headword: 'bird', sense_id: 's-bird', level: 'A1', counted: true, sentence: 0, saved: false, vi_meaning: 'con chim' },
    { text: 'fly', start: 6, end: 9, start16: 6, end16: 9, kind: 'word', headword: 'fly', sense_id: 's-fly', level: 'A1', counted: true, sentence: 0, saved: true, vi_meaning: 'bay' },
    { text: 'south', start: 10, end: 15, start16: 10, end16: 15, kind: 'word', counted: true, sentence: 0, saved: false },
  ],
})

describe('Practice hub', () => {
  it('should offer vocabulary, reading, listening and a disabled "To be continued" tile', () => {
    render(<Practice onSelect={() => {}} />)
    for (const name of ['Từ vựng', 'Đọc', 'Nghe']) {
      expect(screen.getByRole('button', { name: new RegExp(name) })).toBeEnabled()
    }
    expect(screen.getByRole('button', { name: /To be continued/ })).toBeDisabled()
  })
})

describe('Passages list', () => {
  it('should filter by level and Cambridge topic and show source and license', async () => {
    const { api, calls } = apiFor({ 'POST /api/v1/passages/list': () => ({ items: [summary('school')], total: 1, offset: 0, limit: 20, has_more: false }) })
    render(<Passages api={api} kind="reading" onOpen={() => {}} />)
    expect(await screen.findByText('school')).toBeInTheDocument()
    expect(screen.getByText('Public domain · learningenglish.voanews.com')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Nguồn/ })).toHaveAttribute('href', 'https://learningenglish.voanews.com/a/school.html')

    fireEvent.change(screen.getByLabelText('Cấp độ'), { target: { value: 'B1' } })
    fireEvent.change(screen.getByLabelText('Chủ đề'), { target: { value: 'education' } })
    await waitFor(() => expect(calls.at(-1)?.body).toMatchObject({ level: 'B1', topic: 'education', kind: 'reading', offset: 0 }))
  })
})

describe('Reader', () => {
  it('should open a meaning sheet for a tapped word and save it to the deck', async () => {
    const { api, calls } = apiFor({
      'GET /api/v1/passage': () => passage(),
      'POST /api/v1/passage/save-word': () => ({ card_id: 'c1', sense_id: 's-bird', headword: 'bird' }),
    })
    render(<Reader api={api} id="p-birds" mode="reading" onBack={() => {}} />)
    const word = await screen.findByRole('button', { name: 'Birds' })
    expect(word).toHaveAttribute('data-saved', 'false')
    expect(screen.getByRole('button', { name: 'fly' })).toHaveAttribute('data-saved', 'true')

    fireEvent.click(word)
    expect(await screen.findByText('con chim')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Thêm vào ôn tập' }))
    await waitFor(() => expect(calls.find((c) => c.path === '/api/v1/passage/save-word')?.body).toEqual({ passage_id: 'p-birds', token_index: 0 }))
    expect(await screen.findByText('Đã có trong bộ ôn tập')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Birds' })).toHaveAttribute('data-saved', 'true')
  })

  it('should play the audio and reveal a tappable transcript in listening mode', async () => {
    const { api, calls } = apiFor({
      'GET /api/v1/passage': () => passage({ kinds: ['reading', 'listening'], audio_url: 'https://audio.test/birds.mp3' }),
      'POST /api/v1/passage/save-word': () => ({ card_id: 'c2', sense_id: 's-south', headword: 'south' }),
    })
    const { container } = render(<Reader api={api} id="p-birds" mode="listening" onBack={() => {}} />)
    await screen.findByRole('button', { name: 'Hiện transcript' })
    expect(container.querySelector('audio')).toHaveAttribute('src', 'https://audio.test/birds.mp3')
    expect(screen.queryByRole('button', { name: 'Birds' })).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Hiện transcript' }))
    fireEvent.click(await screen.findByRole('button', { name: 'south' }))
    fireEvent.click(await screen.findByRole('button', { name: 'Thêm vào ôn tập' }))
    await waitFor(() => expect(calls.find((c) => c.path === '/api/v1/passage/save-word')?.body).toEqual({ passage_id: 'p-birds', token_index: 2 }))
  })

  it('should hide the player and keep the transcript readable when the audio fails to load', async () => {
    const { api } = apiFor({ 'GET /api/v1/passage': () => passage({ kinds: ['reading', 'listening'], audio_url: 'https://audio.test/broken.mp3' }) })
    const { container } = render(<Reader api={api} id="p-birds" mode="listening" onBack={() => {}} />)
    await screen.findByRole('button', { name: 'Hiện transcript' })
    const audio = container.querySelector('audio')
    expect(audio).not.toBeNull()
    fireEvent.error(audio as HTMLAudioElement)

    expect(await screen.findByText('Không phát được âm thanh.')).toBeInTheDocument()
    expect(container.querySelector('audio')).toBeNull()
    expect(screen.getByRole('button', { name: 'Birds' })).toBeInTheDocument()
  })
})
