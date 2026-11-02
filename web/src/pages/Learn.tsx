import { useMemo, useState } from 'react'
import type { Api, StudyItem, Task } from '../lib/api'
import type { Speech } from '../lib/speech'

type Step = { kind: 'card' | 'mcq' | 'spell'; item: StudyItem; isNew: boolean }

// FSRS ratings sent for a multiple-choice answer.
const RATING_CORRECT = 3
const RATING_WRONG = 1

function buildSteps(tasks: Task[], withListening: boolean): Step[] {
  const items = (type: string) => tasks.find((t) => t.type === type)?.items ?? []
  const steps: Step[] = []
  const add = (item: StudyItem, isNew: boolean) => {
    // A word without a meaning (e.g. one the user saved themselves) can't be quizzed yet.
    if (!item.vi_meaning) return
    if (isNew) steps.push({ kind: 'card', item, isNew })
    steps.push({ kind: 'mcq', item, isNew })
    if (withListening) steps.push({ kind: 'spell', item, isNew })
  }
  items('vocab_review').forEach((it) => add(it, false))
  items('vocab_new').forEach((it) => add(it, true))
  return steps
}

const normalize = (s: string) => s.trim().toLowerCase()

type Props = { api: Api; speech: Speech; tasks: Task[]; onExit: () => void }

export function Learn({ api, speech, tasks, onExit }: Props) {
  const canListen = useMemo(() => speech.hasEnglishVoice(), [speech])
  const steps = useMemo(() => buildSteps(tasks, canListen), [tasks, canListen])
  const [index, setIndex] = useState(0)
  const [answered, setAnswered] = useState<{ correct: boolean } | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [done, setDone] = useState<{ fresh: number; reviewed: number; streak: number | null } | null>(null)

  const next = async () => {
    setAnswered(null)
    setError(null)
    if (index + 1 < steps.length) {
      setIndex(index + 1)
      return
    }
    const fresh = steps.filter((s) => s.kind === 'mcq' && s.isNew).length
    const reviewed = steps.filter((s) => s.kind === 'mcq' && !s.isNew).length
    try {
      const stats = await api.stats()
      setDone({ fresh, reviewed, streak: stats?.streak ?? 0 })
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Không tải được thống kê.')
      setDone({ fresh, reviewed, streak: null })
    }
  }

  if (done) {
    return (
      <section className="stack center">
        <h1>Xong buổi học hôm nay</h1>
        {done.streak !== null && <span className="pill streak">Chuỗi {done.streak} ngày</span>}
        <p>{done.fresh} từ mới</p>
        <p>{done.reviewed} từ đã ôn</p>
        {error && <p role="alert" className="error">{error}</p>}
        <button type="button" className="btn primary" onClick={onExit}>
          Về trang chủ
        </button>
      </section>
    )
  }

  const step = steps[index]
  if (!step) {
    return (
      <section className="stack center">
        <p>Không có từ nào để học.</p>
        <button type="button" className="btn primary" onClick={onExit}>
          Về trang chủ
        </button>
      </section>
    )
  }

  return (
    <section className="stack">
      <header className="row">
        <button type="button" className="btn" onClick={onExit} aria-label="Thoát buổi học">
          ✕
        </button>
        <div className="progress" style={{ flexGrow: 1 }} role="progressbar" aria-label="Tiến độ buổi học" aria-valuemin={0} aria-valuemax={steps.length} aria-valuenow={index}>
          <div style={{ width: `${(index / steps.length) * 100}%` }} />
        </div>
      </header>
      {step.kind === 'card' && <WordCard item={step.item} canListen={canListen} speech={speech} onNext={next} />}
      {step.kind === 'mcq' && (
        <MeaningQuestion
          key={`mcq-${index}`}
          item={step.item}
          answered={answered}
          onAnswer={async (correct) => {
            setAnswered({ correct })
            try {
              await api.review(step.item.sense_id, correct ? RATING_CORRECT : RATING_WRONG)
            } catch (e) {
              setError(e instanceof Error ? e.message : 'Không lưu được kết quả.')
            }
          }}
        />
      )}
      {step.kind === 'spell' && (
        <Spelling key={`spell-${index}`} item={step.item} speech={speech} answered={answered} onCheck={(correct) => setAnswered({ correct })} />
      )}
      {error && <p role="alert" className="error">{error}</p>}
      {answered && (
        <div className="stack">
          <div className={`feedback ${answered.correct ? 'correct' : 'wrong'}`} role="status">
            <span aria-hidden="true">{answered.correct ? '✓ ' : '✕ '}</span>
            {answered.correct ? 'Chính xác!' : 'Chưa đúng'}
            {!answered.correct && <p>Đáp án: {step.kind === 'spell' ? step.item.headword : step.item.vi_meaning}</p>}
          </div>
          <button type="button" className="btn primary" onClick={next}>
            Tiếp tục
          </button>
        </div>
      )}
    </section>
  )
}

