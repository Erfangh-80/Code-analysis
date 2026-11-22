import { useMutation } from "@tanstack/react-query";
import { FileCode2, GitBranch, LoaderCircle, Play, ShieldAlert, Sparkles } from "lucide-react";
import { FormEvent, useMemo, useState } from "react";

import { ApiErrorState, EmptyState } from "@/components/molecules/async-state";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { fileStatusLabels, formatNumber, languageLabels, metricLabels, riskLevelLabels } from "@/lib/analysis";
import { coreApi } from "@/services/api/core-api";
import type { BatchAnalysis, GitAnalysisReport, RiskAnalysisRequest, RiskMetric, StaticAnalysisRequest } from "@/types/api";

const staticMetrics: RiskMetric[] = ["lines_of_code", "max_function_length", "max_class_length"];
const allMetrics: RiskMetric[] = [...staticMetrics, "git_churn", "change_frequency"];

function relativeFiles(value: string) {
  return [...new Set(value.split(/[\n,]/).map((path) => path.trim()).filter(Boolean))];
}

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
  const [staticFiles, setStaticFiles] = useState("");
  const [repositoryPath, setRepositoryPath] = useState("");
  const [historyFrom, setHistoryFrom] = useState("");
  const [historyTo, setHistoryTo] = useState("");
  const [staticReport, setStaticReport] = useState<BatchAnalysis>();
  const [gitReport, setGitReport] = useState<GitAnalysisReport>();

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

  function submitStatic(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    staticAnalysis.mutate({ rootPath: staticRootPath.trim(), files: relativeFiles(staticFiles) });
  }

  function submitGit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const from = historyFrom ? new Date(historyFrom).toISOString() : undefined;
    const to = historyTo ? new Date(historyTo).toISOString() : undefined;
    gitAnalysis.mutate({ repositoryPath: repositoryPath.trim(), history: from || to ? { from, to } : undefined });
  }

  return (
    <div className="space-y-6">
      <section className="rounded-2xl border bg-card/80 p-6 shadow-card">
        <p className="text-sm font-black text-primary">فازهای ۱ تا ۴ — هستهٔ تحلیل</p>
        <h2 className="mt-2 text-2xl font-black tracking-tight sm:text-3xl">از کد محلی تا Hotspot قابل‌توضیح</h2>
        <p className="mt-3 max-w-3xl text-sm leading-6 text-muted-foreground">
          این workspace فقط قابلیت‌های تحلیل ایستا، تاریخچهٔ Git و امتیازدهی Hotspot را نمایش می‌دهد. کد پروژه اجرا یا به سرویس خارجی ارسال نمی‌شود.
        </p>
      </section>

      <section className="grid gap-6 xl:grid-cols-2">
        <Card>
          <CardHeader>
            <FileCode2 className="size-5 text-primary" />
            <CardTitle className="mt-2">۱. تحلیل ایستای فایل‌ها</CardTitle>
            <CardDescription>مسیر محلی root و فایل‌های نسبی Go، JavaScript یا Python را وارد کنید.</CardDescription>
          </CardHeader>
          <CardContent>
            <form className="space-y-4" onSubmit={submitStatic} noValidate>
              <label className="grid gap-2 text-sm font-extrabold">
                مسیر مطلق ریشهٔ پروژه
                <input className="h-10 rounded-md border bg-background px-3 font-mono text-sm" dir="ltr" value={staticRootPath} onChange={(event) => setStaticRootPath(event.target.value)} placeholder="/workspace/project" required />
              </label>
              <label className="grid gap-2 text-sm font-extrabold">
                فایل‌های نسبی
                <textarea className="min-h-28 rounded-md border bg-background p-3 font-mono text-sm" dir="ltr" value={staticFiles} onChange={(event) => setStaticFiles(event.target.value)} placeholder={"src/main.go\nsrc/api.js\npackage/model.py"} required />
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
                <label className="grid gap-2 text-sm font-extrabold">از تاریخ
                  <input className="h-10 rounded-md border bg-background px-3" type="datetime-local" dir="ltr" value={historyFrom} onChange={(event) => setHistoryFrom(event.target.value)} />
                </label>
                <label className="grid gap-2 text-sm font-extrabold">تا تاریخ
                  <input className="h-10 rounded-md border bg-background px-3" type="datetime-local" dir="ltr" value={historyTo} onChange={(event) => setHistoryTo(event.target.value)} />
                </label>
              </div>
              <Button type="submit" variant="outline" disabled={gitAnalysis.isPending}>
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
            {staticReport.files.length === 0 ? <EmptyState title="فایل قابل‌تحلیلی بازنگشت" description="مسیرهای نسبی و پسوندهای پشتیبانی‌شده را بررسی کنید." /> : (
              <Table>
                <TableHeader><TableRow><TableHead>فایل</TableHead><TableHead>زبان</TableHead><TableHead>وضعیت</TableHead><TableHead>LOC</TableHead><TableHead>تابع / کلاس</TableHead></TableRow></TableHeader>
                <TableBody>{staticReport.files.map((file) => <TableRow key={file.path}><TableCell className="font-mono text-xs" dir="ltr">{file.path}</TableCell><TableCell>{languageLabels[file.language]}</TableCell><TableCell>{fileStatusLabels[file.status]}</TableCell><TableCell>{formatNumber(file.metrics.linesOfCode)}</TableCell><TableCell>{formatNumber(file.metrics.functionCount)} / {formatNumber(file.metrics.classCount)}</TableCell></TableRow>)}</TableBody>
              </Table>
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
          <CardContent>
            {gitReport.files.length === 0 ? <EmptyState title="دادهٔ تاریخچه‌ای در بازه پیدا نشد" description="ممکن است مخزن خالی باشد یا بازهٔ انتخاب‌شده commit نداشته باشد." /> : (
              <Table>
                <TableHeader><TableRow><TableHead>فایل</TableHead><TableHead>Churn</TableHead><TableHead>تغییر</TableHead><TableHead>نویسندگان</TableHead><TableHead>مسیر پیشین</TableHead></TableRow></TableHeader>
                <TableBody>{gitReport.files.map((file) => <TableRow key={file.path}><TableCell className="font-mono text-xs" dir="ltr">{file.path}</TableCell><TableCell>{formatNumber(file.gitChurn)}</TableCell><TableCell>{formatNumber(file.changeFrequency)}</TableCell><TableCell>{formatNumber(file.authorCount)}</TableCell><TableCell className="font-mono text-xs" dir="ltr">{file.previousPaths.join(", ") || "—"}</TableCell></TableRow>)}</TableBody>
              </Table>
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
              <TableHeader><TableRow><TableHead>رتبه</TableHead><TableHead>فایل</TableHead><TableHead>امتیاز</TableHead><TableHead>سطح</TableHead><TableHead>مؤثرترین معیار</TableHead></TableRow></TableHeader>
              <TableBody>{riskAnalysis.data.files.map((hotspot) => {
                const leading = hotspot.contributions.slice().sort((left, right) => right.weighted - left.weighted)[0];
                return <TableRow key={hotspot.path}><TableCell>{formatNumber(hotspot.rank)}</TableCell><TableCell className="font-mono text-xs" dir="ltr">{hotspot.path}</TableCell><TableCell className="font-black">{formatNumber(hotspot.score)}</TableCell><TableCell>{riskLevelLabels[hotspot.level]}</TableCell><TableCell>{leading ? metricLabels[leading.metric] : "—"}</TableCell></TableRow>;
              })}</TableBody>
            </Table>
            {riskAnalysis.data.limitations.length > 0 && <div className="rounded-xl border border-warning/25 bg-warning/8 p-4"><p className="flex items-center gap-2 text-sm font-extrabold"><ShieldAlert className="size-4 text-[color:var(--warning)]" />محدودیت‌های مدل</p><ul className="mt-2 space-y-1 text-sm leading-6 text-muted-foreground">{riskAnalysis.data.limitations.map((limitation) => <li key={limitation}>• {limitation}</li>)}</ul></div>}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
