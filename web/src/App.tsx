import { useEffect, useState } from 'react'
import type { Api, Task } from './lib/api'
import type { AuthClient, Session } from './lib/auth'
import { browserSpeech, type Speech } from './lib/speech'
import { Learn } from './pages/Learn'
import { Login } from './pages/Login'
import { Passages } from './pages/Passages'
import { Practice, type PracticeChoice } from './pages/Practice'
import { Reader } from './pages/Reader'
import { SettingsPage } from './pages/Settings'
import { Words } from './pages/Words'
import { Today } from './pages/Today'

// Without an auth client (e.g. a bare render) the app stays on the sign-in screen.
type Tab = 'today' | 'practice' | 'words' | 'settings'

const TABS: { id: Tab; label: string }[] = [
  { id: 'today', label: 'Hôm nay' },
  { id: 'practice', label: 'Luyện tập' },
  { id: 'words', label: 'Từ của tôi' },
  { id: 'settings', label: 'Cài đặt' },
]

type Props = { auth?: AuthClient; api?: Api; speech?: Speech }

function App({ auth, api, speech = browserSpeech() }: Props) {
  const [session, setSession] = useState<Session | null | undefined>(auth ? undefined : null)
  const [learning, setLearning] = useState<Task[] | null>(null)
  const [visit, setVisit] = useState(0)
  const [tab, setTab] = useState<Tab>('today')
  const [practice, setPractice] = useState<Exclude<PracticeChoice, 'vocab'> | null>(null)
  const [reading, setReading] = useState<string | null>(null)

  useEffect(() => {
    if (!auth) return
    auth
      .getSession()
      .then(setSession)
      .catch(() => setSession(null))
    return auth.onChange(setSession)
  }, [auth])

  if (session === undefined) return <p className="muted pad">Đang tải…</p>
  if (session === null || !auth || !api) return <Login auth={auth} />

  return (
    <main className="screen">
      {learning ? (
        <Learn
          api={api}
          speech={speech}
          tasks={learning}
          onExit={() => {
            setLearning(null)
            setVisit((v) => v + 1)
          }}
        />
      ) : (
        <>
          {tab === 'today' && <Today key={visit} api={api} onStart={setLearning} />}
          {tab === 'practice' && reading && practice && (
            <Reader api={api} id={reading} mode={practice} onBack={() => setReading(null)} />
          )}
          {tab === 'practice' && !reading && practice && <Passages api={api} kind={practice} onOpen={setReading} />}
          {tab === 'practice' && !practice && (
            <Practice onSelect={(choice: PracticeChoice) => (choice === 'vocab' ? setTab('today') : setPractice(choice))} />
          )}
          {tab === 'words' && <Words api={api} />}
          {tab === 'settings' && <SettingsPage api={api} />}
          <nav className="nav" aria-label="Điều hướng chính">
            {TABS.map((t) => (
              <button
                key={t.id}
                type="button"
                aria-current={tab === t.id ? 'page' : undefined}
                onClick={() => {
                  setTab(t.id)
                  setPractice(null)
                  setReading(null)
                }}
              >
                {t.label}
              </button>
            ))}
          </nav>
        </>
      )}
    </main>
  )
}

export default App
