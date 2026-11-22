import { FileCode2, GitBranch, Sparkles } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { fileStatusLabels, formatNumber, languageBadgeVariants, languageLabels, metricLabels, riskLevelBadgeVariants, riskLevelLabels, statusBadgeVariants } from "@/lib/analysis";
import { formatJalali } from "@/lib/datetime";
import type { FileAnalysis, GitFileMetrics, Hotspot } from "@/types/api";

interface FileDetailDialogProps {
  path: string | null;
  staticFile?: FileAnalysis;
  gitFile?: GitFileMetrics;
  hotspot?: Hotspot;
  onClose: () => void;
}

function MetricItem({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border bg-background/60 px-3 py-2">
      <p className="text-[0.7rem] font-bold text-muted-foreground">{label}</p>
      <p className="mt-0.5 text-sm font-black">{value}</p>
    </div>
  );
}

/** Per-file detail view combining static, Git-history, and Hotspot facts already returned by the API. */
export function FileDetailDialog({ path, staticFile, gitFile, hotspot, onClose }: FileDetailDialogProps) {
  return (
    <Dialog open={Boolean(path)} onOpenChange={(open) => { if (!open) onClose(); }}>
      <DialogContent className="max-h-[85vh] max-w-2xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="font-mono text-sm break-all" dir="ltr">{path}</DialogTitle>
          <DialogDescription>جزئیات کامل فایل از پاسخ‌های واقعی تحلیل ایستا، Git و Hotspot.</DialogDescription>
        </DialogHeader>

        {staticFile && (
          <section className="space-y-3">
            <p className="flex items-center gap-2 text-sm font-black"><FileCode2 className="size-4 text-primary" />تحلیل ایستا</p>
            <div className="flex flex-wrap gap-2">
              <Badge variant={languageBadgeVariants[staticFile.language]}>{languageLabels[staticFile.language]}</Badge>
              <Badge variant={statusBadgeVariants[staticFile.status]}>{fileStatusLabels[staticFile.status]}</Badge>
              {staticFile.usedFallback && <Badge variant="warning">تحلیل جایگزین</Badge>}
            </div>
            <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
              <MetricItem label="خطوط کد" value={formatNumber(staticFile.metrics.linesOfCode)} />
              <MetricItem label="تعداد تابع" value={formatNumber(staticFile.metrics.functionCount)} />
              <MetricItem label="تعداد کلاس" value={formatNumber(staticFile.metrics.classCount)} />
              <MetricItem label="بیشترین طول تابع" value={formatNumber(staticFile.metrics.functionLength.max)} />
              <MetricItem label="مجموع طول توابع" value={formatNumber(staticFile.metrics.functionLength.total)} />
              <MetricItem label="بیشترین طول کلاس" value={formatNumber(staticFile.metrics.classLength.max)} />
            </div>
            {staticFile.diagnostics.length > 0 && (
              <ul className="space-y-1 rounded-lg border border-warning/25 bg-warning/8 p-3 text-xs leading-5 text-muted-foreground">
                {staticFile.diagnostics.map((diagnostic) => <li key={`${diagnostic.code}-${diagnostic.message}`}>• {diagnostic.message}</li>)}
              </ul>
            )}
          </section>
        )}

        {gitFile && (
          <section className="space-y-3">
            <p className="flex items-center gap-2 text-sm font-black"><GitBranch className="size-4 text-primary" />تاریخچهٔ Git</p>
            <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
              <MetricItem label="حجم تغییرات (Churn)" value={formatNumber(gitFile.gitChurn)} />
              <MetricItem label="دفعات تغییر" value={formatNumber(gitFile.changeFrequency)} />
              <MetricItem label="نویسندگان" value={formatNumber(gitFile.authorCount)} />
              <MetricItem label="نخستین تغییر" value={formatJalali(gitFile.firstChangedAt)} />
              <MetricItem label="آخرین تغییر" value={formatJalali(gitFile.lastChangedAt)} />
              <MetricItem label="وضعیت فعلی" value={gitFile.deleted ? "حذف‌شده" : "موجود"} />
            </div>
            {gitFile.previousPaths.length > 0 && (
              <p className="text-xs leading-5 text-muted-foreground">مسیرهای پیشین: <span className="font-mono" dir="ltr">{gitFile.previousPaths.join(" ، ")}</span></p>
            )}
          </section>
        )}

        {hotspot && (
          <section className="space-y-3">
            <p className="flex items-center gap-2 text-sm font-black"><Sparkles className="size-4 text-primary" />امتیاز Hotspot</p>
            <div className="flex flex-wrap items-center gap-2">
              <Badge variant={riskLevelBadgeVariants[hotspot.level]}>ریسک {riskLevelLabels[hotspot.level]}</Badge>
              <span className="text-sm font-black">امتیاز {formatNumber(hotspot.score)} از ۱۰۰</span>
              <span className="text-xs text-muted-foreground">رتبهٔ {formatNumber(hotspot.rank)}</span>
            </div>
            <Table>
              <TableHeader>
                <TableRow><TableHead>معیار</TableHead><TableHead>مقدار</TableHead><TableHead>سقف نرمال‌سازی</TableHead><TableHead>وزن</TableHead><TableHead>سهم در امتیاز</TableHead></TableRow>
              </TableHeader>
              <TableBody>
                {hotspot.contributions.map((contribution) => (
                  <TableRow key={contribution.metric}>
                    <TableCell>{metricLabels[contribution.metric]}</TableCell>
                    <TableCell>{formatNumber(contribution.raw)}</TableCell>
                    <TableCell>{formatNumber(contribution.normalizationCap)}</TableCell>
                    <TableCell>{formatNumber(contribution.weight)}</TableCell>
                    <TableCell className="font-black">{formatNumber(Math.round(contribution.weighted * 100))}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </section>
        )}

        {!staticFile && !gitFile && !hotspot && (
          <p className="text-sm text-muted-foreground">هنوز داده‌ای برای این فایل از API برنگشته است.</p>
        )}
      </DialogContent>
    </Dialog>
  );
}
