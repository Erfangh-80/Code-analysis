# معماری

## Modular Monolith

یک پردازش Go مالک API، صف تک‌کارگر، تحلیل snapshot، تحلیل Git، امتیازدهی و SQLite است. مرورگر فقط با REST/JSON همان پردازش ارتباط دارد. Next.js در توسعه با rewrite سمت سرور به Go متصل می‌شود؛ در انتشار، `STATIC_EXPORT=1` خروجی static تولید می‌کند و Go آن را از `ANALYZER_WEB_DIR` ارائه می‌دهد. پس از build، Node.js برای اجرا لازم نیست.

```
Next.js (static / RTL UI) ── relative /api/v1 ──> Go server
                                                   ├─ snapshot reader (tracked Git blobs)
                                                   ├─ Tree-sitter static analyzer
                                                   ├─ Git CLI history reader
                                                   ├─ scoring / module aggregation / evaluation
                                                   └─ SQLite
```

## مرزها

- `backend/internal/analyzer/static`: Tree-sitter و شاخص‌های کد؛ هرگز کد هدف را اجرا نمی‌کند.
- `backend/internal/analyzer/snapshot`: فقط `git ls-tree` و `git show` برای commit منتخب.
- `backend/internal/githistory`: تاریخچه ancestorهای commit منتخب، با Git نصب‌شده روی سیستم و بدون shell interpolation. در این محیط محدود، Git CLI جایگزین `go-git` شد تا وابستگی‌های vanity-host غیرقابل دریافت در build آفلاین وارد محصول نشوند؛ Git از قبل پیش‌نیاز محصول است.
- `backend/internal/scoring`: نرمال‌سازی، امتیاز و تجمیع ماژول؛ pure domain logic و قابل تست.
- `backend/internal/jobs`: صف درون‌پردازشی با یک تحلیل هم‌زمان. Jobهای `running` پس از restart به `failed/SERVER_RESTARTED` تبدیل می‌شوند.
- `backend/internal/storage`: schema نسخه 1 و JSONهای شاخص در SQLite. سورس کد ذخیره نمی‌شود.
- `frontend/components`: Atomic Design: `ui` atoms (کامپوننت‌های shadcn-style)، `organisms` (فرم، جدول، Heatmap)، و `templates` (Dashboard).

## قرارداد داده و وضعیت‌ها

`queued → running → completed | partial | failed`. `partial` یعنی parse/read فایل یا تاریخچه ناقص بوده است، نه اینکه صرفاً هیچ معیار قدرت تفکیک نداشته باشد. هشدار «فایل قابل مقایسه کافی نیست» امتیاز را `null` می‌کند، اما تحلیل کامل باقی می‌ماند.

Schema SQLite در startup به صورت idempotent ساخته می‌شود: `schema_migrations`, `analyses`, `files`, `modules`, `evaluations`. نسخه فعلی schema برابر 1 است.

## محدودیت منابع

`ANALYZER_MAX_FILE_BYTES`, `ANALYZER_MAX_COMMITS` و `ANALYZER_ANALYSIS_TIMEOUT` محدودیت‌های عملیاتی‌اند. برخورد با سقف commit یا shallow repository، `history_complete=false` و هشدار می‌سازد؛ صفر churn به اشتباه ثبت نمی‌شود.
