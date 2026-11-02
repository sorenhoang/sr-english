import { useEffect, useState } from 'react'
import type { Api, Settings } from '../lib/api'
import { levelOptions } from '../lib/levels'

export function SettingsPage({ api }: { api: Api }) {
  const [form, setForm] = useState<Settings | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    api
      .settings()
      .then((s) => setForm(s ?? null))
      .catch((e: unknown) => setError(e instanceof Error ? e.message : 'Không tải được cài đặt.'))
  }, [api])

  if (!form) return error ? <p role="alert" className="error">{error}</p> : <p role="status" className="muted">Đang tải…</p>

  const save = async () => {
    if (!Number.isInteger(form.daily_new) || form.daily_new < 1) {
      setError('Số từ mới mỗi ngày phải là số nguyên ≥ 1.')
      return
    }
    setSaving(true)
    setError(null)
    setSaved(false)
    try {
      const updated = await api.updateSettings(form)
      if (updated) setForm(updated)
      setSaved(true)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Không lưu được cài đặt.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <section className="stack">
      <h1>Cài đặt</h1>
      <form
        className="card stack"
        onSubmit={(e) => {
          e.preventDefault()
          void save()
        }}
      >
        <label htmlFor="settings-level">Cấp độ</label>
        <select id="settings-level" value={form.level} onChange={(e) => setForm({ ...form, level: e.target.value })}>
          {levelOptions(form.level).map((l) => (
            <option key={l} value={l}>
              {l}
            </option>
          ))}
        </select>
        <label htmlFor="settings-daily">Số từ mới mỗi ngày</label>
        <input
          id="settings-daily"
          type="number"
          min={1}
          value={form.daily_new || ''}
          onChange={(e) => setForm({ ...form, daily_new: Number(e.target.value) })}
        />
        <label htmlFor="settings-tz">Múi giờ</label>
        <input id="settings-tz" value={form.timezone} onChange={(e) => setForm({ ...form, timezone: e.target.value })} />
        <button
          type="button"
          className="btn"
          onClick={() => setForm({ ...form, timezone: Intl.DateTimeFormat().resolvedOptions().timeZone })}
        >
          Dùng múi giờ của thiết bị
        </button>
        {error && <p role="alert" className="error">{error}</p>}
        {saved && <p role="status" className="muted">Đã lưu.</p>}
        <button type="submit" className="btn primary" disabled={saving}>
          Lưu
        </button>
      </form>
    </section>
  )
}
