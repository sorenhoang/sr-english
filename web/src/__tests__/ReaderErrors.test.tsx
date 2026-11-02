import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { createApi } from '../lib/api'
import { Reader } from '../pages/Reader'

const passage = (audio: string | null) => ({
  id: 'p1', title: 'birds', cefr_level: 'B1', topics: ['education'], kinds: audio ? ['reading', 'listening'] : ['reading'],
  source_url: 'https://learningenglish.voanews.com/a/birds.html', license: 'Public domain · learningenglish.voanews.com',
  audio_url: audio, published_at: null, body: 'Birds fly.', sentences: ['Birds fly.'],
  tokens: [{ text: 'Birds', start16: 0, end16: 5, kind: 'word', headword: 'bird', sense_id: 's1', saved: false, vi_meaning: 'con chim' }],
})

function apiWith(audio: string | null, saveStatus = 200) {
  const fetchFn = async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = new URL(String(input)).pathname
    if (path === '/api/v1/passage') return new Response(JSON.stringify(passage(audio)))
    if (path === '/api/v1/passage/save-word' && init?.method === 'POST') {
      return new Response(JSON.stringify({ error: { message: 'database unavailable' } }), { status: saveStatus })
    }
    return new Response('{}', { status: 404 })
  }
  return createApi({ baseUrl: 'http://api.test', getToken: async () => 't', fetchFn })
}

describe('Reader error handling', () => {
  it('should keep the passage on screen and show the error in the sheet when saving fails', async () => {
    render(<Reader api={apiWith(null, 500)} id="p1" mode="reading" onBack={() => {}} />)
    fireEvent.click(await screen.findByRole('button', { name: 'Birds' }))
    fireEvent.click(screen.getByRole('button', { name: 'Thêm vào ôn tập' }))
    expect(await screen.findByText('database unavailable')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Birds' })).toBeInTheDocument()
    expect(screen.getByText('con chim')).toBeInTheDocument()
  })

  it('should show the transcript when a listening passage has no audio', async () => {
    render(<Reader api={apiWith(null)} id="p1" mode="listening" onBack={() => {}} />)
    expect(await screen.findByRole('button', { name: 'Birds' })).toBeInTheDocument()
    expect(screen.getByText('Bài này chưa có âm thanh.')).toBeInTheDocument()
  })
})
