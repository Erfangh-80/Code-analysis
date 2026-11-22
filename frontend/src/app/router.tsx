import { FileQuestion } from "lucide-react";
import { createBrowserRouter, Link } from "react-router-dom";

import { RouteErrorPage } from "@/app/route-error-page";
import { DashboardLayout } from "@/components/templates/dashboard-layout";
import { Button } from "@/components/ui/button";
import { CoreAnalysisPage } from "@/features/core/pages/core-analysis-page";
import { GuidePage } from "@/features/core/pages/guide-page";

function NotFoundPage() {
  return (
    <div className="flex min-h-96 flex-col items-center justify-center rounded-2xl border border-dashed bg-card p-8 text-center">
      <FileQuestion className="size-9 text-muted-foreground" aria-hidden="true" />
      <h2 className="mt-4 text-xl font-black">این صفحه پیدا نشد</h2>
      <p className="mt-2 text-sm text-muted-foreground">مسیر موردنظر در workspace هستهٔ تحلیل وجود ندارد.</p>
      <Button className="mt-5" asChild><Link to="/">بازگشت به هستهٔ تحلیل</Link></Button>
    </div>
  );
}

export const router = createBrowserRouter([
  {
    element: <DashboardLayout />,
    errorElement: <RouteErrorPage />,
    children: [
      {
        index: true,
        element: <CoreAnalysisPage />,
        handle: {
          title: "هستهٔ تحلیل ریسک کد",
          description: "تحلیل ایستا، تاریخچهٔ Git و رتبه‌بندی Hotspot روی دستگاه محلی.",
        },
      },
      {
        path: "guide",
        element: <GuidePage />,
        handle: {
          title: "راهنمای استفاده",
          description: "توضیح امکانات، معیارها و معنی سطوح ریسک.",
        },
      },
      {
        path: "*",
        element: <NotFoundPage />,
        handle: {
          title: "صفحه پیدا نشد",
          description: "مسیر واردشده در این خروجی فازهای ۱ تا ۴ وجود ندارد.",
        },
      },
    ],
  },
]);
