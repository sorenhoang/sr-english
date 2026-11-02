import { useEffect, useState, type ReactNode } from 'react'
import type { Api, Passage, PassageToken } from '../lib/api'
import { topicLabel } from '../lib/topics'

type Props = { api: Api; id: string; mode: 'reading' | 'listening'; onBack: () => void }

export function Reader({ api, id, mode, onBack }: Props) {
  const [passage, setPassage] = useState<Passage | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [selected, setSelected] = useState<number | null>(null)
  const [saving, setSaving] = useState(false)
  const [showTranscript, setShowTranscript] = useState(mode === 'reading')
  const [audioFailed, setAudioFailed] = useState(false)
  const [completed, setCompleted] = useState(false)

  useEffect(() => {
    let cancelled = false
    api
      .passage(id)
      .then((p) => {
        if (!cancelled) setPassage(p ? { ...p, tokens: p.tokens ?? [] } : null)
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Không tải được bài.')
      })
    return () => {
      cancelled = true
    }
  }, [api, id])

  if (error) {
    return (
      <section className="stack">
        <p role="alert" className="error">{error}</p>
        <button type="button" className="btn" onClick={onBack}>
          Quay lại
        </button>
      </section>
    )
  }
  if (!passage) return <p role="status" className="muted">Đang tải…</p>

  const token = selected === null ? null : passage.tokens[selected]

  const save = async () => {
    if (selected === null || !token) return
    setSaving(true)
    setActionError(null)
    try {
      await api.saveWord(passage.id, selected)
      const sameWord = (t: PassageToken, i: number) =>
        i === selected || (token.sense_id ? t.sense_id === token.sense_id : !t.sense_id && t.text.toLowerCase() === token.text.toLowerCase())
      setPassage({ ...passage, tokens: passage.tokens.map((t, i) => (sameWord(t, i) ? { ...t, saved: true } : t)) })
    } catch (e) {
      setActionError(e instanceof Error ? e.message : 'Không lưu được từ.')
    } finally {
      setSaving(false)
    }
  }

  const complete = async () => {
    try {
      await api.completePassage(passage.id)
      setCompleted(true)
    } catch (e) {
      setActionError(e instanceof Error ? e.message : 'Không đánh dấu được.')
    }
  }

  const hasAudio = mode === 'listening' && !!passage.audio_url
  // Listening without audio (or after it failed) falls back to reading the transcript.
  const transcriptVisible = showTranscript || !hasAudio || audioFailed

  return (
    <section className="stack">
      <header className="row">
        <button type="button" className="btn" onClick={onBack} aria-label="Quay lại">
          ←
        </button>
        <div className="stack tight">
          <h1>{passage.title}</h1>
          <span className="muted small">
            {passage.cefr_level} · {passage.topics.map(topicLabel).join(', ')}
          </span>
        </div>
      </header>

      {hasAudio && !audioFailed && (
        <audio controls preload="none" src={passage.audio_url ?? undefined} onError={() => {
          setAudioFailed(true)
          setShowTranscript(true)
        }} />
      )}
      {hasAudio && audioFailed && <p role="alert" className="error">Không phát được âm thanh.</p>}
      {mode === 'listening' && !passage.audio_url && <p className="muted">Bài này chưa có âm thanh.</p>}
      {hasAudio && !audioFailed && (
        <button type="button" className="btn" onClick={() => setShowTranscript((v) => !v)}>
          {showTranscript ? 'Ẩn transcript' : 'Hiện transcript'}
        </button>
      )}

      {transcriptVisible && (
        <article className="card passage">
          <PassageText passage={passage} onTap={setSelected} />
        </article>
      )}

      <p className="muted small">
        <a href={passage.source_url} target="_blank" rel="noopener noreferrer">
          Nguồn: VOA Learning English
        </a>{' '}
        · {passage.license}
      </p>
      {actionError && !token && <p role="alert" className="error">{actionError}</p>}
      <button type="button" className="btn" onClick={complete} disabled={completed}>
        {completed ? 'Đã đánh dấu xong' : 'Đánh dấu đã xong bài này'}
      </button>

      {token && (
        <div className="sheet card stack" role="dialog" aria-label={`Nghĩa của ${token.headword ?? token.text}`}>
          <div className="row between">
            <strong className="word-sm">{token.headword ?? token.text}</strong>
            <button type="button" className="btn" aria-label="Đóng" onClick={() => {
              setSelected(null)
              setActionError(null)
            }}>
              ✕
            </button>
          </div>
          {token.level && <span className="pill level">{token.level}</span>}
          <p>{token.vi_meaning ?? 'Chưa có nghĩa — từ sẽ được lưu để bổ sung sau.'}</p>
          {actionError && <p role="alert" className="error">{actionError}</p>}
          {token.saved ? (
            <p className="feedback correct">Đã có trong bộ ôn tập</p>
          ) : (
            <button type="button" className="btn primary" disabled={saving} onClick={save}>
              Thêm vào ôn tập
            </button>
          )}
        </div>
      )}
    </section>
  )
}

/** Renders the body with every word token as a tappable button; offsets are UTF-16, matching JS strings. */
function PassageText({ passage, onTap }: { passage: Passage; onTap: (index: number) => void }) {
  const parts: ReactNode[] = []
  let cursor = 0
  passage.tokens.forEach((t, i) => {
    // Skip malformed or overlapping spans instead of corrupting the rest of the text.
    if (!(t.start16 >= cursor && t.end16 > t.start16 && t.end16 <= passage.body.length)) return
    if (t.start16 > cursor) parts.push(passage.body.slice(cursor, t.start16))
    if (t.kind === 'word') {
      parts.push(
        <button key={i} type="button" className={`tok${t.saved ? ' saved' : ''}`} data-saved={String(t.saved)} onClick={() => onTap(i)}>
          {passage.body.slice(t.start16, t.end16)}
        </button>,
      )
    } else {
      parts.push(passage.body.slice(t.start16, t.end16))
    }
    cursor = t.end16
  })
  parts.push(passage.body.slice(cursor))
  return <p className="passage-text">{parts}</p>
}
