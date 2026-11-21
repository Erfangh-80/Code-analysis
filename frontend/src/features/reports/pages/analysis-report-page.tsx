import { ArrowRight, FileText } from "lucide-react";
import { Link, useParams } from "react-router-dom";

import { ApiErrorState, EmptyState, LoadingState } from "@/components/molecules/async-state";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { AnalysisProgressCard } from "@/features/analyses/components/analysis-progress-card";
import { ReportExportCard } from "@/features/reports/components/report-export-card";
import { useAnalysisQuery, useAnalysisReportQuery, useAnalysisSummaryQuery } from "@/hooks/use-analysis-queries";

export function AnalysisReportPage() {
  const { analysisId } = useParams();
  const analysis = useAnalysisQuery(analysisId);
  const run = analysis.data?.analysis;
  const completed = run?.status === "completed";
  const report = useAnalysisReportQuery(analysisId, completed);
  const summary = useAnalysisSummaryQuery(analysisId, completed);

  if (!analysisId) {
    return <EmptyState title="شناسهٔ تحلیل مشخص نیست" description="یک تحلیل تکمیل‌شده را از فهرست انتخاب کنید." />;
  }
  if (analysis.isPending) return <LoadingState />;
  if (analysis.isError || !run) return <ApiErrorState error={analysis.error} onRetry={() => analysis.refetch()} title="وضعیت تحلیل قابل دریافت نیست" />;

  if (!completed) {
    return (
      <div className="space-y-6">
        <Card className="bg-secondary/30"><CardHeader><CardTitle>گزارش هنوز آماده نیست</CardTitle><CardDescription>پس از تکمیل موفق تحلیل، گزارش JSON و CSV از artifact ذخیره‌شده دریافت می‌شود.</CardDescription></CardHeader><CardContent><Button variant="outline" asChild><Link to={`/analyses/${encodeURIComponent(analysisId)}`}><ArrowRight className="size-3.5" />بازگشت به وضعیت تحلیل</Link></Button></CardContent></Card>
        <AnalysisProgressCard analysis={run} />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-6xl space-y-6">
      <section className="flex flex-col justify-between gap-4 rounded-2xl border bg-card/80 p-6 shadow-card sm:flex-row sm:items-end">
        <div><p className="text-sm font-black text-primary">گزارش تکمیل‌شده</p><h2 className="mt-2 text-2xl font-black tracking-tight">بازبینی و خروجی گرفتن از تحلیل</h2><p className="mt-2 text-sm leading-6 text-muted-foreground">خروجی‌ها مستقیماً از گزارش immutable ذخیره‌شده در SQLite سرویس محلی ساخته می‌شوند.</p></div>
        <Button variant="outline" asChild><Link to={`/analyses/${encodeURIComponent(analysisId)}`}><FileText className="size-4" />بازگشت به تحلیل</Link></Button>
      </section>
      {report.isPending && <LoadingState />}
      {report.isError && <ApiErrorState error={report.error} onRetry={() => report.refetch()} title="پیش‌نمایش گزارش قابل دریافت نیست" />}
      {report.data && <ReportExportCard analysisId={analysisId} report={report.data} summary={summary.data} />}
    </div>
  );
}
