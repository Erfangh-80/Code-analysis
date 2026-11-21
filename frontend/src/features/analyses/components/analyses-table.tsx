import { ArrowLeft, ArrowRight, Plus } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router-dom";

import { ApiErrorState, EmptyState, LoadingState } from "@/components/molecules/async-state";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { AnalysisStatusBadge } from "@/features/analyses/components/analysis-status-badge";
import { useAnalysesQuery } from "@/hooks/use-analyses-query";
import { formatDateTime } from "@/lib/utils";
import type { AnalysisRunStatus } from "@/types/api";

interface AnalysesTableProps {
  showAll?: boolean;
}

const statusOptions: Array<{ value: "all" | AnalysisRunStatus; label: string }> = [
  { value: "all", label: "همهٔ وضعیت‌ها" },
  { value: "queued", label: "در صف" },
  { value: "running", label: "در حال اجرا" },
  { value: "completed", label: "تکمیل‌شده" },
  { value: "failed", label: "ناموفق" },
  { value: "cancelled", label: "لغوشده" },
];

export function AnalysesTable({ showAll = false }: AnalysesTableProps) {
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState<"all" | AnalysisRunStatus>("all");
  const pageSize = showAll ? 15 : 5;
  const analyses = useAnalysesQuery({ page, pageSize, status: status === "all" ? undefined : status });
  const totalPages = analyses.data ? Math.max(1, Math.ceil(analyses.data.totalItems / analyses.data.pageSize)) : 1;

  function changeStatus(nextStatus: "all" | AnalysisRunStatus) {
    setStatus(nextStatus);
    setPage(1);
  }

  return (
    <Card>
      <CardHeader className="flex-row items-start justify-between gap-4 space-y-0">
        <div>
          <CardTitle>{showAll ? "فهرست تحلیل‌ها" : "تحلیل‌های اخیر"}</CardTitle>
          <CardDescription>
            {showAll
              ? "وضعیت اجراهای ذخیره‌شده و پیشرفت تحلیل‌های در حال انجام را دنبال کنید."
              : "این داده‌ها فقط پس از پاسخ سرویس محلی نمایش داده می‌شوند."}
          </CardDescription>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          {showAll && (
            <Select value={status} onValueChange={(value) => changeStatus(value as "all" | AnalysisRunStatus)}>
              <SelectTrigger className="w-40" aria-label="فیلتر وضعیت تحلیل">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {statusOptions.map((option) => <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>)}
              </SelectContent>
            </Select>
          )}
          {!showAll && (
            <Button variant="outline" size="sm" asChild>
              <Link to="/analyses">
                مشاهده همه
                <ArrowLeft className="size-3.5" />
              </Link>
            </Button>
          )}
        </div>
      </CardHeader>
      <CardContent>
        {analyses.isPending && <LoadingState />}
        {analyses.isError && <ApiErrorState error={analyses.error} onRetry={() => analyses.refetch()} title="امکان دریافت تحلیل‌ها وجود ندارد" />}

        {analyses.isSuccess && analyses.data.items.length === 0 && (
          <EmptyState
            title={status === "all" ? "هنوز تحلیلی ثبت نشده است" : "تحلیلی با این وضعیت پیدا نشد"}
            description={status === "all" ? "برای شروع، مسیر یک پروژهٔ محلی را در بخش «شروع تحلیل» وارد کنید." : "فیلتر وضعیت را تغییر دهید یا یک تحلیل جدید آغاز کنید."}
            action={<Button size="sm" asChild><Link to="/analyses/new"><Plus className="size-3.5" />شروع تحلیل</Link></Button>}
          />
        )}

        {analyses.isSuccess && analyses.data.items.length > 0 && (
          <>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>شناسه تحلیل</TableHead>
                  <TableHead>وضعیت</TableHead>
                  <TableHead>مرحله / پیشرفت</TableHead>
                  <TableHead className="hidden sm:table-cell">ایجاد شده</TableHead>
                  <TableHead><span className="sr-only">جزئیات</span></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {analyses.data.items.map((analysis) => (
                  <TableRow key={analysis.id}>
                    <TableCell className="max-w-36 truncate font-mono text-xs" dir="ltr">{analysis.id}</TableCell>
                    <TableCell><AnalysisStatusBadge status={analysis.status} /></TableCell>
                    <TableCell>
                      <span className="text-xs font-extrabold">{analysis.progress.percent}٪</span>
                      <div className="mt-1 h-1.5 w-20 overflow-hidden rounded-full bg-muted">
                        <div className="h-full bg-primary" style={{ width: `${analysis.progress.percent}%` }} />
                      </div>
                    </TableCell>
                    <TableCell className="hidden whitespace-nowrap text-muted-foreground sm:table-cell">{formatDateTime(analysis.createdAt)}</TableCell>
                    <TableCell>
                      <Button variant="ghost" size="sm" asChild>
                        <Link to={`/analyses/${encodeURIComponent(analysis.id)}`}>جزئیات<ArrowLeft className="size-3.5" /></Link>
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            {showAll && totalPages > 1 && (
              <div className="mt-5 flex items-center justify-between gap-3 border-t pt-4 text-xs text-muted-foreground">
                <span>صفحهٔ {analyses.data.page} از {totalPages} — {analyses.data.totalItems} تحلیل</span>
                <div className="flex gap-2">
                  <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((current) => current - 1)}>
                    <ArrowRight className="size-3.5" /> قبلی
                  </Button>
                  <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage((current) => current + 1)}>
                    بعدی <ArrowLeft className="size-3.5" />
                  </Button>
                </div>
              </div>
            )}
          </>
        )}
      </CardContent>
    </Card>
  );
}
