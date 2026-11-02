# Plan: sr-english

## Approach (theo design.md)

- **Backend:** Go (chi + pgx) ở `api/`, Postgres trên Supabase. Auth bằng Supabase JWT (verify qua JWKS) cộng allowlist email. FSRS chạy trên server bằng `go-fsrs/v3`. Bộ từ lấy từ CEFR-J, nghĩa được làm giàu bằng LLM qua một tool chạy tay.
- **Nội dung đọc/nghe:** VOA Learning English, nạp bằng tool chạy tay. Pipeline văn bản (lemma bằng `golem/v4`, ghép với CEFR-J) quyết định level và trạng thái lưu của từng token. Chủ đề theo bộ chủ đề Cambridge.
- **Frontend:** React + Vite + TypeScript ở `web/`, test bằng Vitest + Testing Library. Màn hình theo canvas thiết kế.

**Lệch so với design.md** (nêu ở G2):
- Bỏ `sqlc`, viết query bằng pgx tay, giống `go-judge`. Lý do: `sqlc` chưa cài trên máy, và harness không cài thêm tool.
- Package tổ chức theo domain (`internal/<domain>/{model, repository, postgres_repository}.go` + `internal/httpapi`), giống `go-judge`, thay vì `handler/service/repository`.

**Pattern tham chiếu (code thật):**

| Pattern | File mẫu |
|---|---|
| Handler (chi, `RegisterRoutes`, `writeJSON`) | `onlysoren/go/go-judge/internal/httpapi/problems.go` |
| Repository interface + Postgres | `onlysoren/go/go-judge/internal/problem/repository.go`, `postgres_repository.go` |
| Migration golang-migrate (`000001_x.up/down.sql`) | `onlysoren/go/go-judge/migrations/` |
| Vite + React + ESLint | `onlysoren/go/go-judge/frontend/package.json` |

## Affected files

| File | Create/Modify/Delete | Reason |
|---|---|---|
| `.sr/harness.json` | Modify | Thêm `docker compose up -d --wait db &&` vào đầu lệnh `verify`, để test Go có Postgres |
| `docker-compose.yml` | Create | Postgres 16 cho dev và test |
| `api/go.mod`, `api/cmd/api/main.go` | Create | Server, router, `/health` (F-1) |
| `api/internal/config/config.go` | Create | Đọc env: DB URL, JWKS URL, ALLOWED_EMAILS, DAILY_NEW_DEFAULT/MAX, MAX_REVIEWS_PER_DAY, COVERAGE_THRESHOLD, DESIRED_RETENTION |
| `api/migrations/0000NN_*.sql` | Create | Bảng users, user_settings, senses, cards, reviews, passages, passage_tokens, passage_progress |
| `api/cmd/seed/`, `api/data/cefrj.csv` | Create | Nạp CEFR-J (F-2); ghi nguồn CEFR-J |
| `api/cmd/enrich/`, `api/internal/llm/` | Create | Làm giàu nghĩa từ (F-3) |
| `api/internal/auth/` | Create | JWT middleware + allowlist (F-4) |
| `api/internal/settings/`, `vocab/` (senses, cards, fsrs), `stats/` | Create | F-5 → F-9 |
| `api/internal/text/` | Create | Tách từ, lemma, độ phủ, level (F-13) |
| `api/cmd/ingest-voa/`, `api/internal/content/` | Create | Nạp bài VOA, ánh xạ chủ đề (F-14) |
| `api/internal/httpapi/*.go` | Create | Handler cho từng nhóm endpoint |
| `web/` (Vite app), `web/src/lib/api.ts`, `web/src/lib/supabase.ts` | Create | F-10 → F-12, F-17 |
| `web/src/pages/{Today,Learn,Words,Settings,Practice,Reading,Listening}.tsx` | Create | Màn hình theo canvas |

## Dependencies & order

```
F-1 → F-2 → F-3
         ↘
F-1 → F-4 → F-5 → F-6 → F-7 → F-8, F-9
F-2 + F-13 → F-14 → F-15 → F-16
Web: F-10 (sau F-4, F-6) → F-11 (sau F-7) → F-12 (sau F-5, F-9) → F-17 (sau F-15, F-16)
```

- Tuần tự: F-1 → F-2 → F-4 → F-5 → F-6 → F-7.
- Song song được sau F-2: F-3, F-13. Sau F-7: F-8, F-9.
- Thứ tự `implement all` đề xuất: F-1 … F-17 theo số.

## Dependency mới (Go/npm, không phải tool hệ thống)

`go-chi/chi/v5`, `jackc/pgx/v5`, `golang-migrate/migrate/v4`, `open-spaced-repetition/go-fsrs/v3` (đã kiểm tra có bản v3.3.1), `aaaton/golem/v4` (v4.0.2), `MicahParks/keyfunc/v3` + `golang-jwt/jwt/v5`, `anthropics/anthropic-sdk-go` (F-3). Web: `@supabase/supabase-js`, `vitest`, `@testing-library/react`, `jsdom`.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| **VOA ngừng ra bài.** Podcast mới nhất tìm thấy là 31/03/2025; feed podcast không chứa transcript | Đã xảy ra | Cao: chỉ có kho bài cũ | Nạp từ danh sách URL bài viết (archive), lưu text vào DB; kho cũ đủ dùng lâu cho 1 người |
| Cấu trúc HTML trang VOA khác với fixture | Trung bình | Trung bình | Test bằng fixture; selector được đánh dấu (unverified); có thể thêm lệnh `--dry-run` để soát |
| Độ chính xác lemma của `golem` (cụm từ, từ loại) | Trung bình | Thấp | Ưu tiên ghép cụm dài nhất; cho người dùng chọn từ loại khi chạm |
| LLM sinh nghĩa sai | Trung bình | Trung bình | `reviewed=false`, có nút báo sai (sau v1); làm A1–B1 trước |
| Pooler Supabase không hỗ trợ prepared statement | Cao | Cao | pgx dùng `QueryExecModeSimpleProtocol` (đã ghi trong design) |
| Test UI không chạy trình duyệt thật (Vitest + jsdom) | Chắc chắn | Thấp | `speechSynthesis`/audio mock; kiểm tay trên điện thoại trước khi chốt |
| Chi phí LLM cho làm giàu nghĩa | Thấp | Thấp | Chạy theo level, có `--limit`, idempotent |

## Out of scope

Chấm phát âm; import Kindle; bài do AI sinh (A) và dán URL (C); câu hỏi đọc hiểu; chép chính tả theo từng câu (cần mốc thời gian từ whisper); PWA/offline; deploy lên Cloud Run và Cloudflare (làm sau khi v1 chạy local); nút báo nghĩa sai.
