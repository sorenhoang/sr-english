import { useEffect, useState } from 'react'
import { Icon, type IconName } from '../components/Icon'
import { ApiError, type Api, type Stats, type Task } from '../lib/api'
import { LEVELS } from '../lib/levels'

type State =
  | { kind: 'loading' }
  | { kind: 'error'; message: string }
  | { kind: 'ready'; tasks: Task[]; stats: Stats; level: string | null }

const FORBIDDEN = 'Tài khoản này không có quyền truy cập'
const EMPTY_STATS: Stats = { streak: 0, today_new: 0, today_reviews: 0 }

type PassageTask = { kind: 'reading' | 'listening'; id: string }

type Props = {
  api: Api
  onStart: (tasks: Task[]) => void
  onOpenPassage?: (task: PassageTask) => void
}

const dateLabel = () =>
  new Intl.DateTimeFormat('vi-VN', { weekday: 'long', day: 'numeric', month: 'long' }).format(new Date())

export function Today({ api, onStart, onOpenPassage }: Props) {
  const [state, setState] = useState<State>({ kind: 'loading' })

  useEffect(() => {
    let cancelled = false
    setState({ kind: 'loading' })
    // Settings only decorate the level card; the plan still shows if they fail to load.
    const level = api.settings().then((s) => s?.level ?? null, () => null)
    Promise.all([api.today(), api.stats(), level])
      .then(([today, stats, lvl]) => {
        if (!cancelled) setState({ kind: 'ready', tasks: today?.tasks ?? [], stats: stats ?? EMPTY_STATS, level: lvl })
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

  const items = (type: string) => state.tasks.find((t) => t.type === type)?.items ?? []
  const reviews = items('vocab_review').length
  const fresh = items('vocab_new').length
  const nothing = reviews + fresh === 0
  const passage = (kind: 'reading' | 'listening') => (items(kind)[0] as unknown as { id: string; title: string } | undefined) ?? null
  const reading = passage('reading')
  const listening = passage('listening')

  return (
    <section className="stack">
      <header className="row between">
        <div className="stack tight">
          <span className="muted small">{dateLabel()}</span>
          <h1>Hôm nay</h1>
        </div>
        <span className="pill streak row tight-row">
          <Icon name="flame" size={18} />
          <span>{state.stats.streak ?? 0} ngày</span>
        </span>
      </header>

      <div className="card stack">
        <div className="row between">
          <h2>Kế hoạch hôm nay</h2>
        </div>
        <TaskRow icon="repeat" tone="indigo" title="Ôn tập từ" meta={`${reviews} từ đến hạn ôn`} />
        <TaskRow icon="plus" tone="teal" title="Học từ mới" meta={`${fresh} từ mới`} />
        {reading && (
          <TaskRow icon="book" tone="amber" title={`Đọc: ${reading.title}`} meta="Bài đọc gợi ý"
            onClick={onOpenPassage ? () => onOpenPassage({ kind: 'reading', id: reading.id }) : undefined} />
        )}
        {listening && (
          <TaskRow icon="headphones" tone="pink" title={`Nghe: ${listening.title}`} meta="Bài nghe gợi ý"
            onClick={onOpenPassage ? () => onOpenPassage({ kind: 'listening', id: listening.id }) : undefined} />
        )}
        <button type="button" className="btn primary" disabled={nothing} onClick={() => onStart(state.tasks)}>
          {nothing ? 'Hôm nay đã xong' : 'Bắt đầu'}
        </button>
        {nothing && (
          <p className="muted small">
            Không có từ để học: hoặc bạn đã xong, hoặc từ ở cấp độ này chưa có nghĩa tiếng Việt (cần chạy enrich).
          </p>
        )}
      </div>

      {state.level && (
        <div className="card stack">
          <h2>Cấp độ {state.level}</h2>
          <div className="levels">
            {LEVELS.map((l) => {
              const rank = LEVELS.indexOf(l) - LEVELS.indexOf(state.level ?? 'A1')
              return (
                <span key={l} className={`level-chip ${rank < 0 ? 'past' : rank === 0 ? 'current' : ''}`}>
                  {l}
                </span>
              )
            })}
          </div>
        </div>
      )}
    </section>
  )
}

type Tone = 'indigo' | 'teal' | 'amber' | 'pink'

function TaskRow({ icon, tone, title, meta, onClick }: { icon: IconName; tone: Tone; title: string; meta: string; onClick?: () => void }) {
  const body = (
    <>
      <span className={`icon-box ${tone}`}>
        <Icon name={icon} />
      </span>
      <span className="task-text">
        <strong>{title}</strong>
        <span className="muted">{meta}</span>
      </span>
    </>
  )
  return onClick ? (
    <button type="button" className="task task-button" onClick={onClick}>
      {body}
    </button>
  ) : (
    <div className="task">{body}</div>
  )
}
