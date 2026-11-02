export type PracticeChoice = 'vocab' | 'reading' | 'listening'

const TILES: { id: PracticeChoice | null; name: string; hint: string }[] = [
  { id: 'vocab', name: 'Từ vựng', hint: 'Ôn và học từ mới' },
  { id: 'reading', name: 'Đọc', hint: 'Bài theo cấp độ' },
  { id: 'listening', name: 'Nghe', hint: 'Giọng thật, có transcript' },
  { id: null, name: 'To be continued', hint: 'Kỹ năng tiếp theo' },
]

export function Practice({ onSelect }: { onSelect: (choice: PracticeChoice) => void }) {
  return (
    <section className="stack">
      <h1>Luyện tập</h1>
      <div className="tiles">
        {TILES.map((tile) => (
          <button
            key={tile.name}
            type="button"
            className="tile card"
            disabled={tile.id === null}
            onClick={() => tile.id && onSelect(tile.id)}
          >
            <strong>{tile.name}</strong>
            <span className="muted">{tile.hint}</span>
          </button>
        ))}
      </div>
    </section>
  )
}
