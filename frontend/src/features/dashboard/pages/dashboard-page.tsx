import { Activity, CheckCircle2, Files, FolderKanban } from "lucide-react";
import { Link } from "react-router-dom";

import { AnalysesTable } from "@/features/analyses/components/analyses-table";
import { OverviewMetricCard } from "@/features/dashboard/components/overview-metric-card";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { useAnalysesQuery } from "@/hooks/use-analyses-query";
import { useHealthQuery } from "@/hooks/use-health-query";

export function DashboardPage() {
  const health = useHealthQuery();
  const analyses = useAnalysesQuery();
  const totalAnalyses = analyses.data?.totalItems ?? "—";
  const completedAnalyses = analyses.data?.items.filter((item) => item.status === "completed").length ?? "—";

  return (
    <div className="space-y-6">
      <section className="flex flex-col justify-between gap-4 rounded-2xl border bg-card/80 p-6 shadow-card sm:flex-row sm:items-end">
        <div className="max-w-2xl">
          <p className="text-sm font-black text-primary">نمای کلی فضای کاری</p>
          <h2 className="mt-2 text-2xl font-black tracking-tight sm:text-3xl">یک نگاه شفاف به تحلیل‌های محلی</h2>
          <p className="mt-3 text-sm leading-6 text-muted-foreground">
            داده‌های این صفحه فقط از گزارش‌های ذخیره‌شده در سرویس محلی خوانده می‌شوند؛ هیچ شاخص نمونه یا ساختگی نمایش داده
            نمی‌شود.
          </p>
        </div>
        <Button size="lg" asChild>
          <Link to="/analyses/new">
            <Activity className="size-4" />
            شروع تحلیل جدید
          </Link>
        </Button>
      </section>

      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3" aria-label="شاخص‌های فضای کاری">
        <OverviewMetricCard
          label="تحلیل‌های ذخیره‌شده"
          value={totalAnalyses}
          description="تعداد واقعی تحلیل‌هایی که سرویس محلی برگردانده است."
          icon={FolderKanban}
        />
        <OverviewMetricCard
          label="تحلیل‌های کامل در این صفحه"
          value={completedAnalyses}
          description="این مقدار از ده تحلیل اخیر خوانده می‌شود، نه از دادهٔ نمونه."
          icon={CheckCircle2}
          tone="accent"
        />
        <OverviewMetricCard
          label="نسخه سرویس محلی"
          value={health.data?.version ?? "—"}
          description={health.isError ? "برای نمایش نسخه، سرویس Go محلی را اجرا کنید." : "از پاسخ health API خوانده شده است."}
          icon={Files}
          tone="warning"
        />
      </section>

      <section className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_20rem]">
        <AnalysesTable />
        <Card className="h-fit">
          <CardHeader>
            <CardTitle>گام بعدی</CardTitle>
            <CardDescription>مسیر پیشنهادی استفاده از داشبورد</CardDescription>
          </CardHeader>
          <CardContent>
            <ol className="space-y-4 text-sm">
              <li className="flex gap-3">
                <span className="grid size-6 shrink-0 place-items-center rounded-full bg-secondary text-xs font-black text-secondary-foreground">۱</span>
                <span>مسیر مطلق یک پروژهٔ قابل‌دسترسی را وارد کنید.</span>
              </li>
              <li className="flex gap-3">
                <span className="grid size-6 shrink-0 place-items-center rounded-full bg-secondary text-xs font-black text-secondary-foreground">۲</span>
                <span>وضعیت مرحله‌های تحلیل را در فهرست تحلیل‌ها دنبال کنید.</span>
              </li>
              <li className="flex gap-3">
                <span className="grid size-6 shrink-0 place-items-center rounded-full bg-secondary text-xs font-black text-secondary-foreground">۳</span>
                <span>پس از تکمیل، گزارش‌ها و exportها را از همان تحلیل مشاهده کنید.</span>
              </li>
            </ol>
          </CardContent>
        </Card>
      </section>
    </div>
  );
}
