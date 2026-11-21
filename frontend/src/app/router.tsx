import { FileQuestion } from "lucide-react";
import { createBrowserRouter, Link } from "react-router-dom";

import { RouteErrorPage } from "@/app/route-error-page";
import { DashboardLayout } from "@/components/templates/dashboard-layout";
import { Button } from "@/components/ui/button";
import { AnalysisDetailPage } from "@/features/analyses/pages/analysis-detail-page";
import { AnalysisFilesPage } from "@/features/analyses/pages/analysis-files-page";
import { AnalysesPage } from "@/features/analyses/pages/analyses-page";
import { FileDetailPage } from "@/features/analyses/pages/file-detail-page";
import { NewAnalysisPage } from "@/features/analyses/pages/new-analysis-page";
import { AnalysisReportPage } from "@/features/reports/pages/analysis-report-page";
import { DashboardPage } from "@/features/dashboard/pages/dashboard-page";
import { SettingsPage } from "@/features/settings/pages/settings-page";

function NotFoundPage() {
  return (
    <div className="flex min-h-96 flex-col items-center justify-center rounded-2xl border border-dashed bg-card p-8 text-center">
      <FileQuestion className="size-9 text-muted-foreground" aria-hidden="true" />
      <h2 className="mt-4 text-xl font-black">این صفحه پیدا نشد</h2>
      <p className="mt-2 text-sm text-muted-foreground">مسیر موردنظر در داشبورد محلی وجود ندارد.</p>
      <Button className="mt-5" asChild>
        <Link to="/">بازگشت به نمای کلی</Link>
      </Button>
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
        element: <DashboardPage />,
        handle: {
          title: "نمای کلی",
          description: "وضعیت فضای کاری و تحلیل‌های اخیر شما.",
        },
      },
      {
        path: "analyses",
        element: <AnalysesPage />,
        handle: {
          title: "تحلیل‌ها",
          description: "فهرست اجراهای ذخیره‌شده و وضعیت فعلی آن‌ها.",
        },
      },
      {
        path: "analyses/new",
        element: <NewAnalysisPage />,
        handle: {
          title: "شروع تحلیل",
          description: "یک پروژهٔ قابل‌دسترسی روی دستگاه محلی را انتخاب کنید.",
        },
      },
      {
        path: "analyses/:analysisId/report",
        element: <AnalysisReportPage />,
        handle: {
          title: "گزارش و خروجی",
          description: "پیش‌نمایش گزارش تکمیل‌شده و دریافت JSON یا CSV.",
        },
      },
      {
        path: "analyses/:analysisId/files/detail",
        element: <FileDetailPage />,
        handle: {
          title: "جزئیات فایل",
          description: "شاخص‌های ایستا، Git و علت امتیاز ریسک یک فایل.",
        },
      },
      {
        path: "analyses/:analysisId/files",
        element: <AnalysisFilesPage />,
        handle: {
          title: "فایل‌های تحلیل‌شده",
          description: "جست‌وجو و فیلتر فایل‌ها بر اساس شاخص‌های گزارش.",
        },
      },
      {
        path: "analyses/:analysisId",
        element: <AnalysisDetailPage />,
        handle: {
          title: "گزارش تحلیل",
          description: "پیشرفت، خلاصهٔ Repository و Hotspotهای تحلیل محلی.",
        },
      },
      {
        path: "settings",
        element: <SettingsPage />,
        handle: {
          title: "تنظیمات",
          description: "ترجیحات رابط کاربری محلی خود را مدیریت کنید.",
        },
      },
      {
        path: "*",
        element: <NotFoundPage />,
        handle: {
          title: "صفحه پیدا نشد",
          description: "مسیر واردشده در دسترس نیست.",
        },
      },
    ],
  },
]);
