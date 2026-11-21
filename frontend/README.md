# Frontend

داشبورد فارسی و RTL پروژهٔ **Code Risk Analyzer** با React، TypeScript و Vite ساخته شده است. این برنامه فقط با API محلی ارتباط می‌گیرد؛ هیچ مسیر پروژه یا کد منبعی به سرویس خارجی ارسال نمی‌شود.

## فناوری‌ها و معماری

- React 19 + TypeScript + Vite
- Tailwind CSS v4 و design tokenهای قابل استفاده مجدد در `src/styles/globals.css`
- کامپوننت‌های پایهٔ shadcn/ui در `src/components/ui`
- React Router برای routeهای داشبورد
- TanStack Query برای درخواست‌ها و cache API
- Zustand برای state رابط کاربری (مانند sidebar موبایل و ترجیحات نمایش)
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
| `/analyses/new` | ثبت یک تحلیل جدید با مسیر محلی پروژه |
| `/settings` | ترجیحات محلی رابط کاربری |

تمام داده‌های تحلیل در UI از API واقعی خوانده می‌شوند؛ صفحه‌ها دادهٔ نمونه یا نتیجهٔ ساختگی نمایش نمی‌دهند.

## کیفیت

```bash
npm run typecheck
npm test
npm run build
```
