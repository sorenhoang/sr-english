import { useCallback, useEffect, useRef, useState } from 'react'
import type { Api, Word } from '../lib/api'
import { LEVELS } from '../lib/levels'

// Words fetched per "Xem thêm" click.
const PAGE_SIZE = 20

export function Words({ api }: { api: Api }) {
  const [level, setLevel] = useState('')
  const [items, setItems] = useState<Word[]>([])
  const [hasMore, setHasMore] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  // Each load gets an id; responses from superseded loads (e.g. before a filter change) are dropped.
  const latest = useRef(0)

  const load = useCallback(
    async (offset: number, replace: boolean) => {
      const id = ++latest.current
      setLoading(true)
      setError(null)
      try {
        const page = await api.words({ ...(level ? { level } : {}), offset, limit: PAGE_SIZE })
        if (id !== latest.current) return
        const fetched = page?.items ?? []
        setItems((prev) => (replace ? fetched : [...prev, ...fetched]))
        setHasMore(page?.has_more ?? false)
      } catch (e) {
        if (id !== latest.current) return
        if (replace) {
          setItems([])
          setHasMore(false)
        }
        setError(e instanceof Error ? e.message : 'Không tải được danh sách từ.')
      } finally {
        if (id === latest.current) setLoading(false)
      }
    },
    [api, level],
  )

  useEffect(() => {
    void load(0, true)
  }, [load])

  return (
    <section className="stack">
      <h1>Từ của tôi</h1>
      <div className="stack">
        <label htmlFor="words-level">Cấp độ</label>
        <select id="words-level" value={level} onChange={(e) => setLevel(e.target.value)}>
          <option value="">Tất cả</option>
          {LEVELS.map((l) => (
            <option key={l} value={l}>
              {l}
            </option>
          ))}
        </select>
      </div>
      {error && <p role="alert" className="error">{error}</p>}
      {loading && items.length === 0 && <p role="status" className="muted">Đang tải…</p>}
      {!loading && !error && items.length === 0 && <p className="muted">Chưa có từ nào. Học vài từ mới để bắt đầu.</p>}
      <div className="list card">
        {items.map((w) => (
          <div key={w.card_id} className="row between">
            <div>
              <strong>{w.headword}</strong>
              <div className="muted">{w.vi_meaning}</div>
            </div>
            <span className="pill level">{w.cefr_level}</span>
          </div>
        ))}
      </div>
      {hasMore && (
        <button type="button" className="btn" disabled={loading} onClick={() => load(items.length, false)}>
          Xem thêm
        </button>
      )}
    </section>
  )
}
