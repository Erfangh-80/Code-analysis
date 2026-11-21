import { Activity, ChartNoAxesCombined, Files, GitCompareArrows, ScanSearch, ShieldAlert } from "lucide-react";
import { Link } from "react-router-dom";

import { ApiErrorState, EmptyState, LoadingState } from "@/components/molecules/async-state";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { AnalysesTable } from "@/features/analyses/components/analyses-table";
import { RiskHeatmap } from "@/features/analyses/components/risk-heatmap";
import { OverviewMetricCard } from "@/features/dashboard/components/overview-metric-card";
import { RiskDistributionChart } from "@/features/dashboard/components/risk-distribution-chart";
import { TopRiskyFiles } from "@/features/dashboard/components/top-risky-files";
import { useAnalysisHeatmapQuery, useAnalysisHotspotsQuery, useAnalysisReportQuery, useAnalysisSummaryQuery } from "@/hooks/use-analysis-queries";
import { useAnalysesQuery } from "@/hooks/use-analyses-query";
import { useHealthQuery } from "@/hooks/use-health-query";
import { formatNumber } from "@/lib/analysis";
import { formatDateTime } from "@/lib/utils";

function valueOrDash(value: number | undefined) {
  return value === undefined ? "—" : formatNumber(value);
}

export function DashboardPage() {
  const health = useHealthQuery();
  const recentAnalyses = useAnalysesQuery({ page: 1, pageSize: 5 });
  const latestCompleted = useAnalysesQuery({ page: 1, pageSize: 1, status: "completed" });
  const analysisId = latestCompleted.data?.items[0]?.id;
  const summary = useAnalysisSummaryQuery(analysisId);
  const hotspots = useAnalysisHotspotsQuery(analysisId, { page: 1, pageSize: 5, sort: "score", direction: "desc" });
  const heatmap = useAnalysisHeatmapQuery(analysisId);
  const report = useAnalysisReportQuery(analysisId);
  const latest = latestCompleted.data?.items[0];
  const staticFiles = report.data?.artifacts.static?.files ?? [];
  const gitFiles = report.data?.artifacts.git?.files ?? [];
  const averageFunctionLength = staticFiles.reduce((sum, file) => sum + file.metrics.functionLength.total, 0) /
    Math.max(1, staticFiles.reduce((sum, file) => sum + file.metrics.functionCount, 0));
  const totalChurn = gitFiles.reduce((sum, file) => sum + file.gitChurn, 0);
  const totalChangeFrequency = gitFiles.reduce((sum, file) => sum + file.changeFrequency, 0);

  return (
    <div className="space-y-6">
      <section className="flex flex-col justify-between gap-4 rounded-2xl border bg-card/80 p-6 shadow-card sm:flex-row sm:items-end">
        <div className="max-w-2xl">
          <p className="text-sm font-black text-primary">داشبورد گزارش‌های محلی</p>
          <h2 className="mt-2 text-2xl font-black tracking-tight sm:text-3xl">از وضعیت Repository تا دلیل هر Hotspot</h2>
          <p className="mt-3 text-sm leading-6 text-muted-foreground">
            شاخص‌ها از آخرین تحلیل تکمیل‌شده خوانده می‌شوند. داده‌های ناقص Git یا معیارهای در دسترس‌نبوده با مقدار ساختگی جایگزین نمی‌شوند.
          </p>
        </div>
        <Button size="lg" asChild>
          <Link to="/analyses/new"><Activity className="size-4" />شروع تحلیل جدید</Link>
        </Button>
      </section>

      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4" aria-label="وضعیت سرویس و تحلیل‌ها">
        <OverviewMetricCard label="تحلیل‌های ذخیره‌شده" value={recentAnalyses.data?.totalItems ?? "—"} description="تعداد اجراهای واقعی نگهداری‌شده در سرویس محلی." icon={Files} />
        <OverviewMetricCard label="نسخهٔ سرویس محلی" value={health.data?.version ?? "—"} description={health.isError ? "برای مشاهده نسخه، سرویس Go را اجرا کنید." : "از پاسخ health API خوانده شده است."} icon={ScanSearch} tone="accent" />
        <OverviewMetricCard label="آخرین تحلیل" value={latest ? formatDateTime(latest.completedAt ?? latest.createdAt) : "—"} description={latest ? "زمان تکمیل آخرین گزارش قابل‌نمایش." : "هنوز گزارشی تکمیل نشده است."} icon={ChartNoAxesCombined} tone="warning" />
        <OverviewMetricCard label="وضعیت Git" value={summary.data?.summary.gitStatus ?? "—"} description="Git اختیاری است و نبود آن تحلیل ایستا را متوقف نمی‌کند." icon={GitCompareArrows} tone="primary" />
      </section>

      {latestCompleted.isPending && <LoadingState />}
      {latestCompleted.isError && <ApiErrorState error={latestCompleted.error} onRetry={() => latestCompleted.refetch()} title="آخرین تحلیل قابل دریافت نیست" />}
      {latestCompleted.isSuccess && !latest && (
        <EmptyState
          title="هنوز تحلیل تکمیل‌شده‌ای وجود ندارد"
          description="برای دیدن Dashboard جزئیات پروژه، یک مسیر محلی معتبر ثبت کنید و تا پایان تحلیل منتظر بمانید."
          action={<Button asChild><Link to="/analyses/new">شروع اولین تحلیل</Link></Button>}
        />
      )}

      {latest && (
        <>
          {summary.isPending && <LoadingState />}
          {summary.isError && <ApiErrorState error={summary.error} onRetry={() => summary.refetch()} title="خلاصهٔ تحلیل در دسترس نیست" />}
          {summary.data && (
            <>
              <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4" aria-label="خلاصه Repository">
                <OverviewMetricCard label="فایل‌های تحلیل‌شده" value={formatNumber(summary.data.summary.filesAnalyzed)} description={`${formatNumber(summary.data.summary.filesWithErrors)} فایل با خطا یا هشدار ثبت شده است.`} icon={Files} />
                <OverviewMetricCard label="ریسک زیاد" value={formatNumber(summary.data.riskLevels.high)} description="فایل‌هایی که به بازبینی اولویت‌دار نیاز دارند." icon={ShieldAlert} tone="warning" />
                <OverviewMetricCard label="ریسک متوسط / کم" value={`${formatNumber(summary.data.riskLevels.medium)} / ${formatNumber(summary.data.riskLevels.low)}`} description="توزیع واقعی فایل‌های امتیازدهی‌شده." icon={ChartNoAxesCombined} tone="accent" />
                <OverviewMetricCard label="میانگین طول تابع" value={staticFiles.length ? averageFunctionLength.toFixed(1) : "—"} description="Cyclomatic Complexity در قرارداد API v1 تولید نمی‌شود؛ طول تابع نزدیک‌ترین شاخص ساختاری موجود است." icon={ScanSearch} />
              </section>

              <section className="grid gap-4 sm:grid-cols-2" aria-label="خلاصه Git">
                <OverviewMetricCard label="Git Churn کل" value={report.isPending ? "…" : valueOrDash(report.data?.artifacts.git ? totalChurn : undefined)} description={report.data?.artifacts.git ? "مجموع خطوط اضافه/حذف‌شده در تاریخچهٔ انتخاب‌شده." : "دادهٔ Git برای این گزارش در دسترس نیست."} icon={GitCompareArrows} tone="warning" />
                <OverviewMetricCard label="Change Frequency کل" value={report.isPending ? "…" : valueOrDash(report.data?.artifacts.git ? totalChangeFrequency : undefined)} description={report.data?.artifacts.git ? "مجموع commitهای مؤثر بر فایل‌های تحلیل‌شده." : "دادهٔ Git برای این گزارش در دسترس نیست."} icon={Activity} tone="accent" />
              </section>

              <section className="grid gap-6 xl:grid-cols-2">
                <RiskDistributionChart counts={summary.data.riskLevels} />
                {hotspots.isPending ? <LoadingState /> : hotspots.isError ? <ApiErrorState error={hotspots.error} onRetry={() => hotspots.refetch()} title="فایل‌های پرریسک در دسترس نیستند" /> : <TopRiskyFiles analysisId={latest.id} files={hotspots.data?.items ?? []} />}
              </section>

              {heatmap.isPending ? <LoadingState /> : heatmap.isError ? <ApiErrorState error={heatmap.error} onRetry={() => heatmap.refetch()} title="Heatmap این تحلیل قابل دریافت نیست" /> : heatmap.data && <RiskHeatmap analysisId={latest.id} data={heatmap.data} />}

              <Card className="bg-secondary/30">
                <CardHeader><CardTitle>آخرین تحلیل</CardTitle><CardDescription>گزارش تکمیل‌شدهٔ فعلی با شناسهٔ امن و زمان ثبت‌شده در سرویس محلی.</CardDescription></CardHeader>
                <CardContent className="flex flex-wrap items-center justify-between gap-3">
                  <div><p className="font-mono text-xs" dir="ltr">{latest.id}</p><p className="mt-1 text-xs text-muted-foreground">تکمیل: {formatDateTime(latest.completedAt)}</p></div>
                  <Button variant="outline" asChild><Link to={`/analyses/${encodeURIComponent(latest.id)}`}>مشاهدهٔ گزارش کامل</Link></Button>
                </CardContent>
              </Card>
            </>
          )}
        </>
      )}

      <section className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_20rem]">
        <AnalysesTable />
        <Card className="h-fit">
          <CardHeader><CardTitle>راهنمای سریع</CardTitle><CardDescription>مسیر پیشنهادی بررسی ریسک</CardDescription></CardHeader>
          <CardContent><ol className="space-y-4 text-sm"><li>۱. یک تحلیل محلی ثبت کنید.</li><li>۲. مرحله‌ها و پیشرفت آن را دنبال کنید.</li><li>۳. Heatmap و فایل‌های پرریسک را برای تصمیم‌گیری باز کنید.</li></ol></CardContent>
        </Card>
      </section>
    </div>
  );
}
