import { Download, FileText, Layers3, LoaderCircle, Route, ShieldAlert } from "lucide-react";
import { useState } from "react";
import { Link, useParams } from "react-router-dom";

import { ApiErrorState, EmptyState, LoadingState } from "@/components/molecules/async-state";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { AnalysisProgressCard } from "@/features/analyses/components/analysis-progress-card";
import { RiskHeatmap } from "@/features/analyses/components/risk-heatmap";
import { RiskLevelBadge } from "@/features/analyses/components/risk-level-badge";
import { useAnalysisHeatmapQuery, useAnalysisHotspotsQuery, useAnalysisQuery, useAnalysisReportQuery, useAnalysisSummaryQuery, useCancelAnalysisMutation } from "@/hooks/use-analysis-queries";
import { formatNumber, metricLabels } from "@/lib/analysis";
import { formatDateTime } from "@/lib/utils";
import { analysisApi } from "@/services/api/analysis-api";
import { getApiErrorMessage } from "@/services/api/client";

function filenameFromDisposition(contentDisposition: string | null, fallback: string) {
  const match = contentDisposition?.match(/filename="?([^";]+)"?/i);
  return match?.[1] ?? fallback;
}

export function AnalysisDetailPage() {
  const { analysisId } = useParams();
  const analysis = useAnalysisQuery(analysisId);
  const data = analysis.data?.analysis;
  const completed = data?.status === "completed";
  const summary = useAnalysisSummaryQuery(analysisId, completed);
  const hotspots = useAnalysisHotspotsQuery(analysisId, { page: 1, pageSize: 10, sort: "score", direction: "desc" }, completed);
  const heatmap = useAnalysisHeatmapQuery(analysisId, completed);
  const report = useAnalysisReportQuery(analysisId, completed);
  const cancel = useCancelAnalysisMutation(analysisId ?? "");
  const [exporting, setExporting] = useState<"json" | "csv" | null>(null);
  const [exportError, setExportError] = useState<string | null>(null);

  async function download(format: "json" | "csv") {
    if (!analysisId) return;
    setExporting(format);
    setExportError(null);
    try {
      const file = await analysisApi.export(analysisId, format, { timeoutMs: 60_000 });
      const url = URL.createObjectURL(file.blob);
      const anchor = document.createElement("a");
      anchor.href = url;
      anchor.download = filenameFromDisposition(file.contentDisposition, `analysis-${analysisId}.${format}`);
      document.body.append(anchor);
      anchor.click();
      anchor.remove();
      URL.revokeObjectURL(url);
    } catch (error) {
      setExportError(getApiErrorMessage(error, "امکان دریافت خروجی گزارش وجود ندارد."));
    } finally {
      setExporting(null);
    }
  }

  if (!analysisId) {
    return <EmptyState title="شناسهٔ تحلیل مشخص نیست" description="برای مشاهدهٔ نتیجه، یک تحلیل معتبر را از فهرست انتخاب کنید." />;
  }

  if (analysis.isPending) {
    return <LoadingState />;
  }

  if (analysis.isError || !data) {
    return <ApiErrorState error={analysis.error} onRetry={() => analysis.refetch()} title="اطلاعات تحلیل قابل دریافت نیست" />;
  }

  return (
    <div className="space-y-6">
      <section className="flex flex-col justify-between gap-4 rounded-2xl border bg-card/80 p-6 shadow-card lg:flex-row lg:items-start">
        <div className="min-w-0">
          <p className="text-sm font-black text-primary">جزئیات تحلیل محلی</p>
          <h2 className="mt-2 break-all font-mono text-lg font-black tracking-tight sm:text-xl" dir="ltr">{data.id}</h2>
          <p className="mt-2 text-sm text-muted-foreground">شروع: {formatDateTime(data.startedAt ?? data.createdAt)} · پایان: {formatDateTime(data.completedAt)}</p>
        </div>
        {completed && (
          <div className="flex flex-wrap gap-2">
            <Button variant="outline" size="sm" disabled={exporting !== null} onClick={() => download("json")}>
              {exporting === "json" ? <LoaderCircle className="size-3.5 animate-spin" /> : <Download className="size-3.5" />} JSON
            </Button>
            <Button variant="outline" size="sm" disabled={exporting !== null} onClick={() => download("csv")}>
              {exporting === "csv" ? <LoaderCircle className="size-3.5 animate-spin" /> : <Download className="size-3.5" />} CSV
            </Button>
            <Button size="sm" asChild><Link to={`/analyses/${encodeURIComponent(data.id)}/files`}><FileText className="size-3.5" />فایل‌ها</Link></Button>
          </div>
        )}
      </section>

      {exportError && <div className="rounded-lg border border-destructive/25 bg-destructive/7 p-3 text-sm text-destructive" role="alert">{exportError}</div>}

      <AnalysisProgressCard analysis={data} onCancel={() => cancel.mutate()} isCancelling={cancel.isPending} />
      {cancel.isError && <div className="rounded-lg border border-destructive/25 bg-destructive/7 p-3 text-sm text-destructive">{getApiErrorMessage(cancel.error)}</div>}

      {!completed && data.status !== "failed" && data.status !== "cancelled" && (
        <Card className="bg-secondary/30"><CardContent className="p-5 text-sm leading-6 text-muted-foreground">مسیر Repository توسط backend برای ایجاد تحلیل پذیرفته شده است. برای حفظ حریم مسیر محلی، مقدار مطلق آن در پاسخ بازگردانده نمی‌شود. پس از تکمیل، خلاصهٔ Repository، Heatmap، Hotspotها و جدول فایل‌ها در همین صفحه فعال می‌شوند؛ وضعیت هم به‌صورت خودکار به‌روزرسانی می‌شود.</CardContent></Card>
      )}

      {completed && (
        <>
          {summary.isPending && <LoadingState />}
          {summary.isError && <ApiErrorState error={summary.error} onRetry={() => summary.refetch()} title="خلاصهٔ گزارش قابل دریافت نیست" />}
          {summary.data && (
            <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
              <Card><CardContent className="p-5"><p className="text-xs font-bold text-muted-foreground">فایل‌های تحلیل‌شده</p><p className="mt-2 text-3xl font-black">{formatNumber(summary.data.summary.filesAnalyzed)}</p><p className="mt-2 text-xs text-muted-foreground">{formatNumber(summary.data.summary.filesWithErrors)} فایل با خطا</p></CardContent></Card>
              <Card><CardContent className="p-5"><p className="text-xs font-bold text-muted-foreground">فایل‌های امتیازدهی‌شده</p><p className="mt-2 text-3xl font-black">{formatNumber(summary.data.summary.scoredFiles)}</p><p className="mt-2 text-xs text-muted-foreground">{formatNumber(summary.data.summary.unscoredFiles)} فایل بدون امتیاز</p></CardContent></Card>
              <Card><CardContent className="p-5"><p className="text-xs font-bold text-muted-foreground">ماژول‌ها</p><p className="mt-2 text-3xl font-black">{formatNumber(summary.data.moduleCount)}</p><p className="mt-2 text-xs text-muted-foreground">Git: {summary.data.summary.gitStatus}</p></CardContent></Card>
              <Card><CardContent className="p-5"><p className="text-xs font-bold text-muted-foreground">سطوح ریسک</p><div className="mt-3 flex flex-wrap gap-1.5"><RiskLevelBadge level="high" /><span className="text-xs font-black">{formatNumber(summary.data.riskLevels.high)}</span><RiskLevelBadge level="medium" /><span className="text-xs font-black">{formatNumber(summary.data.riskLevels.medium)}</span></div></CardContent></Card>
            </section>
          )}

          {report.data?.artifacts.risk?.limitations.length ? (
            <Card className="border-warning/25 bg-warning/8"><CardHeader><CardTitle className="flex items-center gap-2 text-base"><ShieldAlert className="size-4 text-[color:var(--warning)]" />محدودیت‌های مدل ریسک</CardTitle></CardHeader><CardContent><ul className="space-y-1 text-sm leading-6 text-muted-foreground">{report.data.artifacts.risk.limitations.map((limitation) => <li key={limitation}>• {limitation}</li>)}</ul></CardContent></Card>
          ) : null}

          <section className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_22rem]">
            <Card>
              <CardHeader><CardTitle>Top Risky Files</CardTitle><CardDescription>دلایل ریسک هر فایل از contributionهای محاسبه‌شده در backend می‌آید.</CardDescription></CardHeader>
              <CardContent>
                {hotspots.isPending && <LoadingState />}
                {hotspots.isError && <ApiErrorState error={hotspots.error} onRetry={() => hotspots.refetch()} title="Hotspotها قابل دریافت نیستند" />}
                {hotspots.data && <ol className="divide-y">{hotspots.data.items.map((hotspot) => (
                  <li key={hotspot.path} className="py-3 first:pt-0 last:pb-0"><div className="flex items-start justify-between gap-3"><Link className="min-w-0 truncate font-mono text-xs font-bold hover:text-primary hover:underline" dir="ltr" to={`/analyses/${encodeURIComponent(data.id)}/files/detail?path=${encodeURIComponent(hotspot.path)}`}>{hotspot.path}</Link><div className="flex shrink-0 items-center gap-2"><RiskLevelBadge level={hotspot.level} /><span className="font-black text-destructive">{hotspot.score}</span></div></div><p className="mt-2 text-xs text-muted-foreground">{hotspot.contributions.filter((entry) => entry.weighted > 0).sort((a, b) => b.weighted - a.weighted).slice(0, 2).map((entry) => metricLabels[entry.metric]).join(" · ") || "معیار مؤثری ثبت نشده است"}</p></li>
                ))}</ol>}
              </CardContent>
            </Card>
            <Card className="h-fit"><CardHeader><Layers3 className="size-5 text-primary" /><CardTitle className="mt-2">مسیر بررسی</CardTitle></CardHeader><CardContent className="space-y-3 text-sm leading-6 text-muted-foreground"><p>نقاط Heatmap یا نام هر فایل را انتخاب کنید تا جزئیات شاخص‌ها و علت ریسک را ببینید.</p><Button className="w-full" variant="outline" asChild><Link to={`/analyses/${encodeURIComponent(data.id)}/files`}><Route className="size-3.5" />باز کردن جدول فایل‌ها</Link></Button></CardContent></Card>
          </section>

          {heatmap.isPending && <LoadingState />}
          {heatmap.isError && <ApiErrorState error={heatmap.error} onRetry={() => heatmap.refetch()} title="Heatmap قابل دریافت نیست" />}
          {heatmap.data && <RiskHeatmap analysisId={data.id} data={heatmap.data} />}
        </>
      )}
    </div>
  );
}
