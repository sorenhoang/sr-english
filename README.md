# sr-english

Personal English learning app: daily CEFR vocabulary with FSRS spaced repetition, plus reading and listening
practice from VOA Learning English (public domain). Non-commercial, private use.

- `api/`: Go (chi + pgx) on Postgres (Supabase in production)
- `web/`: React + Vite + TypeScript, Supabase Auth (Google)

## Run locally

```bash
docker compose up -d --wait db                       # Postgres 16 on localhost:55432
export DATABASE_URL="postgres://srenglish:srenglish@localhost:55432/srenglish?sslmode=disable"

cd api
go run ./cmd/seed                                    # migrate + load CEFR-J (A1–B2) and Octanove (C1–C2)
ANTHROPIC_API_KEY=... go run ./cmd/enrich --level A1 --limit 50   # Vietnamese meanings, IPA, examples (costs tokens)
go run ./cmd/ingest-voa --urls urls.txt              # one learningenglish.voanews.com article URL per line

SUPABASE_JWKS_URL=https://<project>.supabase.co/auth/v1/.well-known/jwks.json \
SUPABASE_ISSUER=https://<project>.supabase.co/auth/v1 \
ALLOWED_EMAILS=you@example.com \
go run ./cmd/api                                     # http://localhost:8080

cd ../web
cp .env.example .env.local                           # fill Supabase URL + anon key, API URL
pnpm install && pnpm dev
```

Optional tuning (env): `DAILY_NEW_DEFAULT` (10), `DAILY_NEW_MAX` (50), `MAX_REVIEWS_PER_DAY` (200),
`DESIRED_RETENTION` (0.9), `COVERAGE_THRESHOLD` (0.95).

## Deploy (free: Supabase + 2 Vercel projects)

1. **Supabase** (free): create a project → Authentication → Providers → enable Google.
   Copy the *transaction pooler* connection string (port 6543), the project URL and the anon key.
   Seed from your machine: `DATABASE_URL=<pooler url> go run ./cmd/seed` (then `enrich`, `ingest-voa`).
2. **Vercel project `sr-english-api`**: root directory `api` (Go preset via `api/vercel.json`, runs `cmd/api`).
   Env: `DATABASE_URL`, `SUPABASE_JWKS_URL`, `SUPABASE_ISSUER`, `ALLOWED_EMAILS`.
3. **Vercel project `sr-english`**: root directory `web` (Vite). Env: `VITE_SUPABASE_URL`, `VITE_SUPABASE_ANON_KEY`.
   `web/vercel.json` proxies `/api/*` to `https://sr-english-api.vercel.app` — change it if your API project has another name.
4. Supabase → Authentication → URL configuration: add the web URL to Site URL / Redirect URLs.

Free-tier notes: Supabase pauses after 7 days without activity; the first request after idle is slower (cold start).

## Checks

`python3 ~/.claude/hooks/sr/sr_hooks.py verify` runs vet, golangci-lint, Go tests (against the compose DB),
web lint, build and Vitest.

## Data credits

- CEFR-J Vocabulary Profile 1.5 — Tono Laboratory, Tokyo University of Foreign Studies
- Octanove Vocabulary Profile C1/C2 — CC BY-SA 4.0
- VOA Learning English — public domain, credit learningenglish.voanews.com
