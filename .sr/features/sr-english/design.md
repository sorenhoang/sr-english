# sr-english: design (gộp 2 ADR, cả project là 1 feature)

# Phần 1: Vocab app


_2026-11-02 · slug: `vocab-learning` · trạng thái: chờ G1_

## Problem

Web app cá nhân giúp học từ mới theo cấp độ CEFR (A1→C2), lấy cảm hứng từ Duolingo (bài học ngắn mỗi ngày, streak, nhiều dạng bài tập) và ELSA (nghe, phát âm; ELSA để cho bản sau). Người dùng đầu tiên là chính bạn: lập trình viên Việt Nam, đọc sách trên Kindle. Bản sau sẽ thêm: import từ Kindle `vocab.db`, chấm phát âm, giải thích/ví dụ bằng AI, học từ chuyên ngành dev.

**Phạm vi v1:**
- Chọn level.
- Mỗi ngày học N từ mới và ôn các từ đến hạn.
- Thẻ từ gồm: từ, IPA, từ loại, CEFR, nghĩa tiếng Việt, định nghĩa tiếng Anh, câu ví dụ, nút nghe.
- 3–4 dạng bài tập.
- Lặp lại ngắt quãng (spaced repetition).
- Streak.

**Chưa rõ, cần bạn xác nhận ở G1:**
1. "Web của tôi" là thêm vào site `self-learning` (Next.js, Vercel) hay một app riêng?
2. Bạn có học trên **cả điện thoại lẫn laptop** không? Câu trả lời quyết định có cần backend để đồng bộ tiến độ hay không.
3. Sau này có định **bán** app không? Câu trả lời quyết định nguồn dữ liệu (xem ràng buộc bên dưới).

