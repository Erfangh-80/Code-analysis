# امنیت و استقرار

- `ANALYZER_REPO_ROOT` الزامی است. مسیر ورودی با `Abs`، `EvalSymlinks` و `Rel` بررسی می‌شود؛ prefix رشته‌ای استفاده نشده است.
- git dir و object storage نیز باید پس از resolve داخل ریشه مجاز باشند. این کار `.git`/object store خارجی را رد می‌کند.
- API به جز health نیازمند header `X-Analyzer-Token` برابر با `ANALYZER_API_TOKEN` است. Token در سورس، URL، SQLite یا log ذخیره نمی‌شود. UI آن را فقط در state نشست نگه می‌دارد.
- listen پیش‌فرض `127.0.0.1:8080` است. برای شبکه، هر دو `ANALYZER_TLS_CERT` و `ANALYZER_TLS_KEY` را تنظیم کنید. CORS فقط برای originهای comma-separated در `ANALYZER_ALLOWED_ORIGINS` فعال است؛ wildcard وجود ندارد.
- تحلیل با Git read commands و Tree-sitter انجام می‌شود؛ هیچ فایل هدف نوشته یا اجرا نمی‌شود. برای محیط حساس، اجرا با کاربر محدود و mount read-only مخزن توصیه می‌شود.
- log عملیاتی فقط request id/method/path/status است. response خطا stack trace، token، source و commit message را افشا نمی‌کند.
- اجرای ساخته‌شده offline است: هیچ fetch/CDN در frontend و هیچ outbound API در backend نیست. دریافت npm/Go dependency فقط مرحله build است. برای air-gap، `npm ci` و `go mod download` را روی ماشین متصل اجرا و cache/package mirror را منتقل کنید.
