import { useEffect, useRef, useState } from 'react'
import type { Api, PassageSummary } from '../lib/api'
import { LEVELS } from '../lib/levels'
import { TOPICS, topicLabel } from '../lib/topics'

// Passages fetched per list load.
const PAGE_SIZE = 20

type Props = { api: Api; kind: 'reading' | 'listening'; onOpen: (id: string) => void }

export function Passages({ api, kind, onOpen }: Props) {
  const [level, setLevel] = useState('')
  const [topic, setTopic] = useState('')
  const [items, setItems] = useState<PassageSummary[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const latest = useRef(0)

  useEffect(() => {
    const id = ++latest.current
    setError(null)
    api
      .passages({ ...(level ? { level } : {}), ...(topic ? { topic } : {}), kind, offset: 0, limit: PAGE_SIZE })
      .then((page) => {
        if (id === latest.current) setItems(page?.items ?? [])
      })
      .catch((e: unknown) => {
        if (id !== latest.current) return
        setItems([])
        setError(e instanceof Error ? e.message : 'Không tải được danh sách bài.')
      })
  }, [api, kind, level, topic])

  return (
    <section className="stack">
      <h1>{kind === 'reading' ? 'Đọc' : 'Nghe'}</h1>
      <div className="row">
        <div className="stack grow">
          <label htmlFor="passages-level">Cấp độ</label>
          <select id="passages-level" value={level} onChange={(e) => setLevel(e.target.value)}>
            <option value="">Tất cả</option>
            {LEVELS.map((l) => (
              <option key={l} value={l}>
                {l}
              </option>
            ))}
          </select>
        </div>
        <div className="stack grow">
          <label htmlFor="passages-topic">Chủ đề</label>
          <select id="passages-topic" value={topic} onChange={(e) => setTopic(e.target.value)}>
            <option value="">Tất cả</option>
            {TOPICS.map((t) => (
              <option key={t.id} value={t.id}>
                {t.label}
              </option>
            ))}
          </select>
        </div>
      </div>
      {error && <p role="alert" className="error">{error}</p>}
      {items === null && <p role="status" className="muted">Đang tải…</p>}
      {items?.length === 0 && !error && <p className="muted">Chưa có bài phù hợp với bộ lọc này.</p>}
      {items?.map((p) => (
        <article key={p.id} className="card stack">
          <button type="button" className="link-title" onClick={() => onOpen(p.id)}>
            {p.title}
          </button>
          <div className="row wrap">
            <span className="pill level">{p.cefr_level}</span>
            {p.topics.map((t) => (
              <span key={t} className="pill">{topicLabel(t)}</span>
            ))}
          </div>
          <p className="muted small">
            <a href={p.source_url} target="_blank" rel="noopener noreferrer">
              Nguồn: VOA Learning English
            </a>
          </p>
          <p className="muted small">{p.license}</p>
        </article>
      ))}
    </section>
  )
}
