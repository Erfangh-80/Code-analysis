import { ArrowRight, CalendarRange, Code2, GitCommitHorizontal, ShieldAlert, UsersRound } from "lucide-react";
import { Link, useParams, useSearchParams } from "react-router-dom";

import { ApiErrorState, EmptyState, LoadingState } from "@/components/molecules/async-state";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { RiskLevelBadge } from "@/features/analyses/components/risk-level-badge";
import { useAnalysisFileQuery } from "@/hooks/use-analysis-queries";
import { fileStatusLabels, formatNumber, languageLabels, metricLabels } from "@/lib/analysis";
import { formatDateTime } from "@/lib/utils";

function MetricItem({ label, value, detail }: { label: string; value: string | number; detail?: string }) {
  return <div className="rounded-lg border bg-muted/25 p-3"><p className="text-xs font-bold text-muted-foreground">{label}</p><p className="mt-1 text-xl font-black">{value}</p>{detail && <p className="mt-1 text-xs text-muted-foreground">{detail}</p>}</div>;
}

export function FileDetailPage() {
  const { analysisId } = useParams();
  const [searchParams] = useSearchParams();
  const path = searchParams.get("path") ?? undefined;
  const file = useAnalysisFileQuery(analysisId, path);

  if (!analysisId || !path) {
    return <EmptyState title="فایل مشخص نشده است" description="از جدول فایل‌ها یا Heatmap یک فایل را برای مشاهدهٔ جزئیات انتخاب کنید." />;
  }
  if (file.isPending) return <LoadingState />;
  if (file.isError || !file.data) return <ApiErrorState error={file.error} onRetry={() => file.refetch()} title="جزئیات فایل قابل دریافت نیست" />;

  const { analysis, git, risk } = file.data;
  const previousPaths = git?.previousPaths ?? [];
  const contributions = (risk?.contributions ?? []).slice().sort((left, right) => right.weighted - left.weighted);
  const functions = analysis.functions ?? [];
  const diagnostics = analysis.diagnostics ?? [];

  return (
    <div className="space-y-6">
      <section className="rounded-2xl border bg-card/80 p-6 shadow-card">
        <Button className="mb-4" variant="ghost" size="sm" asChild><Link to={`/analyses/${encodeURIComponent(analysisId)}/files`}><ArrowRight className="size-3.5" />بازگشت به فهرست فایل‌ها</Link></Button>
        <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-start">
          <div className="min-w-0"><p className="text-sm font-black text-primary">جزئیات فایل</p><h2 className="mt-2 break-all font-mono text-base font-black sm:text-xl" dir="ltr">{analysis.path}</h2><p className="mt-2 text-sm text-muted-foreground">{languageLabels[analysis.language]} · {fileStatusLabels[analysis.status]}{analysis.usedFallback ? " · تحلیل جایگزین استفاده شده" : ""}</p></div>
          {risk ? <div className="flex items-center gap-3"><RiskLevelBadge level={risk.level} /><span className="grid size-14 place-items-center rounded-xl bg-destructive/10 text-2xl font-black text-destructive">{risk.score}</span></div> : <span className="rounded-lg bg-muted px-3 py-2 text-sm text-muted-foreground">امتیاز ریسک در دسترس نیست</span>}
        </div>
      </section>

      <section className="grid gap-6 xl:grid-cols-2">
        <Card>
          <CardHeader><div className="flex items-center gap-2"><Code2 className="size-5 text-primary" /><CardTitle>Static Analysis</CardTitle></div><CardDescription>شاخص‌های واقعی استخراج‌شده از فایل بدون اجرای کد.</CardDescription></CardHeader>
          <CardContent className="grid gap-3 sm:grid-cols-2"><MetricItem label="LOC" value={formatNumber(analysis.metrics.linesOfCode)} /><MetricItem label="Cyclomatic Complexity" value="—" detail="در قرارداد API v1 در دسترس نیست" /><MetricItem label="تعداد تابع" value={formatNumber(analysis.metrics.functionCount)} detail={`بیشترین طول: ${formatNumber(analysis.metrics.functionLength.max)}`} /><MetricItem label="تعداد کلاس" value={formatNumber(analysis.metrics.classCount)} detail={`بیشترین طول: ${formatNumber(analysis.metrics.classLength.max)}`} /></CardContent>
        </Card>
        <Card>
          <CardHeader><div className="flex items-center gap-2"><GitCommitHorizontal className="size-5 text-primary" /><CardTitle>Git Analysis</CardTitle></div><CardDescription>این شاخص‌ها فقط زمانی نمایش داده می‌شوند که تاریخچهٔ Git در دسترس بوده باشد.</CardDescription></CardHeader>
          <CardContent>{git ? <div className="grid gap-3 sm:grid-cols-2"><MetricItem label="Git Churn" value={formatNumber(git.gitChurn)} /><MetricItem label="Change Frequency" value={formatNumber(git.changeFrequency)} /><MetricItem label="تعداد نویسندگان" value={formatNumber(git.authorCount)} /><MetricItem label="پراکندگی تغییرات" value={`${Math.round(git.changeDispersion * 100)}٪`} detail={`تمرکز: ${Math.round(git.changeConcentration * 100)}٪`} /></div> : <EmptyState title="دادهٔ Git در دسترس نیست" description="این پروژه ممکن است Git نداشته باشد یا تاریخچهٔ انتخاب‌شده برای این فایل داده‌ای نداشته باشد." className="min-h-52" />}</CardContent>
        </Card>
      </section>

      {git && <Card><CardHeader><div className="flex items-center gap-2"><CalendarRange className="size-5 text-primary" /><CardTitle>بازهٔ تغییرات فایل</CardTitle></div><CardDescription>API فعلی تاریخچهٔ commitهای تک‌فایل را برنمی‌گرداند؛ این بازه از معیارهای Git قابل‌دریافت نمایش داده می‌شود.</CardDescription></CardHeader><CardContent className="grid gap-3 sm:grid-cols-3"><MetricItem label="اولین تغییر" value={formatDateTime(git.firstChangedAt)} /><MetricItem label="آخرین تغییر" value={formatDateTime(git.lastChangedAt)} /><MetricItem label="مسیرهای پیشین" value={formatNumber(previousPaths.length)} detail={previousPaths.length ? previousPaths.join("، ") : "مسیر پیشینی ثبت نشده"} /></CardContent></Card>}

      {risk && <Card>
        <CardHeader><div className="flex items-center gap-2"><ShieldAlert className="size-5 text-destructive" /><CardTitle>چرا این فایل پرریسک است؟</CardTitle></div><CardDescription>سهم هر شاخص از مدل ذخیره‌شدهٔ backend؛ وزن‌های مؤثر را بررسی کنید.</CardDescription></CardHeader>
        <CardContent className="space-y-4">
          {contributions.length === 0 ? <EmptyState title="سهم شاخصی ثبت نشده است" description="مدل ریسک برای این فایل contribution قابل‌نمایش ندارد." /> : contributions.map((contribution) => <div key={contribution.metric} className="rounded-lg border p-4"><div className="flex flex-wrap items-center justify-between gap-2"><p className="font-extrabold">{metricLabels[contribution.metric]}</p><span className="font-mono text-xs text-muted-foreground">سهم: {(contribution.weighted * 100).toFixed(1)}٪</span></div><div className="mt-3 h-2 overflow-hidden rounded-full bg-muted"><div className="h-full rounded-full bg-destructive" style={{ width: `${Math.min(100, contribution.weighted * 100)}%` }} /></div><div className="mt-3 grid gap-2 text-xs text-muted-foreground sm:grid-cols-4"><span>مقدار: {formatNumber(contribution.raw)}</span><span>حد نرمال‌سازی: {formatNumber(contribution.normalizationCap)}</span><span>مقدار نرمال: {contribution.normalized.toFixed(2)}</span><span>وزن مدل: {(contribution.weight * 100).toFixed(0)}٪</span></div></div>)}
        </CardContent>
      </Card>}

      {functions.length > 0 && <Card><CardHeader><CardTitle>توابع ثبت‌شده</CardTitle><CardDescription>بلندترین declarationهای استخراج‌شده از تحلیل ایستا.</CardDescription></CardHeader><CardContent><ul className="grid gap-2 sm:grid-cols-2">{functions.slice().sort((left, right) => right.length - left.length).slice(0, 12).map((declaration) => <li key={`${declaration.name}-${declaration.startLine}`} className="flex items-center justify-between rounded-lg border bg-muted/25 px-3 py-2 text-sm"><span className="truncate font-mono text-xs" dir="ltr">{declaration.name}</span><span className="shrink-0 text-xs text-muted-foreground">خط {formatNumber(declaration.startLine)} · {formatNumber(declaration.length)}</span></li>)}</ul></CardContent></Card>}

      {diagnostics.length > 0 && <Card className="border-warning/25"><CardHeader><CardTitle className="flex items-center gap-2"><UsersRound className="size-4 text-[color:var(--warning)]" />هشدارها و خطاهای فایل</CardTitle></CardHeader><CardContent><ul className="space-y-2 text-sm">{diagnostics.map((diagnostic) => <li key={`${diagnostic.code}-${diagnostic.message}`}><span className="font-mono text-xs">{diagnostic.code}</span> — {diagnostic.message}</li>)}</ul></CardContent></Card>}
    </div>
  );
}
