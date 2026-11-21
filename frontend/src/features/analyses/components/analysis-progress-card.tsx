import { CircleAlert, LoaderCircle, OctagonX, XCircle } from "lucide-react";

import { AnalysisStatusBadge } from "@/features/analyses/components/analysis-status-badge";
import { Button } from "@/components/ui/button";
import { stageLabels, stageStatusLabels } from "@/lib/analysis";
import { formatDateTime, cn } from "@/lib/utils";
import type { AnalysisRun, StageStatus } from "@/types/api";

interface AnalysisProgressCardProps {
  analysis: AnalysisRun;
  onCancel?: () => void;
  isCancelling?: boolean;
}

const stageTone: Record<StageStatus, string> = {
  pending: "bg-muted text-muted-foreground",
  running: "bg-primary text-primary-foreground",
  completed: "bg-success text-[color:var(--primary-foreground)]",
  unavailable: "bg-warning text-[color:var(--foreground)]",
  failed: "bg-destructive text-destructive-foreground",
  cancelled: "bg-warning text-[color:var(--foreground)]",
  skipped: "bg-muted text-muted-foreground",
};

export function AnalysisProgressCard({ analysis, onCancel, isCancelling = false }: AnalysisProgressCardProps) {
  const active = analysis.status === "queued" || analysis.status === "running";
  const diagnostics = analysis.diagnostics.filter((diagnostic) => diagnostic.severity === "error").slice(-3);

  return (
    <div className="space-y-5">
      <div className="rounded-xl border bg-card p-5 shadow-card">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <p className="text-sm font-extrabold">وضعیت اجرای تحلیل</p>
            <p className="mt-1 text-sm text-muted-foreground">
              {analysis.currentStage ? `مرحلهٔ فعلی: ${stageLabels[analysis.currentStage]}` : "در انتظار شروع مراحل تحلیل"}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <AnalysisStatusBadge status={analysis.status} />
            {active && onCancel && (
              <Button
                variant="outline"
                size="sm"
                className="border-destructive/25 text-destructive hover:bg-destructive/10"
                type="button"
                onClick={onCancel}
                disabled={isCancelling}
              >
                {isCancelling ? <LoaderCircle className="size-3.5 animate-spin" /> : <OctagonX className="size-3.5" />}
                لغو تحلیل
              </Button>
            )}
          </div>
        </div>

        <div className="mt-5">
          <div className="mb-2 flex items-center justify-between text-sm">
            <span className="font-black">پیشرفت کلی</span>
            <span className="font-mono text-xs text-muted-foreground" dir="ltr">
              {analysis.progress.percent}%
            </span>
          </div>
          <div className="h-2 overflow-hidden rounded-full bg-muted" aria-label={`پیشرفت کلی ${analysis.progress.percent} درصد`} role="progressbar" aria-valuemin={0} aria-valuemax={100} aria-valuenow={analysis.progress.percent}>
            <div className="h-full rounded-full bg-primary transition-[width] duration-500" style={{ width: `${analysis.progress.percent}%` }} />
          </div>
          <div className="mt-3 grid grid-cols-2 gap-2 text-xs text-muted-foreground sm:grid-cols-4">
            <span>فایل‌های یافت‌شده: {analysis.progress.filesDiscovered}</span>
            <span>فایل‌های تحلیل‌شده: {analysis.progress.filesAnalyzed}</span>
            <span>commitهای بررسی‌شده: {analysis.progress.commitsScanned}</span>
            <span>خطاهای فایل: {analysis.progress.filesWithErrors}</span>
          </div>
        </div>
      </div>

      <ol className="grid gap-2 sm:grid-cols-2 lg:grid-cols-5" aria-label="مراحل تحلیل">
        {analysis.stages.map((stage) => (
          <li key={stage.stage} className="rounded-xl border bg-card p-3">
            <div className="flex items-center gap-2">
              <span className={cn("grid size-6 place-items-center rounded-full text-[0.65rem] font-black", stageTone[stage.status])}>
                {stage.status === "running" ? <LoaderCircle className="size-3.5 animate-spin" /> : stage.progress.completed}
              </span>
              <span className="text-xs font-extrabold">{stageLabels[stage.stage]}</span>
            </div>
            <p className="mt-2 text-xs text-muted-foreground">{stageStatusLabels[stage.status]}</p>
            <p className="mt-1 text-[0.7rem] text-muted-foreground">
              {stage.progress.completed} از {stage.progress.total}
            </p>
          </li>
        ))}
      </ol>

      {diagnostics.length > 0 && (
        <div className="rounded-xl border border-destructive/25 bg-destructive/7 p-4">
          <div className="flex items-center gap-2 text-sm font-extrabold text-destructive">
            <CircleAlert className="size-4" />
            خطاهای ثبت‌شده در تحلیل
          </div>
          <ul className="mt-3 space-y-2 text-xs leading-5 text-muted-foreground">
            {diagnostics.map((diagnostic) => (
              <li key={`${diagnostic.recordedAt}-${diagnostic.code}`}>
                <span className="font-mono text-[0.7rem]">{diagnostic.code}</span> — {diagnostic.message}
                {diagnostic.path && <span className="mr-1 font-mono" dir="ltr">({diagnostic.path})</span>}
              </li>
            ))}
          </ul>
        </div>
      )}

      {analysis.status === "failed" && diagnostics.length === 0 && (
        <div className="flex items-center gap-2 rounded-xl border border-destructive/25 bg-destructive/7 p-4 text-sm text-destructive">
          <XCircle className="size-4" />
          تحلیل با خطا پایان یافته است. جزئیات خطا را از سرویس محلی بررسی کنید.
        </div>
      )}

      {analysis.completedAt && (
        <p className="text-xs text-muted-foreground">آخرین به‌روزرسانی: {formatDateTime(analysis.completedAt)}</p>
      )}
    </div>
  );
}
