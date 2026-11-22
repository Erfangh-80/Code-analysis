import { useMutation } from "@tanstack/react-query";
import { BookOpen, CalendarClock, FileCode2, GitBranch, Info, LoaderCircle, Play, ShieldAlert, Sparkles, TriangleAlert } from "lucide-react";
import { FormEvent, useMemo, useState } from "react";
import type { DateObject } from "react-multi-date-picker";
import { Link } from "react-router-dom";

import { ApiErrorState, EmptyState } from "@/components/molecules/async-state";
import { JalaliDateTimeInput } from "@/components/molecules/jalali-date-time-input";
import { PaginationControls, usePagination } from "@/components/molecules/pagination";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { FileDetailDialog } from "@/features/core/components/file-detail-dialog";
import { fileStatusLabels, formatNumber, languageBadgeVariants, languageLabels, metricLabels, riskLevelBadgeVariants, riskLevelLabels, statusBadgeVariants } from "@/lib/analysis";
import { emptyGitResultCauses, toIsoUtc, validateHistoryRange } from "@/lib/datetime";
import { coreApi } from "@/services/api/core-api";
import type { BatchAnalysis, GitAnalysisReport, RiskAnalysisRequest, RiskMetric, StaticAnalysisRequest } from "@/types/api";

const staticMetrics: RiskMetric[] = ["lines_of_code", "max_function_length", "max_class_length"];
const allMetrics: RiskMetric[] = [...staticMetrics, "git_churn", "change_frequency"];

function sourceRowsForRisk(staticReport: BatchAnalysis, gitReport: GitAnalysisReport | undefined): RiskAnalysisRequest {
  const gitByPath = new Map(gitReport?.files.map((file) => [file.path, file]));
  const hasCompleteGitFacts = Boolean(gitReport) && staticReport.files.every((file) => gitByPath.has(file.path));
  const availableMetrics = hasCompleteGitFacts ? allMetrics : staticMetrics;

  return {
    availableMetrics,
    files: staticReport.files.map((file) => {
      const git = gitByPath.get(file.path);
      return {
        path: file.path,
        metrics: {
          linesOfCode: file.metrics.linesOfCode,
          maxFunctionLength: file.metrics.functionLength.max,
          maxClassLength: file.metrics.classLength.max,
          gitChurn: git?.gitChurn ?? 0,
          changeFrequency: git?.changeFrequency ?? 0,
        },
      };
    }),
  };
}

