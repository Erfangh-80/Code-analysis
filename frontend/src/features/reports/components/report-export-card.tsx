import { CheckCircle2, Download, FileJson2, FileSpreadsheet, LoaderCircle, ShieldCheck } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { formatNumber, metricLabels } from "@/lib/analysis";
import { formatDateTime } from "@/lib/utils";
import { analysisApi } from "@/services/api/analysis-api";
import { getApiErrorMessage } from "@/services/api/client";
import type { AnalysisReport, AnalysisResultSummary } from "@/types/api";

interface ReportExportCardProps {
  analysisId: string;
  report: AnalysisReport;
  summary?: AnalysisResultSummary;
}

function filenameFromDisposition(contentDisposition: string | null, fallback: string) {
  const match = contentDisposition?.match(/filename="?([^";]+)"?/i);
  return match?.[1] ?? fallback;
}

export function ReportExportCard({ analysisId, report, summary }: ReportExportCardProps) {
  const [exporting, setExporting] = useState<"json" | "csv" | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [completedFormat, setCompletedFormat] = useState<"json" | "csv" | null>(null);
  const staticFiles = report.artifacts.static?.files ?? [];
  const riskFiles = report.artifacts.risk?.files ?? [];
  const coverage = report.artifacts.riskCoverage ?? summary?.riskCoverage;

  async function download(format: "json" | "csv") {
    setError(null);
    setCompletedFormat(null);
    setExporting(format);
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
      setCompletedFormat(format);
    } catch (exportError) {
      setError(getApiErrorMessage(exportError, "امکان دریافت گزارش وجود ندارد."));
    } finally {
      setExporting(null);
    }
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-start justify-between gap-4">
          <div>
            <CardTitle>دریافت گزارش تحلیل</CardTitle>
            <CardDescription>خلاصهٔ گزارش را بررسی کنید و سپس خروجی immutable ذخیره‌شده در backend را دریافت کنید.</CardDescription>
          </div>
          <span className="grid size-10 shrink-0 place-items-center rounded-xl bg-secondary text-secondary-foreground"><Download className="size-5" /></span>
        </div>
      </CardHeader>
      <CardContent className="space-y-5">
        <section className="grid gap-3 rounded-xl border bg-muted/25 p-4 sm:grid-cols-2 lg:grid-cols-4" aria-label="خلاصه پیش از خروجی">
          <div><p className="text-xs font-bold text-muted-foreground">فایل‌های گزارش</p><p className="mt-1 text-xl font-black">{formatNumber(staticFiles.length)}</p></div>
          <div><p className="text-xs font-bold text-muted-foreground">فایل‌های امتیازدهی‌شده</p><p className="mt-1 text-xl font-black">{formatNumber(riskFiles.length)}</p></div>
          <div><p className="text-xs font-bold text-muted-foreground">تکمیل تحلیل</p><p className="mt-1 text-sm font-black">{formatDateTime(report.analysis.completedAt)}</p></div>
          <div><p className="text-xs font-bold text-muted-foreground">نسخهٔ پروفایل</p><p className="mt-1 font-mono text-xs font-black" dir="ltr">{report.artifacts.risk?.profile.version ?? "—"}</p></div>
        </section>

        {coverage && (
          <section className="rounded-xl border p-4" aria-label="پوشش معیارهای ریسک">
            <div className="flex items-center gap-2"><ShieldCheck className="size-4 text-primary" /><h3 className="text-sm font-extrabold">پوشش معیارهای ریسک</h3></div>
            <div className="mt-3 grid gap-3 sm:grid-cols-2">
              <div><p className="text-xs font-bold text-muted-foreground">معیارهای در دسترس</p><div className="mt-2 flex flex-wrap gap-1.5">{(coverage.availableMetrics ?? []).map((metric) => <span key={metric} className="rounded-full bg-success/10 px-2 py-1 text-xs font-bold text-[color:var(--success)]">{metricLabels[metric]}</span>)}</div></div>
              <div><p className="text-xs font-bold text-muted-foreground">معیارهای در دسترس‌نبوده</p><div className="mt-2 flex flex-wrap gap-1.5">{(coverage.unavailableMetrics ?? []).length ? (coverage.unavailableMetrics ?? []).map((metric) => <span key={metric} className="rounded-full bg-muted px-2 py-1 text-xs font-bold text-muted-foreground">{metricLabels[metric]}</span>) : <span className="text-xs text-muted-foreground">موردی ندارد</span>}</div></div>
            </div>
          </section>
        )}

        <div className="grid gap-3 sm:grid-cols-2">
          <Button className="h-auto justify-start p-4 text-right" onClick={() => download("json")} disabled={exporting !== null} aria-describedby="json-export-description">
            {exporting === "json" ? <LoaderCircle className="size-5 animate-spin" /> : <FileJson2 className="size-5" />}
            <span><span className="block">دریافت گزارش JSON</span><span id="json-export-description" className="mt-1 block text-xs font-normal text-primary-foreground/80">کل گزارش immutable با ساختار کامل API</span></span>
          </Button>
          <Button className="h-auto justify-start p-4 text-right" variant="outline" onClick={() => download("csv")} disabled={exporting !== null} aria-describedby="csv-export-description">
            {exporting === "csv" ? <LoaderCircle className="size-5 animate-spin" /> : <FileSpreadsheet className="size-5" />}
            <span><span className="block">دریافت فهرست CSV</span><span id="csv-export-description" className="mt-1 block text-xs font-normal text-muted-foreground">ردیف‌های فایل‌ها برای spreadsheet با مقادیر در دسترس</span></span>
          </Button>
        </div>

        <div className="min-h-5" aria-live="polite">
          {error && <p className="rounded-lg border border-destructive/25 bg-destructive/7 p-3 text-sm text-destructive" role="alert">{error}</p>}
          {completedFormat && <p className="flex items-center gap-2 text-sm text-[color:var(--success)]"><CheckCircle2 className="size-4" />دانلود گزارش {completedFormat.toUpperCase()} آغاز شد.</p>}
        </div>
      </CardContent>
    </Card>
  );
}