function WordCard({ item, canListen, speech, onNext }: { item: StudyItem; canListen: boolean; speech: Speech; onNext: () => void }) {
  return (
    <div className="card stack">
      <span className="pill level">Từ mới · {item.cefr_level}</span>
      <h1 className="word">{item.headword}</h1>
      <div className="row">
        <span className="muted">{item.pos}</span>
        {item.ipa_uk && <span className="ipa">{item.ipa_uk}</span>}
        {item.ipa_us && <span className="ipa">{item.ipa_us}</span>}
        {canListen && (
          <button type="button" className="btn" onClick={() => speech.speak(item.headword)}>
            Nghe
          </button>
        )}
      </div>
      <p><strong>{item.vi_meaning}</strong></p>
      {item.en_definition && <p className="muted">{item.en_definition}</p>}
      {item.example_en && <p>{item.example_en}</p>}
      {item.example_vi && <p className="muted">{item.example_vi}</p>}
      <button type="button" className="btn primary" onClick={onNext}>
        Đã hiểu, tiếp tục
      </button>
    </div>
  )
}

function MeaningQuestion({ item, answered, onAnswer }: { item: StudyItem; answered: { correct: boolean } | null; onAnswer: (correct: boolean) => void }) {
  const correct = item.vi_meaning ?? ''
  const choices = useMemo(
    () => [...new Set([correct, ...(item.distractors ?? [])])].filter(Boolean).sort((a, b) => a.localeCompare(b, 'vi')),
    [correct, item.distractors],
  )
  const [picked, setPicked] = useState<string | null>(null)
  return (
    <div className="stack">
      <p className="muted">Chọn nghĩa đúng</p>
      <h1 className="word">{item.headword}</h1>
      {choices.map((choice) => {
        const state = answered && choice === correct ? 'correct' : answered && choice === picked ? 'wrong' : ''
        return (
          <button
            key={choice}
            type="button"
            className={`choice ${state}`}
            disabled={!!answered}
            onClick={() => {
              setPicked(choice)
              onAnswer(choice === correct)
            }}
          >
            {choice}
          </button>
        )
      })}
    </div>
  )
}

function Spelling({ item, speech, answered, onCheck }: { item: StudyItem; speech: Speech; answered: { correct: boolean } | null; onCheck: (correct: boolean) => void }) {
  const [value, setValue] = useState('')
  return (
    <div className="stack">
      <p className="muted">Nghe và gõ lại</p>
      <button type="button" className="btn primary" aria-label="Phát âm thanh" onClick={() => speech.speak(item.headword)}>
        🔊
      </button>
      <label htmlFor="spell-input">Từ bạn nghe được</label>
      <input id="spell-input" autoComplete="off" autoCapitalize="off" spellCheck={false} value={value} onChange={(e) => setValue(e.target.value)} disabled={!!answered} />
      <p className="muted">Gợi ý: {item.vi_meaning}</p>
      {!answered && (
        <button type="button" className="btn primary" onClick={() => onCheck(normalize(value) === normalize(item.headword))}>
          Kiểm tra
        </button>
      )}
    </div>
  )
}
