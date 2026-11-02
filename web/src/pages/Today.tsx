import { useEffect, useState } from 'react'
import { ApiError, type Api, type Stats, type Task } from '../lib/api'

type State =
  | { kind: 'loading' }
  | { kind: 'error'; message: string }
  | { kind: 'ready'; tasks: Task[]; stats: Stats }

const FORBIDDEN = 'Tài khoản này không có quyền truy cập'
const EMPTY_STATS: Stats = { streak: 0, today_new: 0, today_reviews: 0 }

export function Today({ api, onStart }: { api: Api; onStart: (tasks: Task[]) => void }) {
  const [state, setState] = useState<State>({ kind: 'loading' })

  useEffect(() => {
    let cancelled = false
    setState({ kind: 'loading' })
    Promise.all([api.today(), api.stats()])
      .then(([today, stats]) => {
        if (!cancelled) setState({ kind: 'ready', tasks: today?.tasks ?? [], stats: stats ?? EMPTY_STATS })
      })
      .catch((e: unknown) => {
        if (cancelled) return
        const message = e instanceof ApiError && e.status === 403 ? FORBIDDEN : e instanceof Error ? e.message : 'Có lỗi xảy ra.'
        setState({ kind: 'error', message })
      })
    return () => {
      cancelled = true
    }
  }, [api])

  if (state.kind === 'loading') return <p role="status" className="muted pad">Đang tải…</p>
  if (state.kind === 'error') return <p role="alert" className="error pad">{state.message}</p>

  const count = (type: string) => state.tasks.find((t) => t.type === type)?.items?.length ?? 0
  const reviews = count('vocab_review')
  const fresh = count('vocab_new')
  const nothing = reviews + fresh === 0

  return (
    <section className="stack">
      <header className="row between">
        <h1>Hôm nay</h1>
        <span className="pill streak">{state.stats.streak ?? 0} ngày</span>
      </header>
      <div className="card stack">
        <h2>Kế hoạch hôm nay</h2>
        <div className="task">
          <strong>Ôn tập từ</strong>
          <span className="muted">{reviews} từ đến hạn ôn</span>
        </div>
        <div className="task">
          <strong>Học từ mới</strong>
          <span className="muted">{fresh} từ mới</span>
        </div>
        <button type="button" className="btn primary" disabled={nothing} onClick={() => onStart(state.tasks)}>
          {nothing ? 'Hôm nay đã xong' : 'Bắt đầu'}
        </button>
      </div>
    </section>
  )
}
