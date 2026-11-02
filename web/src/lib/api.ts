export type StudyItem = {
  sense_id: string
  card_id: string | null
  headword: string
  pos: string
  cefr_level: string
  ipa_uk: string | null
  ipa_us: string | null
  vi_meaning: string | null
  en_definition: string | null
  example_en: string | null
  example_vi: string | null
  distractors: string[]
}

export type Task = { type: 'vocab_review' | 'vocab_new' | string; items: StudyItem[] }
export type Stats = { streak: number; today_new: number; today_reviews: number }
export type Settings = { level: string; daily_new: number; timezone: string }
export type Word = {
  card_id: string
  sense_id: string
  headword: string
  pos: string
  cefr_level: string
  vi_meaning: string | null
  due: string
  reps: number
  state: number
}
export type Page<T> = { items: T[]; total: number; offset: number; limit: number; has_more: boolean }
export type PassageSummary = {
  id: string
  title: string
  cefr_level: string
  topics: string[]
  kinds: string[]
  source_url: string
  license: string
  audio_url: string | null
  published_at: string | null
}
export type PassageToken = {
  text: string
  start16: number
  end16: number
  kind: 'word' | 'name' | 'number' | 'url' | string
  headword?: string
  sense_id?: string
  level?: string
  saved: boolean
  vi_meaning?: string
}
export type Passage = PassageSummary & { body: string; tokens: PassageToken[]; sentences: string[] }
export type SavedWord = { card_id: string; sense_id: string; headword: string }

export type Card = { card_id: string; sense_id: string; due: string; reps: number; state: number }

/** ApiError carries the backend's {"error": {"message"}} text so screens never show raw HTTP errors. */
export class ApiError extends Error {
  readonly status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

type Options = {
  baseUrl: string
  getToken: () => Promise<string | null>
  fetchFn?: typeof fetch
}

export function createApi({ baseUrl, getToken, fetchFn = fetch }: Options) {
  async function call<T>(method: string, path: string, body?: unknown): Promise<T> {
    const token = await getToken()
    const headers: Record<string, string> = {}
    if (token) headers.Authorization = `Bearer ${token}`
    if (body !== undefined) headers['Content-Type'] = 'application/json'
    let res: Response
    try {
      res = await fetchFn(`${baseUrl}${path}`, {
        method,
        headers,
        body: body === undefined ? undefined : JSON.stringify(body),
      })
    } catch {
      throw new ApiError(0, 'Không kết nối được máy chủ.')
    }
    const data = await res.json().catch(() => null)
    if (!res.ok) {
      const message = (data as { error?: { message?: string } } | null)?.error?.message
      throw new ApiError(res.status, message || `Yêu cầu thất bại (${res.status}).`)
    }
    return data as T
  }

  return {
    today: (extra = 0) => call<{ tasks: Task[] }>('GET', `/api/v1/session/today${extra > 0 ? `?extra=${extra}` : ''}`),
    stats: () => call<Stats>('GET', '/api/v1/stats'),
    review: (senseId: string, rating: number) => call<Card>('POST', '/api/v1/review', { sense_id: senseId, rating }),
    settings: () => call<Settings>('GET', '/api/v1/settings'),
    updateSettings: (s: Settings) => call<Settings>('POST', '/api/v1/settings/update', s),
    words: (req: { level?: string; offset: number; limit: number }) => call<Page<Word>>('POST', '/api/v1/words/list', req),
    passages: (req: { level?: string; topic?: string; kind?: string; offset: number; limit: number }) =>
      call<Page<PassageSummary>>('POST', '/api/v1/passages/list', req),
    passage: (id: string) => call<Passage>('GET', `/api/v1/passage?id=${encodeURIComponent(id)}`),
    completePassage: (id: string) => call<{ completed: boolean }>('POST', `/api/v1/passage/complete?id=${encodeURIComponent(id)}`),
    saveWord: (passageId: string, tokenIndex: number) =>
      call<SavedWord>('POST', '/api/v1/passage/save-word', { passage_id: passageId, token_index: tokenIndex }),
  }
}

export type Api = ReturnType<typeof createApi>
