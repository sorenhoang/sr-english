import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { createApi, type Task } from '../lib/api'
import type { Speech } from '../lib/speech'
import { Learn } from '../pages/Learn'
import { fakeFetch, sampleItem } from '../test/fakes'

const card = (senseId: string) => ({ card_id: `card-${senseId}`, sense_id: senseId, due: '2026-11-03T00:00:00Z', reps: 1, state: 1 })

function setup({ tasks, voice = true }: { tasks: Task[]; voice?: boolean }) {
  const fetchFn = fakeFetch({
    'POST /api/v1/review': { body: card('x') },
    'GET /api/v1/stats': { body: { streak: 13, today_new: 1, today_reviews: 0 } },
  })
  const api = createApi({ baseUrl: 'http://api.test', getToken: async () => 'token', fetchFn })
  const spoken: string[] = []
  const speech: Speech = { hasEnglishVoice: () => voice, speak: (text) => spoken.push(text) }
  let done = false
  render(<Learn api={api} speech={speech} tasks={tasks} onExit={() => (done = true)} />)
  return { fetchFn, spoken, isDone: () => done }
}

const newTask = (headword: string): Task[] => [
  { type: 'vocab_review', items: [] },
  { type: 'vocab_new', items: [sampleItem(headword, { distractors: ['vội vàng', 'hào phóng', 'mơ hồ'] })] },
]

describe('Learn flow', () => {
  it('should introduce a new word then ask its meaning with the distractors as other choices', async () => {
    setup({ tasks: newTask('abandon') })
    expect(screen.getByRole('heading', { name: 'abandon' })).toBeInTheDocument()
    expect(screen.getByText('/uk/')).toBeInTheDocument()
    expect(screen.getByText('nghĩa abandon')).toBeInTheDocument()
    expect(screen.getByText('An example with abandon.')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Đã hiểu, tiếp tục' }))
    const choices = await screen.findAllByRole('button', { name: /^(nghĩa abandon|vội vàng|hào phóng|mơ hồ)$/ })
    expect(choices).toHaveLength(4)
  })

  it('should show correct feedback and send rating 3 when the right meaning is chosen', async () => {
    const { fetchFn } = setup({ tasks: newTask('abandon') })
    fireEvent.click(screen.getByRole('button', { name: 'Đã hiểu, tiếp tục' }))
    fireEvent.click(await screen.findByRole('button', { name: 'nghĩa abandon' }))
    expect(await screen.findByText('Chính xác!')).toBeInTheDocument()
    const review = fetchFn.calls.find((c) => c.path === '/api/v1/review')
    expect(review?.body).toEqual({ sense_id: 'sense-abandon', rating: 3 })
  })

  it('should show wrong feedback with the answer and send rating 1 when a distractor is chosen', async () => {
    const { fetchFn } = setup({ tasks: newTask('abandon') })
    fireEvent.click(screen.getByRole('button', { name: 'Đã hiểu, tiếp tục' }))
    fireEvent.click(await screen.findByRole('button', { name: 'vội vàng' }))
    expect(await screen.findByText('Chưa đúng')).toBeInTheDocument()
    expect(screen.getByText(/Đáp án: nghĩa abandon/)).toBeInTheDocument()
    const review = fetchFn.calls.find((c) => c.path === '/api/v1/review')
    expect(review?.body).toEqual({ sense_id: 'sense-abandon', rating: 1 })
  })

  it('should accept spelling that differs only in case and surrounding spaces', async () => {
    const { spoken } = setup({ tasks: newTask('abandon') })
    fireEvent.click(screen.getByRole('button', { name: 'Đã hiểu, tiếp tục' }))
    fireEvent.click(await screen.findByRole('button', { name: 'nghĩa abandon' }))
    fireEvent.click(await screen.findByRole('button', { name: 'Tiếp tục' }))

    fireEvent.click(await screen.findByRole('button', { name: 'Phát âm thanh' }))
    expect(spoken).toContain('abandon')
    fireEvent.change(screen.getByLabelText('Từ bạn nghe được'), { target: { value: '  ABANDON ' } })
    fireEvent.click(screen.getByRole('button', { name: 'Kiểm tra' }))
    expect(await screen.findByText('Chính xác!')).toBeInTheDocument()
  })

  it('should hide listening and skip the listening exercise when no English voice exists', async () => {
    setup({ tasks: newTask('abandon'), voice: false })
    expect(screen.queryByRole('button', { name: /Nghe/ })).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Đã hiểu, tiếp tục' }))
    fireEvent.click(await screen.findByRole('button', { name: 'nghĩa abandon' }))
    fireEvent.click(await screen.findByRole('button', { name: 'Tiếp tục' }))
    expect(screen.queryByLabelText('Từ bạn nghe được')).not.toBeInTheDocument()
    expect(await screen.findByText('Xong buổi học hôm nay')).toBeInTheDocument()
  })

  it('should finish on a summary with counts and the new streak', async () => {
    const reviewItem = sampleItem('older', { card_id: 'card-older', distractors: ['a', 'b', 'c'] })
    const { isDone } = setup({
      tasks: [{ type: 'vocab_review', items: [reviewItem] }, { type: 'vocab_new', items: [] }],
      voice: false,
    })
    fireEvent.click(await screen.findByRole('button', { name: 'nghĩa older' }))
    fireEvent.click(await screen.findByRole('button', { name: 'Tiếp tục' }))
    expect(await screen.findByText('Xong buổi học hôm nay')).toBeInTheDocument()
    expect(screen.getByText('0 từ mới')).toBeInTheDocument()
    expect(screen.getByText('1 từ đã ôn')).toBeInTheDocument()
    expect(screen.getByText('Chuỗi 13 ngày')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Về trang chủ' }))
    expect(isDone()).toBe(true)
  })
})