**Ràng buộc quan trọng: bản quyền dữ liệu Cambridge**
- **English Vocabulary Profile (EVP)** của Cambridge chỉ miễn phí cho mục đích **phi thương mại**; dùng thương mại phải xin license ([Wikipedia: English Profile](https://en.wikipedia.org/wiki/English_Profile)).
- **Cambridge Dictionary** có API trả phí, phải có license ([dictionary-api.cambridge.org](https://dictionary-api.cambridge.org/index.php/blog/99-english-profile-levels)).
- Cào (scrape) website Cambridge thì rủi ro về điều khoản sử dụng và bản quyền, kể cả khi chỉ dùng cá nhân. **Loại phương án này.**
- **Thay thế hợp pháp:** **CEFR-J Wordlist** (khoảng 7.800 từ, A1–B2, cấp độ theo CEFR, gắn theo từ loại) được dùng **thương mại miễn phí, chỉ cần trích dẫn**. Cùng repo có bộ mở rộng Octanove cho C1–C2 ([openlanguageprofiles/olp-en-cefrj](https://github.com/openlanguageprofiles/olp-en-cefrj)).
- CEFR-J chỉ có từ, từ loại và level, **không có nghĩa hay ví dụ**. Phần đó sinh một lần bằng LLM (nghĩa tiếng Việt, định nghĩa tiếng Anh, ví dụ, IPA), lưu thành JSON và review dần. Nội dung LLM tạo ra là của bạn.
- **Tránh Wiktionary:** license CC BY-SA bắt buộc chia sẻ lại cả bộ dữ liệu theo cùng license.

**Ràng buộc khác:**
- Thời gian: lưu UTC; "một ngày học" tính theo timezone IANA của người dùng (theo TIMEZONE rules).
- Âm thanh v1: Web Speech API `speechSynthesis` (có sẵn, miễn phí; chất lượng tùy trình duyệt và hệ điều hành).
- Thuật toán ôn tập: **FSRS** (Anki dùng mặc định từ bản 23.10), qua thư viện `ts-fsrs`, không tự viết.

## Approaches

### A. Local-first, không backend (thêm route `/vocab` vào `self-learning`)
- **Tóm tắt:** dữ liệu từ là JSON tĩnh lúc build; tiến độ lưu ở IndexedDB trên trình duyệt.
- **Cách chạy:** Next.js page client-side, `ts-fsrs` chạy trên trình duyệt, nút Export/Import JSON để backup.
- **Ưu:** làm trong 2–3 ngày, chi phí 0, không cần auth, chạy offline. Tận dụng được site và Vercel sẵn có.
- **Nhược:** **mỗi thiết bị có tiến độ riêng**, phải export/import tay. Xóa dữ liệu trình duyệt là mất tiến độ (trừ khi đã export). Các tính năng AI sau này vẫn cần thêm server để giấu API key.
- **Hợp khi:** chỉ học trên 1 thiết bị và muốn kiểm chứng thói quen học trước.

### B. Next.js full stack + Postgres (app riêng, hoặc `/vocab` trong `self-learning`)
- **Tóm tắt:** Next.js route handlers, Postgres (Neon hoặc Supabase), đăng nhập đơn giản (magic link hoặc Google).
- **Cách chạy:** dữ liệu từ seed vào DB; lưu bảng trạng thái thẻ FSRS theo từng user; API `/api/session` trả về từ mới và từ đến hạn.
- **Ưu:** **đồng bộ điện thoại và laptop**. Có sẵn server cho AI và import Kindle sau này. Có user nên sẵn sàng nếu muốn bán.
- **Nhược:** thêm khoảng 1–2 ngày cho auth, DB và migration. Có chi phí nhỏ (thường nằm trong free tier).
- **Hợp khi:** học trên nhiều thiết bị, và chắc chắn sẽ thêm AI hoặc bán.

### C. Theo đúng stack chuẩn của bạn: Kotlin + Micronaut + Exposed + Postgres, frontend React + Vite
- **Tóm tắt:** dựng theo `KOTLIN_BACKEND_TEMPLATE` và `FRONTEND_STACK_TEMPLATE`.
- **Ưu:** luyện stack production của bạn, kiến trúc chuẩn để mở rộng; tách hẳn backend và frontend.
- **Nhược:** lâu nhất (khoảng 1–2 tuần mới có bản dùng được), phải deploy 2 service (Railway/AWS). Quá nặng cho một app học cá nhân ở v1.
- **Hợp khi:** mục tiêu chính là luyện Kotlin, hoặc đã xác định làm sản phẩm thương mại.

## Edge cases (phương án nào cũng phải xử lý)

- **Một từ nhiều nghĩa, mỗi nghĩa một level** ("bank" ở A1 và B2; "run" động từ khác "run" danh từ): **mô hình dữ liệu tính theo nghĩa (sense), không theo từ.** Khóa là `headword + pos (+ sense)`.
- **Bỏ học nhiều ngày rồi quay lại thì từ đến hạn dồn hàng trăm:** giới hạn số lượt ôn mỗi ngày, ưu tiên từ quá hạn lâu nhất; từ mới chỉ mở khi đã ôn xong.
- **Ranh giới ngày cho streak:** học lúc 23h50 và 00h10 theo giờ Việt Nam là 2 ngày khác nhau; tính theo timezone của user, không theo UTC.
- **Nghĩa hoặc ví dụ do LLM sinh bị sai:** có nút "báo sai" và trường `reviewed`; ưu tiên review các từ A1–B1 trước.
- **Trình duyệt không có giọng tiếng Anh** (`speechSynthesis` không có voice `en-*`): ẩn nút nghe và bài "nghe chọn từ", không để crash.
- **Đáp án gõ chính tả:** bỏ qua khác biệt hoa/thường và khoảng trắng; chấp nhận cả biến thể Anh–Anh và Anh–Mỹ (colour/color).
- **Câu hỏi trắc nghiệm có phương án nhiễu đồng nghĩa với đáp án:** chọn phương án nhiễu cùng từ loại và cùng level, loại các từ cùng nghĩa tiếng Việt.
- **Seed dữ liệu chạy lại nhiều lần:** phải idempotent (upsert theo khóa sense), không nhân đôi từ và không làm mất tiến độ.
- **Trích dẫn nguồn:** hiển thị credit CEFR-J ở footer (bắt buộc theo license).
- **Phương án A:** IndexedDB có thể bị trình duyệt xóa (Safari xóa sau 7 ngày không mở với site chưa cài như PWA). Phải có nhắc export định kỳ.

## Recommendation

**B: Next.js full stack + Postgres, làm app riêng `vocab`** (không gộp vào `self-learning`, vì site đó là site nội dung tĩnh). Dữ liệu dùng **CEFR-J** cộng phần làm giàu bằng LLM; ôn tập bằng **`ts-fsrs`**.

**Lý do quan trọng nhất:** thói quen học hằng ngày kiểu Duolingo chỉ bền khi tiến độ đi theo bạn trên mọi thiết bị, và mọi tính năng ở bản sau (AI, Kindle) đều cần server. Phương án A rẻ hơn nhưng sẽ phải viết lại phần lưu trữ.

Nếu bạn chỉ học trên laptop, chọn A: nhanh nhất, và có thể chuyển sang B sau bằng export/import JSON.

## Decision

**B″: React + Go backend + Supabase (chốt 2026-11-02).** Thay cho B′ (bản B′ không có backend tự viết).

```
React + Vite (Cloudflare Pages, free)
   │  Supabase Auth (Google login) → access token (JWT)
   ▼
Go API: chi + pgx (Cloud Run, scale-to-zero, free tier)
   │  verify JWT qua JWKS của Supabase → user_id
   ▼
Supabase Postgres (chỉ dùng làm DB, đi qua pooler)
```

| Thành phần | Chọn | Lý do |
|---|---|---|
| Router / DB driver | `go-chi/chi/v5`, `jackc/pgx/v5` | Giống các project Go bạn đang có (`go-judge`, `go-observability-lab`) |
| Query | `sqlc` (SQL viết tay, sinh code Go type-safe) | Không dùng ORM |
| Migration | `golang-migrate` hoặc `goose`, file SQL nằm trong repo | |
| Ôn tập ngắt quãng | `open-spaced-repetition/go-fsrs` chạy **trên server** | Chỉ một nơi quyết định lịch ôn |
| Auth | Supabase Auth (Google) ở frontend; Go verify JWT bằng JWKS | Không tự viết OAuth; giới hạn đúng email của bạn |
| Host API | Google Cloud Run free tier (GCP là nền tảng phụ của bạn) | Scale về 0 nên không tốn tiền khi không dùng; Go khởi động lạnh nhanh. _Kiểm tra lại hạn mức free lúc deploy._ |
| Host web | Cloudflare Pages | Free |

**Cấu trúc package:** `cmd/api`, `internal/{handler, service, repository, auth, fsrs}`. Handler mỏng, service chứa logic, repository lo SQL; cùng nguyên tắc phân tầng với rules backend của bạn.

**API theo kiểu RPC** (theo API_DESIGN rules của bạn):
- `GET /api/v1/session/today`: trả về từ mới và từ đến hạn ôn hôm nay.
- `POST /api/v1/review`: gửi `{sense_id, rating}`; server tính lại lịch FSRS.
- `POST /api/v1/settings/update`: level, số từ mới mỗi ngày, timezone.
- `POST /api/v1/words/list`: phân trang bằng offset/limit trong body.

**Chừa chỗ cho Đọc, Nghe… (chỉ 2 điểm nối, không làm sẵn gì thêm):**
- `GET /session/today` trả về `tasks: [{type: "vocab_review" | "vocab_new", ...}]`. Sau này thêm một skill thì thêm một `type`, trang Hôm nay chỉ hiển thị danh sách task.
- Bảng trạng thái FSRS khóa theo `sense_id`. Mọi skill sau này (Đọc, Nghe, Kindle) đều đẩy từ vào **cùng một bộ ôn tập**. Bảng bài đọc và bài nghe chỉ tạo khi bắt đầu làm skill đó.

**Ràng buộc mới so với B′:**
- **Go kết nối DB bằng role có quyền bỏ qua RLS**, nên mọi query phải tự lọc `user_id` trong code; viết test cho việc này. Vẫn bật RLS, không tạo policy nào, để chặn truy cập trực tiếp bằng `anon key`.
- **Pooler của Supabase** (Supavisor): chế độ transaction (port 6543) không hỗ trợ prepared statement. Cấu hình pgx dùng `QueryExecModeSimpleProtocol`, hoặc dùng chế độ session (port 5432) với pool nhỏ.
- **Khởi động lạnh 2 tầng:** Cloud Run scale về 0 và Supabase tự pause sau 7 ngày không hoạt động. Học hằng ngày thì không bị ảnh hưởng.
- **CORS:** chỉ cho phép domain trên Cloudflare Pages và `localhost`.
- **Secret** (DB URL) để trong Secret Manager của Cloud Run, không để trong biến môi trường dạng plain text.

**Dữ liệu:** CEFR-J làm bộ từ gốc, tra cứu Cambridge EVP để đối chiếu. Mục đích phi thương mại, app private.

**Lịch sử quyết định:**
- B′ (React + Supabase, không backend tự viết): người dùng nói "okay đấy".
- Đổi sang Go: người dùng nói "backend go nhé. cập nhật giúp tôi".
- Phi thương mại: "mục đích của tôi là phi thương mại nhé".

# Phần 2: Nguồn nội dung Đọc/Nghe


_2026-11-02 · slug: `reading-listening-source` · trạng thái: chờ G1_
_Liên quan: Phần 1, quyết định B″ (React + Go + Supabase, phi thương mại, private)_

## Problem

Cần **nguồn bài đọc và bài nghe theo cấp độ CEFR** cho 2 skill mới. Bài phải nối với bộ ôn tập FSRS đã có:
- chạm một từ trong bài là lưu được vào bộ ôn, kèm câu ngữ cảnh;
- từ đã có trong bộ ôn được tô màu.

Hai skill cắm vào khung đã thiết kế: thêm task `reading` / `listening` vào `GET /session/today`, ô Đọc và ô Nghe ở màn Luyện tập.

**Ràng buộc:**
- **Pháp lý:** phi thương mại và private, nhưng vẫn phải dùng nội dung có license rõ ràng. Không cào báo có bản quyền để lưu vào DB.
- **Chi phí ≈ 0:** Supabase free có 500 MB DB và 1 GB file storage; Cloud Run free tier.
- **Chất lượng nghe:** giọng tự nhiên và transcript khớp với audio là điều kiện bắt buộc để luyện nghe có ích.
- **Cấp độ:** một bài phải "vừa sức". Nghiên cứu về đọc hiểu (Hu & Nation, 2000) cho thấy người đọc cần biết khoảng **98% số từ** trong bài để hiểu mà không phải tra; 95% là mức tối thiểu.

**Chưa rõ:**
- Bạn muốn **chủ đề tin tức và đời sống chung**, hay **chủ đề dev/tech**? Câu trả lời quyết định nguồn nào hợp hơn.

## Phần dùng chung (phương án nào cũng cần)

Pipeline xử lý văn bản trong Go, chạy một lần lúc nạp bài vào DB:
1. **Tách từ** (tokenize) và **đưa về dạng gốc** (lemmatize: failed → fail, went → go). Thư viện ứng viên: `aaaton/golem` (cần xác minh license và độ chính xác).
2. **Ghép mỗi từ gốc với CEFR-J** (headword + từ loại) để biết level. Từ không có trong danh sách (tên riêng, số, thuật ngữ) bị loại khỏi phép tính.
3. **Tính độ phủ từ vựng và gán level cho bài:** level thấp nhất mà các từ ở level đó trở xuống phủ ≥ 95–98% số từ trong bài.
4. **Lưu sẵn vị trí từng token** (offset) để frontend biết người dùng chạm vào từ nào. Khi lưu từ, ghi kèm **câu chứa từ đó** làm ví dụ cho thẻ ôn.

Dữ liệu (theo rules DB: UUID, TIMESTAMPTZ, soft delete, không foreign key):
- `passages`: `kind` (reading|listening), `title`, `body`, `cefr_level`, `coverage` (JSONB), `source`, `source_url` (unique), `license`, `audio_url`, `duration_sec`, `published_at`.
- `passage_progress`: `user_id`, `passage_id`, `status`, `completed_at`.
- Thẻ ôn thêm 2 cột: `context_sentence`, `context_passage_id`.

## Approaches

### A. Bài do AI sinh, theo cấp độ và cá nhân hóa
- **Tóm tắt:** LLM viết bài theo level của bạn, cố ý cài các **từ sắp đến hạn ôn** vào bài. Listening dùng chính bài đó đọc bằng TTS.
- **Cách chạy:**
  - Go gọi LLM với prompt gồm level, chủ đề và danh sách từ cần cài. Pipeline dùng chung kiểm tra lại độ phủ; vượt level thì sinh lại.
  - Audio: TTS phía server (OpenAI, Google hoặc Azure) lưu file mp3 trong Supabase Storage, hoặc `speechSynthesis` của trình duyệt (miễn phí).
  - Câu hỏi đọc hiểu cũng do LLM sinh.
- **Ưu:**
  - Bài vô hạn, đúng level, đúng chủ đề bạn thích (kể cả dev).
  - Cá nhân hóa mạnh nhất: bài giúp ôn lại chính các từ đang học.
  - Không vướng bản quyền.
- **Nhược:**
  - Tốn tiền LLM và TTS (ít, nhưng khác 0).
  - Có thể sai ngữ pháp hoặc sai sự thật, văn đọc "nhân tạo".
  - **Giọng TTS kém tự nhiên, chưa đủ tốt để luyện nghe thật.**
  - Phải giấu API key và tự viết prompt và evals.
- **Hợp khi:** ưu tiên đọc hiểu và cá nhân hóa, chấp nhận nghe giọng máy.

### B. Nội dung thật thuộc phạm vi công cộng: VOA Learning English (+ Simple English Wikipedia cho Đọc)
- **Tóm tắt:** nạp bài từ **VOA Learning English**: tin tức và đời sống viết bằng tiếng Anh đơn giản, có **audio người đọc thật ở tốc độ chậm**, kèm transcript.
- **Pháp lý:** VOA là **public domain**, được dùng lại kể cả cho mục đích thương mại, chỉ cần ghi nguồn. Có hơn 1.000 bài.
- **Cách chạy:**
  - Một job Go (CLI hoặc cron) đọc RSS/podcast, lưu text và link mp3, rồi chạy pipeline dùng chung để gán level.
  - Simple English Wikipedia (license CC BY-SA, chỉ dùng cho Đọc) để có thêm chủ đề.
  - **Tùy chọn:** chạy `whisper.cpp` **một lần trên máy bạn** để lấy mốc thời gian từng câu, phục vụ nghe từng câu, chép chính tả và tô sáng câu đang đọc. Đây là tool chạy tay, không deploy.
- **Ưu:**
  - **Giọng thật, transcript chuẩn**, viết cho người học.
  - Không tốn tiền sinh nội dung, không có rủi ro AI bịa.
  - Hợp pháp rõ ràng.
- **Nhược:**
  - Chủ đề là tin tức Mỹ và đời sống, **ít bài về dev**.
  - Không cá nhân hóa theo từ đang học (chỉ tô màu từ đã có).
  - VOA bị cắt giảm mạnh từ 2025, **chưa chắc còn ra bài mới**. Podcast cập nhật gần nhất tìm thấy là 6/4/2026.
  - Audio: chỉ lưu link thì có thể bị gãy link; tự lưu file thì 1 GB storage ≈ 1.000 phút mp3.
  - Không có câu hỏi đọc hiểu: v1 làm chép chính tả và điền từ còn thiếu tự sinh từ transcript; câu hỏi để sau.
- **Hợp khi:** muốn luyện nghe thật, ít tốn chi phí và ít bảo trì.

### C. Tự mang nội dung: dán URL, văn bản hoặc link YouTube
- **Tóm tắt:** bạn dán bài blog hoặc bài viết đang đọc, hoặc video YouTube có phụ đề. App gán level, tô màu từ và cho chạm để lưu.
- **Cách chạy:** Go lấy nội dung chính của trang (readability) hoặc phụ đề YouTube, rồi chạy pipeline dùng chung. Video được nhúng bằng player YouTube, không tải về.
- **Ưu:**
  - Học từ đúng thứ mình quan tâm (blog kỹ thuật, talk), nên động lực cao nhất.
  - Gần như không cần chuẩn bị nội dung.
- **Nhược:**
  - **Không kiểm soát được level.** Nhiều bài sẽ quá khó.
  - Bản quyền: chỉ ổn khi dùng private cho riêng bạn, không được chia sẻ.
  - Việc lấy nội dung dễ hỏng (paywall, trang render bằng JS, phụ đề tự động sai).
  - Không có sẵn task hằng ngày: "kế hoạch hôm nay" phụ thuộc việc bạn có dán bài hay không.
- **Hợp khi:** đã ở mức B2 trở lên và đọc nhiều nguồn tiếng Anh thật.

## Edge cases (phương án nào cũng phải xử lý)

- **Cụm động từ và cụm nhiều từ** ("roll back", "take off"): ưu tiên ghép cụm dài nhất trước rồi mới tới từng từ.
- **Một từ có nhiều từ loại** ("run" danh từ hay động từ): chạm vào thì hiện các lựa chọn để chọn đúng; nếu lúc nạp bài đã gán từ loại thì dùng luôn.
- **Từ không có trong CEFR-J** (thuật ngữ, từ C2 trở lên): vẫn cho lưu thành "từ tự thêm" (nghĩa nhập tay hoặc AI bổ sung sau), không chặn.
- **Không tính vào độ phủ:** tên riêng, số, URL, đoạn code, từ viết tắt.
- **Biến thể chính tả và từ rút gọn:** Anh–Mỹ (colour/color); dạng rút gọn (don't, it's) đưa về dạng gốc.
- **Từ trong bài trên điện thoại:** vùng chạm nhỏ, nên tăng line-height và mở rộng vùng chạm.
- **Nạp lại bài nhiều lần phải idempotent:** unique theo `source_url`, không nhân đôi.
- **Hiển thị nguồn và license** trên mỗi bài (VOA và CC BY-SA bắt buộc ghi nguồn).
- **Audio:** link chết hoặc file lỗi thì ẩn nút nghe; với nguồn không lưu file, kiểm tra link định kỳ.
- **Dung lượng:** text nằm trong DB (500 MB, khoảng vài chục nghìn bài); audio không lưu trong DB.
- **Đổi level:** danh sách bài gợi ý lọc lại theo level mới; tiến độ cũ vẫn giữ.

## Recommendation

**B (VOA Learning English) làm nguồn chính cho cả Đọc lẫn Nghe.** A để sau, làm bổ sung cho Đọc, sinh bài chủ đề dev có cài các từ đang ôn.

**Lý do quan trọng nhất:** luyện nghe chỉ hiệu quả với **giọng người thật kèm transcript chính xác**. VOA cho cả hai, miễn phí và hợp pháp. Đây là thứ phương án A (giọng TTS) và C (không kiểm soát level) không làm được.

**Giảm rủi ro VOA ngừng ra bài:** nạp và lưu trữ phần archive ngay từ đầu (text vào DB, audio cache có chọn lọc). Nếu bạn thấy thiếu chủ đề dev thì thêm A.

## Decision

**B: VOA Learning English làm nguồn chính cho cả Đọc và Nghe** (chốt 2026-11-02). A (bài do AI viết) và C (tự dán URL) không làm ở giai đoạn này.

**Chủ đề:** theo bộ chủ đề của các kỳ thi Cambridge (chủ đề chung của tiếng Anh tổng quát, ví dụ gia đình, du lịch, sức khỏe, công việc, giáo dục, môi trường, công nghệ…), **không** theo chủ đề dev.
- Mỗi bài được gán 1–2 chủ đề trong danh sách cố định này: ánh xạ từ chuyên mục của VOA trước, thiếu thì gán bằng từ khóa.
- Người dùng lọc bài theo chủ đề và level.
- Danh sách chủ đề lấy theo các danh sách từ vựng A2 Key / B1 Preliminary mà Cambridge công bố, chỉ dùng tên chủ đề. Mục đích phi thương mại.

Người dùng: "tôi oke với option B. chủ đề thì là các chủ đề theo cambridge thôi tôi học English mà."
