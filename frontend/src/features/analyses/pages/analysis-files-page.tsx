import { ArrowLeft, ArrowRight, FileSliders, Search, SlidersHorizontal } from "lucide-react";
import { useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";

import { ApiErrorState, EmptyState, LoadingState } from "@/components/molecules/async-state";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { RiskLevelBadge } from "@/features/analyses/components/risk-level-badge";
import { useAnalysisFilesQuery, useAnalysisReportQuery } from "@/hooks/use-analysis-queries";
import { fileStatusLabels, formatNumber, languageLabels } from "@/lib/analysis";
import { fileResultsFromReport, riskMatches } from "@/lib/report";
import type { AnalysisFileResult, FileAnalysisStatus, RiskLevel, SupportedLanguage } from "@/types/api";

const fileStatuses: Array<{ value: "all" | FileAnalysisStatus; label: string }> = [
  { value: "all", label: "همهٔ وضعیت‌ها" },
  ...Object.entries(fileStatusLabels).map(([value, label]) => ({ value: value as FileAnalysisStatus, label })),
];

const languageOptions: Array<{ value: "all" | SupportedLanguage; label: string }> = [
  { value: "all", label: "همهٔ زبان‌ها" },
  { value: "go", label: "Go" },
  { value: "javascript", label: "JavaScript" },
  { value: "python", label: "Python" },
  { value: "unknown", label: "نامشخص" },
];

const sortOptions = [
  ["path", "مسیر فایل"],
  ["language", "زبان"],
  ["linesOfCode", "LOC"],
  ["maxFunctionLength", "طول تابع"],
  ["gitChurn", "Git Churn"],
  ["changeFrequency", "تعداد تغییرات"],
  ["score", "امتیاز ریسک"],
] as const;

type SortKey = (typeof sortOptions)[number][0];

function fileValue(file: AnalysisFileResult, sort: SortKey): number | string {
  switch (sort) {
    case "path": return file.analysis.path;
    case "language": return file.analysis.language;
    case "linesOfCode": return file.analysis.metrics.linesOfCode;
    case "maxFunctionLength": return file.analysis.metrics.functionLength.max;
    case "gitChurn": return file.git?.gitChurn ?? -1;
    case "changeFrequency": return file.git?.changeFrequency ?? -1;
    case "score": return file.risk?.score ?? -1;
  }
}

export function AnalysisFilesPage() {
  const { analysisId } = useParams();
  const [page, setPage] = useState(1);
  const [q, setQ] = useState("");
  const [language, setLanguage] = useState<"all" | SupportedLanguage>("all");
  const [status, setStatus] = useState<"all" | FileAnalysisStatus>("all");
  const [riskLevels, setRiskLevels] = useState<RiskLevel[]>([]);
  const [minScore, setMinScore] = useState("");
  const [maxScore, setMaxScore] = useState("");
  const [sort, setSort] = useState<SortKey>("path");
  const [direction, setDirection] = useState<"asc" | "desc">("asc");
  const pageSize = 20;
  const useClientFilters = language !== "all" || minScore !== "" || maxScore !== "";
  const serverFiles = useAnalysisFilesQuery(
    analysisId,
    { page, pageSize, q, riskLevel: riskLevels, analysisStatus: status === "all" ? undefined : [status], sort, direction },
    !useClientFilters,
  );
  const report = useAnalysisReportQuery(analysisId, useClientFilters);

  function updatePage() {
    setPage(1);
  }

  function toggleRisk(level: RiskLevel) {
    setRiskLevels((current) => current.includes(level) ? current.filter((entry) => entry !== level) : [...current, level]);
    updatePage();
  }

  const clientData = useMemo(() => {
    if (!report.data) return undefined;
    const min = minScore === "" ? Number.NEGATIVE_INFINITY : Number(minScore);
    const max = maxScore === "" ? Number.POSITIVE_INFINITY : Number(maxScore);
    const lowerQuery = q.trim().toLowerCase();
    const filtered = fileResultsFromReport(report.data)
      .filter((file) => language === "all" || file.analysis.language === language)
      .filter((file) => status === "all" || file.analysis.status === status)
      .filter((file) => riskMatches(riskLevels, file.risk?.level))
      .filter((file) => !file.risk ? minScore === "" && maxScore === "" : file.risk.score >= min && file.risk.score <= max)
      .filter((file) => !lowerQuery || file.analysis.path.toLowerCase().includes(lowerQuery))
      .sort((left, right) => {
        const first = fileValue(left, sort);
        const second = fileValue(right, sort);
        const comparison = typeof first === "string" && typeof second === "string" ? first.localeCompare(second) : Number(first) - Number(second);
        return direction === "asc" ? comparison : -comparison;
      });
    return {
      items: filtered.slice((page - 1) * pageSize, page * pageSize),
      page,
      pageSize,
      totalItems: filtered.length,
    };
  }, [direction, language, maxScore, minScore, page, q, report.data, riskLevels, sort, status]);

  const files = useClientFilters ? clientData : serverFiles.data;
  const isPending = useClientFilters ? report.isPending : serverFiles.isPending;
  const error = useClientFilters ? report.error : serverFiles.error;
  const isError = useClientFilters ? report.isError : serverFiles.isError;
  const totalPages = files ? Math.max(1, Math.ceil(files.totalItems / files.pageSize)) : 1;

  if (!analysisId) {
    return <EmptyState title="شناسهٔ تحلیل مشخص نیست" description="از فهرست تحلیل‌ها یک گزارش معتبر را انتخاب کنید." />;
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <CardTitle>فایل‌های تحلیل‌شده</CardTitle>
              <CardDescription>جست‌وجو، فیلتر و مرتب‌سازی نتایج immutable همین تحلیل.</CardDescription>
            </div>
            <Button variant="outline" size="sm" asChild><Link to={`/analyses/${encodeURIComponent(analysisId)}`}><ArrowRight className="size-3.5" />بازگشت به گزارش</Link></Button>
          </div>
        </CardHeader>
        <CardContent className="space-y-5">
          <div className="grid gap-3 rounded-xl border bg-muted/25 p-4 lg:grid-cols-4">
            <label className="grid gap-1.5 text-xs font-extrabold lg:col-span-2">جست‌وجوی نام یا مسیر
              <span className="relative"><Search className="pointer-events-none absolute top-1/2 right-3 size-4 -translate-y-1/2 text-muted-foreground" /><Input className="pr-9" value={q} onChange={(event) => { setQ(event.target.value); updatePage(); }} placeholder="src/features/..." dir="ltr" /></span>
            </label>
            <label className="grid gap-1.5 text-xs font-extrabold">زبان
              <Select value={language} onValueChange={(value) => { setLanguage(value as "all" | SupportedLanguage); updatePage(); }}><SelectTrigger aria-label="فیلتر زبان"><SelectValue /></SelectTrigger><SelectContent>{languageOptions.map((option) => <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>)}</SelectContent></Select>
            </label>
            <label className="grid gap-1.5 text-xs font-extrabold">وضعیت تحلیل
              <Select value={status} onValueChange={(value) => { setStatus(value as "all" | FileAnalysisStatus); updatePage(); }}><SelectTrigger aria-label="فیلتر وضعیت فایل"><SelectValue /></SelectTrigger><SelectContent>{fileStatuses.map((option) => <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>)}</SelectContent></Select>
            </label>
            <label className="grid gap-1.5 text-xs font-extrabold">حداقل امتیاز ریسک
              <Input type="number" min="0" max="100" dir="ltr" value={minScore} onChange={(event) => { setMinScore(event.target.value); updatePage(); }} placeholder="۰" />
            </label>
            <label className="grid gap-1.5 text-xs font-extrabold">حداکثر امتیاز ریسک
              <Input type="number" min="0" max="100" dir="ltr" value={maxScore} onChange={(event) => { setMaxScore(event.target.value); updatePage(); }} placeholder="۱۰۰" />
            </label>
            <label className="grid gap-1.5 text-xs font-extrabold">مرتب‌سازی
              <Select value={sort} onValueChange={(value) => { setSort(value as SortKey); updatePage(); }}><SelectTrigger aria-label="مرتب‌سازی فایل‌ها"><SelectValue /></SelectTrigger><SelectContent>{sortOptions.map(([value, label]) => <SelectItem key={value} value={value}>{label}</SelectItem>)}</SelectContent></Select>
            </label>
            <label className="grid gap-1.5 text-xs font-extrabold">جهت
              <Select value={direction} onValueChange={(value) => { setDirection(value as "asc" | "desc"); updatePage(); }}><SelectTrigger aria-label="جهت مرتب‌سازی"><SelectValue /></SelectTrigger><SelectContent><SelectItem value="asc">صعودی</SelectItem><SelectItem value="desc">نزولی</SelectItem></SelectContent></Select>
            </label>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <span className="inline-flex items-center gap-1.5 text-xs font-extrabold"><SlidersHorizontal className="size-3.5" />سطح ریسک:</span>
            {(["low", "medium", "high"] as RiskLevel[]).map((level) => <Button key={level} type="button" size="sm" variant={riskLevels.includes(level) ? "secondary" : "outline"} aria-pressed={riskLevels.includes(level)} onClick={() => toggleRisk(level)}>{level === "low" ? "کم" : level === "medium" ? "متوسط" : "زیاد"}</Button>)}
            {useClientFilters && <span className="text-xs text-muted-foreground">فیلتر زبان و بازهٔ امتیاز با گزارش کامل همین تحلیل اعمال می‌شود.</span>}
          </div>

          {isPending && <LoadingState />}
          {isError && <ApiErrorState error={error} onRetry={() => useClientFilters ? report.refetch() : serverFiles.refetch()} title="فهرست فایل‌ها قابل دریافت نیست" />}
          {files && files.items.length === 0 && <EmptyState title="فایلی با این فیلترها پیدا نشد" description="فیلترها را تغییر دهید یا بازهٔ امتیاز ریسک را پاک کنید." />}
          {files && files.items.length > 0 && (
            <>
              <Table>
                <caption className="sr-only">فهرست فایل‌های تحلیل‌شده با شاخص‌های ایستا، Git و ریسک</caption>
                <TableHeader><TableRow><TableHead>فایل / مسیر</TableHead><TableHead>زبان</TableHead><TableHead className="hidden lg:table-cell">Cyclomatic</TableHead><TableHead>LOC / تابع</TableHead><TableHead className="hidden md:table-cell">Git</TableHead><TableHead>Hotspot</TableHead></TableRow></TableHeader>
                <TableBody>{files.items.map((file) => (
                  <TableRow key={file.analysis.path}>
                    <TableCell className="max-w-52"><Link className="block truncate font-mono text-xs font-bold hover:text-primary hover:underline" dir="ltr" to={`/analyses/${encodeURIComponent(analysisId)}/files/detail?path=${encodeURIComponent(file.analysis.path)}`}>{file.analysis.path}</Link><span className="mt-1 block text-[0.7rem] text-muted-foreground">{fileStatusLabels[file.analysis.status]}</span></TableCell>
                    <TableCell>{languageLabels[file.analysis.language]}</TableCell>
                    <TableCell className="hidden text-muted-foreground lg:table-cell" title="این شاخص در API v1 موجود نیست">—</TableCell>
                    <TableCell><span className="font-bold">{formatNumber(file.analysis.metrics.linesOfCode)}</span><span className="block text-xs text-muted-foreground">تابع: {formatNumber(file.analysis.metrics.functionLength.max)}</span></TableCell>
                    <TableCell className="hidden md:table-cell"><span>{formatNumber(file.git?.gitChurn)}</span><span className="block text-xs text-muted-foreground">تغییر: {formatNumber(file.git?.changeFrequency)}</span></TableCell>
                    <TableCell>{file.risk ? <div className="flex items-center gap-2"><span className="font-black text-destructive">{file.risk.score}</span><RiskLevelBadge level={file.risk.level} /></div> : <span className="text-muted-foreground">—</span>}</TableCell>
                  </TableRow>
                ))}</TableBody>
              </Table>
              <div className="flex flex-wrap items-center justify-between gap-3 border-t pt-4 text-xs text-muted-foreground"><span>{formatNumber(files.totalItems)} فایل · صفحهٔ {files.page} از {totalPages}</span><div className="flex gap-2"><Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((current) => current - 1)}><ArrowRight className="size-3.5" />قبلی</Button><Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage((current) => current + 1)}>بعدی<ArrowLeft className="size-3.5" /></Button></div></div>
            </>
          )}
        </CardContent>
      </Card>
      <div className="flex items-center gap-2 text-xs text-muted-foreground"><FileSliders className="size-3.5" />شاخص Cyclomatic Complexity در API فعلی backend تولید نمی‌شود؛ سایر اعداد فقط از گزارش واقعی تحلیل می‌آیند.</div>
    </div>
  );
}
