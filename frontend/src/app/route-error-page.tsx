import { AlertTriangle, Home, RefreshCw } from "lucide-react";
import { isRouteErrorResponse, Link, useRouteError } from "react-router-dom";

import { Button } from "@/components/ui/button";

function safeMessage(error: unknown) {
  if (isRouteErrorResponse(error)) {
    return error.status === 404 ? "صفحه یا تحلیل موردنظر پیدا نشد." : "در دریافت این صفحه مشکلی رخ داد.";
  }
  return "نمایش این بخش با خطای غیرمنتظره مواجه شد.";
}

// The router catches render failures here instead of exposing React's developer
// stack screen to an end user. The actionable error text stays intentionally
// short and never serializes an internal stack trace.
export function RouteErrorPage() {
  const error = useRouteError();

  return (
    <main className="grid min-h-screen place-items-center bg-background p-5 text-foreground" dir="rtl">
      <section className="w-full max-w-xl rounded-2xl border bg-card p-7 text-center shadow-card" role="alert">
        <span className="mx-auto grid size-12 place-items-center rounded-2xl bg-destructive/10 text-destructive">
          <AlertTriangle className="size-6" aria-hidden="true" />
        </span>
        <h1 className="mt-5 text-xl font-black">نمایش این صفحه ممکن نیست</h1>
        <p className="mt-2 text-sm leading-6 text-muted-foreground">{safeMessage(error)}</p>
        <p className="mt-2 text-xs leading-5 text-muted-foreground">می‌توانید دوباره تلاش کنید یا به نمای کلی برگردید. جزئیات فنی خطا در رابط کاربری نمایش داده نمی‌شود.</p>
        <div className="mt-6 flex flex-wrap justify-center gap-3">
          <Button onClick={() => window.location.reload()}>
            <RefreshCw className="size-4" />
            تلاش دوباره
          </Button>
          <Button variant="outline" asChild>
            <Link to="/"><Home className="size-4" />بازگشت به نمای کلی</Link>
          </Button>
        </div>
      </section>
    </main>
  );
}
