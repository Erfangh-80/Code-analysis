# Frontend

داشبورد فارسی و RTL پروژهٔ **Code Risk Analyzer** با React، TypeScript و Vite ساخته شده است. این برنامه فقط با API محلی ارتباط می‌گیرد؛ هیچ مسیر پروژه یا کد منبعی به سرویس خارجی ارسال نمی‌شود.

## فناوری‌ها و معماری

- React 19 + TypeScript + Vite
- Tailwind CSS v4 و design tokenهای قابل استفاده مجدد در `src/styles/globals.css`
- کامپوننت‌های پایهٔ shadcn/ui در `src/components/ui`
- React Router برای routeهای داشبورد
- TanStack Query برای درخواست‌ها و cache API
- Zustand برای state رابط کاربری (مانند sidebar موبایل، حالت روشن/تاریک و ترجیحات نمایش)
- Recharts برای نمودار توزیع ریسک و Heatmap تعاملی با محورهای ذخیره‌شدهٔ backend
- ساختار Atomic Design:
  - `components/atoms` برای اجزای هویتی کوچک
  - `components/molecules` برای ترکیب‌های کوچک UI
  - `components/organisms` برای sidebar و header
  - `components/templates` برای shell اصلی داشبورد
  - `features/*` برای صفحه‌ها و اجزای دامنه‌ای

## اجرا

ابتدا API محلی Go را از پوشهٔ `../backend` اجرا کنید. سپس:

```bash
cd frontend
npm install
npm run dev
```

Vite روی همهٔ interfaceها bind می‌شود. مسیرهای نسبی `/api/...` در dev server به `http://127.0.0.1:8080` proxy می‌شوند. کد مرورگر باید فقط از مسیرهای نسبی مانند `/api/v1/analyses` استفاده کند و نباید به `localhost` یا سرویس خارجی درخواست مستقیم بفرستد.

## مسیرهای فعلی

| مسیر | کاربرد |
| --- | --- |
| `/` | نمای کلی و وضعیت اتصال سرویس محلی |
| `/analyses` | فهرست تحلیل‌های ذخیره‌شده |
| `/analyses/new` | ثبت تحلیل با مسیر محلی، بازهٔ Git و وزن‌های Hotspot |
| `/analyses/:analysisId` | پیشرفت زنده، Dashboard گزارش و Heatmap |
| `/analyses/:analysisId/report` | پیش‌نمایش گزارش immutable و دریافت JSON/CSV با وضعیت خطا |
| `/analyses/:analysisId/files` | جدول فایل‌ها با جست‌وجو، فیلتر، مرتب‌سازی و صفحه‌بندی |
| `/analyses/:analysisId/files/detail?path=…` | جزئیات Static/Git/Risk و contributionهای فایل |
| `/settings` | ترجیحات محلی رابط کاربری |

تمام داده‌های تحلیل در UI از API واقعی خوانده می‌شوند؛ صفحه‌ها دادهٔ نمونه یا نتیجهٔ ساختگی نمایش نمی‌دهند. خطاهای render مسیرها با `errorElement` اختصاصی به صفحه‌ای قابل‌اقدام تبدیل می‌شوند؛ جدول‌ها، کنترل‌های خروجی و Heatmap نیز نام/جایگزین قابل‌دسترسی دارند.

## کیفیت

```bash
npm run typecheck
npm test
npm run build
```
