# تست و نتایج واقعی

رویکرد توسعه backend test-first بوده است: ابتدا testهای scoring/static/Git/storage/job/API نوشته و سپس implementation اضافه شده است. frontend نیز تست واحد تاریخ شمسی و رفتار Heatmap دارد.

## اجراهای انجام‌شده در محیط توسعه

- `go test ./...` با Go 1.23.12 و `CGO_ENABLED=1` — موفق.
- `go build ./cmd/server ./cmd/demo-fixture` — موفق.
- `npm run test` — 2 فایل / 4 تست موفق.
- `npm run typecheck` — موفق.
- `npm run build` — موفق (Next.js 16.4.6).

پوشش‌های مهم: LOC/comment/string و parse error سه زبان، nested function، وزن نامعتبر و معیار ثابت، root commit/author/range/limit Git، persistence/recovery SQLite، snapshot بدون dirty/untracked، token API، خطای JSON کنترل‌شده، CSV formula injection، تاریخ جلالی و non-selectable empty Heatmap cell.

## کارایی

benchmark ادعایی ثبت نشده است. این محیط روی مجموعه نزدیک 300k LOC آزمایش نشده؛ بنابراین زمان پاسخ یا ظرفیت قطعی برای آن اعلام نمی‌شود. قبل از production، روی مخازن مجاز کوچک/متوسط/بزرگ همراه CPU/RAM، تعداد فایل، حجم تاریخچه و جداگانه زمان static/Git اندازه‌گیری کنید.