export function CoreAnalysisPage() {
  const [staticRootPath, setStaticRootPath] = useState("");
  const [repositoryPath, setRepositoryPath] = useState("");
  const [historyFrom, setHistoryFrom] = useState<DateObject | null>(null);
  const [historyTo, setHistoryTo] = useState<DateObject | null>(null);
  const [staticReport, setStaticReport] = useState<BatchAnalysis>();
  const [gitReport, setGitReport] = useState<GitAnalysisReport>();
  const [detailPath, setDetailPath] = useState<string | null>(null);

  const staticAnalysis = useMutation({
    mutationFn: (request: StaticAnalysisRequest) => coreApi.staticAnalysis(request),
    onSuccess: (report) => {
      setStaticReport(report);
      setGitReport(undefined);
    },
  });
  const gitAnalysis = useMutation({
    mutationFn: (request: Parameters<typeof coreApi.gitAnalysis>[0]) => coreApi.gitAnalysis(request),
    onSuccess: setGitReport,
  });
  const riskAnalysis = useMutation({
    mutationFn: (request: RiskAnalysisRequest) => coreApi.riskAnalysis(request),
  });

  const riskRequest = useMemo(
    () => (staticReport ? sourceRowsForRisk(staticReport, gitReport) : undefined),
    [gitReport, staticReport],
  );
  const isStaticOnlyScore = riskRequest ? (riskRequest.availableMetrics?.length ?? 0) === staticMetrics.length : false;

  const rangeValidation = validateHistoryRange(historyFrom, historyTo);

  const staticPagination = usePagination(staticReport?.files ?? [], staticReport);
  const gitPagination = usePagination(gitReport?.files ?? [], gitReport);
  const hotspotPagination = usePagination(riskAnalysis.data?.files ?? [], riskAnalysis.data);

  const detailStaticFile = staticReport?.files.find((file) => file.path === detailPath);
  const detailGitFile = gitReport?.files.find((file) => file.path === detailPath);
  const detailHotspot = riskAnalysis.data?.files.find((file) => file.path === detailPath);

  function submitStatic(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    staticAnalysis.mutate({ rootPath: staticRootPath.trim() });
  }

  function submitGit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (rangeValidation.error) return;
    const from = toIsoUtc(historyFrom);
    const to = toIsoUtc(historyTo);
    gitAnalysis.mutate({ repositoryPath: repositoryPath.trim(), history: from || to ? { from, to } : undefined });
  }

  return (
    <div className="space-y-6">
      <section className="rounded-2xl border bg-card/80 p-6 shadow-card">
        <p className="text-sm font-black text-primary">فازهای ۱ تا ۴ — هستهٔ تحلیل</p>
        <h2 className="mt-2 text-2xl font-black tracking-tight sm:text-3xl">از کد محلی تا Hotspot قابل‌توضیح</h2>
        <p className="mt-3 max-w-3xl text-sm leading-6 text-muted-foreground">
          مسیر پروژه را می‌دهید؛ فایل‌ها به‌صورت خودکار تحلیل ایستا می‌شوند، تاریخچهٔ Git خوانده می‌شود و Backend برای هر فایل امتیاز Hotspot شفاف می‌سازد. کد پروژه اجرا یا به سرویس خارجی ارسال نمی‌شود.
        </p>
        <Button variant="outline" size="sm" className="mt-4" asChild>
          <Link to="/guide"><BookOpen className="size-4" />راهنمای کامل و معنی معیارها</Link>
        </Button>
      </section>

      <section className="grid gap-6 xl:grid-cols-2">
        <Card>
          <CardHeader>
            <FileCode2 className="size-5 text-primary" />
            <CardTitle className="mt-2">۱. تحلیل ایستای فایل‌ها</CardTitle>
            <CardDescription>فقط مسیر مطلق ریشهٔ پروژه کافی است؛ فایل‌های Go، JavaScript و Python به‌صورت خودکار و امن کشف می‌شوند.</CardDescription>
          </CardHeader>
          <CardContent>
            <form className="space-y-4" onSubmit={submitStatic} noValidate>
              <label className="grid gap-2 text-sm font-extrabold">
                مسیر مطلق ریشهٔ پروژه
                <input className="h-10 rounded-md border bg-background px-3 font-mono text-sm" dir="ltr" value={staticRootPath} onChange={(event) => setStaticRootPath(event.target.value)} placeholder="/workspace/project" required />
              </label>
              <Button type="submit" disabled={staticAnalysis.isPending}>
                {staticAnalysis.isPending ? <LoaderCircle className="size-4 animate-spin" /> : <Play className="size-4" />}
                تحلیل ایستا
              </Button>
            </form>
            {staticAnalysis.isError && <ApiErrorState className="mt-4 min-h-0" error={staticAnalysis.error} title="تحلیل ایستا انجام نشد" />}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <GitBranch className="size-5 text-primary" />
            <CardTitle className="mt-2">۲. تحلیل تاریخچهٔ Git</CardTitle>
            <CardDescription>این مرحله فقط دادهٔ commit و diff یک worktree محلی موجود را به‌صورت read-only می‌خواند.</CardDescription>
          </CardHeader>
          <CardContent>
            <form className="space-y-4" onSubmit={submitGit} noValidate>
              <label className="grid gap-2 text-sm font-extrabold">
                مسیر مطلق Repository
                <input className="h-10 rounded-md border bg-background px-3 font-mono text-sm" dir="ltr" value={repositoryPath} onChange={(event) => setRepositoryPath(event.target.value)} placeholder="/workspace/project" required />
              </label>
              <div className="grid gap-4 sm:grid-cols-2">
                <JalaliDateTimeInput label="از تاریخ" value={historyFrom} onChange={setHistoryFrom} />
                <JalaliDateTimeInput label="تا تاریخ" value={historyTo} onChange={setHistoryTo} />
              </div>
              <p className="flex items-start gap-2 rounded-lg border border-info/25 bg-info/8 p-3 text-xs leading-6 text-muted-foreground">
                <Info className="mt-0.5 size-4 shrink-0 text-info" />
                انتخاب بازهٔ تاریخ اختیاری است. اگر می‌دانید تغییرات مدنظرتان در چه دوره‌ای اعمال شده‌اند، همان بازه را انتخاب کنید تا تحلیل متمرکزتر شود؛ در غیر این صورت هر دو فیلد را خالی بگذارید تا کل تاریخچهٔ مخزن بررسی شود.
              </p>
              {rangeValidation.error && (
                <p className="rounded-md border border-destructive/30 bg-destructive/10 p-3 text-sm font-bold text-destructive" role="alert">
                  {rangeValidation.error}
                </p>
              )}
              {rangeValidation.warnings.length > 0 && (
                <ul className="space-y-1 rounded-md border border-warning/25 bg-warning/8 p-3 text-sm leading-6 text-muted-foreground">
                  {rangeValidation.warnings.map((warning) => (
                    <li key={warning} className="flex items-start gap-2">
                      <TriangleAlert className="mt-1 size-4 shrink-0 text-[color:var(--warning)]" />
                      {warning}
                    </li>
                  ))}
                </ul>
              )}
              <Button type="submit" variant="outline" disabled={gitAnalysis.isPending || Boolean(rangeValidation.error)}>
                {gitAnalysis.isPending ? <LoaderCircle className="size-4 animate-spin" /> : <GitBranch className="size-4" />}
                تحلیل Git
              </Button>
            </form>
            {gitAnalysis.isError && <ApiErrorState className="mt-4 min-h-0" error={gitAnalysis.error} title="تحلیل تاریخچهٔ Git انجام نشد" />}
          </CardContent>
        </Card>
      </section>

      {staticReport && (
        <Card>
          <CardHeader>
            <CardTitle>نتیجهٔ تحلیل ایستا</CardTitle>
            <CardDescription>{formatNumber(staticReport.files.length)} فایل و {formatNumber(staticReport.modules.length)} ماژول از پاسخ واقعی API.</CardDescription>
          </CardHeader>
          <CardContent>
            {staticReport.files.length === 0 ? <EmptyState title="فایل قابل‌تحلیلی پیدا نشد" description="زیر این ریشه فایل Go، JavaScript یا Python وجود ندارد؛ مسیر مطلق و محدودهٔ مجاز سرور را بررسی کنید." /> : (
              <>
                <Table>
                  <TableHeader><TableRow><TableHead>فایل</TableHead><TableHead>زبان</TableHead><TableHead>وضعیت</TableHead><TableHead>LOC</TableHead><TableHead>تابع / کلاس</TableHead><TableHead><span className="sr-only">جزئیات</span></TableHead></TableRow></TableHeader>
                  <TableBody>{staticPagination.pageItems.map((file) => (
                    <TableRow key={file.path}>
                      <TableCell className="font-mono text-xs" dir="ltr">{file.path}</TableCell>
                      <TableCell><Badge variant={languageBadgeVariants[file.language]}>{languageLabels[file.language]}</Badge></TableCell>
                      <TableCell><Badge variant={statusBadgeVariants[file.status]}>{fileStatusLabels[file.status]}</Badge></TableCell>
                      <TableCell>{formatNumber(file.metrics.linesOfCode)}</TableCell>
                      <TableCell>{formatNumber(file.metrics.functionCount)} / {formatNumber(file.metrics.classCount)}</TableCell>
                      <TableCell><Button type="button" variant="ghost" size="sm" onClick={() => setDetailPath(file.path)}>جزئیات</Button></TableCell>
                    </TableRow>
                  ))}</TableBody>
                </Table>
                <PaginationControls label="صفحه‌بندی نتایج تحلیل ایستا" page={staticPagination.page} pageCount={staticPagination.pageCount} rangeStart={staticPagination.rangeStart} rangeEnd={staticPagination.rangeEnd} total={staticReport.files.length} onPageChange={staticPagination.setPage} />
              </>
            )}
          </CardContent>
        </Card>
      )}

      {gitReport && (
        <Card>
          <CardHeader>
            <CardTitle>نتیجهٔ تاریخچهٔ Git</CardTitle>
            <CardDescription>{formatNumber(gitReport.commitCount)} commit و {formatNumber(gitReport.files.length)} فایل با معیار واقعی تاریخچه.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {gitReport.commitCount === 0 && (
              <div className="rounded-xl border border-warning/25 bg-warning/8 p-4">
                <p className="flex items-center gap-2 text-sm font-extrabold">
                  <CalendarClock className="size-4 text-[color:var(--warning)]" />
                  در این بازه هیچ تحلیلی به‌دست نیامد — علت‌های محتمل:
                </p>
                <ul className="mt-2 space-y-1 text-sm leading-6 text-muted-foreground">
                  {emptyGitResultCauses(Boolean(gitReport.history.from || gitReport.history.to)).map((cause) => <li key={cause}>• {cause}</li>)}
                </ul>
              </div>
            )}
            {gitReport.diagnostics.length > 0 && (
              <ul className="space-y-1 rounded-md border p-3 text-sm leading-6 text-muted-foreground">
                {gitReport.diagnostics.map((diagnostic) => <li key={`${diagnostic.code}-${diagnostic.message}`}>• {diagnostic.message}</li>)}
              </ul>
            )}
            {gitReport.files.length > 0 && (
              <>
                <Table>
                  <TableHeader><TableRow><TableHead>فایل</TableHead><TableHead>حجم تغییرات</TableHead><TableHead>دفعات تغییر</TableHead><TableHead>نویسندگان</TableHead><TableHead><span className="sr-only">جزئیات</span></TableHead></TableRow></TableHeader>
                  <TableBody>{gitPagination.pageItems.map((file) => (
                    <TableRow key={file.path}>
                      <TableCell className="font-mono text-xs" dir="ltr">{file.path}</TableCell>
                      <TableCell>{formatNumber(file.gitChurn)}</TableCell>
                      <TableCell>{formatNumber(file.changeFrequency)}</TableCell>
                      <TableCell>{formatNumber(file.authorCount)}</TableCell>
                      <TableCell><Button type="button" variant="ghost" size="sm" onClick={() => setDetailPath(file.path)}>جزئیات</Button></TableCell>
                    </TableRow>
                  ))}</TableBody>
                </Table>
                <p className="flex items-start gap-2 text-xs leading-6 text-muted-foreground">
                  <Info className="mt-0.5 size-3.5 shrink-0 text-info" />
                  حجم تغییرات (Churn) مجموع خطوط افزوده/حذف‌شدهٔ فایل در بازهٔ انتخاب‌شده است و دفعات تغییر یعنی چند commit متفاوت روی آن کار کرده‌اند؛ مقادیر بزرگ‌تر یعنی فایل داغ‌تر و حساس‌تر است. تاریخ اولین/آخرین تغییر و مسیرهای پیشین را در «جزئیات» ببینید.
                </p>
                <PaginationControls label="صفحه‌بندی نتایج تاریخچهٔ Git" page={gitPagination.page} pageCount={gitPagination.pageCount} rangeStart={gitPagination.rangeStart} rangeEnd={gitPagination.rangeEnd} total={gitReport.files.length} onPageChange={gitPagination.setPage} />
              </>
            )}
          </CardContent>
        </Card>
      )}

      <Card className="border-primary/20 bg-primary/5">
        <CardHeader>
          <Sparkles className="size-5 text-primary" />
          <CardTitle className="mt-2">۳. امتیازدهی و رتبه‌بندی Hotspot</CardTitle>
          <CardDescription>Frontend فقط واقعیت‌های برگشتی API را برای موتور Backend ارسال می‌کند؛ امتیاز، وزن، رتبه و Heatmap همگی در Go محاسبه می‌شوند.</CardDescription>
        </CardHeader>
        <CardContent>
          {!riskRequest ? <EmptyState title="ابتدا تحلیل ایستا را اجرا کنید" description="برای ساخت ورودی معتبر Score، حداقل نتیجهٔ Static Analysis لازم است." className="min-h-32" /> : (
            <div className="space-y-4">
              <div className="rounded-xl border bg-card p-4 text-sm leading-6 text-muted-foreground">
                {isStaticOnlyScore ? "برای همهٔ فایل‌های ایستا دادهٔ Git موجود نیست؛ موتور Backend فقط معیارهای ایستا را با وزن‌های مؤثر بازتنظیم‌شده استفاده می‌کند." : "معیارهای ایستا و Git برای همهٔ فایل‌های انتخاب‌شده موجود است و موتور Backend هر پنج معیار را امتیازدهی می‌کند."}
              </div>
              <Button onClick={() => riskAnalysis.mutate(riskRequest)} disabled={riskAnalysis.isPending}>
                {riskAnalysis.isPending ? <LoaderCircle className="size-4 animate-spin" /> : <Sparkles className="size-4" />}
                محاسبهٔ Hotspot در Backend
              </Button>
            </div>
          )}
          {riskAnalysis.isError && <ApiErrorState className="mt-4 min-h-0" error={riskAnalysis.error} title="محاسبهٔ Hotspot انجام نشد" />}
        </CardContent>
      </Card>

      {riskAnalysis.data && (
        <Card>
          <CardHeader>
            <CardTitle>Hotspotهای رتبه‌بندی‌شده</CardTitle>
            <CardDescription>{riskAnalysis.data.formula}</CardDescription>
          </CardHeader>
          <CardContent className="space-y-5">
            <Table>
              <TableHeader><TableRow><TableHead>رتبه</TableHead><TableHead>فایل</TableHead><TableHead>امتیاز</TableHead><TableHead>سطح ریسک</TableHead><TableHead>مؤثرترین معیار</TableHead><TableHead><span className="sr-only">جزئیات</span></TableHead></TableRow></TableHeader>
              <TableBody>{hotspotPagination.pageItems.map((hotspot) => {
                const leading = hotspot.contributions.slice().sort((left, right) => right.weighted - left.weighted)[0];
                return (
                  <TableRow key={hotspot.path}>
                    <TableCell>{formatNumber(hotspot.rank)}</TableCell>
                    <TableCell className="font-mono text-xs" dir="ltr">{hotspot.path}</TableCell>
                    <TableCell className="font-black">{formatNumber(hotspot.score)}</TableCell>
                    <TableCell><Badge variant={riskLevelBadgeVariants[hotspot.level]}>{riskLevelLabels[hotspot.level]}</Badge></TableCell>
                    <TableCell>{leading ? metricLabels[leading.metric] : "—"}</TableCell>
                    <TableCell><Button type="button" variant="ghost" size="sm" onClick={() => setDetailPath(hotspot.path)}>جزئیات</Button></TableCell>
                  </TableRow>
                );
              })}</TableBody>
            </Table>
            <PaginationControls label="صفحه‌بندی Hotspotها" page={hotspotPagination.page} pageCount={hotspotPagination.pageCount} rangeStart={hotspotPagination.rangeStart} rangeEnd={hotspotPagination.rangeEnd} total={riskAnalysis.data.files.length} onPageChange={hotspotPagination.setPage} />
            {riskAnalysis.data.limitations.length > 0 && <div className="rounded-xl border border-warning/25 bg-warning/8 p-4"><p className="flex items-center gap-2 text-sm font-extrabold"><ShieldAlert className="size-4 text-[color:var(--warning)]" />محدودیت‌های مدل</p><ul className="mt-2 space-y-1 text-sm leading-6 text-muted-foreground">{riskAnalysis.data.limitations.map((limitation) => <li key={limitation}>• {limitation}</li>)}</ul></div>}
          </CardContent>
        </Card>
      )}

      <FileDetailDialog path={detailPath} staticFile={detailStaticFile} gitFile={detailGitFile} hotspot={detailHotspot} onClose={() => setDetailPath(null)} />
    </div>
  );
}
