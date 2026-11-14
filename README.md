# Code Hotspot Analyzer

**Code Hotspot Analyzer** یک MVP محلی برای اولویت‌بندی بازبینی و بازآرایی کد است. ابزار، snapshot ثبت‌شده یک مخزن Git، پیچیدگی تابع/LOC و تاریخچه تغییرات را می‌خواند و فایل‌ها و ماژول‌ها را با امتیاز **نسبی و توضیح‌پذیر** رتبه‌بندی می‌کند.

> این عدد درصد احتمال باگ، تضمین آسیب‌پذیری یا قضاوت درباره کیفیت افراد نیست. هیچ کد هدفی اجرا یا تغییر داده نمی‌شود.

## قابلیت‌های پیاده‌سازی‌شده

- Backend و تحلیل‌گر با Go، SQLite پایدار و صف تک‌کارگر Job.
- تحلیل Tree-sitter برای **Go، Python و JavaScript/JSX**: LOC، توابع، کلاس‌های Python/JS، طول و cyclomatic complexity.
- خواندن فقط-خواندنی blobهای commit منتخب؛ dirty/untracked وارد تحلیل نمی‌شوند.
- تاریخچه Git با بازه مشخص UTC `[since, until)`، churn، frequency، author count، concentration/dispersion؛ root commit و shallow/limit هشدار دارند.
- امتیاز log1p/min-max با وزن قابل تنظیم، contribution هر شاخص و دلیل `null` بودن score.
- تجمیع ماژول، Heatmap واقعی 2D (شبکه 10×10)، فیلتر خانه و «هر دو شاخص بالا».
- API token، اعتبارسنجی symlink/object store در root مجاز، CSV formula escaping، JSON/CSV report و OpenAPI.
- ارزیابی asynchronous و صادقانه Spearman با proxy کلیدواژه‌های احتمالی Bug-fix.
- Next.js + TypeScript + Tailwind + کامپوننت‌های shadcn-style، Atomic Design، `TanStack Query`، `zod`، `react-hook-form`، RTL، dark/light mode و نمایش شمسی با `jalaali-js`.
- fixture محلی Git برای دمو، بدون GitHub.

## پیش‌نیازها

- Go **1.23+** و CGO فعال (`gcc` یا compiler C سازگار): Tree-sitter و `go-sqlite3` به CGO نیاز دارند.
- Git command line.
- Node.js 20+ و npm فقط برای توسعه/build frontend. اجرای build نهایی static به Node نیاز ندارد.

وابستگی‌ها در زمان build دریافت می‌شوند؛ اجرای برنامه هیچ درخواست بیرونی، CDN یا API ابری ندارد. برای محیط air-gapped، cache `npm ci` و `go mod download` را روی محیط متصل آماده و منتقل کنید.

## نصب و اجرا

```bash
cp .env.example .env
# .env را با مسیر مطلق ریشه مجاز و token واقعی پر کنید
# برنامه dotenv را خودکار نمی‌خواند؛ متغیرها را صریحاً وارد shell کنید:
set -a; . ./.env; set +a

cd frontend && npm ci && cd ..
# اگر Go در PATH است:
make test
make build

export ANALYZER_REPO_ROOT=/home/user/repositories
export ANALYZER_API_TOKEN='یک-token-بلند-و-تصادفی'
export ANALYZER_WEB_DIR="$PWD/frontend/out"
./backend/bin/code-hotspot-analyzer
```

سپس `http://127.0.0.1:8080` را باز کنید و token را در نوار بالای UI وارد کنید. Token فقط در حافظه نشست UI باقی می‌ماند. frontend static با همان origin API سرو می‌شود. برای توسعه جداگانه:

```bash
# Terminal 1
export ANALYZER_REPO_ROOT=/home/user/repositories ANALYZER_API_TOKEN=dev-token
cd backend && go run ./cmd/server
# Terminal 2; rewrite سمت Next درخواست /api را به Go منتقل می‌کند
cd frontend && BACKEND_URL=http://127.0.0.1:8080 npm run dev
```

در production پیش‌فرض روی loopback است. برای شبکه TLS را با `ANALYZER_TLS_CERT` و `ANALYZER_TLS_KEY` فعال و originهای محدود را در `ANALYZER_ALLOWED_ORIGINS` مشخص کنید. جزئیات امنیت در [docs/security.md](docs/security.md) است.

## ساخت مخزن دمو

ابتدا root مجاز را شامل محل دمو قرار دهید، سپس:

```bash
make demo
# یا: ./backend/bin/demo-fixture --path /home/user/repositories/hotspot-demo
```

fixture بدون `--force` هرگز پوشه موجود را overwrite نمی‌کند و این موارد را چاپ می‌کند: مسیر، SHA تاریخی، بازه پیشنهادی تحلیل و بازه پیشنهادی evaluation. در فرم، SHA تاریخی را در تنظیمات پیشرفته و بازه تحلیل پیشنهادی وارد کنید؛ سپس با بازه evaluation پیشنهادی، دکمه ارزیابی Bug-fix را بزنید.

## API و گزارش

تمام endpointها زیر `/api/v1` هستند. به جز `GET /health` باید این header را داشته باشند:

```text
X-Analyzer-Token: <ANALYZER_API_TOKEN>
```

نمونه ثبت تحلیل:

```bash
curl -X POST http://127.0.0.1:8080/api/v1/analyses \
  -H 'Content-Type: application/json' -H "X-Analyzer-Token: $ANALYZER_API_TOKEN" \
  -d '{"repository_path":"/home/user/repositories/my-project","ref":"HEAD","languages":["go","python","javascript"],"history":{"since":"2025-01-01T00:00:00Z","until":"2026-01-01T00:00:00Z"},"weights":{"complexity":0.5,"churn":0.3,"frequency":0.2}}'
```

گزارش‌ها: `GET /analyses/{id}/reports?format=json&level=file`، `format=csv&level=file`، `format=csv&level=module`. قرارداد کامل در [docs/openapi.yaml](docs/openapi.yaml) است.

## مستندات فنی

- [معماری و schema](docs/architecture.md)
- [تعریف شاخص‌ها و فرمول](docs/metrics.md)
- [امنیت و استقرار](docs/security.md)
- [ارزیابی Bug-fix و محدودیت‌ها](docs/evaluation.md)
- [تست‌های اجراشده و محدودیت benchmark](docs/testing.md)

## محدودیت‌های آگاهانه MVP

- Merge commitها برای جلوگیری از دوباره‌شماری در churn کنار گذاشته می‌شوند.
- rename path-based است؛ تاریخ پیش از rename ممکن است تعلق نگیرد.
- history shallow، binary diff، timeout و commit limit نتیجه را partial/incomplete نشان می‌دهند.
- `max function complexity` منطق سطح فایل بدون تابع را کامل نشان نمی‌دهد.
- تحلیل parser برای TS/JSX از grammar JavaScript است؛ TypeScript خارج از دامنه است.
- پیام `fix`/`bug`/`resolve` فقط یک نشانه احتمالی Bug-fix است. Spearman درصد صحت یا رابطه علّی نیست.
- برای کارایی روی مخازن بسیار بزرگ benchmark واقعی این پروژه انجام نشده و ادعایی ارائه نمی‌شود.
